// Package vulnintel enriches already-detected CVEs using public exploitation signals.
// It never discovers asset exposure and never sends source code, tokens or package data.
package vulnintel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	CISAFeedURL = "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json"
	FIRSTAPIURL = "https://api.first.org/data/v1/epss"
)

var cvePattern = regexp.MustCompile(`(?i)\bCVE-[0-9]{4}-[0-9]{4,}\b`)

// ExtractCVE returns a canonical CVE identifier, not scanner-controlled URL content.
func ExtractCVE(values ...string) string {
	for _, value := range values {
		if found := cvePattern.FindString(value); found != "" {
			return strings.ToUpper(found)
		}
	}
	return ""
}

type KEVEntry struct {
	CVEID          string `json:"cveID"`
	DateAdded      string `json:"dateAdded"`
	DueDate        string `json:"dueDate"`
	RequiredAction string `json:"requiredAction"`
}
type Result struct {
	CVE            string    `json:"cve"`
	KEVKnown       bool      `json:"kev_known"`
	KEVChecked     bool      `json:"kev_checked"`
	KEVDateAdded   string    `json:"kev_date_added,omitempty"`
	KEVDueDate     string    `json:"kev_due_date,omitempty"`
	EPSS           *float64  `json:"epss,omitempty"`
	EPSSPercentile *float64  `json:"epss_percentile,omitempty"`
	EPSSChecked    bool      `json:"epss_checked"`
	Priority       string    `json:"priority"`
	Message        string    `json:"message"`
	Warnings       []string  `json:"warnings,omitempty"`
	RetrievedAt    time.Time `json:"retrieved_at"`
}

type cachedEPSS struct {
	score      float64
	percentile float64
	found      bool
	until      time.Time
}

// Client holds bounded TTL caches. Every request uses fixed official upstream hosts
// unless explicit test-only URLs are injected by the caller.
type Client struct {
	httpClient *http.Client
	kevURL     string
	epssURL    string
	mu         sync.Mutex
	kev        map[string]KEVEntry
	kevUntil   time.Time
	epss       map[string]cachedEPSS
	now        func() time.Time
}

func NewClient() *Client {
	return NewClientWithSources(&http.Client{Timeout: 5 * time.Second}, CISAFeedURL, FIRSTAPIURL)
}

// NewClientWithSources allows httptest injection; production uses NewClient.
func NewClientWithSources(httpClient *http.Client, kevURL, epssURL string) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &Client{httpClient: httpClient, kevURL: kevURL, epssURL: epssURL,
		epss: make(map[string]cachedEPSS), now: time.Now}
}

func (c *Client) Lookup(ctx context.Context, cve string) (Result, error) {
	id := ExtractCVE(cve)
	if id == "" || id != strings.ToUpper(strings.TrimSpace(cve)) {
		return Result{}, errors.New("invalid CVE identifier")
	}
	out := Result{CVE: id, RetrievedAt: c.now().UTC()}
	kev, kevErr := c.lookupKEV(ctx, id)
	if kevErr == nil {
		out.KEVChecked = true
		if kev.CVEID != "" {
			out.KEVKnown = true
			out.KEVDateAdded = kev.DateAdded
			out.KEVDueDate = kev.DueDate
		}
	} else {
		out.Warnings = append(out.Warnings, "CISA KEV lookup unavailable: "+kevErr.Error())
	}
	score, percentile, found, epssErr := c.lookupEPSS(ctx, id)
	if epssErr == nil {
		out.EPSSChecked = true
		if found {
			out.EPSS, out.EPSSPercentile = &score, &percentile
		}
	} else {
		out.Warnings = append(out.Warnings, "FIRST EPSS lookup unavailable: "+epssErr.Error())
	}
	switch {
	case out.KEVKnown:
		out.Priority = "Known exploited — investigate urgently"
		out.Message = "CISA lists active exploitation. Confirm the affected version and asset exposure; apply the vendor fix or mitigation promptly."
	case out.EPSS != nil && *out.EPSS >= 0.10:
		out.Priority = "Elevated exploitation likelihood"
		out.Message = "EPSS indicates elevated 30-day exploitation probability. Verify reachability and patch availability."
	case !out.KEVChecked && !out.EPSSChecked:
		out.Priority = "Unknown — intelligence unavailable"
		out.Message = "Unable to obtain current external signals. Do not interpret this as low risk."
	default:
		out.Priority = "Review deployment exposure"
		out.Message = "No positive KEV match in available data. This does not establish safety; prioritize using exposure, impact, and the scanner evidence."
	}
	return out, nil
}

func (c *Client) fetchJSON(ctx context.Context, endpoint string, limit int64, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upstream request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upstream HTTP %d", resp.StatusCode)
	}
	dec := json.NewDecoder(io.LimitReader(resp.Body, limit+1))
	if err := dec.Decode(target); err != nil {
		return fmt.Errorf("invalid upstream JSON")
	}
	// LimitReader prevents excessive bodies even for hostile or accidental redirects.
	// json.Decoder can buffer more than consumed; verify its observed input offset.
	if dec.InputOffset() > limit {
		return errors.New("upstream response too large")
	}
	return nil
}

func (c *Client) lookupKEV(ctx context.Context, cve string) (KEVEntry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.kev != nil && c.now().Before(c.kevUntil) {
		return c.kev[cve], nil
	}
	var feed struct {
		Vulnerabilities []KEVEntry `json:"vulnerabilities"`
	}
	if err := c.fetchJSON(ctx, c.kevURL, 12<<20, &feed); err != nil {
		return KEVEntry{}, err
	}
	if len(feed.Vulnerabilities) == 0 {
		return KEVEntry{}, errors.New("CISA feed has no vulnerability records")
	}
	items := make(map[string]KEVEntry, len(feed.Vulnerabilities))
	for _, row := range feed.Vulnerabilities {
		id := ExtractCVE(row.CVEID)
		if id == strings.ToUpper(row.CVEID) && id != "" {
			items[id] = row
		}
	}
	c.kev, c.kevUntil = items, c.now().Add(6*time.Hour)
	return items[cve], nil
}

func (c *Client) lookupEPSS(ctx context.Context, cve string) (float64, float64, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry, ok := c.epss[cve]; ok && c.now().Before(entry.until) {
		return entry.score, entry.percentile, entry.found, nil
	}
	endpoint, err := url.Parse(c.epssURL)
	if err != nil {
		return 0, 0, false, err
	}
	q := endpoint.Query()
	q.Set("cve", cve)
	endpoint.RawQuery = q.Encode()
	var response struct {
		Data []struct {
			CVE        string `json:"cve"`
			Score      string `json:"epss"`
			Percentile string `json:"percentile"`
		} `json:"data"`
	}
	if err := c.fetchJSON(ctx, endpoint.String(), 256<<10, &response); err != nil {
		return 0, 0, false, err
	}
	entry := cachedEPSS{until: c.now().Add(time.Hour)}
	for _, row := range response.Data {
		if row.CVE != cve {
			continue
		}
		score, serr := strconv.ParseFloat(row.Score, 64)
		perc, perr := strconv.ParseFloat(row.Percentile, 64)
		if serr != nil || perr != nil || !(score >= 0 && score <= 1 && perc >= 0 && perc <= 1) {
			return 0, 0, false, errors.New("invalid EPSS probability")
		}
		entry.score, entry.percentile, entry.found = score, perc, true
		break
	}
	c.epss[cve] = entry
	return entry.score, entry.percentile, entry.found, nil
}

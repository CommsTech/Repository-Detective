package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// standardCronParser accepts 5-field cron: minute hour day month weekday.
var standardCronParser = cron.NewParser(
	cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
)

// ValidateCronExpression checks a standard 5-field cron expression.
func ValidateCronExpression(expr string) error {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return fmt.Errorf("empty cron expression")
	}
	if len(expr) > 128 {
		return fmt.Errorf("cron expression too long (max 128 chars)")
	}
	if _, err := standardCronParser.Parse(expr); err != nil {
		return fmt.Errorf("invalid cron: %w", err)
	}
	return nil
}

// ParseCronSchedule parses a cron expression into a schedule.
func ParseCronSchedule(expr string) (cron.Schedule, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty cron expression")
	}
	schedule, err := standardCronParser.Parse(expr)
	if err != nil {
		return nil, err
	}
	return schedule, nil
}

// NextCronRun returns the next run time strictly after after.
func NextCronRun(expr string, after time.Time) (time.Time, error) {
	schedule, err := ParseCronSchedule(expr)
	if err != nil {
		return time.Time{}, err
	}
	next := schedule.Next(after)
	if next.IsZero() {
		return time.Time{}, fmt.Errorf("no next run for cron %q", expr)
	}
	return next, nil
}

// CronDescription returns a human-readable validation result for UI/API.
type CronDescription struct {
	Valid   bool   `json:"valid"`
	Error   string `json:"error,omitempty"`
	NextRun string `json:"next_run,omitempty"`
}

// DescribeCron validates expr and optionally computes the next run from now.
func DescribeCron(expr string, baseline time.Time) CronDescription {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return CronDescription{Valid: false, Error: "empty cron expression"}
	}
	if err := ValidateCronExpression(expr); err != nil {
		return CronDescription{Valid: false, Error: err.Error()}
	}
	if baseline.IsZero() {
		baseline = time.Now().UTC()
	}
	next, err := NextCronRun(expr, baseline)
	if err != nil {
		return CronDescription{Valid: false, Error: err.Error()}
	}
	return CronDescription{Valid: true, NextRun: next.UTC().Format(time.RFC3339)}
}

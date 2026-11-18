(function () {
  "use strict";

  function debounce(fn, ms) {
    var t;
    return function () {
      var ctx = this;
      var args = arguments;
      clearTimeout(t);
      t = setTimeout(function () { fn.apply(ctx, args); }, ms);
    };
  }

  // On narrow screens collapse the large operations navigation without
  // making the page unusable when JavaScript is disabled.
  function initMobileNav() {
    var button = document.querySelector("[data-mobile-nav-toggle]");
    var sidebar = document.querySelector(".rd-sidebar");
    var nav = document.getElementById("rd-primary-nav");
    if (!button || !sidebar || !nav) return;
    document.documentElement.classList.add("rd-js");
    function setOpen(open) {
      sidebar.classList.toggle("rd-menu-open", open);
      button.setAttribute("aria-expanded", String(open));
      button.querySelector("[aria-hidden]").textContent = open ? "✕" : "☰";
    }
    button.addEventListener("click", function () {
      setOpen(!sidebar.classList.contains("rd-menu-open"));
    });
    nav.addEventListener("click", function (event) {
      if (event.target.closest("a") && window.matchMedia("(max-width: 768px)").matches) {
        setOpen(false);
      }
    });
    document.addEventListener("keydown", function (event) {
      if (event.key === "Escape" && sidebar.classList.contains("rd-menu-open")) {
        setOpen(false);
        button.focus();
      }
    });
    window.addEventListener("resize", function () {
      if (!window.matchMedia("(max-width: 768px)").matches) setOpen(false);
    });
  }

  function initTableSearch() {
    var input = document.getElementById("rd-table-search");
    var container = document.getElementById("rd-searchable-table");
    if (!input || !container) return;
    var rows = container.querySelectorAll(".rd-fleet-row[data-search], tbody tr[data-search]");
    var filter = debounce(function () {
      var q = input.value.toLowerCase().trim();
      rows.forEach(function (row) {
        var text = row.getAttribute("data-search") || "";
        row.style.display = !q || text.indexOf(q) !== -1 ? "" : "none";
      });
    }, 180);
    input.addEventListener("input", filter);
  }

  function initActionMenus() {
    document.querySelectorAll(".rd-action-menu").forEach(function (menu) {
      var trigger = menu.querySelector(".rd-menu-trigger");
      var panel = menu.querySelector(".rd-action-menu-panel");
      if (!trigger || !panel) return;

      trigger.addEventListener("click", function (e) {
        e.stopPropagation();
        var open = panel.hidden;
        document.querySelectorAll(".rd-action-menu-panel").forEach(function (p) {
          p.hidden = true;
        });
        document.querySelectorAll(".rd-menu-trigger").forEach(function (t) {
          t.setAttribute("aria-expanded", "false");
        });
        if (open) {
          panel.hidden = false;
          trigger.setAttribute("aria-expanded", "true");
        }
      });

      panel.addEventListener("click", function (e) {
        e.stopPropagation();
      });
    });

    document.addEventListener("click", function () {
      document.querySelectorAll(".rd-action-menu-panel").forEach(function (p) {
        p.hidden = true;
      });
      document.querySelectorAll(".rd-menu-trigger").forEach(function (t) {
        t.setAttribute("aria-expanded", "false");
      });
    });

    document.addEventListener("keydown", function (e) {
      if (e.key === "Escape") {
        document.querySelectorAll(".rd-action-menu-panel").forEach(function (p) {
          p.hidden = true;
        });
        document.querySelectorAll(".rd-menu-trigger").forEach(function (t) {
          t.setAttribute("aria-expanded", "false");
        });
      }
    });
  }

  function initConfirmForms() {
    document.querySelectorAll("form[data-confirm]").forEach(function (form) {
      if (form.getAttribute("data-scan-form") !== null) {
        return;
      }
      form.addEventListener("submit", function (e) {
        var msg = form.getAttribute("data-confirm");
        if (msg && !window.confirm(msg)) {
          e.preventDefault();
        }
      });
    });
  }

  function initRiskWarnings() {
    document.querySelectorAll("[data-risk-warn]").forEach(function (el) {
      el.addEventListener("change", function () {
        var warn = el.closest("label") && el.closest("label").querySelector(".rd-risk-hint");
        if (!warn) return;
        var risky = el.value === "false" || el.value === "monitor_only";
        warn.hidden = !risky;
      });
    });
  }

  function openModal(modal) {
    if (!modal) return;
    modal.hidden = false;
    modal.setAttribute("aria-hidden", "false");
    document.body.classList.add("rd-modal-open");
    var focusTarget = modal.querySelector("input[name=ref], button, [href]");
    if (focusTarget && focusTarget.focus) {
      focusTarget.focus();
    }
  }

  function closeModal(modal) {
    if (!modal) return;
    modal.hidden = true;
    modal.setAttribute("aria-hidden", "true");
    document.body.classList.remove("rd-modal-open");
    var result = modal.querySelector("[data-scan-result]");
    if (result) {
      result.hidden = true;
      result.textContent = "";
      result.className = "rd-scan-result";
    }
  }

  function updateScanPreflight(form, issueFiling, dryRunChecked) {
    var issuesLine = form.querySelector("#rd-repos-scan-issues-line");
    var dryRunLine = form.querySelector("#rd-scan-dry-run-line");
    var filingNote = form.querySelector("#rd-repos-scan-filing-note");
    var willFile = issueFiling && !dryRunChecked;

    if (filingNote) {
      filingNote.hidden = issueFiling;
    }
    if (issuesLine) {
      if (willFile) {
        issuesLine.className = "rd-warn";
        issuesLine.textContent =
          "This scan will file or update Gitea issues for eligible findings (duplicates prevented by fingerprint).";
      } else if (!issueFiling) {
        issuesLine.className = "rd-safe";
        issuesLine.textContent =
          "Issue filing is disabled by repo/global policy — this scan will not create or update forge issues.";
      } else {
        issuesLine.className = "rd-safe";
        issuesLine.textContent =
          "This scan is report-only and will not file or update Gitea issues.";
      }
    }
    if (dryRunLine) {
      if (!issueFiling) {
        dryRunLine.textContent = "Report-only enforced — issue filing is off in policy.";
      } else if (dryRunChecked) {
        dryRunLine.textContent =
          "Dry run selected — findings persist, reports generate, issue_sync_status will be skipped.";
      } else {
        dryRunLine.textContent =
          "Dry run not selected — issue filing follows repo policy when eligible.";
      }
    }
  }

  function wireScanPreflightToggle(form, issueFiling) {
    if (!form) return;
    var reportEl = form.querySelector("[data-preflight-toggle]");
    if (!reportEl) return;
    reportEl.addEventListener("change", function () {
      updateScanPreflight(form, issueFiling, reportEl.checked);
    });
  }

  function setEnabledLine(el, label, enabled) {
    if (!el) return;
    if (enabled) {
      el.className = "rd-safe";
      el.innerHTML = label + " is enabled.";
    } else {
      el.className = "rd-safe";
      el.innerHTML = label + " is <strong>disabled</strong>.";
    }
  }

  function updateProfileAILines(form) {
    if (!form) return;
    var select = form.querySelector("#rd-repos-scan-profile");
    if (!select) return;
    var gateLine = form.querySelector("#rd-scan-llm-gate-line");
    var aiLine = form.querySelector("#rd-scan-ai-auditors-line");
    var value = select.value || "";
    var gateOn = false;
    var aiOn = false;
    if (!value) {
      gateOn = select.getAttribute("data-inherit-llm-gate") === "true";
      aiOn = select.getAttribute("data-inherit-ai") === "true";
    } else {
      var opt = select.options[select.selectedIndex];
      gateOn = opt && opt.getAttribute("data-llm-gate") === "true";
      // Deep is the only preset with AI on; Custom follows inherit/platform.
      if (value === "deep") {
        aiOn = true;
      } else if (value === "custom") {
        aiOn = select.getAttribute("data-inherit-ai") === "true";
      } else {
        aiOn = false;
      }
    }
    setEnabledLine(gateLine, "LLM sanity gate", gateOn);
    setEnabledLine(aiLine, "AI auditors", aiOn);
  }

  function wireScanProfileAILines(form) {
    if (!form) return;
    var select = form.querySelector("#rd-repos-scan-profile");
    if (!select || select.getAttribute("data-ai-wired") === "1") return;
    select.setAttribute("data-ai-wired", "1");
    select.addEventListener("change", function () {
      updateProfileAILines(form);
    });
    updateProfileAILines(form);
  }

  function wireScanFormModal(modal) {
    if (!modal) return;
    modal.querySelectorAll("[data-scan-cancel]").forEach(function (el) {
      el.addEventListener("click", function () {
        closeModal(modal);
      });
    });

    var form = modal.querySelector("[data-scan-form]");
    if (!form) return;

    form.addEventListener("submit", function (e) {
      e.preventDefault();
      var msg = form.getAttribute("data-confirm");
      if (msg && !window.confirm(msg)) {
        return;
      }

      var resultEl = form.querySelector("[data-scan-result]");
      var submitBtn = form.querySelector('button[type="submit"]');
      if (submitBtn) {
        submitBtn.disabled = true;
      }

      var body = new FormData(form);
      fetch(form.action, {
        method: "POST",
        body: body,
        headers: {
          Accept: "application/json",
          "X-Requested-With": "XMLHttpRequest"
        },
        credentials: "same-origin"
      })
        .then(function (res) {
          return res.json().then(function (data) {
            if (!res.ok) {
              throw new Error((data && data.error) || "Failed to start scan");
            }
            return data;
          });
        })
        .then(function (data) {
          var dest = data.redirect || data.scan_url;
          if (dest) {
            window.location.assign(dest);
            return;
          }
          if (!resultEl) return;
          resultEl.hidden = false;
          resultEl.className = "rd-scan-result success";
          resultEl.innerHTML =
            "<strong>Scan started</strong><br>" +
            "Scan ID: <code>" + (data.scan_id || "—") + "</code>";
        })
        .catch(function (err) {
          if (!resultEl) return;
          resultEl.hidden = false;
          resultEl.className = "rd-scan-result error";
          resultEl.textContent = err.message || "Failed to start scan";
        })
        .finally(function () {
          if (submitBtn) {
            submitBtn.disabled = false;
          }
        });
    });
  }

  function initScanNowModal() {
    var detailModal = document.getElementById("rd-scan-now-modal");
    var listModal = document.getElementById("rd-repos-scan-modal");
    var openBtn = document.getElementById("rd-scan-now-open");

    if (openBtn && detailModal) {
      openBtn.addEventListener("click", function () {
        openModal(detailModal);
      });
    }

    wireScanFormModal(detailModal);
    wireScanFormModal(listModal);

    document.querySelectorAll("[data-scan-form]").forEach(function (form) {
      var issueFiling = !form.querySelector("#rd-repos-scan-report-only[disabled]");
      wireScanPreflightToggle(form, issueFiling);
      wireScanProfileAILines(form);
    });

    document.querySelectorAll("[data-scan-open]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        if (!listModal) return;
        var repoID = btn.getAttribute("data-repo-id");
        var repoName = btn.getAttribute("data-repo-name") || "";
        var ref = btn.getAttribute("data-default-ref") || "main";
        var profile = btn.getAttribute("data-scan-profile") || "";
        var issueFiling = btn.getAttribute("data-issue-filing") === "true";
        var reportOnly = btn.getAttribute("data-report-only") !== "false";
        var form = document.getElementById("rd-repos-scan-form");
        if (!form || !repoID) return;

        var basePath = form.getAttribute("data-base-path") || "";
        var qs = window.location.search || "";
        form.action = basePath + "/repos/" + repoID + "/scan" + qs;
        form.setAttribute("data-confirm", "Start manual scan for " + repoName + "?");

        var nameEl = document.getElementById("rd-repos-scan-name");
        var idEl = document.getElementById("rd-repos-scan-id");
        var refEl = document.getElementById("rd-repos-scan-ref");
        var profileEl = document.getElementById("rd-repos-scan-profile");
        var reportEl = document.getElementById("rd-repos-scan-report-only");
        var filingNote = document.getElementById("rd-repos-scan-filing-note");
        var issuesLine = document.getElementById("rd-repos-scan-issues-line");

        if (nameEl) nameEl.textContent = repoName;
        if (idEl) idEl.textContent = repoID;
        if (refEl) refEl.value = ref;
        if (profileEl) {
          profileEl.value = profile;
          // Fleet rows carry the repo's resolved profile for inherit display.
          if (profile) {
            profileEl.setAttribute("data-inherit-profile", profile);
            profileEl.setAttribute("data-inherit-llm-gate", profile === "deep" ? "true" : "false");
            profileEl.setAttribute("data-inherit-ai", profile === "deep" ? "true" : "false");
          }
        }
        if (reportEl) {
          reportEl.checked = !issueFiling || reportOnly;
          reportEl.disabled = !issueFiling;
        }
        updateScanPreflight(form, issueFiling, reportEl ? reportEl.checked : true);
        wireScanPreflightToggle(form, issueFiling);
        wireScanProfileAILines(form);
        updateProfileAILines(form);
        openModal(listModal);
      });
    });

    document.addEventListener("keydown", function (e) {
      if (e.key !== "Escape") return;
      [detailModal, listModal].forEach(function (m) {
        if (m && !m.hidden) closeModal(m);
      });
    });
  }

  function softRefreshScanPage() {
    // Avoid full document reload — in dark mode location.reload() flashes bright white.
    return fetch(window.location.href, {
      credentials: "same-origin",
      headers: { Accept: "text/html", "X-Requested-With": "RDScanRefresh" },
      cache: "no-store"
    })
      .then(function (resp) {
        if (!resp.ok) throw new Error("refresh status " + resp.status);
        return resp.text();
      })
      .then(function (html) {
        var doc = new DOMParser().parseFromString(html, "text/html");
        var nextMain = doc.getElementById("main-content");
        var curMain = document.getElementById("main-content");
        if (!nextMain || !curMain) {
          throw new Error("scan refresh response was incomplete");
        }
        // Preserve the operator's location and open diagnostic disclosures on
        // periodic updates. Replacing the whole page would discard their work.
        var scrollLeft = window.scrollX;
        var scrollTop = window.scrollY;
        var expanded = Array.prototype.slice.call(curMain.querySelectorAll('details[open][id]')).map(function (el) { return el.id; });
        var previousFocus = document.activeElement;
        var focusedID = previousFocus && curMain.contains(previousFocus) ? previousFocus.id : '';
        curMain.replaceWith(document.importNode(nextMain, true));
        var refreshedMain = document.getElementById('main-content');
        expanded.forEach(function (id) {
          var el = refreshedMain && refreshedMain.querySelector('details[id="' + CSS.escape(id) + '"]');
          if (el) el.open = true;
        });
        if (focusedID) {
          var focusTarget = document.getElementById(focusedID);
          if (focusTarget && refreshedMain.contains(focusTarget)) focusTarget.focus({preventScroll: true});
        }
        window.scrollTo(scrollLeft, scrollTop);
        if (doc.title) {
          document.title = doc.title;
        }
        initScanAutoRefresh();
        initScanEvidenceSummary();
      })
      .catch(function () {
        // Never fall back to a full reload here: the white browser navigation
        // frame can be painful for dark-mode operators. Keep last known state.
        var banner = document.querySelector("[data-scan-auto-refresh]");
        var label = document.querySelector("[data-scan-refresh-label]");
        if (label) {
          label.textContent = "Update unavailable. Retrying shortly; last displayed data is unchanged.";
        }
        if (banner) {
          banner.removeAttribute("data-scan-refresh-armed");
          setTimeout(function () {
            if (document.contains(banner)) initScanAutoRefresh();
          }, 10000);
        }
      });
  }

  // A lightweight execution-outcome bar. It is deliberately NOT a code coverage
  // percentage or a claim that the scanned repository is safe.
  function initScanEvidenceSummary() {
    var panel = document.querySelector('[data-scanner-coverage]');
    if (!panel) return;
    var rows = Array.prototype.slice.call(document.querySelectorAll('.rd-scanner-results [data-scanner-status]'));
    if (!rows.length) return;
    var reported = 0;
    rows.forEach(function (row) {
      var status = (row.getAttribute('data-scanner-status') || '').toLowerCase().trim();
      if (!row.hasAttribute('data-scanner-error') && ['found', 'clean', 'passed', 'ok', 'completed', 'success'].indexOf(status) !== -1) reported += 1;
    });
    var attention = rows.length - reported;
    var label = panel.querySelector('[data-scanner-coverage-label]');
    if (label) {
      label.replaceChildren();
      var ok = document.createElement('span');
      var okCount = document.createElement('strong');
      okCount.textContent = String(reported);
      ok.appendChild(okCount);
      ok.appendChild(document.createTextNode(' recorded complete'));
      var warn = document.createElement('span');
      warn.className = 'rd-attention';
      var warnCount = document.createElement('strong');
      warnCount.textContent = String(attention);
      warn.appendChild(warnCount);
      warn.appendChild(document.createTextNode(' need inspection'));
      var total = document.createElement('span');
      total.textContent = String(rows.length) + ' scanner/stage records';
      label.appendChild(ok);
      label.appendChild(warn);
      label.appendChild(total);
    }
    var okBar = panel.querySelector('[data-coverage-success]');
    var warnBar = panel.querySelector('[data-coverage-problem]');
    if (okBar) okBar.style.width = (100 * reported / rows.length).toFixed(2) + '%';
    if (warnBar) warnBar.style.width = (100 * attention / rows.length).toFixed(2) + '%';
    panel.hidden = false;
  }

  function initScanAutoRefresh() {
    var banner = document.querySelector("[data-scan-auto-refresh]");
    if (!banner) return;
    if (banner.getAttribute("data-scan-refresh-armed") === "1") return;
    banner.setAttribute("data-scan-refresh-armed", "1");
    var secs = parseInt(banner.getAttribute("data-scan-auto-refresh"), 10);
    if (!secs || secs < 1) secs = 5;
    var label = document.querySelector("[data-scan-refresh-label]");
    var left = secs;
    if (label) {
      label.textContent = "Refreshing in " + left + "s…";
    }
    var timer = setInterval(function () {
      left -= 1;
      if (left <= 0) {
        if (document.hidden) { left = secs; return; }
        clearInterval(timer);
        if (label) {
          label.textContent = "Updating…";
        }
        softRefreshScanPage();
        return;
      }
      if (label) {
        label.textContent = "Refreshing in " + left + "s…";
      }
    }, 1000);
  }

  function initCopyButtons() {
    document.querySelectorAll("[data-copy-target]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var id = btn.getAttribute("data-copy-target");
        var src = id ? document.getElementById(id) : null;
        if (!src) return;
        var text = src.value || src.textContent || "";
        var done = function () {
          var prev = btn.textContent;
          btn.textContent = "Copied";
          setTimeout(function () { btn.textContent = prev; }, 1400);
        };
        if (navigator.clipboard && navigator.clipboard.writeText) {
          navigator.clipboard.writeText(text).then(done).catch(function () {
            src.hidden = false;
            src.select();
            try { document.execCommand("copy"); done(); } catch (err) {
              btn.textContent = "Copy failed";
              setTimeout(function () { btn.textContent = prev; }, 2000);
            }
            src.hidden = true;
          });
          return;
        }
        src.hidden = false;
        src.select();
        try { document.execCommand("copy"); done(); } catch (e) {}
        src.hidden = true;
      });
    });
  }

  function initPrintButtons() {
    document.querySelectorAll("[data-rd-print]").forEach(function (btn) {
      btn.addEventListener("click", function (e) {
        e.preventDefault();
        window.print();
      });
    });
  }

  // Doctor page: CSP blocks inline scripts (script-src 'self'), so the Run
  // button must live here. Fetch the UI-auth report path so rd_ui_sess works.
  function initDoctorPage() {
    var out = document.getElementById("rd-doctor-out");
    var btn = document.getElementById("rd-doctor-run");
    if (!btn || !out) return;

    btn.addEventListener("click", function () {
      var url = btn.getAttribute("data-report-url") || "/ui/doctor/report";
      out.textContent = "Running…";
      btn.disabled = true;
      fetch(url, {
        headers: { Accept: "application/json" },
        credentials: "same-origin"
      })
        .then(function (r) {
          return r.text().then(function (text) {
            var payload = null;
            try {
              payload = text ? JSON.parse(text) : null;
            } catch (e) {
              payload = null;
            }
            return { ok: r.ok, status: r.status, payload: payload, text: text };
          });
        })
        .then(function (x) {
          if (!x.ok) {
            var errMsg = (x.payload && (x.payload.error || x.payload.message)) ||
              ("Doctor request failed (HTTP " + x.status + ")");
            out.textContent = errMsg;
            return;
          }
          var j = x.payload || {};
          var lines = [
            "Overall: " + (j.overall || ""),
            "Summary: " + (j.summary || ""),
            ""
          ];
          (j.checks || []).forEach(function (c) {
            lines.push("[" + c.state + "] " + c.id + " — " + c.summary);
            if (c.detail) {
              lines.push("    " + c.detail);
            }
          });
          lines.push("");
          lines.push("These results describe readiness, not whether repositories are safe or secure.");
          out.textContent = lines.join("\n");
        })
        .catch(function (e) {
          out.textContent = String(e && e.message ? e.message : e);
        })
        .finally(function () {
          btn.disabled = false;
        });
    });
  }

  // Local-only rule builder. No forge writes, eval, dynamic scripts, or remote calls.
  // The Go scanner remains the authority for RE2 syntax and resource limits.
  function initCustomRuleBuilder() {
    var root = document.querySelector("[data-rule-builder]");
    if (!root) return;
    var rules = [];
    var status = root.querySelector("[data-rule-status]");
    var preview = root.querySelector("[data-rule-json]");
    var list = root.querySelector("[data-rule-list]");
    var download = root.querySelector("[data-rule-download]");
    function value(name) { return (root.querySelector("[data-rule-" + name + "]")?.value || "").trim(); }
    function redraw() {
      var json = rules.length ? JSON.stringify({ version: 1, rules: rules }, null, 2) + "\n" : "";
      preview.value = json;
      download.disabled = !rules.length;
      list.replaceChildren();
      rules.forEach(function (rule, index) {
        var row = document.createElement("li");
        var summary = document.createElement("span");
        summary.textContent = rule.id + " · " + rule.title + " (" + rule.severity + ")";
        var remove = document.createElement("button");
        remove.type = "button";
        remove.className = "rd-btn rd-btn-sm rd-btn-ghost";
        remove.textContent = "Remove";
        remove.setAttribute("aria-label", "Remove " + rule.id);
        remove.addEventListener("click", function () {
          rules.splice(index, 1);
          redraw();
          status.textContent = "Rule removed.";
        });
        row.append(summary, remove);
        list.appendChild(row);
      });
    }
    root.querySelector("[data-rule-add]").addEventListener("click", function () {
      var id = value("id").toUpperCase();
      var title = value("title");
      var pattern = value("pattern");
      var path = value("path");
      if (!/^[A-Z][A-Z0-9_-]{2,47}$/.test(id)) {
        status.textContent = "Use a unique rule ID of 3–48 uppercase letters, digits, - or _, starting with a letter.";
        return;
      }
      if (rules.some(function (r) { return r.id === id; })) {
        status.textContent = "That rule ID already exists in this configuration.";
        return;
      }
      if (!title || title.length > 160 || !pattern || pattern.length > 320) {
        status.textContent = "Provide a short title and a pattern up to 320 characters.";
        return;
      }
      if (path.length > 200 || path.startsWith("/") || path.includes("\\") || path.includes("..")) {
        status.textContent = "Use a relative path filter without parent-directory references.";
        return;
      }
      if (rules.length >= 32) { status.textContent = "Maximum 32 rules per repository."; return; }
      var rule = { id: id, title: title, category: value("category"), severity: value("severity") };
      if (path) rule.path_glob = path;
      rule.pattern = pattern;
      rules.push(rule);
      redraw();
      if (new Blob([preview.value]).size > 64 * 1024) {
        rules.pop(); redraw();
        status.textContent = "Configuration exceeds the 64 KiB server limit.";
        return;
      }
      status.textContent = "Added " + id + ". Download the JSON, review it, then commit it to your repository.";
    });
    root.querySelector("[data-rule-clear]").addEventListener("click", function () {
      rules.length = 0;
      redraw();
      status.textContent = "Builder reset; existing repository rules remain unchanged.";
    });
    download.addEventListener("click", function () {
      if (!rules.length) return;
      var blob = new Blob([preview.value], { type: "application/json" });
      var url = URL.createObjectURL(blob);
      var a = document.createElement("a");
      a.href = url;
      a.download = ".repository-detective.rules.json";
      document.body.appendChild(a);
      a.click();
      a.remove();
      setTimeout(function () { URL.revokeObjectURL(url); }, 1000);
      status.textContent = "Configuration downloaded. Nothing was changed on the server.";
    });
    redraw();
  }

  // Purely client-side page switcher. Reuses already rendered, authorized
  // navigation links; never fetches content or makes up a new route.
  function initQuickNavigation() {
    var dialog = document.getElementById('rd-quicknav');
    var input = document.getElementById('rd-quicknav-search');
    var results = document.getElementById('rd-quicknav-results');
    var empty = document.getElementById('rd-quicknav-empty');
    var openButton = document.querySelector('[data-rd-quicknav-open]');
    if (!dialog || !input || !results || !empty || !openButton || !dialog.showModal) return;
    var lastFocus = null;
    var destinations = Array.prototype.map.call(
      document.querySelectorAll('#rd-primary-nav a[href]'),
      function (link) {
        return { label: link.textContent.trim().replace(/\s+/g, ' '), href: link.href };
      }
    );
    // Do not add anything not already present in the authenticated sidebar.
    function render() {
      var term = input.value.trim().toLowerCase();
      var count = 0;
      results.replaceChildren();
      destinations.forEach(function (item) {
        if (term && item.label.toLowerCase().indexOf(term) < 0) return;
        var link = document.createElement('a');
        link.href = item.href;
        link.textContent = item.label;
        link.className = 'rd-quicknav-result';
        results.appendChild(link);
        count++;
      });
      empty.hidden = count > 0;
    }
    function open() {
      if (dialog.open) return;
      lastFocus = document.activeElement;
      input.value = '';
      render();
      dialog.showModal();
      input.focus();
    }
    function close() { if (dialog.open) dialog.close(); }
    openButton.addEventListener('click', open);
    var closeButton = dialog.querySelector('[data-rd-quicknav-close]');
    if (closeButton) closeButton.addEventListener('click', close);
    dialog.addEventListener('keydown', function (event) {
      if (event.key === 'Escape') {
        event.preventDefault();
        event.stopPropagation();
        close();
      }
    });
    dialog.addEventListener('close', function () {
      if (lastFocus && lastFocus.isConnected && lastFocus.focus) lastFocus.focus();
    });
    dialog.addEventListener('click', function (event) {
      if (event.target === dialog) close();
    });
    input.addEventListener('input', render);
    input.addEventListener('keydown', function (event) {
      if (event.key === 'Enter') {
        var first = results.querySelector('a[href]');
        if (first) { event.preventDefault(); window.location.assign(first.href); }
      } else if (event.key === 'ArrowDown') {
        var result = results.querySelector('a[href]');
        if (result) { event.preventDefault(); result.focus(); }
      }
    });
    results.addEventListener('keydown', function (event) {
      if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
        var links = Array.prototype.slice.call(results.querySelectorAll('a[href]'));
        var current = links.indexOf(document.activeElement);
        if (current < 0 || !links.length) return;
        var index = event.key === 'ArrowDown' ? Math.min(current + 1, links.length - 1) : current - 1;
        event.preventDefault();
        if (index < 0) input.focus(); else links[index].focus();
      }
    });
    document.addEventListener('keydown', function (event) {
      if (event.defaultPrevented || event.isComposing) return;
      var target = event.target;
      var tag = target && target.tagName;
      var editing = target && (target.isContentEditable || tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT');
      if ((event.ctrlKey || event.metaKey) && !event.altKey && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        if (dialog.open) close(); else open();
      } else if (event.key === '/' && !editing && !event.ctrlKey && !event.metaKey && !event.altKey && !dialog.open) {
        event.preventDefault(); open();
      }
    });
  }

  document.addEventListener("DOMContentLoaded", function () {
    initMobileNav();
    initQuickNavigation();
    initTableSearch();
    initConfirmForms();
    initRiskWarnings();
    initActionMenus();
    initScanNowModal();
    initScanAutoRefresh();
    initScanEvidenceSummary();
    initCopyButtons();
    initPrintButtons();
    initDoctorPage();
    initCustomRuleBuilder();
  });
})();

// Optional CVE intelligence: explicit operator action, never an automatic page-load request.
// All server-supplied strings enter textContent, not innerHTML.
document.addEventListener("click", async function (event) {
  const btn = event.target.closest("[data-cve-intelligence]");
  if (!btn || btn.disabled) return;
  const result = btn.closest(".rd-intel-panel")?.querySelector("[data-cve-intelligence-result]");
  if (!result) return;
  const endpoint = btn.getAttribute("data-intel-url");
  if (!endpoint) return;
  btn.disabled = true;
  result.textContent = "Checking CISA and FIRST...";
  try {
    const resp = await fetch(endpoint, { credentials: "same-origin", headers: { "Accept": "application/json" } });
    const data = await resp.json();
    if (!resp.ok) throw new Error(data.error || "Intelligence lookup failed");
    const kev = data.kev_checked ? (data.kev_known ? "Listed in CISA KEV" : "Not listed in current CISA KEV") : "CISA status unavailable";
    const epss = data.epss_checked ? (typeof data.epss === "number" ? `EPSS ${(data.epss * 100).toFixed(1)}%` : "EPSS score not available") : "EPSS status unavailable";
    const warning = (data.warnings || []).length ? " Some sources unavailable; do not treat missing data as low risk." : "";
    result.textContent = `${data.cve}: ${data.priority}. ${kev}${data.kev_date_added ? " (added " + data.kev_date_added + ")" : ""}; ${epss}. ${data.message}${warning}`;
  } catch (err) {
    result.textContent = `Intelligence unavailable: ${err.message}. Continue triage using scanner evidence and exposure.`;
  } finally {
    btn.disabled = false;
  }
});

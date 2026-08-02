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
        if (profileEl) profileEl.value = profile;
        if (reportEl) {
          reportEl.checked = !issueFiling || reportOnly;
          reportEl.disabled = !issueFiling;
        }
        updateScanPreflight(form, issueFiling, reportEl ? reportEl.checked : true);
        wireScanPreflightToggle(form, issueFiling);
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

  function initScanAutoRefresh() {
    var banner = document.querySelector("[data-scan-auto-refresh]");
    if (!banner) return;
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
        clearInterval(timer);
        window.location.reload();
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
            try { document.execCommand("copy"); done(); } catch (e) {}
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

  document.addEventListener("DOMContentLoaded", function () {
    initTableSearch();
    initConfirmForms();
    initRiskWarnings();
    initActionMenus();
    initScanNowModal();
    initScanAutoRefresh();
    initCopyButtons();
    initPrintButtons();
  });
})();

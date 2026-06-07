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
    var table = document.getElementById("rd-searchable-table");
    if (!input || !table) return;
    var rows = table.querySelectorAll("tbody tr[data-search]");
    var filter = debounce(function () {
      var q = input.value.toLowerCase().trim();
      rows.forEach(function (row) {
        var text = row.getAttribute("data-search") || "";
        row.style.display = !q || text.indexOf(q) !== -1 ? "" : "none";
      });
    }, 180);
    input.addEventListener("input", filter);
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

  function initScanNowModal() {
    var modal = document.getElementById("rd-scan-now-modal");
    var openBtn = document.getElementById("rd-scan-now-open");
    if (!modal) return;

    if (openBtn) {
      openBtn.addEventListener("click", function () {
        openModal(modal);
      });
    }

    modal.querySelectorAll("[data-scan-cancel]").forEach(function (el) {
      el.addEventListener("click", function () {
        closeModal(modal);
      });
    });

    document.addEventListener("keydown", function (e) {
      if (e.key === "Escape" && !modal.hidden) {
        closeModal(modal);
      }
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
          if (!resultEl) return;
          resultEl.hidden = false;
          resultEl.className = "rd-scan-result success";
          var scanLink = data.scan_url
            ? '<a class="rd-link" href="' + data.scan_url + '">View scan status →</a>'
            : "";
          resultEl.innerHTML =
            "<strong>Scan started</strong><br>" +
            "Scan ID: <code>" + (data.scan_id || "—") + "</code><br>" +
            "Trigger: <code>" + (data.trigger_type || "manual") + "</code><br>" +
            "Report-only: <strong>" + (data.report_only_dry_run ? "yes" : "no") + "</strong><br>" +
            scanLink;
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

  document.addEventListener("DOMContentLoaded", function () {
    initTableSearch();
    initConfirmForms();
    initRiskWarnings();
    initScanNowModal();
  });
})();

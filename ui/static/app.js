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

  document.addEventListener("DOMContentLoaded", function () {
    initTableSearch();
    initConfirmForms();
    initRiskWarnings();
  });
})();

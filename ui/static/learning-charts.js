(function () {
  "use strict";

  var chartHandles = [];

  function cssVar(name, fallback) {
    var v = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
    return v || fallback;
  }

  function readThemePalette() {
    return {
      teal: "#0ea5a4",
      blue: "#2563eb",
      orange: "#f97316",
      red: "#ef4444",
      amber: "#f59e0b",
      violet: "#8b5cf6",
      slate: "#64748b",
      grid: cssVar("--rd-border", "rgba(148, 163, 184, 0.12)"),
      text: cssVar("--rd-text-dim", "#94a3b8"),
      card: cssVar("--rd-card", "#111827"),
    };
  }

  var palette = readThemePalette();
  var eventColors = [];

  function refreshEventColors() {
    eventColors = [
      palette.teal, palette.blue, palette.orange, palette.violet,
      palette.amber, palette.red, palette.slate, "#14b8a6"
    ];
  }

  function readPayload() {
    var el = document.getElementById("rd-learning-data");
    if (!el || !el.textContent) return null;
    try {
      var data = JSON.parse(el.textContent.trim());
      if (typeof data === "string") data = JSON.parse(data);
      return data;
    } catch (e) {
      console.warn("Repository Detective: invalid learning chart data", e);
      return null;
    }
  }

  function hasValues(values) {
    if (!values || !values.length) return false;
    for (var i = 0; i < values.length; i++) {
      if (Number(values[i]) > 0) return true;
    }
    return false;
  }

  function destroyCharts() {
    chartHandles.forEach(function (c) {
      try { c.destroy(); } catch (e) { /* ignore */ }
    });
    chartHandles = [];
  }

  function showEmpty(canvas, message) {
    if (!canvas || !canvas.parentNode) return;
    var wrap = canvas.parentNode;
    canvas.style.display = "none";
    if (wrap.querySelector(".rd-chart-empty")) return;
    var note = document.createElement("p");
    note.className = "rd-muted rd-chart-empty";
    note.textContent = message;
    wrap.appendChild(note);
  }

  function clearEmptyNotes() {
    document.querySelectorAll(".rd-learn-chart-wrap .rd-chart-empty").forEach(function (el) { el.remove(); });
    document.querySelectorAll(".rd-learn-chart-wrap canvas").forEach(function (c) { c.style.display = ""; });
  }

  function showError(message) {
    var section = document.querySelector(".rd-learn-charts");
    if (!section || section.querySelector(".rd-chart-error")) return;
    var alert = document.createElement("div");
    alert.className = "rd-alert rd-chart-error";
    alert.setAttribute("role", "alert");
    alert.textContent = message;
    section.insertBefore(alert, section.firstChild);
  }

  function initEventsChart(canvas, data) {
    if (!hasValues(data.eventValues)) {
      showEmpty(canvas, "No learning events recorded yet.");
      return;
    }
    chartHandles.push(new Chart(canvas, {
      type: "doughnut",
      data: {
        labels: data.eventLabels,
        datasets: [{
          data: data.eventValues,
          backgroundColor: eventColors.slice(0, data.eventLabels.length),
          borderColor: palette.card,
          borderWidth: 2,
          hoverOffset: 6,
        }],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        cutout: "58%",
        plugins: {
          legend: {
            position: "right",
            labels: { color: palette.text, boxWidth: 12, font: { size: 11 } },
          },
        },
      },
    }));
  }

  function initNoisyChart(canvas, data) {
    if (!data.noisyRuleLabels || !data.noisyRuleLabels.length) {
      showEmpty(canvas, "No noisy rule stats yet — complete more scans.");
      return;
    }
    chartHandles.push(new Chart(canvas, {
      type: "bar",
      data: {
        labels: data.noisyRuleLabels,
        datasets: [{
          label: "False-positive rate %",
          data: data.noisyRuleFPRates,
          backgroundColor: data.noisyRuleFPRates.map(function (v) {
            if (v >= 40) return "rgba(239, 68, 68, 0.75)";
            if (v >= 20) return "rgba(249, 115, 22, 0.75)";
            return "rgba(14, 165, 164, 0.7)";
          }),
          borderRadius: 6,
        }],
      },
      options: {
        indexAxis: "y",
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { display: false },
          tooltip: {
            callbacks: {
              afterLabel: function (ctx) {
                var i = ctx.dataIndex;
                var n = data.noisyRuleFindings && data.noisyRuleFindings[i];
                return n != null ? n + " findings tracked" : "";
              },
            },
          },
        },
        scales: {
          x: {
            min: 0,
            max: 100,
            ticks: { color: palette.text, callback: function (v) { return v + "%"; } },
            grid: { color: palette.grid },
          },
          y: {
            ticks: { color: palette.text, font: { size: 10 } },
            grid: { display: false },
          },
        },
      },
    }));
  }

  function mount() {
    destroyCharts();
    clearEmptyNotes();
    var err = document.querySelector(".rd-learn-charts .rd-chart-error");
    if (err) err.remove();

    palette = readThemePalette();
    refreshEventColors();

    if (typeof Chart === "undefined") {
      showError("Charts could not load (Chart.js missing). Learning metrics below are still available.");
      return;
    }
    var data = readPayload();
    if (!data) {
      showError("Learning chart data is unavailable.");
      return;
    }
    var events = document.getElementById("rd-learn-events-chart");
    var noisy = document.getElementById("rd-learn-noisy-chart");
    if (events) initEventsChart(events, data);
    if (noisy) initNoisyChart(noisy, data);
  }

  if (document.readyState === "complete") {
    mount();
  } else {
    window.addEventListener("load", mount);
  }
  document.documentElement.addEventListener("rd-theme-change", mount);
})();

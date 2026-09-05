(function () {
  "use strict";

  var chartHandle = null;

  function cssColor(name, fallback) {
    var v = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
    return v || fallback;
  }

  function readPayload() {
    var el = document.getElementById("rd-report-chart-data");
    if (!el || !el.textContent) return null;
    try {
      return JSON.parse(el.textContent.trim());
    } catch (e) {
      return null;
    }
  }

  function initRadar(ctx, data) {
    if (!data.categoryLabels || !data.categoryLabels.length) {
      if (ctx.parentElement && !ctx.parentElement.querySelector(".rd-chart-empty")) {
        ctx.style.display = "none";
        var note = document.createElement("p");
        note.className = "rd-muted rd-chart-empty";
        note.textContent = "No category data for radar chart.";
        ctx.parentElement.appendChild(note);
      }
      return;
    }
    var max = Math.max.apply(null, data.categoryValues.concat([1]));
    var text = cssColor("--rd-text-dim", "#94a3b8");
    var grid = cssColor("--rd-border", "rgba(148, 163, 184, 0.12)");
    chartHandle = new Chart(ctx, {
      type: "radar",
      data: {
        labels: data.categoryLabels,
        datasets: [{
          label: "Open findings by category",
          data: data.categoryValues,
          backgroundColor: "rgba(37, 99, 235, 0.25)",
          borderColor: "#2563eb",
          pointBackgroundColor: "#0ea5a4",
          borderWidth: 2,
        }],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          r: {
            angleLines: { color: grid },
            grid: { color: grid },
            pointLabels: { color: text, font: { size: 10 } },
            ticks: { display: false },
            suggestedMin: 0,
            suggestedMax: max * 1.15,
          },
        },
        plugins: { legend: { display: false } },
      },
    });
  }

  function mount() {
    if (chartHandle) {
      try { chartHandle.destroy(); } catch (e) { /* ignore */ }
      chartHandle = null;
    }
    var empty = document.querySelector(".rd-chart-empty");
    if (empty) empty.remove();
    var radar = document.getElementById("rd-report-radar");
    if (radar) radar.style.display = "";
    if (typeof Chart === "undefined") return;
    var data = readPayload();
    if (radar && data) {
      initRadar(radar, data);
    }
  }

  if (document.readyState === "complete") {
    mount();
  } else {
    window.addEventListener("load", mount);
  }
  document.documentElement.addEventListener("rd-theme-change", mount);
})();

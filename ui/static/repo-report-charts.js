(function () {
  "use strict";

  var palette = {
    teal: "#0ea5a4",
    blue: "#2563eb",
    grid: "rgba(148, 163, 184, 0.12)",
    text: "#94a3b8",
  };

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
      ctx.parentElement.innerHTML = "<p class=\"rd-muted\">No category data for radar chart.</p>";
      return;
    }
    var max = Math.max.apply(null, data.categoryValues.concat([1]));
    palette.text = cssColor("--rd-text-dim", palette.text);
    palette.grid = cssColor("--rd-border", palette.grid);
    new Chart(ctx, {
      type: "radar",
      data: {
        labels: data.categoryLabels,
        datasets: [{
          label: "Open findings by category",
          data: data.categoryValues,
          backgroundColor: "rgba(37, 99, 235, 0.25)",
          borderColor: palette.blue,
          pointBackgroundColor: palette.teal,
          borderWidth: 2,
        }],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          r: {
            angleLines: { color: palette.grid },
            grid: { color: palette.grid },
            pointLabels: { color: palette.text, font: { size: 10 } },
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
    if (typeof Chart === "undefined") return;
    var data = readPayload();
    var radar = document.getElementById("rd-report-radar");
    if (radar && data) {
      initRadar(radar, data);
    }
  }

  if (document.readyState === "complete") {
    mount();
  } else {
    window.addEventListener("load", mount);
  }
})();

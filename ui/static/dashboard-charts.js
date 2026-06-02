(function () {
  "use strict";

  var palette = {
    critical: "#ef4444",
    high: "#f97316",
    medium: "#f59e0b",
    low: "#0ea5a4",
    info: "#64748b",
    teal: "#0ea5a4",
    blue: "#2563eb",
    grid: "rgba(148, 163, 184, 0.12)",
    text: "#94a3b8",
  };

  function readPayload() {
    var el = document.getElementById("rd-dashboard-data");
    if (!el || !el.textContent) return null;
    try {
      return JSON.parse(el.textContent);
    } catch (e) {
      console.warn("Repository Detective: invalid dashboard chart data", e);
      return null;
    }
  }

  function severityColors(labels) {
    return labels.map(function (label) {
      var key = (label || "").toLowerCase();
      return palette[key] || palette.blue;
    });
  }

  function baseOptions() {
    return {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: {
          labels: { color: palette.text, boxWidth: 12, font: { size: 11 } },
        },
      },
      scales: {
        x: {
          ticks: { color: palette.text, maxRotation: 45, minRotation: 0 },
          grid: { color: palette.grid },
        },
        y: {
          ticks: { color: palette.text },
          grid: { color: palette.grid },
          beginAtZero: true,
        },
      },
    };
  }

  function initSeverityChart(ctx, data) {
    if (!data.severityLabels || !data.severityLabels.length) return;
    new Chart(ctx, {
      type: "doughnut",
      data: {
        labels: data.severityLabels,
        datasets: [{
          data: data.severityValues,
          backgroundColor: severityColors(data.severityLabels),
          borderColor: "#111827",
          borderWidth: 2,
          hoverOffset: 8,
        }],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        cutout: "62%",
        plugins: {
          legend: { position: "right", labels: { color: palette.text } },
        },
      },
    });
  }

  function initCategoryChart(ctx, data) {
    if (!data.categoryLabels || !data.categoryLabels.length) return;
    new Chart(ctx, {
      type: "bar",
      data: {
        labels: data.categoryLabels,
        datasets: [{
          label: "Open findings",
          data: data.categoryValues,
          backgroundColor: "rgba(14, 165, 164, 0.65)",
          borderColor: palette.teal,
          borderWidth: 1,
          borderRadius: 6,
        }],
      },
      options: baseOptions(),
    });
  }

  function initRadarChart(ctx, data) {
    if (!data.categoryLabels || !data.categoryLabels.length) return;
    var max = Math.max.apply(null, data.categoryValues.concat([1]));
    new Chart(ctx, {
      type: "radar",
      data: {
        labels: data.categoryLabels,
        datasets: [{
          label: "Finding categories",
          data: data.categoryValues,
          backgroundColor: "rgba(37, 99, 235, 0.25)",
          borderColor: palette.blue,
          pointBackgroundColor: palette.teal,
          pointBorderColor: "#fff",
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
            ticks: { display: false, backdropColor: "transparent" },
            suggestedMin: 0,
            suggestedMax: max * 1.15,
          },
        },
        plugins: { legend: { display: false } },
      },
    });
  }

  function initTrendChart(ctx, data) {
    if (!data.scanTrendLabels) return;
    new Chart(ctx, {
      type: "line",
      data: {
        labels: data.scanTrendLabels,
        datasets: [{
          label: "Raw findings per scan day",
          data: data.scanTrendValues,
          borderColor: palette.teal,
          backgroundColor: "rgba(14, 165, 164, 0.15)",
          fill: true,
          tension: 0.35,
          pointRadius: 3,
          pointHoverRadius: 6,
        }],
      },
      options: baseOptions(),
    });
  }

  function initRepoMap(ctx, data) {
    if (!data.repoMapLabels || !data.repoMapLabels.length) return;
    var colors = data.repoMapLabels.map(function (_, i) {
      return data.repoMapFailed && data.repoMapFailed[i]
        ? "rgba(239, 68, 68, 0.75)"
        : "rgba(37, 99, 235, 0.7)";
    });
    new Chart(ctx, {
      type: "bar",
      data: {
        labels: data.repoMapLabels,
        datasets: [{
          label: "Open findings",
          data: data.repoMapValues,
          backgroundColor: colors,
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
                if (data.repoMapFailed && data.repoMapFailed[i]) {
                  return "Last scan failed";
                }
                return "";
              },
            },
          },
        },
        scales: {
          x: {
            ticks: { color: palette.text },
            grid: { color: palette.grid },
            beginAtZero: true,
          },
          y: {
            ticks: { color: palette.text, font: { size: 10 } },
            grid: { display: false },
          },
        },
      },
    });
  }

  function init() {
    if (typeof Chart === "undefined") return;
    var data = readPayload();
    if (!data) return;

    Chart.defaults.color = palette.text;
    Chart.defaults.borderColor = palette.grid;
    Chart.defaults.font.family = '"DM Sans", system-ui, sans-serif';

    var severity = document.getElementById("rd-chart-severity");
    var category = document.getElementById("rd-chart-category");
    var radar = document.getElementById("rd-chart-radar");
    var trend = document.getElementById("rd-chart-trend");
    var repoMap = document.getElementById("rd-chart-repos");

    if (severity) initSeverityChart(severity, data);
    if (category) initCategoryChart(category, data);
    if (radar) initRadarChart(radar, data);
    if (trend) initTrendChart(trend, data);
    if (repoMap) initRepoMap(repoMap, data);
  }

  document.addEventListener("DOMContentLoaded", init);
})();

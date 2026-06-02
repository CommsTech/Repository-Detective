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
      var data = JSON.parse(el.textContent.trim());
      if (typeof data === "string") {
        data = JSON.parse(data);
      }
      return data;
    } catch (e) {
      console.warn("Repository Detective: invalid dashboard chart data", e);
      return null;
    }
  }

  function hasPositiveValues(values) {
    if (!values || !values.length) return false;
    for (var i = 0; i < values.length; i++) {
      if (Number(values[i]) > 0) return true;
    }
    return false;
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
          ticks: { color: palette.text, precision: 0 },
          grid: { color: palette.grid },
          beginAtZero: true,
        },
      },
    };
  }

  function initSeverityChart(ctx, data) {
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
            ticks: { color: palette.text, precision: 0 },
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

  function showEmptyChart(canvas, message) {
    if (!canvas || !canvas.parentElement) return;
    var wrap = canvas.parentElement;
    if (wrap.querySelector(".rd-chart-empty")) return;
    canvas.style.display = "none";
    var note = document.createElement("p");
    note.className = "rd-chart-empty rd-muted";
    note.textContent = message;
    wrap.appendChild(note);
  }

  function showChartError(message) {
    var grid = document.querySelector(".rd-charts-grid");
    if (!grid || grid.querySelector(".rd-chart-error")) return;
    var alert = document.createElement("div");
    alert.className = "rd-alert rd-chart-error";
    alert.setAttribute("role", "alert");
    alert.textContent = message;
    grid.parentNode.insertBefore(alert, grid);
  }

  function mountCharts() {
    if (typeof Chart === "undefined") {
      console.warn("Repository Detective: Chart.js not loaded");
      showChartError("Charts could not load (Chart.js missing). Other dashboard data is still available.");
      return;
    }

    var data = readPayload();
    if (!data) {
      showChartError("Chart data is unavailable. Other dashboard metrics are still shown below.");
      return;
    }

    Chart.defaults.color = palette.text;
    Chart.defaults.borderColor = palette.grid;
    Chart.defaults.font.family = '"DM Sans", system-ui, sans-serif';

    var severity = document.getElementById("rd-chart-severity");
    var category = document.getElementById("rd-chart-category");
    var radar = document.getElementById("rd-chart-radar");
    var trend = document.getElementById("rd-chart-trend");
    var repoMap = document.getElementById("rd-chart-repos");

    if (severity) {
      if (data.severityLabels && data.severityLabels.length) {
        initSeverityChart(severity, data);
      } else {
        showEmptyChart(severity, "No open findings by severity.");
      }
    }
    if (category) {
      if (data.categoryLabels && data.categoryLabels.length) {
        initCategoryChart(category, data);
      } else {
        showEmptyChart(category, "No category breakdown yet.");
      }
    }
    if (radar) {
      if (data.categoryLabels && data.categoryLabels.length) {
        initRadarChart(radar, data);
      } else {
        showEmptyChart(radar, "No category data for radar.");
      }
    }
    if (trend) {
      if (data.scanTrendLabels && data.scanTrendLabels.length && hasPositiveValues(data.scanTrendValues)) {
        initTrendChart(trend, data);
      } else {
        showEmptyChart(trend, "No completed scan activity in the last 14 days.");
      }
    }
    if (repoMap) {
      if (data.repoMapLabels && data.repoMapLabels.length && hasPositiveValues(data.repoMapValues)) {
        initRepoMap(repoMap, data);
      } else {
        showEmptyChart(repoMap, "No repository risk data yet.");
      }
    }
  }

  if (document.readyState === "complete") {
    mountCharts();
  } else {
    window.addEventListener("load", mountCharts);
  }
})();

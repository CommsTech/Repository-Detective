(function () {
  "use strict";

  var chartHandles = [];

  function cssVar(name, fallback) {
    var v = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
    return v || fallback;
  }

  function readThemePalette() {
    return {
      critical: "#ef4444",
      high: "#f97316",
      medium: "#f59e0b",
      low: "#0ea5a4",
      info: "#64748b",
      teal: "#0ea5a4",
      blue: "#2563eb",
      grid: cssVar("--rd-border", "rgba(148, 163, 184, 0.12)"),
      text: cssVar("--rd-text-dim", "#94a3b8"),
      card: cssVar("--rd-card", "#111827"),
    };
  }

  var palette = readThemePalette();

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

  function trackChart(chart) {
    if (chart) chartHandles.push(chart);
    return chart;
  }

  function destroyCharts() {
    chartHandles.forEach(function (c) {
      try { c.destroy(); } catch (e) { /* ignore */ }
    });
    chartHandles = [];
    document.querySelectorAll(".rd-chart-empty").forEach(function (el) { el.remove(); });
    document.querySelectorAll(".rd-charts-grid canvas").forEach(function (c) {
      c.style.display = "";
    });
    var err = document.querySelector(".rd-chart-error");
    if (err) err.remove();
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
    trackChart(new Chart(ctx, {
      type: "doughnut",
      data: {
        labels: data.severityLabels,
        datasets: [{
          data: data.severityValues,
          backgroundColor: severityColors(data.severityLabels),
          borderColor: palette.card,
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
    }));
  }

  function initCategoryChart(ctx, data) {
    trackChart(new Chart(ctx, {
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
    }));
  }

  function initRadarChart(ctx, data) {
    var max = Math.max.apply(null, data.categoryValues.concat([1]));
    trackChart(new Chart(ctx, {
      type: "radar",
      data: {
        labels: data.categoryLabels,
        datasets: [{
          label: "Finding categories",
          data: data.categoryValues,
          backgroundColor: "rgba(37, 99, 235, 0.25)",
          borderColor: palette.blue,
          pointBackgroundColor: palette.teal,
          pointBorderColor: palette.card,
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
    }));
  }

  function initTrendChart(ctx, data) {
    var remediation = data.remediationTrendValues || [];
    var plans = data.planTrendValues || [];
    var datasets = [{
      label: "Completed scans",
      data: data.scanTrendValues,
      borderColor: palette.teal,
      backgroundColor: "rgba(14, 165, 164, 0.15)",
      fill: true,
      tension: 0.35,
      pointRadius: 3,
      pointHoverRadius: 6,
      yAxisID: "y",
    }];
    if (plans.length) {
      datasets.push({
        label: "Remediation plans",
        data: plans,
        borderColor: palette.blue,
        backgroundColor: "rgba(37, 99, 235, 0.08)",
        fill: false,
        tension: 0.35,
        pointRadius: 2,
        pointHoverRadius: 5,
        borderDash: [5, 4],
        yAxisID: "yRemediation",
      });
    }
    if (remediation.length) {
      datasets.push({
        label: "Auto-remediated findings",
        data: remediation,
        borderColor: "#22c55e",
        backgroundColor: "rgba(34, 197, 94, 0.12)",
        fill: false,
        tension: 0.35,
        pointRadius: 3,
        pointHoverRadius: 6,
        yAxisID: "yRemediation",
      });
    }
    var opts = baseOptions();
    opts.plugins.legend = {
      display: true,
      position: "bottom",
      labels: { color: palette.text, boxWidth: 12, font: { size: 11 } },
    };
    opts.scales.yRemediation = {
      position: "right",
      ticks: { color: palette.text, precision: 0 },
      grid: { drawOnChartArea: false },
      beginAtZero: true,
    };
    trackChart(new Chart(ctx, {
      type: "line",
      data: {
        labels: data.scanTrendLabels,
        datasets: datasets,
      },
      options: opts,
    }));
  }

  var riskSegmentColors = [
    "rgba(239, 68, 68, 0.85)",
    "rgba(249, 115, 22, 0.85)",
    "rgba(234, 179, 8, 0.85)",
    "rgba(139, 92, 246, 0.85)",
    "rgba(37, 99, 235, 0.85)",
    "rgba(20, 184, 166, 0.85)",
    "rgba(107, 114, 128, 0.85)",
  ];

  function initRepoMap(ctx, data) {
    if (data.repoMapStacks && data.repoMapStacks.length && data.repoMapStackLabels && data.repoMapStackLabels.length) {
      var datasets = data.repoMapStackLabels.map(function (label, si) {
        return {
          label: label,
          data: data.repoMapStacks.map(function (row) { return row[si] || 0; }),
          backgroundColor: riskSegmentColors[si % riskSegmentColors.length],
          borderRadius: si === data.repoMapStackLabels.length - 1 ? 6 : 0,
          stack: "risk",
        };
      });
      trackChart(new Chart(ctx, {
        type: "bar",
        data: { labels: data.repoMapLabels, datasets: datasets },
        options: {
          indexAxis: "y",
          responsive: true,
          maintainAspectRatio: false,
          plugins: { legend: { position: "bottom", labels: { color: palette.text, boxWidth: 12 } } },
          scales: {
            x: { stacked: true, ticks: { color: palette.text, precision: 0 }, grid: { color: palette.grid }, beginAtZero: true },
            y: { stacked: true, ticks: { color: palette.text, font: { size: 10 } }, grid: { display: false } },
          },
        },
      }));
      return;
    }
    var borderColors = (data.repoMapFailed || []).map(function (failed) {
      return failed ? "#ef4444" : palette.teal;
    });
    trackChart(new Chart(ctx, {
      type: "bar",
      data: {
        labels: data.repoMapLabels,
        datasets: [{
          label: "Open findings",
          data: data.repoMapValues,
          backgroundColor: "rgba(14, 165, 164, 0.55)",
          borderColor: borderColors,
          borderWidth: 2,
          borderRadius: 6,
        }],
      },
      options: Object.assign(baseOptions(), { indexAxis: "y" }),
    }));
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
    destroyCharts();
    palette = readThemePalette();

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
      var hasStacks = false;
      if (data.repoMapStacks) {
        for (var i = 0; i < data.repoMapStacks.length; i++) {
          if (hasPositiveValues(data.repoMapStacks[i])) { hasStacks = true; break; }
        }
      }
      if (data.repoMapLabels && data.repoMapLabels.length && (hasPositiveValues(data.repoMapValues) || hasStacks)) {
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
  document.documentElement.addEventListener("rd-theme-change", mountCharts);
})();

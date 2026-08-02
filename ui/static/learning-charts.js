(function () {
  "use strict";

  var palette = {
    teal: "#0ea5a4",
    blue: "#2563eb",
    orange: "#f97316",
    red: "#ef4444",
    amber: "#f59e0b",
    violet: "#8b5cf6",
    slate: "#64748b",
    grid: "rgba(148, 163, 184, 0.12)",
    text: "#94a3b8",
  };

  var eventColors = [
    palette.teal, palette.blue, palette.orange, palette.violet,
    palette.amber, palette.red, palette.slate, "#14b8a6"
  ];

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

  function showEmpty(canvas, message) {
    if (!canvas || !canvas.parentNode) return;
    var note = document.createElement("p");
    note.className = "rd-muted";
    note.textContent = message;
    canvas.parentNode.replaceChild(note, canvas);
  }

  function initEventsChart(canvas, data) {
    if (!hasValues(data.eventValues)) {
      showEmpty(canvas, "No learning events recorded yet.");
      return;
    }
    new Chart(canvas, {
      type: "doughnut",
      data: {
        labels: data.eventLabels,
        datasets: [{
          data: data.eventValues,
          backgroundColor: eventColors.slice(0, data.eventLabels.length),
          borderColor: "#111827",
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
    });
  }

  function initNoisyChart(canvas, data) {
    if (!data.noisyRuleLabels || !data.noisyRuleLabels.length) {
      showEmpty(canvas, "No noisy rule stats yet — complete more scans.");
      return;
    }
    new Chart(canvas, {
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
    });
  }

  function mount() {
    if (typeof Chart === "undefined") return;
    var data = readPayload();
    if (!data) return;
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
})();

(function () {
  "use strict";

  var STORAGE_KEY = "rd-theme";
  var VALID = { system: true, light: true, dark: true };

  function readPreference() {
    try {
      var stored = localStorage.getItem(STORAGE_KEY);
      return VALID[stored] ? stored : "system";
    } catch (e) {
      return "system";
    }
  }

  function writePreference(mode) {
    if (!VALID[mode]) {
      return;
    }
    try {
      localStorage.setItem(STORAGE_KEY, mode);
    } catch (e) {
      /* ignore quota / private mode */
    }
  }

  function resolvedTheme(mode) {
    if (mode === "light" || mode === "dark") {
      return mode;
    }
    if (window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches) {
      return "dark";
    }
    return "light";
  }

  function emitThemeChange(mode) {
    document.documentElement.dispatchEvent(
      new CustomEvent("rd-theme-change", {
        detail: { preference: mode, resolved: resolvedTheme(mode) }
      })
    );
  }

  function updateThemeColorMeta(mode) {
    var meta = document.querySelector('meta[name="theme-color"]');
    if (!meta) {
      return;
    }
    meta.setAttribute("content", resolvedTheme(mode) === "dark" ? "#0d1b2a" : "#f0f4f8");
  }

  function syncToggleUI(mode) {
    document.querySelectorAll(".rd-theme-switch [data-theme-choice]").forEach(function (btn) {
      var choice = btn.getAttribute("data-theme-choice");
      var pressed = choice === mode;
      btn.setAttribute("aria-pressed", pressed ? "true" : "false");
      btn.classList.toggle("active", pressed);
    });
  }

  function applyTheme(mode, options) {
    if (!VALID[mode]) {
      mode = "system";
    }
    var opts = options || {};
    document.documentElement.dataset.theme = mode;
    var resolved = resolvedTheme(mode);
    document.documentElement.style.colorScheme = resolved;
    document.documentElement.style.backgroundColor = resolved === "dark" ? "#0a1018" : "#f0f4f8";
    updateThemeColorMeta(mode);
    syncToggleUI(mode);
    if (opts.emit !== false) {
      emitThemeChange(mode);
    }
  }

  function initThemeToggle() {
    document.querySelectorAll(".rd-theme-switch").forEach(function (group) {
      if (group.getAttribute("data-rd-theme-bound") === "true") {
        return;
      }
      group.setAttribute("data-rd-theme-bound", "true");
      group.addEventListener("click", function (e) {
        var btn = e.target.closest("[data-theme-choice]");
        if (!btn || !group.contains(btn)) {
          return;
        }
        var mode = btn.getAttribute("data-theme-choice");
        if (!VALID[mode]) {
          return;
        }
        writePreference(mode);
        applyTheme(mode);
      });
      group.addEventListener("keydown", function (e) {
        if (e.key !== "ArrowLeft" && e.key !== "ArrowRight") {
          return;
        }
        var buttons = Array.prototype.slice.call(group.querySelectorAll("[data-theme-choice]"));
        var idx = buttons.indexOf(document.activeElement);
        if (idx < 0) {
          return;
        }
        e.preventDefault();
        var next = e.key === "ArrowRight" ? idx + 1 : idx - 1;
        if (next < 0) {
          next = buttons.length - 1;
        }
        if (next >= buttons.length) {
          next = 0;
        }
        buttons[next].focus();
      });
    });
  }

  function watchSystemTheme() {
    if (!window.matchMedia) {
      return;
    }
    var mq = window.matchMedia("(prefers-color-scheme: dark)");
    var handler = function () {
      if (readPreference() === "system") {
        updateThemeColorMeta("system");
        emitThemeChange("system");
      }
    };
    if (typeof mq.addEventListener === "function") {
      mq.addEventListener("change", handler);
    } else if (typeof mq.addListener === "function") {
      mq.addListener(handler);
    }
  }

  function initTheme() {
    applyTheme(readPreference(), { emit: false });
    initThemeToggle();
    watchSystemTheme();
    emitThemeChange(readPreference());
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initTheme);
  } else {
    initTheme();
  }

  window.RDTheme = {
    readPreference: readPreference,
    applyTheme: applyTheme,
    resolvedTheme: resolvedTheme
  };
})();

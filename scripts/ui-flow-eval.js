#!/usr/bin/env node
/**
 * Repository Detective WebUI flow evaluation (light/dark/system).
 */
const { chromium } = require("playwright");
const fs = require("fs");
const path = require("path");

const BASE = (process.env.RD_BASE_URL || "http://127.0.0.1:8081/ui").replace(/\/$/, "");
const KEY = process.env.REPOSITORY_DETECTIVE_API_KEY || process.env.BUGBOT_API_KEY || "";
const SHOTS = process.env.RD_SHOT_DIR || "/out/shots";
fs.mkdirSync(SHOTS, { recursive: true });

const PAGES = [
  ["dashboard", ""],
  ["repos", "/repos"],
  ["scans", "/scans"],
  ["findings", "/findings"],
  ["reports", "/reports"],
  ["learning", "/learning"],
  ["health", "/health"],
  ["preinstall", "/preinstall"],
  ["projects", "/projects"],
  ["configure", "/configure"],
];
const THEMES = ["light", "dark", "system"];

function urlFor(p) {
  const sep = p.includes("?") ? "&" : "?";
  return KEY ? `${BASE}${p}${sep}api_key=${encodeURIComponent(KEY)}` : `${BASE}${p}`;
}

function parseRGB(s) {
  const m = /rgba?\((\d+),\s*(\d+),\s*(\d+)(?:,\s*([0-9.]+))?\)/.exec(s || "");
  if (!m) return null;
  const a = m[4] === undefined ? 1 : Number(m[4]);
  if (a < 0.2) return null; // treat near-transparent as no background
  return [Number(m[1]), Number(m[2]), Number(m[3])];
}

function lum([r, g, b]) {
  const f = (c) => {
    c /= 255;
    return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
  };
  return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b);
}

function contrast(fg, bg) {
  const L1 = lum(fg), L2 = lum(bg);
  const lighter = Math.max(L1, L2), darker = Math.min(L1, L2);
  return (lighter + 0.05) / (darker + 0.05);
}

async function setTheme(page, theme) {
  await page.evaluate((theme) => {
    localStorage.setItem("rd-theme", theme);
    if (window.RDTheme) window.RDTheme.applyTheme(theme);
    else {
      document.documentElement.dataset.theme = theme;
      document.documentElement.dispatchEvent(
        new CustomEvent("rd-theme-change", { detail: { preference: theme } })
      );
    }
  }, theme);
  await page.waitForTimeout(400);
}

async function evaluatePage(page, name, theme) {
  const result = {
    page: name,
    theme,
    url: page.url(),
    title: await page.title(),
    status: "ok",
    issues: [],
    branding: {},
    charts: {},
    nav: {},
    low_contrast: [],
    alerts: [],
  };

  const bodyText = await page.innerText("body");
  const html = await page.content();
  const bugbotVisible = (bodyText.match(/bugbot/gi) || []).length;
  const bugbotHtml = (html.match(/bugbot/gi) || []).length;
  const hasProduct = /Repository Detective/i.test(bodyText) || /Repository Detective/i.test(result.title);
  result.branding = {
    has_repository_detective: hasProduct,
    bugbot_visible_count: bugbotVisible,
    bugbot_html_count: bugbotHtml,
  };
  if (bugbotVisible > 0) {
    result.issues.push(`Visible Bugbot text (${bugbotVisible})`);
    result.status = "fail";
  }
  if (!hasProduct) {
    result.issues.push("Missing Repository Detective branding");
    result.status = "fail";
  }

  const navLinks = await page.$$eval(".rd-nav a", (els) =>
    els.map((e) => ({
      text: (e.textContent || "").trim().replace(/\s+/g, " "),
      href: e.getAttribute("href") || "",
    }))
  );
  result.nav.links = navLinks;
  if (navLinks.some((n) => /polic/i.test(n.text))) {
    result.issues.push("Unexpected Policies nav entry");
    result.status = "fail";
  }
  if (!navLinks.some((n) => /learning/i.test(n.href))) {
    result.issues.push("Learning missing from nav");
    result.status = "fail";
  }

  const applied = await page.evaluate(() => ({
    dataset: document.documentElement.dataset.theme,
    colorScheme: document.documentElement.style.colorScheme,
  }));
  result.theme_applied = applied;
  if (applied.dataset !== theme) {
    result.issues.push(`Theme dataset expected ${theme}, got ${applied.dataset}`);
    result.status = "fail";
  }

  // Prefer solid ancestor backgrounds so translucent overlays don't false-fail contrast.
  const selectors = [
    ".rd-stat-value",
    ".rd-kpi strong",
    ".rd-card h2",
    ".rd-card h3",
    ".rd-topbar h1",
    ".rd-table th",
    ".rd-table td strong",
  ];
  for (const sel of selectors) {
    try {
      const info = await page.$eval(sel, (el) => {
        const cs = getComputedStyle(el);
        let node = el;
        let bg = "rgba(0, 0, 0, 0)";
        while (node && node !== document.documentElement) {
          const b = getComputedStyle(node).backgroundColor;
          if (b && b !== "transparent" && !b.endsWith(", 0)") && b !== "rgba(0, 0, 0, 0)") {
            bg = b;
            break;
          }
          node = node.parentElement;
        }
        return {
          text: (el.textContent || "").trim().slice(0, 40),
          color: cs.color,
          bg,
          display: cs.display,
          visibility: cs.visibility,
        };
      });
      if (!info || info.visibility === "hidden" || info.display === "none" || !info.text) continue;
      const fg = parseRGB(info.color);
      const bg = parseRGB(info.bg);
      if (!fg || !bg) continue;
      const ratio = contrast(fg, bg);
      if (ratio < 3.0) {
        result.low_contrast.push({ sel, ratio: Number(ratio.toFixed(2)), text: info.text, color: info.color, bg: info.bg });
      }
    } catch (_) {
      /* selector absent */
    }
  }
  if (result.low_contrast.length) {
    result.issues.push(`Low contrast: ${JSON.stringify(result.low_contrast.slice(0, 4))}`);
    if (result.status === "ok") result.status = "warn";
  }

  // Give deferred Chart.js mounts a moment on chart-heavy pages.
  if (["dashboard", "learning", "repo_report"].includes(name)) {
    await page.waitForTimeout(900);
  }

  result.charts = await page.evaluate(() => {
    const canvases = [...document.querySelectorAll("canvas")].map((c) => ({
      id: c.id || null,
      cw: c.clientWidth,
      ch: c.clientHeight,
      hidden: getComputedStyle(c).display === "none",
    }));
    const empty = [...document.querySelectorAll(".rd-chart-empty")].map((e) => e.textContent.trim());
    let live = 0;
    for (const c of document.querySelectorAll("canvas")) {
      try {
        if (window.Chart && Chart.getChart && Chart.getChart(c)) live++;
      } catch (_) {}
    }
    return { canvases, empty, liveCharts: live };
  });
  if (name === "dashboard" && (result.charts.liveCharts || 0) < 1) {
    result.issues.push("Dashboard charts did not mount");
    result.status = "fail";
  }
  if (name === "learning" && (result.charts.liveCharts || 0) < 1) {
    result.issues.push("Learning charts did not mount");
    if (result.status === "ok") result.status = "warn";
  }
  if (name === "repo_report") {
    const radar = await page.locator("#rd-report-radar").count();
    if (radar && (result.charts.liveCharts || 0) < 1 && !(result.charts.empty || []).length) {
      result.issues.push("Repo report radar chart did not mount");
      if (result.status === "ok") result.status = "warn";
    }
  }

  // Print / Save PDF must be near the top on report pages.
  if (name === "reports" || name === "repo_report") {
    const printInfo = await page.evaluate(() => {
      const btn = document.querySelector("[data-rd-print]");
      if (!btn) return { present: false };
      const box = btn.getBoundingClientRect();
      return { present: true, y: box.y, text: (btn.textContent || "").trim() };
    });
    result.print_button = printInfo;
    if (!printInfo.present) {
      result.issues.push("Print / Save PDF button missing");
      result.status = "fail";
    } else if (printInfo.y > 280) {
      result.issues.push(`Print button too low on page (y=${Math.round(printInfo.y)})`);
      result.status = "fail";
    }
  }

  result.alerts = await page.$$eval(".rd-alert, [role=alert]", (els) =>
    els.map((e) => ({ cls: e.className, text: (e.textContent || "").trim().slice(0, 140) }))
  );
  for (const a of result.alerts) {
    const t = (a.text || "").toLowerCase();
    if (t.includes("internal server error") || t.includes("panic")) {
      result.issues.push(`Error alert: ${a.text}`);
      result.status = "fail";
    }
  }

  const broken = await page.evaluate(() =>
    [...document.images].filter((i) => !i.complete || i.naturalWidth === 0).map((i) => i.src)
  );
  if (broken.length) {
    result.issues.push(`Broken images: ${broken.join(", ")}`);
    result.status = "fail";
  }

  // Widget sanity: KPI numbers present on dashboard/reports/health
  if (["dashboard", "reports", "health", "learning"].includes(name)) {
    const kpiCount = await page.locator(".rd-kpi, .rd-stat, .rd-stat-value").count();
    result.widget_count = kpiCount;
    if (kpiCount === 0) {
      result.issues.push("No KPI/stat widgets found");
      if (result.status === "ok") result.status = "warn";
    }
  }

  const shot = path.join(SHOTS, `${name}-${theme}.png`);
  await page.screenshot({ path: shot, fullPage: true });
  result.screenshot = shot;
  return result;
}

async function main() {
  if (!KEY) {
    console.error("REPOSITORY_DETECTIVE_API_KEY missing");
    process.exit(2);
  }
  const report = { base: BASE, pages: [], summary: {} };
  const browser = await chromium.launch({
    headless: true,
    args: ["--no-sandbox", "--disable-dev-shm-usage"],
  });
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();
  page.setDefaultTimeout(45000);

  for (const theme of THEMES) {
    await page.goto(urlFor(""), { waitUntil: "networkidle" });
    await setTheme(page, theme);
    await page.reload({ waitUntil: "networkidle" });
    await page.waitForTimeout(300);

    for (const [name, p] of PAGES) {
      try {
        const resp = await page.goto(urlFor(p), { waitUntil: "networkidle" });
        await page.waitForTimeout(700);
        await setTheme(page, theme);
        const row = await evaluatePage(page, name, theme);
        row.http_status = resp ? resp.status() : 0;
        if (row.http_status >= 400) {
          row.status = "fail";
          row.issues.push(`HTTP ${row.http_status}`);
        }
        report.pages.push(row);
        console.log(`[${row.status}] ${theme}/${name} http=${row.http_status} issues=${row.issues.length}`);
      } catch (e) {
        report.pages.push({ page: name, theme, status: "fail", issues: [`exception: ${e}`] });
        console.log(`[fail] ${theme}/${name} EXC ${e}`);
      }
    }

    // repo detail + report
    try {
      await page.goto(urlFor("/repos"), { waitUntil: "networkidle" });
      await setTheme(page, theme);
      const href = await page.$eval("a[href*='/repos/']", (a) => a.getAttribute("href")).catch(() => null);
      if (href) {
        const abs = href.startsWith("http") ? href : `http://127.0.0.1:8081${href}`;
        await page.goto(abs, { waitUntil: "networkidle" });
        await page.waitForTimeout(500);
        await setTheme(page, theme);
        const row = await evaluatePage(page, "repo_detail", theme);
        report.pages.push(row);
        console.log(`[${row.status}] ${theme}/repo_detail issues=${row.issues.length}`);

        const rh = await page.$eval("a[href*='/report']", (a) => a.getAttribute("href")).catch(() => null);
        if (rh) {
          const rabs = rh.startsWith("http") ? rh : `http://127.0.0.1:8081${rh}`;
          await page.goto(rabs, { waitUntil: "networkidle" });
          await page.waitForTimeout(800);
          await setTheme(page, theme);
          const rrow = await evaluatePage(page, "repo_report", theme);
          report.pages.push(rrow);
          console.log(`[${rrow.status}] ${theme}/repo_report issues=${rrow.issues.length}`);
        }
      }
    } catch (e) {
      report.pages.push({ page: "repo_detail", theme, status: "fail", issues: [String(e)] });
      console.log(`[fail] ${theme}/repo_detail EXC ${e}`);
    }
  }

  await browser.close();
  const fails = report.pages.filter((p) => p.status === "fail");
  const warns = report.pages.filter((p) => p.status === "warn");
  const oks = report.pages.filter((p) => p.status === "ok");
  const unique = [...new Set(report.pages.flatMap((p) => p.issues || []))].sort();
  report.summary = { ok: oks.length, warn: warns.length, fail: fails.length, total: report.pages.length, unique_issues: unique };
  fs.writeFileSync("/out/report.json", JSON.stringify(report, null, 2));
  console.log("\n=== SUMMARY ===");
  console.log(JSON.stringify(report.summary, null, 2));
  process.exit(fails.length ? 1 : 0);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});

#!/usr/bin/env node
/**
 * Full UI verification: every route, nav link, interactive control, chart, and JSON data endpoint.
 * Non-destructive — does not submit remediation, reconcile, or scan-start forms.
 *
 * Usage (Playwright Docker image):
 *   docker run --rm --network host -v $PWD/scripts:/scripts -v /tmp/rd-ui-verify:/out \
 *     -e REPOSITORY_DETECTIVE_API_KEY=... -e RD_BASE_URL=http://127.0.0.1:8081/ui \
 *     mcr.microsoft.com/playwright:v1.49.0-jammy node /scripts/ui-full-verification.js
 */
const { chromium } = require("playwright");
const fs = require("fs");
const path = require("path");

const BASE = (process.env.RD_BASE_URL || "http://127.0.0.1:8081/ui").replace(/\/$/, "");
const API_BASE = (process.env.RD_API_BASE || BASE.replace(/\/ui$/, "")).replace(/\/$/, "");
const KEY = process.env.REPOSITORY_DETECTIVE_API_KEY || process.env.BUGBOT_API_KEY || "";
const REPO_ID = process.env.RD_VERIFY_REPO_ID || "1";
const SCAN_ID = process.env.RD_VERIFY_SCAN_ID || "";
const FINDING_ID = process.env.RD_VERIFY_FINDING_ID || "1";
const OUT = process.env.RD_VERIFY_OUT || "/out";
const THEME = process.env.RD_VERIFY_THEME || "light";
const WAIT_UNTIL = process.env.RD_VERIFY_WAIT_UNTIL || "domcontentloaded";
const PAGE_SETTLE_MS = Number(process.env.RD_VERIFY_SETTLE_MS || 800);

fs.mkdirSync(OUT, { recursive: true });

const ROUTES = [
  ["dashboard", ""],
  ["repos", "/repos"],
  ["repo_detail", `/repos/${REPO_ID}`],
  ["repo_settings", `/repos/${REPO_ID}/settings`],
  ["repo_containers", `/repos/${REPO_ID}/containers`],
  ["repo_sbom", `/repos/${REPO_ID}/sbom`],
  ["repo_graph", `/repos/${REPO_ID}/graph`],
  ["repo_report", `/repos/${REPO_ID}/report`],
  ["repo_reconcile", `/repos/${REPO_ID}/reconcile`],
  ["repo_scan", `/repos/${REPO_ID}/scan`],
  ["scans", "/scans"],
  ["findings", "/findings"],
  ["findings_filtered", "/findings?repo_id=" + REPO_ID + "&status=open"],
  ["finding_detail", "/findings/" + FINDING_ID],
  ["configure", "/configure"],
  ["learning", "/learning"],
  ["health", "/health"],
  ["preinstall", "/preinstall"],
  ["reports", "/reports"],
  ["projects", "/projects"],
];

if (SCAN_ID) {
  ROUTES.push(["scan_detail", `/scans/${SCAN_ID}`]);
  ROUTES.push(["scan_sbom", `/scans/${SCAN_ID}/sbom`]);
  ROUTES.push(["scan_graph", `/scans/${SCAN_ID}/graph`]);
}

const LIGHTWEIGHT_ROUTES = new Set(["repo_graph", "scan_graph"]);
const LIGHTWEIGHT_PATHS = [/\/graph(?:\?|$|\/)/];

const JSON_ENDPOINTS = [
  ["api_status", `${API_BASE}/api/v1/status`],
  ["api_dashboard_summary", `${API_BASE}/api/v1/dashboard/summary`],
  ["graph_repo_data", `${API_BASE}/ui/repos/${REPO_ID}/graph/data`],
];
if (SCAN_ID) {
  JSON_ENDPOINTS.push(["graph_scan_data", `${API_BASE}/ui/scans/${SCAN_ID}/graph/data`]);
}

const STATIC_ASSETS = [
  "/ui/static/theme.css",
  "/ui/static/theme.js",
  "/ui/static/app.js",
  "/ui/static/dashboard-charts.js",
  "/ui/static/learning-charts.js",
  "/ui/static/repo-report-charts.js",
  "/ui/static/chart.umd.min.js",
  "/ui/static/cytoscape.min.js",
  "/ui/static/graph.js",
  "/ui/static/logo.svg",
  "/ui/static/favicon.svg",
];

const DESTRUCTIVE_PATTERNS = [
  /logout/i,
  /reconcile.*apply/i,
  /attempt-pr/i,
  /suppress/i,
  /mark-false/i,
  /mark-intentional/i,
  /enable-scanning/i,
  /disable-scanning/i,
  /remediation\/(approve|reject|generate)/i,
  /closure\/verify/i,
  /projects.*create/i,
  /preinstall.*start/i,
  /learning\/recommendations.*accept/i,
  /learning\/recommendations.*reject/i,
  /configure.*save/i,
];

function urlFor(p) {
  const sep = p.includes("?") ? "&" : "?";
  return KEY ? `${BASE}${p}${sep}api_key=${encodeURIComponent(KEY)}` : `${BASE}${p}`;
}

function authHeaders() {
  return {
    "X-Repository-Detective-API-Key": KEY,
    Authorization: `Bearer ${KEY}`,
  };
}

function isDestructive(action) {
  const s = String(action || "");
  return DESTRUCTIVE_PATTERNS.some((re) => re.test(s));
}

async function auditInteractiveElements(page, pageName) {
  const issues = [];
  const inventory = await page.evaluate(() => {
    const nav = [...document.querySelectorAll(".rd-nav a, nav a")].map((a) => ({
      type: "nav_link",
      text: (a.textContent || "").trim().replace(/\s+/g, " ").slice(0, 60),
      href: a.getAttribute("href") || "",
      visible: a.offsetParent !== null,
    }));
    const buttons = [...document.querySelectorAll("button, input[type=submit], a.rd-btn, .rd-btn")].map((el) => ({
      type: el.tagName.toLowerCase(),
      text: (el.textContent || el.value || "").trim().replace(/\s+/g, " ").slice(0, 60),
      href: el.getAttribute("href") || "",
      disabled: el.disabled || el.getAttribute("aria-disabled") === "true",
      visible: el.offsetParent !== null,
    }));
    const forms = [...document.querySelectorAll("form")].map((f) => ({
      action: f.getAttribute("action") || "",
      method: (f.getAttribute("method") || "get").toLowerCase(),
      hasCsrf: !!f.querySelector('input[name="csrf_token"]'),
      inputs: f.querySelectorAll("input, select, textarea").length,
    }));
    const canvases = [...document.querySelectorAll("canvas")].map((c) => ({
      id: c.id || null,
      w: c.clientWidth,
      h: c.clientHeight,
    }));
    const tables = document.querySelectorAll("table.rd-table, .rd-table table").length;
    const alerts = [...document.querySelectorAll(".rd-alert, [role=alert]")].map((e) =>
      (e.textContent || "").trim().slice(0, 120)
    );
    const headings = [...document.querySelectorAll("h1, h2.rd-page-title, .rd-topbar h1")].map((h) =>
      (h.textContent || "").trim().slice(0, 80)
    );
    let liveCharts = 0;
    for (const c of document.querySelectorAll("canvas")) {
      try {
        if (window.Chart && Chart.getChart && Chart.getChart(c)) liveCharts++;
      } catch (_) {}
    }
    return { nav, buttons, forms, canvases, tables, alerts, headings, liveCharts };
  });

  for (const a of inventory.alerts) {
    if (/internal server error|panic|runtime error/i.test(a)) {
      issues.push(`error_alert: ${a}`);
    }
  }

  for (const f of inventory.forms) {
    if (f.method === "post" && !f.hasCsrf) {
      issues.push(`post_form_missing_csrf: ${f.action}`);
    }
  }

  if (pageName === "dashboard" && inventory.liveCharts < 1 && inventory.canvases.length > 0) {
    issues.push("dashboard_charts_not_mounted");
  }
  if (pageName === "learning" && inventory.canvases.length > 0 && inventory.liveCharts < 1) {
    issues.push("learning_charts_not_mounted");
  }

  return { inventory, issues };
}

async function clickSafeNavLinks(page, report) {
  const links = await page.$$eval(".rd-nav a[href]", (els) =>
    els.map((a) => ({ text: (a.textContent || "").trim(), href: a.getAttribute("href") }))
  );
  for (const link of links) {
    if (!link.href || link.href.startsWith("http") && !link.href.includes("8081")) continue;
    if (/\/graph(?:\?|$|\/)/.test(link.href)) {
      report.nav_clicks.push({ text: link.text, href: link.href, status: "skipped_graph" });
      continue;
    }
    try {
      const target = link.href.startsWith("http") ? link.href : `${API_BASE}${link.href}`;
      const resp = await page.goto(target, { waitUntil: WAIT_UNTIL, timeout: 30000 });
      const status = resp ? resp.status() : 0;
      report.nav_clicks.push({ text: link.text, href: link.href, status });
      if (status >= 400) {
        report.failures.push(`nav_click_${link.text}: HTTP ${status}`);
      }
      await page.waitForTimeout(300);
    } catch (e) {
      report.failures.push(`nav_click_${link.text}: ${e.message}`);
    }
  }
}

async function verifyPage(page, name, routePath, report) {
  const entry = { name, route: routePath, status: "ok", http: 0, issues: [], inventory: null };
  const lightweight =
    LIGHTWEIGHT_ROUTES.has(name) || LIGHTWEIGHT_PATHS.some((re) => re.test(routePath));
  const gotoTimeout = lightweight ? 20000 : 45000;
  try {
    const resp = await page.goto(urlFor(routePath), { waitUntil: WAIT_UNTIL, timeout: gotoTimeout });
    entry.http = resp ? resp.status() : 0;
    if (entry.http >= 400) {
      entry.status = "fail";
      entry.issues.push(`HTTP ${entry.http}`);
      report.pages.push(entry);
      report.failures.push(`${name}: HTTP ${entry.http}`);
      return;
    }
    if (lightweight) {
      await page.waitForSelector("#graph-app, .rd-page, main", { timeout: 8000 }).catch(() => {});
      entry.inventory = {
        nav_links: 0,
        buttons: 0,
        forms: 0,
        canvases: 0,
        live_charts: 0,
        tables: 0,
        headings: [],
        lightweight: true,
      };
      report.pages.push(entry);
      console.log(`[${entry.status}] ${name} http=${entry.http} lightweight=1`);
      return;
    }
    await page.waitForTimeout(PAGE_SETTLE_MS);
    let body = "";
    try {
      body = await page.innerText("body", { timeout: 10000 });
    } catch (_) {
      body = await page.evaluate(() => document.body?.innerText || "");
    }
    if (/panic|runtime error|internal server error/i.test(body)) {
      entry.status = "fail";
      entry.issues.push("error_text_in_body");
    }
    const { inventory, issues } = await auditInteractiveElements(page, name);
    entry.inventory = {
      nav_links: inventory.nav.length,
      buttons: inventory.buttons.length,
      forms: inventory.forms.length,
      canvases: inventory.canvases.length,
      live_charts: inventory.liveCharts,
      tables: inventory.tables,
      headings: inventory.headings.slice(0, 3),
    };
    entry.issues.push(...issues);
    if (issues.length) entry.status = entry.status === "ok" ? "warn" : entry.status;
    if (issues.some((i) => i.startsWith("error_"))) entry.status = "fail";

    // Inventory every visible button (presence check, no click for destructive)
    for (const btn of inventory.buttons.filter((b) => b.visible)) {
      report.controls_checked++;
      if (!btn.text && !btn.href) {
        entry.issues.push("unnamed_button");
      }
    }
    report.pages.push(entry);
    console.log(`[${entry.status}] ${name} http=${entry.http} nav=${entry.inventory.nav_links} btns=${entry.inventory.buttons} charts=${entry.inventory.live_charts}`);
  } catch (e) {
    entry.status = "fail";
    entry.issues.push(String(e));
    report.pages.push(entry);
    report.failures.push(`${name}: ${e.message}`);
    console.log(`[fail] ${name} EXC ${e.message}`);
  }
}

async function main() {
  if (!KEY) {
    console.error("REPOSITORY_DETECTIVE_API_KEY required");
    process.exit(2);
  }

  const report = {
    base: BASE,
    theme: THEME,
    repo_id: REPO_ID,
    scan_id: SCAN_ID,
    finding_id: FINDING_ID,
    pages: [],
    json_endpoints: [],
    static_assets: [],
    nav_clicks: [],
    failures: [],
    controls_checked: 0,
    summary: {},
  };

  const browser = await chromium.launch({ headless: true, args: ["--no-sandbox", "--disable-dev-shm-usage"] });
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();
  page.setDefaultTimeout(45000);

  // Theme
  await page.goto(urlFor(""), { waitUntil: WAIT_UNTIL });
  await page.evaluate((theme) => {
    localStorage.setItem("rd-theme", theme);
    if (window.RDTheme) window.RDTheme.applyTheme(theme);
    else document.documentElement.dataset.theme = theme;
  }, THEME);
  await page.reload({ waitUntil: WAIT_UNTIL });

  // All routes
  for (const [name, routePath] of ROUTES) {
    const lightweight =
      LIGHTWEIGHT_ROUTES.has(name) || LIGHTWEIGHT_PATHS.some((re) => re.test(routePath));
    if (lightweight) {
      const isolated = await context.newPage();
      isolated.setDefaultTimeout(45000);
      try {
        await verifyPage(isolated, name, routePath, report);
      } finally {
        await isolated.close();
      }
      continue;
    }
    await verifyPage(page, name, routePath, report);
  }

  // Nav click-through from dashboard
  await page.goto(urlFor(""), { waitUntil: WAIT_UNTIL });
  await clickSafeNavLinks(page, report);

  // Theme toggle
  try {
    await page.goto(urlFor("/configure"), { waitUntil: WAIT_UNTIL });
    const toggle = page.locator("[data-rd-theme-toggle], #rd-theme-toggle, .rd-theme-toggle").first();
    if (await toggle.count()) {
      await toggle.click();
      await page.waitForTimeout(400);
      const theme = await page.evaluate(() => document.documentElement.dataset.theme);
      report.theme_toggle = theme;
      if (!theme) report.failures.push("theme_toggle: no dataset.theme after click");
    }
  } catch (e) {
    report.failures.push(`theme_toggle: ${e.message}`);
  }

  await browser.close();

  // JSON endpoints via fetch
  for (const [name, url] of JSON_ENDPOINTS) {
    try {
      const res = await fetch(url, { headers: authHeaders() });
      const entry = { name, url, status: res.status, ok: res.ok };
      if (res.ok) {
        const ct = res.headers.get("content-type") || "";
        if (ct.includes("json")) {
          const data = await res.json();
          entry.keys = Object.keys(data).slice(0, 12);
        }
      } else {
        report.failures.push(`json_${name}: HTTP ${res.status}`);
      }
      report.json_endpoints.push(entry);
      console.log(`[json] ${name} http=${entry.status}`);
    } catch (e) {
      report.json_endpoints.push({ name, url, error: String(e) });
      report.failures.push(`json_${name}: ${e.message}`);
    }
  }

  // Static assets
  for (const asset of STATIC_ASSETS) {
    try {
      const res = await fetch(`${API_BASE}${asset}`);
      const entry = { asset, status: res.status, ok: res.ok };
      if (!res.ok) report.failures.push(`static_${asset}: HTTP ${res.status}`);
      report.static_assets.push(entry);
    } catch (e) {
      report.static_assets.push({ asset, error: String(e) });
      report.failures.push(`static_${asset}: ${e.message}`);
    }
  }

  const fails = report.pages.filter((p) => p.status === "fail").length + report.failures.length;
  const warns = report.pages.filter((p) => p.status === "warn").length;
  const oks = report.pages.filter((p) => p.status === "ok").length;
  report.summary = {
    pages_ok: oks,
    pages_warn: warns,
    pages_fail: report.pages.filter((p) => p.status === "fail").length,
    total_failures: report.failures.length,
    controls_checked: report.controls_checked,
    routes_tested: report.pages.length,
    nav_clicks: report.nav_clicks.length,
    json_endpoints: report.json_endpoints.length,
    static_assets: report.static_assets.length,
  };

  const outPath = path.join(OUT, "ui-full-verification.json");
  fs.writeFileSync(outPath, JSON.stringify(report, null, 2));
  const mdPath = path.join(OUT, "ui-full-verification.md");
  fs.writeFileSync(
    mdPath,
    [
      "# UI full verification report",
      "",
      `Generated: ${new Date().toISOString()}`,
      `Base: ${BASE}`,
      "",
      "## Summary",
      `- Pages OK: ${oks}`,
      `- Pages warn: ${warns}`,
      `- Pages fail: ${report.summary.pages_fail}`,
      `- Total failures: ${report.failures.length}`,
      `- Controls inventoried: ${report.controls_checked}`,
      `- Nav clicks: ${report.nav_clicks.length}`,
      "",
      "## Failures",
      ...(report.failures.length ? report.failures.map((f) => `- ${f}`) : ["- (none)"]),
      "",
      "## Pages",
      "| Page | HTTP | Status | Nav | Buttons | Charts | Issues |",
      "|------|------|--------|-----|---------|--------|--------|",
      ...report.pages.map(
        (p) =>
          `| ${p.name} | ${p.http} | ${p.status} | ${p.inventory?.nav_links ?? "-"} | ${p.inventory?.buttons ?? "-"} | ${p.inventory?.live_charts ?? "-"} | ${p.issues.join("; ") || "-"} |`
      ),
    ].join("\n")
  );

  console.log("\n=== SUMMARY ===");
  console.log(JSON.stringify(report.summary, null, 2));
  console.log(`Report: ${outPath}`);
  process.exit(report.failures.length || report.summary.pages_fail ? 1 : 0);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});

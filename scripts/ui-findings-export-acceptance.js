#!/usr/bin/env node
/**
 * Read-only browser acceptance check for Findings JSON export.
 * Usage (staging only):
 *   RD_BASE_URL=http://127.0.0.1:8081/ui \
 *   REPOSITORY_DETECTIVE_API_KEY=<scoped test key> \
 *   node scripts/ui-findings-export-acceptance.js
 * Alternatively provide RD_STORAGE_STATE=/path/to/authenticated-playwright-storage.json.
 * Requires Playwright installed in the CI image; does not install software or submit changes.
 */
'use strict';
const fs = require('fs');
const { chromium } = require('playwright');
const base = (process.env.RD_BASE_URL || '').replace(/\/$/, '');
const key = process.env.REPOSITORY_DETECTIVE_API_KEY || '';
const state = process.env.RD_STORAGE_STATE || '';
if (!base || !/^https?:\/\//.test(base)) {
  console.error('RD_BASE_URL must be an HTTP(S) staging UI URL');
  process.exit(2);
}
(async function run() {
  const browser = await chromium.launch({headless:true});
  try {
    const context = await browser.newContext({acceptDownloads:true, ...(state ? {storageState:state} : {})});
    const page = await context.newPage();
    const errs = [];
    page.on('pageerror', e => errs.push(e.message));
    const pageURL = base + '/findings' + (key ? '?api_key=' + encodeURIComponent(key) : '');
    const response = await page.goto(pageURL, {waitUntil:'domcontentloaded',timeout:40000});
    if (!response || response.status() !== 200 || !(await page.locator('[data-finding-export-format="json"]').count())) {
      throw new Error('Findings page not available; sign in or provide an authenticated test context');
    }
    const downloadPromise = page.waitForEvent('download', {timeout:120000});
    await page.locator('[data-finding-export-format="json"]').click();
    const download = await downloadPromise;
    const filePath = await download.path();
    const payload = JSON.parse(fs.readFileSync(filePath,'utf8'));
    if (!Array.isArray(payload.findings) || payload.count !== payload.findings.length) {
      throw new Error('JSON export incomplete or invalid');
    }
    const status = await page.locator('[data-finding-export-status]').textContent();
    if (!/export ready/.test(status || '')) throw new Error('Missing completion message');
    if (errs.length) throw new Error('Browser JS errors: ' + errs.join('; '));
    console.log('PASS: Findings JSON export valid; rows=' + payload.count);
  } finally { await browser.close(); }
})().catch(err => { console.error('FAIL:',err.message); process.exitCode = 1; });

#!/usr/bin/env node
/**
 * Read-only staging QA for the two workbench pages.
 * Example:
 *   RD_BASE_URL=https://staging.example/ui RD_SCAN_ID=<scan> RD_REPO_ID=<id> node scripts/ui-scan-settings-check.js
 * Install the existing optional Playwright dev dependency before running.
 * Authentication must already be available to the browser. No credentials are logged.
 */
'use strict';
const { chromium } = require('playwright');
const base = (process.env.RD_BASE_URL || '').replace(/\/$/, '');
const scanID = process.env.RD_SCAN_ID || '';
const repoID = process.env.RD_REPO_ID || '';
if (!base || !scanID || !repoID || !/^https?:\/\//.test(base) || !/^[a-zA-Z0-9_-]+$/.test(scanID) || !/^[0-9]+$/.test(repoID)) {
  console.error('Required: RD_BASE_URL (http/https), RD_SCAN_ID and numeric RD_REPO_ID');
  process.exit(2);
}
const failures = [];
(async () => {
  const browser = await chromium.launch({ headless: true });
  try {
    for (const theme of ['dark', 'light']) {
      for (const width of [390, 768, 1280, 1600]) {
        const context = await browser.newContext({viewport: {width, height:900}, colorScheme: theme});
        await context.addInitScript(value => { localStorage.setItem('rd-theme', value); }, theme);
        const page = await context.newPage();
        const errors = [];
        page.on('pageerror', e => errors.push(e.message));
        for (const [path, selector] of [
          ['/scans/' + scanID, '.rd-scan-hero'],
          ['/repos/' + repoID + '/settings', '.rd-settings-start'],
        ]) {
          try {
            const response = await page.goto(base + path, {waitUntil:'networkidle',timeout:30000});
            if (!response || response.status() !== 200) throw new Error('page unavailable or unauthenticated');
            if (await page.locator(selector).count() !== 1) throw new Error('workbench not rendered');
            const info = await page.evaluate(() => {
              const h = document.querySelector('.rd-scan-heading h2, .rd-settings-hero h2');
              const r = h?.getBoundingClientRect();
              return {overflow:document.documentElement.scrollWidth>window.innerWidth+1,
                titleOverflow:r && r.right>window.innerWidth+1,
                brokenAnchors:[...document.querySelectorAll('.rd-workbench-nav a')].filter(a=>!document.querySelector(a.getAttribute('href'))).length};
            });
            if (info.overflow || info.titleOverflow || info.brokenAnchors) throw new Error(JSON.stringify(info));
            if (path.includes('/scans/')) {
              const evidence = page.locator('#rd-scan-coverage');
              const ai = page.locator('#rd-scan-ai');
              const above = await evidence.evaluate((el, aiId) => (el.compareDocumentPosition(document.getElementById(aiId)) & Node.DOCUMENT_POSITION_FOLLOWING) !== 0, 'rd-scan-ai');
              if (!above || !await ai.count()) throw new Error('evidence/AI sections missing or out of order');
            } else {
              if (!await page.locator('[name="scan_profile"]').count() || !await page.locator('button[type="submit"]').count()) throw new Error('settings controls missing');
              await page.locator('#rd-repo-advanced > summary').click();
              if (!await page.locator('#rd-repo-advanced').evaluate(el => el.open)) throw new Error('advanced panel not interactive');
            }
          } catch (err) { failures.push(`${theme}/${width} ${path}: ${err.message}`); }
        }
        if (errors.length) failures.push(`${theme}/${width}: ${errors.join('; ')}`);
        await context.close();
      }
    }
  } finally { await browser.close(); }
  if (failures.length) { console.error('UI QA FAILED\n'+failures.join('\n')); process.exitCode=1; }
  else console.log('UI QA PASSED: 16 workbench/theme/viewport checks (read-only staging)');
})().catch(e => { console.error('UI QA runner failed:',e.message); process.exitCode=1; });

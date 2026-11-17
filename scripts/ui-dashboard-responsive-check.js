#!/usr/bin/env node
/**
 * Non-destructive dashboard layout gate. Run against a staging deployment using
 * Playwright (`npm install --no-save playwright`) or the existing Playwright CI image.
 * Example: RD_BASE_URL=http://127.0.0.1:8081/ui node scripts/ui-dashboard-responsive-check.js
 * Auth: REPOSITORY_DETECTIVE_API_KEY (local-only staging; never log or capture URL).
 * This checks layout mechanics; user acceptance still requires visual inspection.
 */
'use strict';
const { chromium } = require('playwright');
const BASE = (process.env.RD_BASE_URL || 'http://127.0.0.1:8081/ui').replace(/\/$/, '');
const KEY = process.env.REPOSITORY_DETECTIVE_API_KEY || '';
const widths = [390, 768, 1024, 1280, 1600];
const themes = ['light', 'dark'];
const failures = [];
function note(message) { failures.push(message); }
async function run() {
  const browser = await chromium.launch({ headless: true });
  try {
    for (const theme of themes) {
      for (const width of widths) {
        const ctx = await browser.newContext({ viewport: { width, height: 900 }, colorScheme: theme });
        const page = await ctx.newPage();
        const pageErrors = [];
        page.on('pageerror', e => pageErrors.push(e.message));
        await page.addInitScript(t => { localStorage.setItem('rd-theme', t); }, theme);
        const target = KEY ? BASE + '/?api_key=' + encodeURIComponent(KEY) : BASE + '/';
        try {
          const response = await page.goto(target, { waitUntil: 'networkidle', timeout: 35000 });
          if (!response || response.status() >= 400) throw new Error('HTTP ' + (response && response.status()));
          if (await page.locator('.rd-dashboard-v2').count() !== 1) throw new Error('dashboard workbench not rendered (check authentication)');
          const result = await page.evaluate(() => {
            const vw = window.innerWidth;
            const r = sel => document.querySelector(sel)?.getBoundingClientRect();
            const intersects = (a, b) => a && b && Math.min(a.right,b.right)-Math.max(a.left,b.left)>3 && Math.min(a.bottom,b.bottom)-Math.max(a.top,b.top)>3;
            const heads = [...document.querySelectorAll('.rd-dashboard-v2 .rd-card-head')];
            const overlappingHeads = heads.filter(h => intersects(h.querySelector('h2')?.getBoundingClientRect(),h.querySelector('a')?.getBoundingClientRect())).length;
            const canvases = [...document.querySelectorAll('.rd-dashboard-v2 canvas:not(#rd-chart-repos)')];
            const brokenCanvases = canvases.filter(c => c.clientWidth === 0 || c.clientHeight === 0).map(c => c.id);
            return {
              viewport: vw, scrollWidth: document.documentElement.scrollWidth,
              overlappingHeads, brokenCanvases,
              overlappingHeader: intersects(r('.rd-topbar h1'),r('.rd-topbar-meta')),
              chartError: Boolean(document.querySelector('.rd-chart-error')),
              chartCount: canvases.length,
              themeBackground: getComputedStyle(document.documentElement).backgroundColor,
              bodyBackground: getComputedStyle(document.body).backgroundColor,
              bootGuardPresent: Boolean(document.getElementById('rd-theme-boot')),
            };
          });
          if (result.scrollWidth > width + 1) note(theme + '/' + width + ': body horizontal overflow ' + result.scrollWidth);
          if (result.overlappingHeads) note(theme + '/' + width + ': ' + result.overlappingHeads + ' card headings overlap actions');
          if (result.overlappingHeader) note(theme + '/' + width + ': header title overlaps controls');
          if (result.brokenCanvases.length) note(theme + '/' + width + ': zero-size chart(s): ' + result.brokenCanvases.join(','));
          if (result.chartError) note(theme + '/' + width + ': chart loader error');
          if (result.chartCount < 3) note(theme + '/' + width + ': expected three visible analytical charts');
          if (result.bootGuardPresent) note(theme + '/' + width + ': flash-prevention guard not released after CSS load');
          const expected = theme === 'dark' ? 'rgb(10, 16, 24)' : 'rgb(240, 244, 248)';
          if (result.themeBackground !== expected || result.bodyBackground !== expected) note(theme + '/' + width + ': theme background mismatch');
          if (pageErrors.length) note(theme + '/' + width + ': ' + pageErrors.join('; '));
          // Expand / collapse reveals the advanced chart without adding more page overflow.
          if (width === 1280) {
            const d = page.locator('#rd-dash-advanced');
            await d.locator('summary').click();
            await page.waitForTimeout(250);
            if (!await d.evaluate(e => e.open)) note(theme + '/1280: advanced disclosure did not open');
            if (!await page.locator('#rd-chart-repos').evaluate(c => !!(window.Chart && Chart.getChart(c)))) {
              note(theme + '/1280: repository chart was not initialized on expansion');
            }
          }
          console.log('checked ' + theme + ' ' + width + 'px');
        } catch (e) { note(theme + '/' + width + ': ' + e.message); }
        finally { await ctx.close(); }
      }
    }
  } finally { await browser.close(); }
  if (failures.length) { console.error('DASHBOARD QA FAIL\n' + failures.join('\n')); process.exitCode=1; }
  else console.log('DASHBOARD QA PASS: 10 viewport/theme combinations; details and charts checked');
}
run().catch(e => { console.error('QA runner error:',e.message);process.exitCode=1; });

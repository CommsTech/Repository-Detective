#!/usr/bin/env node
/**
 * Read-only, authenticated staging QA for a finding's remediation workbench.
 * RD_FINDING_URL must point to a prepared finding fixture on staging.
 * REPOSITORY_DETECTIVE_API_KEY is sent as a header, never appended to the URL.
 * Requires Playwright in the test environment; makes no configuration changes.
 */
'use strict';
const { chromium } = require('playwright');
const target = process.env.RD_FINDING_URL || '';
const key = process.env.REPOSITORY_DETECTIVE_API_KEY || '';
const widths = [390, 768, 1280];
const themes = ['dark', 'light'];
async function main() {
  if (!/^https?:\/\//.test(target)) throw new Error('set RD_FINDING_URL to an authenticated staging finding URL');
  const browser = await chromium.launch({ headless: true });
  const errors = [];
  try {
    for (const theme of themes) for (const width of widths) {
      const ctx = await browser.newContext({
        viewport: { width, height: 900 }, colorScheme: theme,
        extraHTTPHeaders: key ? { 'X-Repository-Detective-API-Key': key } : {},
      });
      try {
        const page = await ctx.newPage();
        const pageErrors = [];
        page.on('pageerror', e => pageErrors.push(e.message));
        await page.addInitScript(t => localStorage.setItem('rd-theme', t), theme);
        const response = await page.goto(target, { waitUntil: 'networkidle', timeout: 30000 });
        if (!response || response.status() >= 400) throw new Error('HTTP ' + (response && response.status()));
        if (await page.locator('#rd-remediation').count() !== 1) throw new Error('remediation workbench not present');
        const data = await page.evaluate(() => ({
          scrollWidth: document.documentElement.scrollWidth,
          viewportWidth: window.innerWidth,
          reviewForms: [...document.querySelectorAll('#rd-remediation form')].every(f => !!f.querySelector('input[name="csrf_token"]')),
          attempts: document.querySelectorAll('.rd-patch-attempt-item').length,
          theme: document.documentElement.getAttribute('data-theme'),
          brokenAttemptCards: [...document.querySelectorAll('.rd-patch-attempt-item')].some(el => el.getBoundingClientRect().width > window.innerWidth),
        }));
        if (data.scrollWidth > data.viewportWidth + 1) throw new Error('document horizontal overflow');
        if (!data.reviewForms) throw new Error('missing CSRF in remediation action form');
        if (data.brokenAttemptCards) throw new Error('attempt card exceeds viewport');
        if (data.theme !== theme) throw new Error(`theme mismatch: ${data.theme}`);
        if (pageErrors.length) throw new Error('JS errors: ' + pageErrors.join('; '));
        if (data.attempts) {
          const panel = page.locator('.rd-patch-attempts');
          const open = await panel.getAttribute('open');
          await panel.locator('summary').click();
          if (Boolean(await panel.getAttribute('open')) === (open !== null)) throw new Error('attempt evidence disclosure did not toggle');
        }
        console.log(`PASS ${theme}/${width}px`);
      } catch (e) { errors.push(`${theme}/${width}px: ${e.message}`); }
      finally { await ctx.close(); }
    }
  } finally { await browser.close(); }
  if (errors.length) throw new Error(errors.join('\n'));
  console.log('Remediation evidence QA: 6 responsive/theme checks passed');
}
main().catch(e => { console.error('QA FAILED:', e.message); process.exitCode = 1; });

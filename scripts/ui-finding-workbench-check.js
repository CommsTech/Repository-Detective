#!/usr/bin/env node
/**
 * Read-only finding investigation acceptance, for authenticated staging.
 * Prerequisites: Playwright available to Node, and a representative finding ID.
 * Use dedicated QA data: screenshot filenames contain no secrets, but screenshots
 * can still contain private repository names and redacted scanner evidence.
 */
'use strict';
const { chromium } = require('playwright');
const base = (process.env.RD_BASE_URL || 'http://127.0.0.1:8081/ui').replace(/\/$/, '');
const findingID = process.env.RD_FINDING_ID || '';
const key = process.env.REPOSITORY_DETECTIVE_API_KEY || '';
const widths = [390, 768, 1024, 1280, 1600];
const themes = ['light', 'dark'];
if (!/^[1-9]\d*$/.test(findingID)) {
  console.error('Set RD_FINDING_ID to an existing numeric finding ID in a staging instance');
  process.exit(2);
}
const failures = [];
function check(ok, context, message) { if (!ok) failures.push(`${context}: ${message}`); }
function redactError(error) {
  let message = String(error && error.message || error);
  if (key) {
    message = message.replaceAll(encodeURIComponent(key), "[redacted]").replaceAll(key, "[redacted]");
  }
  return message;
}
(async function run() {
  const browser = await chromium.launch({ headless: true });
  try {
    for (const theme of themes) {
      for (const width of widths) {
        const label = `${theme}/${width}`;
        const context = await browser.newContext({ viewport: { width, height: 900 }, colorScheme: theme });
        const page = await context.newPage();
        const errors = [];
        page.on('pageerror', error => errors.push(error.message));
        await page.addInitScript(preference => { localStorage.setItem('rd-theme', preference); }, theme);
        try {
          const url = `${base}/findings/${findingID}` + (key ? `?api_key=${encodeURIComponent(key)}` : '');
          const response = await page.goto(url, { waitUntil: 'networkidle', timeout: 30000 });
          check(response && response.status() === 200, label, 'response not 200');
          const result = await page.evaluate(() => {
            const el = s => document.querySelector(s);
            const rect = s => el(s)?.getBoundingClientRect();
            const intersects = (a,b) => Boolean(a && b && Math.min(a.right,b.right) - Math.max(a.left,b.left) > 3 && Math.min(a.bottom,b.bottom) - Math.max(a.top,b.top) > 3);
            const ids = ['rd-remediation', 'rd-closure', 'rd-evidence'];
            return {
              overflow: document.documentElement.scrollWidth > innerWidth + 1,
              finding: Boolean(el('.rd-case-page') && el('#rd-finding-headline')),
              confidence: Boolean(el('.rd-case-confidence meter')),
              linkToPlan: el('.rd-case-primary-link')?.getAttribute('href') === '#rd-remediation',
              sections: ids.map(id => Boolean(el(`#${id}`))),
              order: [el('.rd-case-page'),...ids.map(id => el(`#${id}`))].every((item,i,arr) => i === 0 || !item || !arr[i-1] || (arr[i-1].compareDocumentPosition(item) & Node.DOCUMENT_POSITION_FOLLOWING) !== 0),
              introOverlap: intersects(rect('.rd-case-heading h2'), rect('.rd-case-primary-link')),
              columnsOverlap: intersects(rect('.rd-case-primary'),rect('.rd-case-aside')),
              protectedForms: [...document.querySelectorAll('main form[method="post"]')].every(f => f.querySelector('input[name="csrf_token"]')),
              remediationStages: [...document.querySelectorAll('.rd-remediation-stage')].map(node => node.textContent.trim()),
              remediationWidthsInView: [...document.querySelectorAll('.rd-remediation-stage')].every(node => {
                const box = node.getBoundingClientRect();
                return box.left >= -1 && box.right <= innerWidth + 1;
              }),
              verifiedStage: Boolean(el('.rd-remediation-stage-verified')),
              backendVerified: Boolean([...document.querySelectorAll('#rd-closure p')].some(node => /Status:\s*verified\b/i.test(node.textContent))),
              evidenceCollapsed: !el('#rd-evidence')?.open,
              dispositionCollapsed: !el('.rd-case-disposition')?.open,
              bootGuardRemoved: !el('#rd-theme-boot'),
              theme: document.documentElement.dataset.theme,
            };
          });
          check(result.finding, label, 'finding workbench missing');
          check(result.confidence, label, 'confidence explanation missing');
          check(result.linkToPlan && result.sections[0] && result.sections[2], label, 'workflow link or required section missing');
          check(result.order, label, 'review sections are out of expected order');
          check(!result.overflow, label, 'document horizontal overflow');
          check(!result.introOverlap && !result.columnsOverlap, label, 'heading or assessment collision');
          check(result.remediationStages.length === 4 && result.remediationWidthsInView, label, 'remediation lifecycle missing or overflowing');
          check(!result.verifiedStage || result.backendVerified, label, 'verified stage without matching closure evidence');
          check(result.protectedForms, label, 'state-changing form missing CSRF field');
          check(result.evidenceCollapsed && result.dispositionCollapsed, label, 'secondary detail should begin collapsed');
          check(result.bootGuardRemoved && result.theme === theme, label, 'theme preference did not settle');
          check(!errors.length, label, `JS errors: ${errors.join('; ')}`);
          if (width === 1280) {
            await page.locator('#rd-evidence > summary').click();
            check(await page.locator('#rd-evidence').evaluate(e => e.open), label, 'evidence disclosure did not open');
            await page.locator('.rd-case-capabilities > summary').click();
            check(await page.locator('.rd-case-capabilities').evaluate(e => e.open), label, 'capability disclosure did not open');
            await page.locator('.rd-case-primary-link').click();
            check((await page.evaluate(() => location.hash)) === '#rd-remediation', label, 'remediation navigation failed');
          }
          console.log(`checked ${label}`);
        } catch (error) { failures.push(`${label}: ${redactError(error)}`); }
        finally { await context.close(); }
      }
    }
  } finally { await browser.close(); }
  if (failures.length) {
    console.error('FINDING WORKBENCH QA FAIL:\n' + failures.join('\n'));
    process.exitCode = 1;
  } else {
    console.log('FINDING WORKBENCH QA PASS: 10 viewport/theme combinations; read-only operator flow checks');
  }
})().catch(error => { console.error('QA runner error:', redactError(error)); process.exitCode = 1; });

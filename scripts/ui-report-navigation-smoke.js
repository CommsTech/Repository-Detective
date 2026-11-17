#!/usr/bin/env node
/* Read-only browser acceptance: execute against a running, non-production instance.
   Run in an environment with `playwright` installed and an authenticated UI.
   RD_BASE_URL=http://127.0.0.1:8081/ui RD_VERIFY_REPO_ID=1 node scripts/ui-report-navigation-smoke.js
*/
const { chromium } = require('playwright');
const BASE = (process.env.RD_BASE_URL || 'http://127.0.0.1:8081/ui').replace(/\/$/, '');
const KEY = process.env.REPOSITORY_DETECTIVE_API_KEY || '';
const REPO = process.env.RD_VERIFY_REPO_ID || '';
const paths = ['/reports', ...(REPO ? [`/repos/${encodeURIComponent(REPO)}/report`] : [])];
const pathURL = (path) => `${BASE}${path}${KEY ? (path.includes('?') ? '&' : '?') + 'api_key=' + encodeURIComponent(KEY) : ''}`;
(async () => {
  const browser = await chromium.launch({headless: true});
  const results = [];
  try {
    for (const path of paths) {
      const page = await browser.newPage({ viewport: {width: 1365, height: 900} });
      const pageErrors = [];
      page.on('pageerror', err => pageErrors.push(err.message));
      const start = Date.now();
      const response = await page.goto(pathURL(path), { waitUntil:'domcontentloaded', timeout:30000 });
      if (!response || response.status() !== 200) throw Error(`${path} returned ${response && response.status()}`);
      if (!await page.locator('.rd-report-intro').count()) throw Error(`${path} report intro missing`);
      for (const href of await page.locator('.rd-report-index a').evaluateAll(links => links.map(a => a.getAttribute('href')))) {
        if (!await page.locator(`#${href.slice(1)}`).count()) throw Error(`${path} broken section anchor ${href}`);
      }
      await page.keyboard.press('Control+k');
      if (!await page.locator('#rd-quicknav').evaluate(d => d.open)) throw Error('Ctrl+K did not open switcher');
      await page.locator('#rd-quicknav-search').fill('reports');
      if (!await page.locator('.rd-quicknav-result').count()) throw Error('Search returned no Reports destination');
      await page.keyboard.press('Escape');
      if (await page.locator('#rd-quicknav').evaluate(d => d.open)) throw Error('Escape did not close switcher');
      const theme = await page.locator('html').getAttribute('data-theme');
      await page.evaluate(() => localStorage.setItem('rd-theme', 'dark'));
      await page.reload({ waitUntil:'domcontentloaded' });
      const scheme = await page.locator('html').evaluate(el => getComputedStyle(el).colorScheme);
      if (scheme !== 'dark') throw Error(`Expected dark color scheme, got ${scheme}`);
      const overflows = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth + 1);
      if (overflows) throw Error('Unexpected horizontal overflow');
      if (pageErrors.length) throw Error(`${path} JS exceptions: ${pageErrors.join('; ')}`);
      results.push({path, domReadyMs:Date.now()-start, themeBefore:theme, darkScheme:scheme, errors:pageErrors.length});
      await page.close();
    }
    console.log(JSON.stringify({ok:true, results}, null, 2));
  } finally { await browser.close(); }
})().catch(e => { console.error(e); process.exitCode=1; });

// read-only layout regression; use an existing Playwright installation.
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const assert = require('node:assert/strict');
const fs = require('node:fs/promises');
const base = process.env.QA_URL || 'http://127.0.0.1:3058';

(async () => {
  const browser = await chromium.launch({ channel: 'chrome', headless: true });
  const page = await browser.newPage({ reducedMotion: 'reduce', serviceWorkers: 'block' });
  const errors = [];
  const results = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.goto(base, { waitUntil: 'networkidle' });
  // Repeat desktop last to cover recalculation after a mobile resize.
  for (const [width, height] of [[1440,900],[1280,720],[1920,1080],[768,1024],[375,812],[320,568],[375,320],[1440,900]]) {
    await page.setViewportSize({ width, height });
    await page.waitForTimeout(150);
    const positions = [];
    for (let slide = 0; slide < 3; slide++) {
      await page.locator(`[data-slide="${slide}"]`).click();
      await page.waitForFunction(n => document.querySelector(`[data-slide="${n}"]`).getAttribute('aria-pressed') === 'true', slide);
      await page.waitForTimeout(50);
      const layout = await page.evaluate(() => {
        const top = document.querySelector('#hero').getBoundingClientRect().top;
        const rect = selector => {
          const r = document.querySelector(selector).getBoundingClientRect();
          return { y: r.top - top, height: r.height };
        };
        return { hero: rect('#hero'), title: rect('#hero-title'), description: rect('#hero-description'), actions: rect('.hero-actions'), controls: rect('.hero-bottom'), font: getComputedStyle(document.querySelector('#hero-title')).fontFamily, overflow: document.documentElement.scrollWidth > innerWidth };
      });
      assert.equal(layout.overflow, false, `${width}px slide ${slide + 1} overflow`);
      assert.match(layout.font, /Arial/);
      positions.push(layout);
    }
    for (const position of positions.slice(1)) {
      for (const key of ['hero','title','description','actions','controls']) {
        assert.ok(Math.abs(position[key].y - positions[0][key].y) <= 1, `${width}px ${key} moved`);
        assert.ok(Math.abs(position[key].height - positions[0][key].height) <= 1, `${width}px ${key} changed height`);
      }
    }
    results.push({ width, height, stable: true, positions });
  }
  assert.deepEqual(errors, []);
  await browser.close();
  await fs.writeFile('docs/landing-preview/carousel-layout.json', JSON.stringify({ base, results, errors }, null, 2));
  console.log(JSON.stringify(results.map(({width,height,stable}) => ({width,height,stable}))));
})().catch(error => { console.error(error); process.exit(1); });

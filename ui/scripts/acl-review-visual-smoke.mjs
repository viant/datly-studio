import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';
import { createServer } from 'vite';
import { chromium } from '../../../forge/node_modules/playwright/index.mjs';

const root = fileURLToPath(new URL('../', import.meta.url));
const server = await createServer({ root, server: { host: '127.0.0.1', port: 0 } });
let browser;
try {
  await server.listen();
  browser = await chromium.launch({ headless: true, channel: process.env.PLAYWRIGHT_CHANNEL || 'chrome' });
  const page = await browser.newPage({ viewport: { width: 390, height: 844 } });
  const apiRequests = [];
  page.on('request', (request) => { if (request.url().includes('/v1/studio/')) apiRequests.push(request.url()); });
  await page.goto(`http://127.0.0.1:${server.httpServer.address().port}/acl-review.html?frame=1&kind=report&scenario=editable`);
  await page.getByText('Policy revision 7').waitFor();
  const action = page.getByLabel('Permission action', { exact: true });
  assert.equal(await action.locator('option[value="preview"]').count(), 1);
  assert.equal(await action.locator('option[value="execute"]').count(), 1);
  await action.selectOption('preview');
  assert.equal(await page.getByLabel('Required entity scope').inputValue(), 'project');
  assert.equal(await page.getByRole('button', { name: /retrieve/ }).count(), 0);
  assert.deepEqual(apiRequests, []);
  if (process.env.ACL_REVIEW_SCREENSHOT) await page.screenshot({ path: process.env.ACL_REVIEW_SCREENSHOT, fullPage: true });
  await page.getByLabel('rule.1 value').selectOption('reader');
  await page.getByRole('button', { name: 'Review changes' }).click();
  const dialog = page.getByRole('dialog', { name: 'Review permission changes' });
  await dialog.waitFor();
  assert.match(await dialog.innerText(), /Policy revision 7|policy revision 7/);
  assert.match(await dialog.innerText(), /Current/);
  assert.match(await dialog.innerText(), /Proposed/);
  assert.deepEqual(apiRequests, []);
  await page.waitForTimeout(400); // Let the Blueprint dialog opening transition finish before visual QA.
  if (process.env.ACL_REVIEW_DIALOG_SCREENSHOT) await page.screenshot({ path: process.env.ACL_REVIEW_DIALOG_SCREENSHOT, fullPage: true });
  console.log('ACL review ✓ report preview/execute policies and pre-save diff at 390 px, no backend requests');
} finally {
  await browser?.close();
  await server.close();
}

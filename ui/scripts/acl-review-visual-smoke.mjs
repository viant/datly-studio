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

  const origin = `http://127.0.0.1:${server.httpServer.address().port}`;
  const resource = { kind: 'report', id: 'operations', tenant: 'one', version: '3' };
  const operations = [];
  await page.route('**/studio-config.json', route => route.fulfill({ status: 200, contentType: 'application/json',
    body: JSON.stringify({ mode: 'authenticated', apiBaseURL: origin }) }));
  await page.route('**/v1/studio/sdk/*', route => {
    const request = route.request();
    const operation = new URL(request.url()).pathname.split('/').pop();
    operations.push({ operation, body: request.postDataJSON() });
    const payload = operation === 'access.get'
      ? { resource, revision: 12, policies: { preview: { mode: 'protected', rule: { kind: 'role', value: 'analyst' } } } }
      : operation === 'access.context' ? { canManage: true, choices: { role: [{ id: 'analyst', label: 'Analyst' }] } } : null;
    return route.fulfill({ status: payload ? 200 : 404, contentType: 'application/json', body: JSON.stringify(payload) });
  });
  await page.goto(`${origin}/acl-review.html?mode=live&kind=report&id=operations&tenant=one&version=3`);
  await page.getByText('Policy revision 12').waitFor();
  assert.equal(await page.getByRole('heading', { name: 'Permissions review' }).count(), 1);
  assert.equal(await page.getByText('Live · read-only').count(), 1);
  assert.equal(await page.getByLabel('Permission action', { exact: true }).inputValue(), 'preview');
  assert.equal(await page.getByRole('button', { name: 'Review changes' }).count(), 0);
  assert.equal(await page.getByText('Synthetic ACL preview').count(), 0);
  assert.deepEqual(operations.map(item => item.operation).sort(), ['access.context', 'access.get']);
  assert.deepEqual(operations.map(item => item.body), [resource, resource]);
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  assert.ok(overflow <= 1, `live ACL review overflows the 390 px viewport by ${overflow} px`);
  if (process.env.ACL_LIVE_REVIEW_SCREENSHOT) await page.screenshot({ path: process.env.ACL_LIVE_REVIEW_SCREENSHOT, fullPage: true });
  for (const width of [768, 1200]) {
    await page.setViewportSize({ width, height: 900 });
    const extra = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
    assert.ok(extra <= 1, `live ACL review overflows the ${width} px viewport by ${extra} px`);
  }
  console.log('ACL review ✓ live read-only report policy at 390/768/1200 px, exact SDK reads, no horizontal overflow');
} finally {
  await browser?.close();
  await server.close();
}

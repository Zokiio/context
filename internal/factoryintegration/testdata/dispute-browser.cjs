// Run against TestDisputeBrowserFixture with a private fixture directory argument.
// Playwright drives the real service; every worker is a deterministic subprocess.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const { execFileSync } = require('node:child_process');
const { chromium } = require('playwright');

async function main() {
  const directory = process.argv[2];
  assert(directory, 'Pass the private fixture directory');
  const readyBy = Date.now() + 60000;
  while (!fs.existsSync(path.join(directory, 'fixture.json')) && Date.now() < readyBy) await new Promise(resolve => setTimeout(resolve, 200));
  const fixture = JSON.parse(fs.readFileSync(path.join(directory, 'fixture.json'), 'utf8'));
  const token = fs.readFileSync(fixture.tokenFile, 'utf8').trim();
  const evidence = path.join(directory, 'evidence');
  fs.mkdirSync(evidence, { recursive: true, mode: 0o700 });
  const api = async endpoint => {
    const response = await fetch(fixture.url + '/v1' + endpoint, { headers: { Authorization: 'Bearer ' + token }, signal: AbortSignal.timeout(10000) });
    const result = await response.json();
    assert(response.ok, endpoint + ': ' + JSON.stringify(result));
    return result;
  };
  const waitFor = async (read, ready, label, milliseconds = 60000) => {
    const deadline = Date.now() + milliseconds;
    let result;
    while (Date.now() < deadline) {
      result = await read();
      if (ready(result)) return result;
      await new Promise(resolve => setTimeout(resolve, 200));
    }
    throw new Error('Timed out waiting for ' + label + ': ' + JSON.stringify(result).slice(0, 2000));
  };
  const defaultChrome = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
  const executablePath = process.env.BROWSER_EXECUTABLE || (fs.existsSync(defaultChrome) ? defaultChrome : undefined);
  const browser = await chromium.launch({ headless: true, executablePath });
  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
  await context.addInitScript(value => sessionStorage.setItem('waymark-token', value), token);
  const page = await context.newPage();
  const errors = [], results = [];
  page.on('pageerror', error => errors.push(error.message));
  page.on('dialog', dialog => dialog.accept());
  const item = '/projects/' + fixture.projectId + '/work-items/' + fixture.workItemId;
  const screenshot = name => page.screenshot({ path: path.join(evidence, name + '.png'), fullPage: true });
  const record = (name, detail = {}) => { results.push({ name, ...detail }); console.log('PASS ' + name); };
  try {
    const original = await waitFor(() => api('/jobs/' + fixture.jobId), job => job.status === 'verification_failed', 'original protected check to fail');
    assert(original.checks.some(check => check.status === 'failed'));
    await page.goto(fixture.url + '/#run/' + fixture.workItemId);
    await page.getByRole('button', { name: 'Report a dispute', exact: true }).click();
    await page.getByRole('textbox', { name: 'Dispute summary', exact: true }).fill('The protected expectation requires empty content, contradicting requirement 1.');
    await page.getByRole('textbox', { name: 'Affected requirement numbers', exact: true }).fill('1');
    await page.getByRole('textbox', { name: 'Affected test paths', exact: true }).fill('tests/marker.expect');
    await page.getByRole('textbox', { name: 'Observed failure', exact: true }).fill('The independent check rejected the candidate because the protected expectation requires empty marker content.');
    await page.getByRole('textbox', { name: 'Evidence references', exact: true }).fill(original.attempts.at(-1).id + '\ntests/marker.expect');
    // A reporter can identify the defect without knowing its correction.
    await page.getByRole('button', { name: 'Send dispute to judge', exact: true }).click();
    await page.locator('.dispute-list .next-owner').getByText('Judge', { exact: true }).waitFor();
    await page.getByRole('button', { name: /Judge the dispute/ }).waitFor();
    await screenshot('judge-running');
    record('Human report starts an independent judge and appears as a conditional Run step');

    await page.locator('.dispute-list .next-owner').getByText('Test author', { exact: true }).waitFor();
    await screenshot('judge-to-test-author');
    let activity = await waitFor(() => api(item + '/activity'), value => value.disputes?.[0]?.status === 'tests_proposed', 'test proposal');
    let dispute = activity.disputes[0];
    assert.equal(dispute.decision.kind, 'revise_tests');
    assert.equal(dispute.nextOwner, 'you');
    assert.equal(activity.jobs.length, 1, 'human approval precedes replacement admission');
    await page.locator('.dispute-list .next-owner').getByText('You', { exact: true }).waitFor();
    await page.locator('.dispute-list').getByText('The protected expectation contradicts the approved requirement for actual marker content.', { exact: true }).waitFor();
    await page.locator('.dispute-list').getByText('Corrected the protected expectation to require nonempty marker content.', { exact: true }).waitFor();
    await screenshot('judge-awaiting-test-approval');
    await page.setViewportSize({ width: 390, height: 844 });
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1));
    await screenshot('judge-awaiting-test-approval-mobile');
    await page.setViewportSize({ width: 1440, height: 1000 });
    record('Judge routes to test author, then waits for human approval with rationale, evidence and patch visible', { disputeId: dispute.id, expectationId: dispute.expectationId });

    await page.getByRole('textbox', { name: 'Reason for your decision', exact: true }).fill('The corrected test matches the approved requirement and preserves acceptance strength.');
    await page.getByRole('button', { name: 'Approve tests and start replacement run', exact: true }).click();
    await page.locator('.dispute-list .next-owner').getByText('Builder', { exact: true }).waitFor();
    activity = await waitFor(() => api(item + '/activity'), value => value.disputes?.[0]?.replacementJobId, 'linked replacement run');
    dispute = activity.disputes[0];
    assert.equal(dispute.status, 'resolved');
    assert.equal(dispute.resolution, 'accept_tests');
    const replacement = await waitFor(() => api('/jobs/' + dispute.replacementJobId), job => job.status === 'ready_for_review', 'fresh replacement verification');
    const retained = await api('/jobs/' + original.id);
    assert.equal(retained.status, 'superseded');
    assert.equal(retained.admission.baseRevision, original.admission.baseRevision);
    assert.equal(retained.admission.ticketSHA256, original.admission.ticketSHA256);
    assert(retained.checks.some(check => check.status === 'failed'));
    assert.equal(replacement.admission.parentJobId, original.id);
    assert.equal(replacement.admission.ticketSHA256, original.admission.ticketSHA256);
    assert(replacement.checks.some(check => check.status === 'passed'));
    assert.equal(replacement.reviews.length, 0, 'old approvals are never copied');
    const gitFile = revision => execFileSync('git', ['show', revision + ':tests/marker.expect'], { cwd: fixture.repository, encoding: 'utf8' });
    assert.equal(gitFile(original.admission.baseRevision), 'empty\n');
    assert.equal(gitFile(replacement.admission.baseRevision), 'nonempty\n');
    const card = page.locator('.dispute-list');
    assert.equal(await card.getByRole('link', { name: /Original run/ }).getAttribute('href'), '#job/' + original.id);
    assert.equal(await card.getByRole('link', { name: /Replacement run/ }).getAttribute('href'), '#job/' + replacement.id);
    await screenshot('judge-replacement-verified');
    record('Approval creates a linked replacement, preserves original failed evidence, and runs fresh checks', { originalJobId: original.id, replacementJobId: replacement.id, originalBase: original.admission.baseRevision, replacementBase: replacement.admission.baseRevision });

    await page.getByRole('link', { name: 'History', exact: true }).click();
    await page.locator('.history-entry > summary').filter({ hasText: 'Judge · Correct the tests' }).click();
    await page.locator('[aria-label="History"]').getByText('The corrected test matches the approved requirement and preserves acceptance strength.', { exact: true }).waitFor();
    assert.equal(await page.locator('[aria-label="History"]').getByRole('link', { name: /Replacement run/ }).getAttribute('href'), '#job/' + replacement.id);
    await screenshot('judge-retained-history');
    await page.getByRole('link', { name: 'Run', exact: true }).click();
    await page.setViewportSize({ width: 390, height: 844 });
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1));
    await screenshot('judge-mobile');
    assert.deepEqual(errors, []);
    record('History retains judge and human decisions; Run fits a narrow viewport');
    fs.writeFileSync(path.join(evidence, 'results.json'), JSON.stringify({ results, errors }, null, 2));
  } catch (error) {
    fs.writeFileSync(path.join(evidence, 'failure.txt'), String(error.stack) + '\n' + await page.locator('body').innerText());
    await screenshot('failure');
    throw error;
  } finally {
    await browser.close();
    fs.writeFileSync(path.join(directory, 'done'), 'done\n', { mode: 0o600 });
  }
}
main().catch(error => { console.error(error); process.exitCode = 1; });

// Run against TestWorkspaceBrowserFixture; see docs/factory-platform.md.
// Requires Playwright and an installed Chromium. All agents are offline fixtures.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const { execFileSync } = require('node:child_process');
const { chromium } = require('playwright');

async function main() {
  const directory = process.argv[2];
  assert(directory, 'Pass the private fixture directory');
  const readyBy = Date.now() + 30000;
  while (!fs.existsSync(path.join(directory, 'fixture.json')) && Date.now() < readyBy) await new Promise(resolve => setTimeout(resolve, 200));
  const fixture = JSON.parse(fs.readFileSync(path.join(directory, 'fixture.json'), 'utf8'));
  const token = fs.readFileSync(fixture.tokenFile, 'utf8').trim();
  const evidence = path.join(directory, 'evidence');
  fs.mkdirSync(evidence, { recursive: true, mode: 0o700 });
  const results = [];
  const record = (name, detail = {}) => { results.push({ name, ...detail }); console.log('PASS ' + name); };
  const api = async (endpoint, body, method = body ? 'POST' : 'GET') => {
    const response = await fetch(fixture.url + '/v1' + endpoint, {
      method, headers: { Authorization: 'Bearer ' + token, 'Content-Type': 'application/json' },
      body: body ? JSON.stringify(body) : undefined, signal: AbortSignal.timeout(10000),
    });
    const value = await response.json();
    assert(response.ok, method + ' ' + endpoint + ': ' + JSON.stringify(value));
    return value;
  };
  const waitFor = async (read, check, label, timeout = 45000) => {
    const deadline = Date.now() + timeout;
    let value;
    while (Date.now() < deadline) {
      value = await read();
      if (check(value)) return value;
      await new Promise(resolve => setTimeout(resolve, 200));
    }
    throw new Error('Timed out: ' + label + ', last: ' + JSON.stringify(value).slice(0,1500));
  };
  const project = '/projects/' + fixture.projectId;
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
  await context.addInitScript(value => sessionStorage.setItem('waymark-token', value), token);
  const page = await context.newPage();
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  let acceptDialog = true;
  page.on('dialog', dialog => acceptDialog ? dialog.accept() : dialog.dismiss());
  const navigate = async hash => { await page.evaluate(value => { location.hash = value; }, hash); };
  const screenshot = async name => page.screenshot({ path: path.join(evidence, name + '.png'), fullPage: true });
  let taskID, item;
  try {
    await page.goto(fixture.url);
    await page.getByRole('button', { name: 'Add work', exact: true }).first().click();
    await page.getByRole('textbox', { name: 'Title', exact: true }).fill('Task workspace browser acceptance');
    const description = '## A readable task\n\nKeep **important details** and `code` easy to read.\n\n- Save before queueing\n- Follow the running loop\n\n[Source](https://example.com/spec)\n\n```go\n' + 'fmt.Println("fixture output")\n'.repeat(28) + '```\n\n<script>window.markdownExecuted = true</script>\n\n[Unsafe](javascript:alert(1))';
    await page.getByRole('textbox', { name: 'What and why', exact: true }).fill(description);
    const criterion = page.getByRole('textbox', { name: 'Requirement 1', exact: true });
    await criterion.fill('Produce waymark-fixture.txt with the admitted job and requirement identities.');
    await criterion.evaluate(node => { node.focus(); node.setSelectionRange(17, 17); });
    await page.waitForTimeout(1000);
    assert.equal(await criterion.evaluate(node => document.activeElement === node && node.selectionStart), 17);
    await page.getByRole('button', { name: 'Preview', exact: true }).click();
    assert.equal(await page.locator('.description-preview').getByRole('heading', { name: 'A readable task', exact: true }).textContent(), 'A readable task');
    assert.equal(await page.locator('.description-preview li').count(), 2);
    assert.equal(await page.locator('.description-preview pre code').count(), 1);
    assert.equal(await page.locator('.description-preview a[href^="javascript:"]').count(), 0);
    assert.equal(await page.evaluate(() => window.markdownExecuted), undefined);
    await screenshot('task-preview-desktop');
    await page.getByRole('button', { name: 'Save draft', exact: true }).click();
    await page.waitForURL(/#item\//);
    taskID = page.url().split('#item/')[1]; item = project + '/work-items/' + taskID;
    let task = await api(item);
    assert.equal(task.queued, false);
    assert.equal((await api(item + '/activity')).jobs.length, 0);
    record('Save creates a draft without execution; Markdown is safe; scoring preserves caret', { taskID });

    await page.getByRole('button', { name: 'Edit task', exact: true }).click();
    await page.getByRole('textbox', { name: 'Title', exact: true }).fill('Task workspace edited locally');
    acceptDialog = false;
    await page.getByRole('link', { name: 'Back to board', exact: true }).click();
    await page.waitForTimeout(200);
    assert(page.url().endsWith('#item/' + taskID));
    assert.equal(await page.getByRole('textbox', { name: 'Title', exact: true }).inputValue(), 'Task workspace edited locally');
    acceptDialog = true;
    await page.getByRole('link', { name: 'Run', exact: true }).click();
    await page.getByRole('link', { name: 'Task', exact: true }).click();
    assert.equal(await page.getByRole('textbox', { name: 'Title', exact: true }).inputValue(), 'Task workspace edited locally');
    task = await api(item);
    await api(item, { requestId: 'browser-concurrent-edit', workItemId: taskID, ifTicketSHA256: task.ticketSHA256, draft: { ...task.draft, description: 'Saved concurrently by another editor.' } }, 'PUT');
    await page.getByRole('button', { name: 'Save changes', exact: true }).click();
    await page.getByRole('heading', { name: 'Latest saved task' }).waitFor();
    assert.equal(await page.getByRole('textbox', { name: 'Title', exact: true }).inputValue(), 'Task workspace edited locally');
    await screenshot('stale-save-comparison');
    await page.getByRole('button', { name: 'Keep my draft and use this revision', exact: true }).click();
    await page.getByRole('button', { name: 'Save changes', exact: true }).click();
    await page.getByRole('button', { name: 'Edit task', exact: true }).waitFor();
    assert.equal((await api(item)).draft.title, 'Task workspace edited locally');
    record('Unsaved navigation and tab changes preserve draft; stale save supports comparison and reapply');

    await page.getByRole('button', { name: 'Queue task', exact: true }).click();
    await waitFor(() => api(item), value => value.queued, 'task queued');
    await page.getByRole('button', { name: 'Remove from queue', exact: true }).click();
    await waitFor(() => api(item), value => !value.queued, 'task unqueued');
    await page.getByRole('button', { name: 'Archive', exact: true }).click();
    await waitFor(() => api(item), value => value.archived, 'task archived');
    await page.getByRole('button', { name: 'Restore as draft', exact: true }).click();
    await waitFor(() => api(item), value => !value.archived && !value.queued, 'task restored unqueued');
    record('Queue, unqueue, archive and restore retain identity and restore unqueued');

    for (const width of [768, 390]) {
      await page.setViewportSize({ width, height: 900 });
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), 'Task horizontal overflow at ' + width);
      await screenshot('task-' + width);
    }
    await page.setViewportSize({ width: 1440, height: 1000 });
    await page.getByRole('button', { name: 'Queue task', exact: true }).click();
    await waitFor(() => api(item), value => value.queued, 'task requeued');
    // As with ordinary authored record edits, commit the fixture requirements
    // before using them as a Git base. The service never commits human edits.
    execFileSync('git', ['add', 'records'], { cwd: fixture.repository });
    execFileSync('git', ['commit', '-m', 'Author the browser acceptance task'], { cwd: fixture.repository, stdio: 'pipe' });
    fs.writeFileSync(path.join(directory, 'fail-next-check'), 'controlled failure', { mode: 0o600 });
    await navigate('decision/' + taskID);
    await page.getByRole('button', { name: 'Agree and start', exact: true }).click();
    await navigate('run/' + taskID);
    let activity = await waitFor(() => api(item + '/activity'), value => value.jobs.some(job => job.attempts.some(attempt => attempt.progress)), 'actual worker progress');
    const jobID = activity.jobs[0].id;
    await page.getByText('Preparing the fixture change', { exact: false }).first().waitFor();
    await screenshot('run-live-progress');
    const frozen = await api(item);
    assert(frozen.frozen, 'Active task must be frozen');
    await navigate('item/' + taskID);
    await waitFor(() => page.getByRole('button', { name: 'Edit task', exact: true }).isDisabled(), Boolean, 'frozen editor');
    await navigate('run/' + taskID);
    await page.locator('.stage-step').first().click();
    const selectedStage = await page.locator('.stage-step[aria-pressed="true"]').textContent();
    await waitFor(() => api(item + '/activity'), value => value.jobs[0]?.status === 'verification_failed', 'controlled check failure');
    await page.getByRole('button', { name: 'Request repair', exact: true }).waitFor();
    assert.equal(await page.locator('.stage-step[aria-pressed="true"]').textContent(), selectedStage);
    assert.equal(await page.getByRole('checkbox', { name: 'Follow active step' }).isChecked(), false);
    await screenshot('run-failed-check');
    await page.getByRole('button', { name: 'Request repair', exact: true }).click();
    activity = await waitFor(() => api(item + '/activity'), value => value.jobs[0]?.status === 'ready_for_review', 'repaired checks pass');
    assert.equal(activity.jobs[0].repairCount, 1);
    assert(activity.jobs[0].attempts.some(attempt => attempt.checkStatus === 'failed'));
    record('Real worker progress, frozen requirements, failed check and repair; manual stage selection persists', { jobID });

    await navigate('decision/' + taskID);
    await page.getByRole('button', { name: 'Approve and merge', exact: true }).click();
    activity = await waitFor(() => api(item + '/activity'), value => value.jobs[0]?.merge?.status === 'merged', 'verified merge', 60000);
    const job = activity.jobs[0];
    const mainRevision = execFileSync('git', ['rev-parse', 'main'], { cwd: fixture.repository, encoding: 'utf8' }).trim();
    assert.equal(job.merge.revision, mainRevision);
    assert(job.attempts.some(attempt => attempt.purpose === 'merge' && attempt.checkStatus === 'passed'));
    await navigate('history/' + taskID);
    await page.getByText('Original requirements and context', { exact: true }).click();
    const output = page.locator('.history-entry .artifact summary').first();
    await output.click();
    await page.locator('.history-entry .artifact[open] pre').first().waitFor();
    await screenshot('retained-history-and-output');
    record('Approval performs merge verification; history retains original failure, revisions and full output', { candidate: job.candidateRevision, merge: mainRevision });

    await navigate('run/' + taskID);
    await context.setOffline(true);
    await page.locator('.freshness.disconnected').waitFor({ timeout: 12000 });
    await screenshot('connection-lost');
    await context.setOffline(false);
    await waitFor(() => page.locator('.freshness').textContent(), value => value.startsWith('Connected'), 'reconnect');
    for (const width of [768, 390]) {
      await page.setViewportSize({ width, height: 900 });
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), 'Run horizontal overflow at ' + width);
      await screenshot('run-' + width);
    }
    await page.setViewportSize({ width: 1440, height: 1000 });
    record('Disconnect/reconnect is explicit; task and run views fit tablet and mobile widths');

    // The existing committed task provides a separate cancellable build.
    const stopItem = project + '/work-items/' + fixture.workItemId;
    await navigate('decision/' + fixture.workItemId);
    await page.getByRole('button', { name: 'Agree and start', exact: true }).click();
    await navigate('run/' + fixture.workItemId);
    await waitFor(() => api(stopItem + '/activity'), value => value.jobs.some(job => job.attempts.some(attempt => attempt.progress)), 'second worker starts');
    await page.getByRole('button', { name: 'Stop run', exact: true }).click();
    const stopped = await waitFor(() => api(stopItem + '/activity'), value => value.jobs[0]?.status === 'cancelled', 'confirmed stop');
    assert.equal(stopped.held, true);
    assert.equal((await api(stopItem)).stopped, true);
    await page.getByText('Run stopped. Queue this task explicitly to allow another run.', { exact: true }).waitFor();
    await screenshot('run-confirmed-stop');
    record('Stop remains distinct from termination and retains an explicit pickup hold');

    await navigate('new');
    await page.getByRole('textbox', { name: 'Title', exact: true }).fill('Disposable never-run draft');
    await page.getByRole('textbox', { name: 'Requirement 1', exact: true }).fill('Delete this unreferenced draft.');
    await page.getByRole('button', { name: 'Save draft', exact: true }).click();
    await page.waitForURL(/#item\//);
    const disposable = page.url().split('#item/')[1];
    await page.getByRole('button', { name: 'Delete draft', exact: true }).click();
    await page.waitForURL(/#board$/);
    assert(!(await api(project + '/work-items')).some(value => value.workItemId === disposable));
    assert.equal(errors.length, 0, errors.join('\n'));
    record('Unreferenced never-run draft deletes; no browser JavaScript errors');
    fs.writeFileSync(path.join(evidence, 'results.json'), JSON.stringify({ fixture: 'offline', results, errors }, null, 2));
    fs.writeFileSync(path.join(directory, 'done'), 'complete');
  } catch (error) {
    await screenshot('failure').catch(() => {});
    fs.writeFileSync(path.join(evidence, 'failure.txt'), error.stack + '\n\n' + await page.locator('body').innerText());
    throw error;
  } finally { await browser.close(); }
}

main().catch(error => { console.error(error.stack); process.exitCode = 1; });

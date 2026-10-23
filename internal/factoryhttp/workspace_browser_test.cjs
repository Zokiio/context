// Run with Node and Playwright installed. BROWSER_EXECUTABLE can select local Chrome.
// These browser regressions use controlled API responses; the real-service trial
// separately establishes execution, authorization and Git publication behavior.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const http = require('node:http');
const path = require('node:path');
const { chromium } = require('playwright');

const project = { id: 'project', title: 'Workspace fixture', policy: { ask: { autopilot: false }, allowedExecutors: ['fixture'], maxRepairAttempts: 2, maxJobDurationSeconds: 3600 } };
const now = () => new Date().toISOString();
const store = { records: [], activity: { jobs: [], refinements: [], expectations: [], held: false }, offline: false, createBodies: [], scoreRequests: 0, updates: [], staleStop: false, stopAttempts: 0, jobReads: 0, admin: true, capabilities: ['read', 'coordinate', 'review', 'accept', 'merge'] };
const draftRecord = (id, draft) => ({ projectId: project.id, workItemId: id, path: id + '.md', ticketSHA256: 'revision-1', draft, queued: false, archived: false, frozen: false, canDelete: true, dependents: [], actions: ['queue', 'archive', 'delete'] });
const board = () => ({ project, publicationPending: !!store.publicationPending, pilot: false, running: 0, mergedToday: 0, generatedAt: now(), cards: store.records.filter((record) => record.queued).map((record) => ({ workItemId: record.workItemId, title: record.draft.title, column: 'next', state: 'waiting', status: 'Ready', step: 0, decision: ['ready_for_review', 'ready_for_acceptance'].includes(store.activity.jobs[0]?.status) ? 'approve' : 'plan', jobId: store.activity.jobs[0]?.id, updatedAt: '2026-10-23T00:00:00Z', line: 'Ready to start' })) });
const stages = () => ({ workItemId: 'task-1', title: store.records[0]?.draft.title, costUsd: 0, steps: [{ key: 'sharpen', title: 'Sharpen requirements', kind: 'agent', status: 'done', summary: 'Requirements are clear.' }, { key: 'build', title: 'Build the change', kind: 'agent', status: store.activity.jobs[0]?.status === 'cancelled' ? 'failed' : 'running', summary: 'Fixture builder', notes: ['## Agent note\n\n- Real reported text\n\n```go\nreturn 1\n```'] }] });
const server = http.createServer(async (request, response) => {
	const url = new URL(request.url, 'http://localhost');
	if (!url.pathname.startsWith('/v1')) {
		const file = url.pathname === '/' ? 'index.html' : url.pathname.replace('/assets/', '');
		if (!/^(index\.html|app\.(js|css)|fonts\/[\w.-]+)$/.test(file)) { response.writeHead(404).end(); return; }
		response.setHeader('Content-Type', file.endsWith('.js') ? 'text/javascript' : file.endsWith('.css') ? 'text/css' : file.endsWith('.woff2') ? 'font/woff2' : 'text/html');
		response.end(fs.readFileSync(path.join(__dirname, 'assets', file))); return;
	}
	const route = url.pathname.slice(3);
	let body = ''; for await (const chunk of request) body += chunk;
	body = body ? JSON.parse(body) : {};
	const send = (value, status = 200) => { response.writeHead(status, { 'Content-Type': 'application/json' }); response.end(JSON.stringify(value)); };
	if (route === '/me') return send({ userId: 'operator', username: 'Operator', kind: 'human', installationAdmin: store.admin });
	if (route === '/projects') return send([project]);
	if (route.endsWith('/capabilities')) return send(store.capabilities);
	if (route.endsWith('/criteria/score')) {
		store.scoreRequests++;
		// Older scoring replies deliberately arrive after newer requests.
		await new Promise((resolve) => setTimeout(resolve, body.criteria[0]?.includes('older') ? 1200 : 20));
		return send({ status: 'observed', criteria: body.criteria.map((text) => ({ text, flagged: text.includes('older') })) });
	}
	if (store.offline) return send({ message: 'Fixture disconnected' }, 503);
	if (route.endsWith('/board')) return send(board());
	if (route === '/projects/project/work-items' && request.method === 'GET') return send(store.records);
	if (route === '/projects/project/work-items' && request.method === 'POST') {
		store.createBodies.push(body); const record = draftRecord('task-' + (store.records.length + 1), body.draft); store.records.push(record); return send(record);
	}
	if (route.endsWith('/activity')) return send({ projectId: project.id, workItemId: 'task-1', generatedAt: now(), truncated: false, ...store.activity });
	if (route.endsWith('/run')) return send(stages());
	if (route === '/projects/project/work-items/reconcile') { store.publicationPending = false; return send(store.records); }
	if (route === '/projects/project/plan') { store.activity.jobs[0].status = 'implementing'; store.activity.jobs[0].version++; return send({ job: store.activity.jobs[0] }); }
	if (route === '/projects/project/orientation') return send({ criterionSignals: {} });
	if (route === '/jobs/job-1/change') return send({ files: [], patch: '' });
	if (route === '/jobs/job-1/artifacts/log-1') return send({ id: 'log-1', content: 'Fixture build output\nline 2\n' });
	if (route === '/jobs/job-1') { store.jobReads++; return send(store.activity.jobs[0]); }
	if (route === '/jobs/job-1/actions') {
		store.stopAttempts++;
		if (store.staleStop) { store.staleStop = false; store.activity.jobs[0].version++; return send({ code: 'stale', message: 'job version changed' }, 409); }
		if (body.ifVersion !== store.activity.jobs[0].version) return send({ code: 'stale', message: 'job version changed' }, 409);
		store.activity.jobs[0].status = 'cancel_requested'; store.activity.jobs[0].version++; store.activity.held = true; return send(store.activity.jobs[0]); }
	const match = route.match(/^\/projects\/project\/work-items\/([^/]+)(?:\/(disposition))?$/);
	if (match) {
		const record = store.records.find((entry) => entry.workItemId === match[1]);
		if (!record) return send({ message: 'Missing task' }, 404);
		if (request.method === 'PUT') {
			store.updates.push(body);
			if (body.ifTicketSHA256 !== record.ticketSHA256) return send({ code: 'conflict', message: 'The task changed since it was read.' }, 409);
			record.draft = body.draft; record.ticketSHA256 += '-saved'; return send(record);
		}
		if (match[2]) {
			if (body.action === 'queue') { record.queued = true; record.actions = ['unqueue', 'archive']; }
			if (body.action === 'unqueue') { record.queued = false; record.actions = ['queue', 'archive', 'delete']; }
			if (body.action === 'archive') { record.queued = false; record.archived = true; record.actions = ['restore']; }
			if (body.action === 'restore') { record.archived = false; record.queued = false; record.actions = ['queue', 'archive', 'delete']; }
			if (body.action === 'delete') { store.records = store.records.filter((entry) => entry !== record); return send({ ...record, deleted: true }); }
			return send(record);
		}
		return send(record);
	}
	send({ message: 'Unexpected fixture request: ' + route }, 404);
});

(async () => {
	await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
	const defaultChrome = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
	const executablePath = process.env.BROWSER_EXECUTABLE || (fs.existsSync(defaultChrome) ? defaultChrome : undefined);
	const browser = await chromium.launch({ headless: true, executablePath });
	const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });
	const evidenceDir = process.env.WORKSPACE_BROWSER_EVIDENCE_DIR;
	if (evidenceDir) fs.mkdirSync(evidenceDir, { recursive: true });
	const capture = async (name) => { if (evidenceDir) await page.screenshot({ path: path.join(evidenceDir, name + '.png'), fullPage: true }); };
	const errors = []; page.on('pageerror', (error) => errors.push(error.message));
	let acceptDialog = true;
	page.on('dialog', (dialog) => acceptDialog ? dialog.accept() : dialog.dismiss());
	try {
		await page.goto('http://127.0.0.1:' + server.address().port);
		await page.getByLabel('Your access token').fill('fixture-token');
		await page.getByRole('button', { name: 'Open the board', exact: true }).click();
		await page.getByRole('button', { name: 'Add work', exact: true }).first().click();
		await page.getByRole('textbox', { name: 'Title', exact: true }).fill('Persistent task');
		await page.getByRole('textbox', { name: 'What and why', exact: true }).fill('# Readable task\n\n- First point\n- Second point\n\n[Docs](https://example.com/docs) [unsafe](javascript:alert)\n\n<script>window.unsafe=true</script>\n\n```js\nconst safe = true;\n```');
		await page.getByRole('textbox', { name: 'Requirement 1', exact: true }).fill('Return the checked result.');
		await page.getByRole('button', { name: 'Preview', exact: true }).click();
		await page.getByRole('heading', { name: 'Readable task', exact: true }).waitFor();
		assert.equal(await page.locator('.description-preview a[href^="javascript:"]').count(), 0);
		assert.equal(await page.evaluate(() => window.unsafe), undefined);
		assert.equal(await page.locator('.description-preview pre code').textContent(), 'const safe = true;');
		await page.getByRole('button', { name: 'Save draft', exact: true }).click();
		await page.getByRole('button', { name: 'Edit task', exact: true }).waitFor();
		assert.equal(store.createBodies[0].queue, false, 'saving must explicitly request an unqueued draft');
		assert.equal(store.activity.jobs.length, 0);
		assert.equal((await page.locator('.task-document').innerText()).split('\n').includes('null'), false, 'optional content must not render a null text node');
		await capture('task-desktop');
		await page.getByRole('button', { name: 'Edit task', exact: true }).click();
		const criterion = page.getByRole('textbox', { name: 'Requirement 1', exact: true });
		await criterion.fill('older score response');
		await page.waitForTimeout(680);
		await criterion.fill('Current checkable requirement');
		await criterion.evaluate((node) => { window.retainedCriterion = node; node.setSelectionRange(7, 7); });
		await page.waitForTimeout(1700);
		assert.equal(await criterion.evaluate((node) => node === window.retainedCriterion && document.activeElement === node && node.selectionStart === 7), true, 'scoring preserves textarea identity, focus and caret');
		assert.equal(await page.getByText('Looks vague', { exact: true }).count(), 0, 'an older scoring response must not replace current feedback');
		await page.getByRole('link', { name: 'Run', exact: true }).click();
		await page.getByRole('button', { name: /Sharpen requirements/ }).click();
		await page.getByRole('link', { name: 'History', exact: true }).click();
		await page.getByRole('link', { name: 'Task', exact: true }).click();
		assert.equal(await criterion.inputValue(), 'Current checkable requirement');
		assert.equal(await criterion.evaluate((node) => node === window.retainedCriterion), true, 'internal tabs retain the editor DOM');
		acceptDialog = false;
		await page.getByRole('link', { name: 'Board', exact: true }).click();
		await page.waitForTimeout(100);
		assert.equal(new URL(page.url()).hash, '#item/task-1', 'declining navigation keeps the task open');
		acceptDialog = true;
		await page.getByRole('link', { name: 'Board', exact: true }).click();
		await page.getByRole('link', { name: 'Persistent task Draft', exact: true }).click();
		assert.equal(await criterion.inputValue(), 'Current checkable requirement', 'accepted internal navigation retains the cached draft');
		store.records[0].ticketSHA256 = 'remote-revision'; store.records[0].draft.description = 'Latest remote description';
		await page.getByRole('button', { name: 'Save changes', exact: true }).click();
		await page.getByRole('heading', { name: 'Latest saved task', exact: true }).waitFor();
		assert.equal(await criterion.inputValue(), 'Current checkable requirement');
		await page.getByRole('button', { name: 'Keep my draft and use this revision', exact: true }).click();
		await page.getByRole('button', { name: 'Save changes', exact: true }).click();
		await page.getByRole('button', { name: 'Edit task', exact: true }).waitFor();
		assert.equal(store.updates.at(-1).ifTicketSHA256, 'remote-revision');
		store.records[0].frozen = true; store.records[0].frozenReason = 'A worker is running.';
		await page.waitForFunction(() => document.querySelector('.task-document button')?.disabled === true);
		store.records[0].frozen = false;
		await page.waitForFunction(() => document.querySelector('.task-document button')?.disabled === false);
		store.activity.jobs = [{ id: 'job-1', version: 1, workItemId: 'task-1', projectId: project.id, status: 'implementing', createdAt: now(), updatedAt: now(), admission: { ticketSHA256: 'original-requirements', baseRevision: 'base', sources: [{ path: 'original.md', text: 'Original admitted requirements' }] }, attempts: [{ id: 'attempt-1', stage: 'implementation', ordinal: 1, status: 'running', startedAt: now(), lastHeartbeatAt: now(), inputRevision: 'base', progress: 'First heartbeat', artifacts: [{ id: 'log-1', name: 'stdout', bytes: 28 }] }], events: [], reviews: [], checks: [], availableActions: ['cancel'] }];
		await page.getByRole('link', { name: 'Run', exact: true }).click();
		await page.locator('.live-activity').getByText('First heartbeat', { exact: true }).waitFor();
		assert.equal(await page.getByRole('button', { name: /Sharpen requirements/ }).getAttribute('aria-pressed'), 'true', 'manual stage selection survives tab changes');
		await page.locator('.live-activity details.artifact summary').click();
		await page.getByText('Fixture build output', { exact: false }).waitFor();
		store.activity.jobs[0].attempts[0].progress = 'Second heartbeat'; store.activity.jobs[0].version++;
		await page.locator('.live-activity').getByText('Second heartbeat', { exact: true }).waitFor({ timeout: 7000 });
		assert.equal(await page.locator('.live-activity details.artifact').getAttribute('open'), '', 'open output survives heartbeat updates');
		assert.equal(await page.getByRole('button', { name: /Sharpen requirements/ }).getAttribute('aria-pressed'), 'true');
		await capture('run-desktop');
		store.offline = true;
		await page.getByText(/Disconnected · last received/).waitFor({ timeout: 7000 });
		store.offline = false;
		await page.getByText(/Connected · updated/).waitFor({ timeout: 7000 });
		store.staleStop = true; store.activity.jobs[0].version++;
		await page.getByRole('button', { name: 'Stop run', exact: true }).click();
		await page.getByText('Stop requested. Waiting for workers to confirm termination.', { exact: true }).waitFor({ timeout: 7000 });
		assert.equal(store.stopAttempts, 2, 'a stop races safely with one worker version update');
		assert.equal(store.jobReads, 2, 'each stop attempt reads the current job version');
		store.activity.jobs[0].status = 'cancelled'; store.activity.jobs[0].version++; store.activity.jobs[0].availableActions = []; store.records[0].stopped = true;
		await page.getByText('Run stopped. Queue this task explicitly to allow another run.', { exact: true }).waitFor({ timeout: 7000 });
		await page.getByRole('link', { name: 'History', exact: true }).click();
		await page.getByText('original-requirements', { exact: true }).waitFor();
		await capture('history-desktop');
		await page.locator('.history-entry > summary').first().click();
		store.activity.jobs[0].version++;
		await page.waitForTimeout(3300);
		assert.equal(await page.locator('.history-entry').first().evaluate((node) => node.open), false, 'a manually collapsed job remains closed across polling');
		await page.setViewportSize({ width: 390, height: 844 });
		await page.getByRole('link', { name: 'Task', exact: true }).click();
		await capture('task-mobile');
		assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), true, 'workspace fits a narrow viewport');

		// Permission checks use fresh browser sessions, just as a new token does.
		store.admin = false; store.records[0].queued = true; store.records[0].stopped = false; store.records[0].actions = ['edit', 'start', 'unqueue', 'archive']; store.activity.held = false;
		const decisionJob = store.activity.jobs[0]; decisionJob.candidateRevision = 'candidate-1'; decisionJob.coordinatorId = 'coordinator'; decisionJob.implementerId = 'worker'; decisionJob.admission.requiredChecks = ['fixture']; decisionJob.availableActions = ['cancel'];
		let session = 0;
		const asCapabilities = async (capabilities, status, view = 'item') => {
			store.capabilities = ['read', ...capabilities]; decisionJob.status = status; decisionJob.version++;
			await page.goto('http://127.0.0.1:' + server.address().port + '/?permissions=' + (++session) + '#' + view + '/task-1');
			await page.getByText(/Connected · updated/).waitFor();
		};
		await page.setViewportSize({ width: 1440, height: 1000 });
		await asCapabilities(['accept'], 'ready_for_review');
		assert.equal(await page.getByRole('link', { name: 'Review result', exact: true }).count(), 0, 'accept alone cannot approve an unreviewed result');
		await page.evaluate(() => { location.hash = '#decision/task-1'; });
		await page.getByText('Approving requires review and accept access, and an independent human reviewer.', { exact: true }).waitFor();
		assert.equal(await page.getByRole('button', { name: 'Approve', exact: true }).count(), 0);
		assert.equal(await page.getByRole('button', { name: 'Reject', exact: true }).count(), 0);
		await asCapabilities(['review'], 'ready_for_review');
		await page.getByRole('link', { name: 'Review result', exact: true }).click();
		await page.getByRole('button', { name: 'Ask for a fix', exact: true }).click();
		await page.getByRole('textbox', { name: 'What should change', exact: true }).fill('Preserve this unsent review note.');
		assert.equal(await page.getByRole('button', { name: 'Approve', exact: true }).count(), 0, 'review alone does not expose combined acceptance');
		assert.equal(await page.getByRole('button', { name: 'Reject', exact: true }).count(), 0, 'reject requires coordinate access');
		decisionJob.status = 'ready_for_acceptance'; decisionJob.version++;
		await page.getByText('Decision changed; reload to continue', { exact: true }).waitFor({ timeout: 7000 });
		assert.equal(await page.getByRole('textbox', { name: 'What should change', exact: true }).inputValue(), 'Preserve this unsent review note.');
		assert.equal(await page.getByRole('button', { name: 'Send back for a fix', exact: true }).isDisabled(), true, 'stale decision actions are disabled');
		await page.getByRole('button', { name: 'Reload decision', exact: true }).click();
		await page.getByText('Accepting this reviewed result requires accept access and a human account.', { exact: true }).waitFor();
		assert.equal(await page.getByRole('button', { name: 'Ask for a fix', exact: true }).count(), 0, 'completed reviews cannot be sent back through the review endpoint');
		decisionJob.coordinatorId = 'operator';
		await asCapabilities(['accept'], 'ready_for_acceptance');
		await page.getByRole('link', { name: 'Review result', exact: true }).click();
		await page.getByRole('button', { name: 'Accept result', exact: true }).waitFor();
		assert.equal(await page.getByRole('button', { name: 'Accept result', exact: true }).isEnabled(), true, 'the coordinator can accept an already reviewed candidate');
		decisionJob.coordinatorId = 'coordinator';
		await asCapabilities(['review', 'accept'], 'ready_for_review');
		await page.getByRole('link', { name: 'Review result', exact: true }).click();
		await page.getByRole('button', { name: 'Approve', exact: true }).waitFor();
		await asCapabilities([], 'implementing', 'decision');
		await page.getByText('You can read this task, but making planning decisions requires coordinate access and a human account.', { exact: true }).waitFor();
		assert.equal(await page.getByRole('button', { name: 'Agree and start', exact: true }).count(), 0, 'direct planning route does not grant coordinate access');
		await asCapabilities(['coordinate', 'review', 'accept'], 'implementing', 'decision');
		await page.getByRole('button', { name: 'Agree and start', exact: true }).click();
		await page.waitForURL('**#run/task-1');
		decisionJob.status = 'ready_for_review'; decisionJob.version++;
		await page.getByRole('link', { name: 'Review result', exact: true }).waitFor({ timeout: 7000 });
		await page.getByRole('link', { name: 'Review result', exact: true }).click();
		await page.getByRole('button', { name: 'Approve', exact: true }).waitFor();
		assert.equal(await page.getByRole('button', { name: 'Agree and start', exact: true }).count(), 0, 'returning after a completed planning decision loads current approval');
		assert.equal(await page.getByText('Decision changed; reload to continue', { exact: true }).isVisible(), false, 'submitted actions are not unsaved input');
		store.publicationPending = true;
		await page.getByRole('link', { name: 'Board', exact: true }).click();
		await page.getByRole('button', { name: 'Resume pending save', exact: true }).click();
		await page.getByRole('heading', { name: 'Task library', exact: true }).waitFor();
		assert.equal(store.publicationPending, false, 'pending creation can be recovered from the board without opening a task');
		decisionJob.merge = { status: 'verifying' };
		await asCapabilities(['merge'], 'accepted', 'run');
		assert.equal(await page.getByRole('button', { name: 'Retry merge', exact: true }).count(), 0);
		assert.equal(await page.getByRole('button', { name: 'Merge', exact: true }).count(), 0, 'an in-flight merge cannot start again');
		decisionJob.merge = null; decisionJob.version++;
		await page.getByRole('button', { name: 'Merge', exact: true }).waitFor({ timeout: 7000 });
		decisionJob.merge = { status: 'failed' }; decisionJob.version++;
		await page.getByRole('button', { name: 'Retry merge', exact: true }).waitFor({ timeout: 7000 });

		assert.deepEqual(errors, []);
		console.log('Workspace browser regressions passed: safe Markdown, draft save, focus/caret, stale replies and saves, tab state, live polling, output expansion, reconnect, stop states, history, narrow layout, permission matrix, stale decision input and merge-state controls.');
	} catch (error) { console.error('Browser errors:', errors); console.error((await page.locator('body').innerText()).slice(-6000)); throw error; } finally { await browser.close(); server.close(); }
})().catch((error) => { console.error(error); process.exitCode = 1; server.close(); });

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
const store = { records: [], activity: { jobs: [], refinements: [], expectations: [], disputes: [], held: false }, offline: false, createBodies: [], disputeBodies: [], resolutions: [], scoreRequests: 0, updates: [], staleStop: false, stopAttempts: 0, jobReads: 0, admin: true, capabilities: ['read', 'coordinate', 'review', 'accept', 'merge'] };
const draftRecord = (id, draft) => ({ projectId: project.id, workItemId: id, path: id + '.md', ticketSHA256: 'revision-1', draft, queued: false, archived: false, frozen: false, canDelete: true, dependents: [], actions: ['queue', 'archive', 'delete'] });
const board = () => ({ project, publicationPending: !!store.publicationPending, pilot: false, running: 0, mergedToday: 0, generatedAt: now(), cards: store.records.filter((record) => record.queued).map((record) => ({ workItemId: record.workItemId, title: record.draft.title, column: 'next', state: 'waiting', status: 'Ready', step: 0, decision: ['ready_for_review', 'ready_for_acceptance'].includes(store.activity.jobs[0]?.status) ? 'approve' : 'plan', jobId: store.activity.jobs[0]?.id, updatedAt: '2026-10-23T00:00:00Z', line: 'Ready to start' })) });
const stages = () => ({ workItemId: 'task-1', title: store.records[0]?.draft.title, costUsd: 0, steps: [{ key: 'sharpen', title: 'Sharpen requirements', kind: 'agent', status: 'done', summary: 'Requirements are clear.' }, { key: 'build', title: 'Build the change', kind: 'agent', status: store.activity.jobs[0]?.status === 'cancelled' ? 'failed' : 'running', summary: 'Fixture builder', notes: ['## Agent note\n\n- Real reported text\n\n```go\nreturn 1\n```'] }, ...(store.activity.disputes.length ? [{ key: 'judge', title: store.activity.disputes[0].status === 'reviewing_tests' || store.activity.disputes[0].testReviews?.length ? 'Review the test proposal' : 'Judge the dispute', kind: 'agent', status: ['judging', 'reviewing_tests', 'authoring_tests'].includes(store.activity.disputes[0].status) ? 'running' : store.activity.disputes[0].status === 'resolved' ? 'done' : 'needs_you', summary: 'Recorded judge routing.' }] : [])] });
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
	if (route.endsWith('/activity')) return send({ projectId: project.id, workItemId: 'task-1', generatedAt: now(), truncated: false, ...store.activity, disputes: store.activity.disputes.map((dispute) => ({ ...dispute, availableActions: store.capabilities.includes('coordinate') ? dispute.availableActions : [] })) });
	if (route.endsWith('/run')) return send(stages());
	if (route === '/projects/project/work-items/reconcile') { store.publicationPending = false; return send(store.records); }
	if (route === '/projects/project/plan') { store.activity.jobs[0].status = 'implementing'; store.activity.jobs[0].version++; return send({ job: store.activity.jobs[0] }); }
	if (route === '/projects/project/orientation') return send({ criterionSignals: {} });
	if (route === '/jobs/job-1/change') return send({ files: [], patch: '' });
	if (route === '/jobs/job-1/artifacts/log-1') return send({ id: 'log-1', content: 'Fixture build output\nline 2\n' });
	if (route === '/jobs/job-1') { store.jobReads++; return send(store.activity.jobs[0]); }
	if (route === '/jobs/job-1/disputes') {
		store.disputeBodies.push(body);
		if (store.staleReport) { store.staleReport = false; store.activity.jobs[0].version++; return send({ code: 'stale', message: 'Run changed.' }, 409); }
		if (body.ifVersion !== store.activity.jobs[0].version) return send({ code: 'stale', message: 'Run changed.' }, 409);
		const dispute = { id: 'dispute-1', projectId: project.id, workItemId: 'task-1', jobId: 'job-1', version: 1, status: 'judging', input: body.dispute, nextOwner: 'judge', createdAt: now(), updatedAt: now(), availableActions: [] };
		store.activity.disputes = [dispute]; store.activity.jobs[0].status = 'disputed'; store.activity.jobs[0].version++; store.activity.jobs[0].availableActions = ['cancel']; return send(dispute);
	}
	if (route === '/disputes/dispute-1/resolve') {
		store.resolutions.push(body); const dispute = store.activity.disputes[0];
		if (store.staleResolve) { store.staleResolve = false; dispute.version++; return send({ code: 'stale', message: 'Decision changed.' }, 409); }
		if (body.ifVersion !== dispute.version) return send({ code: 'stale', message: 'Decision changed.' }, 409);
		if (body.action === 'review_tests') { dispute.status = 'reviewing_tests'; dispute.nextOwner = 'judge'; dispute.version++; dispute.availableActions = []; return send(dispute); }
		dispute.status = 'resolved'; dispute.nextOwner = 'implementer'; dispute.replacementJobId = 'job-2'; dispute.version++; dispute.availableActions = []; return send(dispute);
	}
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

		decisionJob.merge = null; decisionJob.availableActions = ['report_dispute'];
		await asCapabilities(['coordinate'], 'verification_failed', 'run');
		await page.getByRole('button', { name: 'Report a dispute', exact: true }).click();
		await page.getByRole('textbox', { name: 'Dispute summary', exact: true }).fill('The protected test contradicts requirement 1.');
		await page.getByRole('textbox', { name: 'Affected requirement numbers', exact: true }).fill('1');
		await page.getByRole('textbox', { name: 'Affected test paths', exact: true }).fill('strutil/wordfreq_holdout_test.go');
		await page.getByRole('textbox', { name: 'Observed failure', exact: true }).fill('The test counts empty words.');
		await page.getByRole('textbox', { name: 'Evidence references', exact: true }).fill('attempt-1/log-1');
		const correction = page.getByRole('textbox', { name: 'Proposed correction', exact: true });
		await correction.fill('Keep the requirement and correct the empty-word assertion.');
		await correction.evaluate((node) => { window.disputeInput = node; node.setSelectionRange(4, 4); });
		decisionJob.version++;
		await page.getByText('Run changed; review before sending', { exact: true }).waitFor({ timeout: 7000 });
		assert.equal(await correction.evaluate((node) => node === window.disputeInput && document.activeElement === node && node.selectionStart === 4), true, 'polling preserves dispute input DOM and caret');
		assert.equal(await page.getByRole('button', { name: 'Send dispute to judge', exact: true }).isDisabled(), true);
		await page.getByRole('link', { name: 'History', exact: true }).click();
		await page.getByRole('link', { name: 'Run', exact: true }).click();
		assert.equal(await correction.inputValue(), 'Keep the requirement and correct the empty-word assertion.');
		await page.getByRole('button', { name: 'Use current run version', exact: true }).click();
		store.staleReport = true;
		await page.getByRole('button', { name: 'Send dispute to judge', exact: true }).click();
		await page.getByText('Run changed; review before sending', { exact: true }).waitFor();
		assert.equal(await page.getByRole('button', { name: 'Send dispute to judge', exact: true }).isDisabled(), true, 'a stale submission keeps the report disabled until reviewed');
		assert.equal(await correction.inputValue(), 'Keep the requirement and correct the empty-word assertion.');
		await page.getByRole('button', { name: 'Use current run version', exact: true }).click();
		const submittedVersion = decisionJob.version;
		await page.getByRole('button', { name: 'Send dispute to judge', exact: true }).click();
		await page.locator('.dispute-list .next-owner').getByText('Judge', { exact: true }).waitFor();
		assert.equal(store.disputeBodies.at(-1).ifVersion, submittedVersion);
		assert.deepEqual(store.disputeBodies[0].dispute.criterionRefs, [1]);
		assert.ok(store.disputeBodies[0].requestId, 'dispute submission is an idempotent command');
		await page.getByRole('button', { name: /Judge the dispute/ }).waitFor();
		const dispute = store.activity.disputes[0];
		dispute.status = 'needs_human'; dispute.nextOwner = 'you'; dispute.version++;
		dispute.decision = { kind: 'revise_tests', rationale: 'The approved criterion excludes empty tokens.', evidenceRefs: ['frozen-ticket:1', 'attempt-1/log-1'], preservesIntent: true, weakensAcceptance: false, directive: 'Correct the empty-token case without changing the intended behavior.' };
		dispute.availableActions = ['revise_tests', 'return_to_planning', 'dismiss'];
		await page.locator('.dispute-list .next-owner').getByText('You', { exact: true }).waitFor({ timeout: 7000 });
		assert.equal(await page.locator('.dispute-list').getByText('The approved criterion excludes empty tokens.', { exact: true }).count(), 1);
		const rationale = page.getByRole('textbox', { name: 'Reason for your decision', exact: true });
		await rationale.fill('I reviewed the original criterion and the reported failure.');
		await rationale.evaluate((node) => { window.judgeInput = node; });
		dispute.status = 'authoring_tests'; dispute.nextOwner = 'test_author'; dispute.availableActions = []; dispute.version++;
		await page.locator('.dispute-list .next-owner').getByText('Test author', { exact: true }).waitFor({ timeout: 7000 });
		dispute.status = 'tests_proposed'; dispute.nextOwner = 'you'; dispute.version++; dispute.expectationId = 'expectation-2'; dispute.availableActions = ['accept_tests', 'dismiss'];
		store.activity.expectations = [{ id: 'expectation-2', status: 'proposed', createdAt: now(), notes: 'Changed only the incorrect expectation.', patch: 'diff --git a/wordfreq_test.go b/wordfreq_test.go\n--- a/wordfreq_test.go\n+++ b/wordfreq_test.go\n@@ -1 +1 @@\n-want := 1\n+want := 0\n' }];
		await page.locator('.dispute-list .next-owner').getByText('You', { exact: true }).waitFor({ timeout: 7000 });
		await page.getByText('Judge decision changed; review the latest state', { exact: true }).waitFor();
		assert.equal(await rationale.evaluate((node) => node === window.judgeInput), true);
		assert.equal(await rationale.inputValue(), 'I reviewed the original criterion and the reported failure.');
		assert.equal(await page.getByRole('button', { name: 'Ask test author to correct tests', exact: true }).isDisabled(), true);
		await page.getByRole('button', { name: 'Review latest judge decision', exact: true }).click();
		await page.getByRole('button', { name: 'Approve tests and start replacement run', exact: true }).waitFor();
		assert.equal(await page.locator('.dispute-list').getByText('Changed only the incorrect expectation.', { exact: true }).isVisible(), true, 'test patch and author notes are visible before approval');
		store.staleResolve = true;
		await page.getByRole('button', { name: 'Approve tests and start replacement run', exact: true }).click();
		await page.getByText('Judge decision changed; review the latest state', { exact: true }).waitFor();
		assert.equal(await page.getByRole('button', { name: 'Approve tests and start replacement run', exact: true }).isDisabled(), true, 'a server stale response blocks reapplying an outdated decision');
		await page.getByRole('button', { name: 'Review latest judge decision', exact: true }).click();
		await page.getByRole('button', { name: 'Approve tests and start replacement run', exact: true }).click();
		await page.locator('.dispute-list .next-owner').getByText('Builder', { exact: true }).waitFor();
		assert.equal(store.resolutions.at(-1).ifVersion, dispute.version - 1);
		assert.equal(store.resolutions.at(-1).rationale, 'I reviewed the original criterion and the reported failure.');
		assert.equal(await page.locator('.dispute-list').getByRole('link', { name: 'Original run job-1', exact: true }).getAttribute('href'), '#job/job-1');
		assert.equal(await page.locator('.dispute-list').getByRole('link', { name: 'Replacement run job-2', exact: true }).getAttribute('href'), '#job/job-2');
		await capture('judge-desktop');
		await page.getByRole('link', { name: 'History', exact: true }).click();
		await page.locator('.history-entry > summary').filter({ hasText: 'Judge · Correct the tests' }).click();
		assert.equal(await page.locator('[aria-label="History"]').getByText('The approved criterion excludes empty tokens.', { exact: true }).isVisible(), true);
		dispute.status = 'needs_human'; dispute.nextOwner = 'you'; dispute.availableActions = ['return_to_planning']; dispute.version++;
		await asCapabilities([], 'disputed', 'run');
		await page.locator('.dispute-list .next-owner').getByText('You', { exact: true }).waitFor();
		assert.equal(await page.getByRole('button', { name: 'Return task to planning', exact: true }).count(), 0, 'read-only viewers cannot resolve a dispute');
		assert.equal(await page.getByRole('textbox', { name: 'Reason for your decision', exact: true }).isVisible(), false);
		await page.setViewportSize({ width: 390, height: 844 });
		assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), true, 'judge view fits a narrow viewport');
		await capture('judge-mobile');

		// A saved legacy proposal can enter the new judge review loop. Its original
		// verdict and both reviewed revisions remain distinct as ownership changes.
		dispute.status = 'tests_proposed'; dispute.nextOwner = 'you'; dispute.availableActions = ['review_tests']; dispute.replacementJobId = ''; dispute.testProposalCount = 1; dispute.version++;
		await asCapabilities([], 'disputed', 'run');
		assert.equal(await page.getByRole('button', { name: 'Ask judge to review tests', exact: true }).count(), 0, 'read-only access cannot route a legacy proposal');
		await asCapabilities(['coordinate'], 'disputed', 'run');
		await page.getByRole('textbox', { name: 'Reason for your decision', exact: true }).fill('Ask the independent judge to assess the existing proposal.');
		const legacyVersion = dispute.version;
		await page.getByRole('button', { name: 'Ask judge to review tests', exact: true }).click();
		await page.locator('.dispute-list').getByText('Judge is reviewing the finished test proposal', { exact: true }).waitFor();
		assert.equal(store.resolutions.at(-1).action, 'review_tests');
		assert.equal(store.resolutions.at(-1).ifVersion, legacyVersion);
		await page.getByRole('button', { name: /Review the test proposal/ }).waitFor();
		assert.equal(await page.getByRole('button', { name: 'Approve tests and start replacement run', exact: true }).count(), 0, 'active judge review does not expose a competing manual action');
		const firstReview = { expectationId: 'expectation-2', commit: 'first-reviewed-commit', round: 1, decision: { kind: 'revise_tests', rationale: 'The first proposal still omits empty input.', directive: 'Preserve the corrected case and add empty-input coverage.', evidenceRefs: ['tests/empty_case'], preservesIntent: true, weakensAcceptance: false }, run: { model: 'fixture-judge', turns: 2, costUsd: .02 }, nextOwner: 'test_author', createdAt: now() };
		dispute.testReviews = [firstReview]; dispute.status = 'authoring_tests'; dispute.nextOwner = 'test_author'; dispute.testProposalCount = 2; dispute.expectationId = 'expectation-3'; dispute.version++;
		store.activity.expectations.push({ id: 'expectation-3', status: 'running', createdAt: now() });
		await page.locator('.dispute-list .next-owner').getByText('Test author', { exact: true }).waitFor({ timeout: 7000 });
		const reviews = page.locator('.dispute-list .test-proposal-review');
		await reviews.getByRole('heading', { name: 'Improve the tests', exact: true }).waitFor();
		assert.equal(await reviews.getByText('first-reviewed-commit', { exact: true }).count(), 1);
		assert.equal(await page.locator('.dispute-list .original-dispute-verdict').getByText('The approved criterion excludes empty tokens.', { exact: true }).count(), 1);
		const priorProposalSummary = reviews.getByText('Test author proposal expectation-', { exact: true });
		if (!await priorProposalSummary.evaluate((node) => node.parentElement.open)) await priorProposalSummary.click();
		await reviews.getByText('Changed only the incorrect expectation.', { exact: true }).waitFor();
		const improved = store.activity.expectations.at(-1); improved.status = 'proposed'; improved.notes = 'Added empty-input coverage without dropping the original correction.'; improved.patch = 'diff --git a/test.go b/test.go\n--- a/test.go\n+++ b/test.go\n@@ -1 +1,2 @@\n keepOriginalCase()\n+checkEmptyInput()\n';
		dispute.status = 'reviewing_tests'; dispute.nextOwner = 'judge'; dispute.version++;
		await page.locator('.dispute-list .next-owner').getByText('Judge', { exact: true }).waitFor({ timeout: 7000 });
		dispute.testReviews.push({ expectationId: 'expectation-3', commit: 'second-reviewed-commit', round: 2, decision: { kind: 'accept_tests', rationale: 'The improved tests preserve both the original case and the empty-input requirement.', evidenceRefs: ['tests/original_case', 'tests/empty_case'], preservesIntent: true, weakensAcceptance: false }, run: { model: 'fixture-judge', turns: 1, costUsd: .01 }, nextOwner: 'implementer', createdAt: now() });
		dispute.status = 'resolved'; dispute.nextOwner = 'implementer'; dispute.replacementJobId = 'job-3'; dispute.resolution = 'accept_tests'; dispute.delegated = 'judge_test_review'; dispute.rationale = 'Judge accepted the exact second proposal.'; dispute.version++;
		await page.locator('.dispute-list .next-owner').getByText('Builder', { exact: true }).waitFor({ timeout: 7000 });
		assert.equal(await reviews.count(), 2);
		await reviews.getByRole('heading', { name: 'Accept the tests', exact: true }).waitFor();
		assert.equal(await reviews.getByText('second-reviewed-commit', { exact: true }).count(), 1);
		assert.equal(await page.getByText('Recorded human decision', { exact: true }).count(), 0, 'delegated judge routing is not described as human approval');
		await page.locator('.dispute-list').getByText('Delegated: judge_test_review', { exact: true }).waitFor();
		await capture('proposal-review-mobile');
		await page.setViewportSize({ width: 1440, height: 1000 });
		await capture('proposal-review-desktop');
		await page.getByRole('link', { name: 'History', exact: true }).click();
		await page.locator('.history-entry > summary').filter({ hasText: 'Judge · Correct the tests' }).click();
		const historyReviews = page.locator('[aria-label="History"] .test-proposal-review');
		assert.equal(await historyReviews.count(), 2, 'history retains each proposal review');
		await page.locator('[aria-label="History"]').getByText('Earlier test proposal review 1', { exact: true }).click();
		await historyReviews.getByText('The first proposal still omits empty input.', { exact: true }).waitFor();
		await historyReviews.getByText('The improved tests preserve both the original case and the empty-input requirement.', { exact: true }).waitFor();

		assert.deepEqual(errors, []);
		console.log('Workspace browser regressions passed: task editing, safe formatting, live runs, permissions, stop, disputes, judge handoffs, stale decisions, retained input and mobile layout.');
	} catch (error) { console.error('Browser errors:', errors); console.error((await page.locator('body').innerText()).slice(-6000)); throw error; } finally { await browser.close(); server.close(); }
})().catch((error) => { console.error(error); process.exitCode = 1; server.close(); });

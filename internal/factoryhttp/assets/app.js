const $ = (selector) => document.querySelector(selector);
const list = (value) => Array.isArray(value) ? value : [];
const state = { token: '', me: null, projects: [], permissions: new Map(), job: null, tab: 'overview', view: '', generation: 0 };
const pending = new Map();
let mutationsInFlight = 0;
try { state.token = sessionStorage.getItem('waymark-token') || ''; } catch {}

function element(tag, attributes = {}, ...children) {
	const result = document.createElement(tag);
	for (const [name, value] of Object.entries(attributes)) {
		if (name.startsWith('on')) result.addEventListener(name.slice(2), value);
		else if (name === 'class') result.className = value;
		else if (name in result) result[name] = value;
		else result.setAttribute(name, value);
	}
	for (const child of children.flat()) if (child !== null && child !== undefined) result.append(child instanceof Node ? child : document.createTextNode(String(child)));
	return result;
}
const button = (text, action, kind = '') => element('button', { type: 'button', class: kind, onclick: (event) => run(action, event.currentTarget) }, text);
const code = (value) => element('span', { class: 'mono' }, value || 'Not available');
const short = (value) => value ? String(value).slice(0, 12) : 'Pending';
const time = (value) => value ? new Date(value).toLocaleString() : 'Not recorded';
const human = (value) => String(value || 'unknown').replaceAll('_', ' ');
function badge(status) {
	let kind = '';
	if (['accepted', 'passed', 'ready', 'approved', 'succeeded'].includes(status)) kind = 'good';
	if (/unknown|pending|queued|requested|implementing|verifying|stale|unavailable/.test(status)) kind = 'warn';
	if (/failed|rejected|changes_requested/.test(status)) kind = 'bad';
	if (status === 'fixture') kind = 'fixture';
	return element('span', { class: 'badge ' + kind }, human(status));
}
function panel(title, content, note = '') {
	return element('section', { class: 'panel' }, element('div', { class: 'panel-head' }, element('h2', {}, title), element('small', {}, note)), content);
}
function empty(title, detail) {
	return element('div', { class: 'empty' }, element('div', { class: 'empty-symbol', 'aria-hidden': 'true' }, '◇'), element('strong', {}, title), detail);
}
function table(headers, rows) {
	return element('div', { class: 'table-scroll' }, element('table', {}, element('thead', {}, element('tr', {}, headers.map((title) => element('th', {}, title)))), element('tbody', {}, rows.map((cells) => element('tr', {}, cells.map((cell) => element('td', {}, cell)))))));
}
function metadata(entries) {
	return element('dl', { class: 'metadata' }, entries.flatMap(([label, value]) => [element('dt', {}, label), element('dd', {}, value ?? 'Not available')]));
}
function pageHead(title, description, actions = []) {
	return element('div', { class: 'page-head' }, element('div', {}, element('span', { class: 'eyebrow' }, 'WAYMARK WORKSPACE'), element('h1', {}, title), element('p', {}, description)), element('div', { class: 'head-actions' }, actions));
}
function field(label, name, options = {}) {
	const { choices, help, full, ...attributes } = options;
	let input;
	if (choices) input = element('select', { name, ...attributes }, choices.map((choice) => element('option', { value: choice.value }, choice.label)));
	else if (attributes.type === 'textarea') { delete attributes.type; input = element('textarea', { name, ...attributes }); }
	else input = element('input', { type: 'text', name, ...attributes });
	const id = 'field-' + crypto.randomUUID(); input.id = id;
	return element('div', { class: 'field' + (full ? ' full' : '') }, element('label', { htmlFor: id }, label), input, help ? element('span', { class: 'fine' }, help) : null);
}
function formGrid(...fields) { return element('div', { class: 'form-grid' }, fields); }
function data(form) { return Object.fromEntries(new FormData(form)); }
function notice(message, error = false) {
	const target = $('#notice'); target.hidden = !message; target.textContent = message; target.className = error ? 'error' : '';
}
async function run(action, target) {
	if (target) target.disabled = true;
	try { await action(); }
	catch (error) { notice(error.message || 'The operation could not be completed.', true); }
	finally { if (target) target.disabled = false; }
}
async function api(path, options = {}) {
	const controller = new AbortController(); const timeout = setTimeout(() => controller.abort(), 30000);
	try {
		const response = await fetch('/v1' + path, { method: options.method || 'GET', headers: { Authorization: 'Bearer ' + state.token, Accept: 'application/json', ...(options.body ? { 'Content-Type': 'application/json' } : {}) }, body: options.body ? JSON.stringify(options.body) : undefined, credentials: 'omit', redirect: 'error', signal: controller.signal });
		const value = response.status === 204 ? null : await response.json();
		if (!response.ok) {
			const error = new Error((value.message || 'Request failed') + (value.code ? ' [' + value.code + ']' : '')); error.code = value.code; error.status = response.status;
			if (response.status === 401 && state.me) signout();
			throw error;
		}
		return value;
	} catch (error) {
		if (error.name === 'AbortError' || error instanceof TypeError) throw new Error('Connection interrupted. The outcome may be pending. Refresh before retrying; an unchanged request keeps its request identity.');
		throw error;
	} finally { clearTimeout(timeout); }
}
async function command(path, body, method = 'POST') {
	const identity = method + path + JSON.stringify(body);
	if (!pending.has(identity)) pending.set(identity, crypto.randomUUID());
	mutationsInFlight++;
	try {
		const result = await api(path, { method, body: { ...body, requestId: pending.get(identity) } });
		pending.delete(identity); notice(''); return result;
	} catch (error) {
		if (error.status >= 400 && error.status < 500) pending.delete(identity);
		throw error;
	} finally { mutationsInFlight--; }
}
function navigate(fragment) { if (location.hash === '#' + fragment) return route(); location.hash = fragment; }
function signout() {
	state.token = ''; state.me = null; state.job = null; state.generation++; pending.clear();
	try { sessionStorage.removeItem('waymark-token'); } catch {}
	$('#workspace').hidden = true; $('#signin').hidden = false; $('#signin-error').hidden = true; $('#token').value = '';
}
async function connect() {
	state.me = await api('/me');
	try { sessionStorage.setItem('waymark-token', state.token); } catch {}
	$('#token').value = ''; $('#signin').hidden = true; $('#workspace').hidden = false;
	$('#user-name').textContent = state.me.username; $('#user-role').textContent = state.me.installationAdmin ? 'Installation administrator' : human(state.me.kind);
	$('#user-initial').textContent = state.me.username.slice(0, 1).toUpperCase(); $('#nav-admin').hidden = !state.me.installationAdmin;
	await refreshProjects(); await route();
}
async function refreshProjects() {
	state.projects = list(await api('/projects'));
	const permissions = await Promise.all(state.projects.map(async (project) => [project.id, list(await api('/projects/' + encodeURIComponent(project.id) + '/capabilities'))]));
	state.permissions = new Map(permissions);
	$('#project-navigation').replaceChildren(...state.projects.map((project) => element('button', { class: 'nav-item', onclick: () => navigate('project/' + encodeURIComponent(project.id)) }, element('span', { 'aria-hidden': 'true' }, '◈'), project.title)));
	if (!state.projects.length) $('#project-navigation').append(element('div', { class: 'fine' }, 'No accessible projects'));
}
async function route() {
	if (!state.me) return;
	const generation = ++state.generation; const parts = location.hash.slice(1).split('/');
	state.view = parts[0] || 'queue'; state.job = null; notice('');
	$('#content').replaceChildren(element('div', { class: 'loading' }, element('span', { class: 'spinner', 'aria-hidden': 'true' }), 'Loading workspace…'));
	$('#nav-queue').classList.toggle('active', state.view === 'queue'); $('#nav-admin').classList.toggle('active', state.view === 'admin');
	try {
		let content;
		if (state.view === 'project') content = await projectPage(decodeURIComponent(parts[1] || ''), parts[2] || 'work');
		else if (state.view === 'job') content = await jobPage(decodeURIComponent(parts[1] || ''), parts[2] || 'overview');
		else if (state.view === 'admin') content = await adminPage(parts[1] || 'users');
		else content = await queuePage();
		if (generation === state.generation) $('#content').replaceChildren(...content);
	} catch (error) {
		if (generation !== state.generation) return;
		$('#content').replaceChildren(pageHead('Unable to open this view', 'The service has not returned an accessible result.'), element('div', { class: 'error', role: 'alert' }, error.message), button('Try again', route));
	}
}
async function queuePage() {
	$('#breadcrumb').textContent = 'Work queue';
	const jobs = list(await api('/jobs?limit=100'));
	state.queueVersion = JSON.stringify(jobs.map((job) => [job.id, job.version]));
	const active = jobs.filter((job) => !['accepted', 'cancelled', 'failed'].includes(job.status));
	const writable = state.projects.find((project) => state.permissions.get(project.id)?.includes('coordinate'));
	const statistics = [['Jobs shown', jobs.length], ['In progress', active.filter((job) => !/unknown|stale|review|acceptance/.test(job.status)).length], ['Awaiting decision', active.filter((job) => /review|acceptance/.test(job.status)).length], ['Need attention', active.filter((job) => /unknown|stale|failed/.test(job.status)).length]];
	const head = pageHead('Work queue', 'One place to see active work, evidence, and the decision that comes next.', [button('↻ Refresh', route), ...(writable ? [button('+ New job', () => navigate('project/' + encodeURIComponent(writable.id) + '/submit'), 'primary')] : [])]);
	const stats = element('div', { class: 'stat-grid' }, statistics.map(([label, value]) => element('div', { class: 'stat' }, element('div', { class: 'stat-label' }, label), element('div', { class: 'stat-value' }, value))));
	return [head, stats, panel('All accessible jobs', jobsTable(jobs), 'Latest 100 · checks for updates every 5 seconds'), element('p', { class: 'section-note' }, 'Acceptance follows independent evidence and review. An execution finishing does not mark its work accepted.')];
}
function jobsTable(jobs) {
	if (!jobs.length) return empty('Your queue is clear', state.projects.length ? 'Open a project to inspect ready work and submit its first job.' : 'An administrator must register a project and grant you access.');
	return table(['WORK ITEM', 'PROJECT', 'STATE', 'CANDIDATE', 'UPDATED', ''], jobs.map((job) => [element('div', {}, element('a', { class: 'row-link', href: '#job/' + encodeURIComponent(job.id) }, job.workItemId), element('div', { class: 'subline mono' }, short(job.id))), state.projects.find((project) => project.id === job.projectId)?.title || job.projectId, badge(job.status), code(short(job.candidateRevision)), time(job.updatedAt), element('a', { href: '#job/' + encodeURIComponent(job.id), 'aria-label': 'Inspect job ' + job.id }, '↗')]));
}
function tabs(choices, selected, change) { return element('div', { class: 'tabs' }, choices.map(([key, label]) => button(label, () => change(key), 'tab' + (selected === key ? ' active' : '')))); }
async function projectPage(id, tab) {
	const [view, capabilities] = await Promise.all([api('/projects/' + encodeURIComponent(id) + '/orientation'), api('/projects/' + encodeURIComponent(id) + '/capabilities')]);
	const project = view.project; const items = list(view.result.workItems); const can = (capability) => list(capabilities).includes(capability);
	$('#breadcrumb').textContent = project.title;
	const content = [pageHead(project.title, 'Authoritative records stay in the project bundle. The factory coordinates their execution.', [button('↻ Refresh', route), project.policy.fixture ? badge('fixture') : null].filter(Boolean)), tabs([['work', 'Work items'], ['jobs', 'Project jobs'], ...(can('coordinate') ? [['submit', 'Submit a job']] : []), ...(can('project_admin') ? [['access', 'Project access']] : [])], tab, (next) => navigate('project/' + encodeURIComponent(id) + '/' + next))];
	if (tab === 'submit') {
		if (!can('coordinate')) throw new Error('Coordination access is required to submit work.');
		content.push(submitForm(project, items));
	} else if (tab === 'jobs') content.push(panel('Project jobs', jobsTable(list(await api('/jobs?projectId=' + encodeURIComponent(id))))));
	else if (tab === 'access') { if (!can('project_admin')) throw new Error('Project administration access is required.'); content.push(await grantsForm(project)); }
	else {
		content.push(panel('Work items', items.length ? table(['WORK ITEM', 'READINESS', 'ELIGIBILITY', 'EXECUTION', ''], items.map((item) => [element('div', {}, element('strong', {}, item.title || item.id || 'Unidentified item'), element('div', { class: 'subline mono' }, item.id || item.source)), badge(item.readiness), item.eligible ? badge('eligible') : badge('not eligible'), human(item.execution), button('Inspect', () => inspectItem(item, project, can('coordinate')))])) : empty('No work items found', 'The project bundle has no visible work items.')));
		content.push(panel('Project policy', element('div', { class: 'panel-body' }, metadata([['Required checks', list(project.policy.requiredChecks).join(', ') || 'None configured'], ['Allowed executors', list(project.policy.allowedExecutors).join(', ') || 'None configured'], ['Repair maximum', project.policy.maxRepairAttempts], ['Job duration', project.policy.maxJobDurationSeconds + ' seconds'], ['Execution mode', project.policy.fixture ? 'Fixture project, simulated workers are labelled' : 'Ordinary project, fixture acceptance prohibited']]))));
		if (list(view.result.diagnostics).length) content.push(panel('Reader diagnostics', element('div', { class: 'panel-body' }, list(view.result.diagnostics).map((diagnostic) => element('p', { class: 'check-text' }, diagnostic.code + ': ' + diagnostic.message)))));
	}
	return content;
}
function inspectItem(item, project, canSubmit) {
	const root = $('#content');
	root.replaceChildren(pageHead(item.title || item.id, 'Current work readiness and selected requirement identities.', [button('Back to project', route)]), panel('Work item', element('div', { class: 'panel-body' }, metadata([['Stable ID', code(item.id)], ['Source', code(item.source)], ['Readiness', badge(item.readiness)], ['Eligible', String(item.eligible)], ['Ticket fingerprint', code(item.ticketSHA256)], ['Criteria fingerprint', code(item.criteriaSHA256)]]))), panel('Readiness checks', table(['CHECK', 'STATUS', 'REASONS'], list(item.checks).map((check) => [human(check.name), badge(check.status), list(check.reasons).map((reason) => reason.message).join(' · ')]))));
	if (list(item.exclusionReasons).length) root.append(element('div', { class: 'notice' }, item.exclusionReasons.map((reason) => reason.message).join(' ')));
	if (canSubmit && item.eligible && item.readiness === 'ready') root.append(submitForm(project, [item]));
}
function submitForm(project, items) {
	const allowed = items.filter((item) => item.id && !item.identityAmbiguous && item.eligible && item.readiness === 'ready');
	const form = element('form', {});
	if (!allowed.length) form.append(element('div', { class: 'notice' }, 'No ready and eligible work items are available. Inspect the work-item readiness checks before submitting.'));
	if (!list(project.policy.allowedExecutors).length) form.append(element('div', { class: 'notice' }, 'An administrator must allow an executor in project policy before submission.'));
	form.append(formGrid(field('Work item', 'workItemId', { choices: allowed.map((item) => ({ value: item.id, label: (item.title || item.id) + ' · ' + human(item.readiness) + (item.eligible ? '' : ' · not eligible') })), required: true, full: true }), field('Base revision', 'baseRevision', { required: true, value: 'HEAD', help: 'The service resolves and retains the exact commit.' }), field('Executor', 'executorKey', { choices: list(project.policy.allowedExecutors).map((key) => ({ value: key, label: key })), required: true }), field('Repair limit', 'repairLimit', { type: 'number', min: 0, max: project.policy.maxRepairAttempts, value: 0, required: true }), field('Deadline', 'deadline', { type: 'datetime-local', required: true, value: localDate(new Date(Date.now() + Math.min(project.policy.maxJobDurationSeconds || 3600, 3600) * 1000)) })));
	form.append(element('div', { class: 'notice' }, 'Required checks: ' + (list(project.policy.requiredChecks).join(', ') || 'none configured') + '. Project policy owns these checks; submission cannot remove them.'), element('p', { class: 'fine' }, project.policy.fixture ? 'FIXTURE PROJECT. This flow tests platform behavior and does not claim model execution.' : 'Fixture executions cannot satisfy acceptance for this ordinary project.'));
	const submit = element('button', { type: 'submit', class: 'primary', disabled: !allowed.length || !list(project.policy.allowedExecutors).length }, 'Admit and submit job');
	form.append(element('div', { class: 'form-actions' }, submit));
	form.addEventListener('submit', (event) => { event.preventDefault(); run(async () => { const values = data(form); const job = await command('/jobs', { projectId: project.id, workItemId: values.workItemId, baseRevision: values.baseRevision, executorKey: values.executorKey, repairLimit: Number(values.repairLimit), deadline: new Date(values.deadline).toISOString() }); await navigate('job/' + encodeURIComponent(job.id)); }, submit); });
	return panel('Submit ready work', element('div', { class: 'panel-body' }, element('p', { class: 'check-text' }, 'The service refreshes complete context and readiness, checks your permission, and acquires one implementation claim before dispatch.'), form));
}
function localDate(value) { return new Date(value.getTime() - value.getTimezoneOffset() * 60000).toISOString().slice(0, 16); }
async function jobPage(id, tab) {
	const job = await api('/jobs/' + encodeURIComponent(id));
	const capabilities = list(await api('/projects/' + encodeURIComponent(job.projectId) + '/capabilities'));
	state.job = job; state.tab = tab; $('#breadcrumb').textContent = 'Job · ' + short(job.id);
	const project = state.projects.find((entry) => entry.id === job.projectId);
	const headline = pageHead(job.workItemId, 'Job ' + job.id + ' · version ' + job.version, [badge(job.status), ...(job.status === 'accepted' && job.acceptanceStatus !== 'valid' ? [badge(job.acceptanceStatus || 'unknown')] : []), ...(project?.policy.fixture ? [badge('fixture')] : []), button('↻ Refresh', route)]);
	const content = [headline, tabs([['overview', 'Overview'], ['context', 'Requirements'], ['checks', 'Checks & evidence'], ['history', 'Event history'], ['reviews', 'Review & accept']], tab, (next) => navigate('job/' + encodeURIComponent(job.id) + '/' + next))];
	if (tab === 'context') {
		content.push(panel('Admitted context', element('div', {}, list(job.admission.sources).map((source) => element('details', { class: 'source' }, element('summary', {}, source.path), element('div', { class: 'fine mono' }, source.sha256), element('pre', {}, source.text)))), job.admission.contextComplete ? 'Complete captured sources' : 'INCOMPLETE'));
		content.push(panel('Requirement identities', element('div', { class: 'panel-body' }, metadata([['Ticket', code(job.admission.ticketSHA256)], ['Criteria', code(job.admission.criteriaSHA256)], ['Context', code(job.admission.contextSHA256)], ['Fingerprint version', job.admission.fingerprintVersion]]))));
	} else if (tab === 'checks') content.push(checksPanel(job));
	else if (tab === 'history') content.push(historyPanel(job));
	else if (tab === 'reviews') content.push(...reviewPanels(job, capabilities));
	else {
		const detail = element('div', {}, panel('Candidate & admission', element('div', { class: 'panel-body' }, metadata([['Candidate', code(job.candidateRevision)], ['Recorded base', code(job.admission.baseRevision)], ['Executor', job.admission.executorKey], ['Coordinator', code(job.coordinatorId)], ['Repairs used', job.repairCount + ' / ' + job.admission.repairLimit], ['Deadline', time(job.admission.deadline)], ['Acceptance', job.acceptanceStatus || 'Not recorded'], ...(job.acceptanceRecordPath ? [['Record', code(job.acceptanceRecordPath)]] : [])]))), attemptsPanel(job));
		const action = element('div', {}, actionsPanel(job, capabilities), panel('What comes next', element('div', { class: 'panel-body check-text' }, nextAction(job))), historyPanel({ ...job, events: list(job.events).slice(-4) }));
		content.push(element('div', { class: 'two-columns' }, detail, action), checksPanel(job));
	}
	if (project?.policy.fixture) content.push(element('p', { class: 'section-note' }, 'Fixture execution is visible throughout this project. Its evidence demonstrates the offline platform flow only.'));
	return content;
}
function nextAction(job) {
	if (job.status === 'accepted' && job.acceptanceStatus !== 'valid') {
		return job.acceptanceStatus === 'stale' ? 'Historical acceptance is retained, but current requirements or evidence no longer match it. Reassess the authored decision before relying on this work.' : 'Historical acceptance is retained, but its current decision or evidence cannot be validated. Restore the original decision and evidence together, or reassess the work.';
	}
	const descriptions = { queued: 'Waiting for an enrolled implementation worker to claim the recorded dispatch.', implementing: 'Implementation is active. Inspect its retained attempt or request cancellation.', implementation_unknown: 'The worker outcome is unknown. Keep the claim until an authorized reconciliation establishes what happened.', verifying: 'Waiting for an independent verification worker. Implementation success is not accepted evidence.', verification_failed: 'A required result has not passed. Inspect the original evidence before requesting a bounded repair.', ready_for_review: 'Independent checks passed. A different authorized reviewer must inspect this exact candidate.', ready_for_acceptance: 'A current independent review is recorded. An authorized acceptor can assess the exact candidate and requirements.', accepting: 'Acceptance publication is pending. The service must validate the authored record before declaring acceptance.', accepted: 'The authored acceptance record was validated. Its original revision and evidence remain inspectable.', cancel_requested: 'Stop requested. An acknowledgement or expired lease does not prove the execution stopped.', cancellation_unknown: 'Cancellation remains uncertain. No replacement should run while the old execution may still exist.', cancelled: 'The recorded execution is confirmed cancelled. Its workspace and observations remain retained.', failed: 'The job has a known failure. Inspect retained evidence and original requirements before new work.', stale: 'Requirements or candidate identity changed. Reassess before relying on previous checks or review.' };
	return descriptions[job.status] || 'Inspect retained state and ask an authorized coordinator for the next action.';
}
function attemptsPanel(job) {
	return panel('Execution attempts', list(job.attempts).length ? table(['STAGE', 'STATE', 'IDENTITY', 'WORKSPACE'], job.attempts.map((attempt) => [element('div', {}, human(attempt.stage), element('div', { class: 'subline' }, attempt.checkKey || 'Attempt ' + attempt.ordinal)), badge(attempt.status), element('div', { class: 'mono' }, attempt.nativeRunId || 'Awaiting native run', element('div', { class: 'subline mono' }, attempt.dispatchId)), code(attempt.workspacePath)])) : empty('No attempts yet', 'The service has not dispatched this job.'));
}
function checksPanel(job) {
	const checks = list(job.checks); const required = list(job.admission.requiredChecks);
	const rows = required.map((key) => {
		const check = [...checks].reverse().find((item) => item.checkKey === key && item.candidateRevision === job.candidateRevision && item.requirementFingerprint === job.admission.ticketSHA256);
		return [key, badge(check?.status || 'unknown'), code(short(check?.candidateRevision)), check?.environment || 'Not observed', check ? element('details', {}, element('summary', {}, 'Inspect evidence'), element('pre', { class: 'output' }, JSON.stringify(check, null, 2))) : 'Required result missing'];
	});
	const body = element('div', {}, rows.length ? table(['REQUIRED CHECK', 'RESULT', 'CANDIDATE', 'ENVIRONMENT', 'EVIDENCE'], rows) : empty('No required checks configured', 'The service still evaluates independent review and acceptance policy.'));
	if (checks.length) body.append(element('details', { class: 'source' }, element('summary', {}, 'All retained check observations'), element('pre', {}, JSON.stringify(checks, null, 2))));
	return panel('Checks & evidence', body, 'Missing, skipped, unavailable, and unknown results do not pass');
}
function historyPanel(job) {
	return panel('Event history', list(job.events).length ? element('div', { class: 'panel-body' }, element('ol', { class: 'timeline' }, [...job.events].reverse().map((event) => element('li', {}, element('strong', {}, human(event.kind)), element('time', {}, '#' + event.sequence + ' · ' + time(event.createdAt)), element('details', {}, element('summary', { class: 'fine' }, 'Details'), element('pre', { class: 'output' }, JSON.stringify(event.data, null, 2))))))) : empty('No events recorded', 'Events appear after a durable service transition.'));
}
function actionsPanel(job, capabilities) {
	const body = element('div', { class: 'panel-body action-stack' });
	for (const kind of list(job.availableActions)) {
		if (kind === 'resolve_execution') continue;
		const title = { cancel: 'Request cancellation', repair: 'Request bounded repair', reconcile: 'Reconcile execution' }[kind] || human(kind);
		body.append(button(title, async () => { await command('/jobs/' + encodeURIComponent(job.id) + '/actions', { jobId: job.id, ifVersion: job.version, action: kind }); await route(); }, kind === 'cancel' ? 'danger' : ''));
	}
	if (capabilities.includes('review') || capabilities.includes('accept')) body.append(button('Review & acceptance', () => navigate('job/' + encodeURIComponent(job.id) + '/reviews'), 'primary'));
	if (!body.children.length) body.append(element('p', { class: 'fine' }, 'No state actions are currently available to this account.'));
	if (list(job.availableActions).includes('resolve_execution')) {
		const form = element('form', {}, field('Uncertain attempt', 'attemptId', { choices: list(job.attempts).filter((attempt) => /unknown|running|assigned|registered|acknowledged/.test(attempt.status)).map((attempt) => ({ value: attempt.id, label: attempt.id })) }), field('Observed resolution', 'resolution', { choices: ['still_running', 'not_started', 'stopped_failed', 'stopped_cancelled'].map((value) => ({ value, label: human(value) })) }), field('Termination or execution evidence', 'evidence', { type: 'textarea', required: true }));
		const submit = element('button', { type: 'submit' }, 'Record explicit resolution'); form.append(submit);
		form.addEventListener('submit', (event) => { event.preventDefault(); run(async () => { await command('/jobs/' + encodeURIComponent(job.id) + '/actions', { ...data(form), jobId: job.id, ifVersion: job.version, action: 'resolve_execution' }); await route(); }, submit); }); body.append(form);
	}
	return panel('Available actions', body);
}
function reviewPanels(job, capabilities) {
	const reviews = list(job.reviews);
	const result = [panel('Independent review history', reviews.length ? table(['REVIEWER', 'DECISION', 'CANDIDATE', 'SUMMARY'], reviews.map((review) => [code(review.reviewerId), element('div', { class: 'pill-row' }, badge(review.decision), ...(review.candidateRevision !== job.candidateRevision || review.requirementFingerprint !== job.admission.ticketSHA256 ? [badge('stale')] : [])), code(review.candidateRevision), review.summary])) : empty('Independent review is pending', 'An authorized reviewer must differ from the coordinating and implementation identities.'))];
	const independent = state.me.userId !== job.coordinatorId && state.me.userId !== job.implementerId;
	if (capabilities.includes('review') && independent && job.candidateRevision && job.status === 'ready_for_review') {
		const form = element('form', {}, formGrid(field('Decision', 'decision', { choices: [{ value: 'approved', label: 'Approve this candidate' }, { value: 'changes_requested', label: 'Request changes' }] }), field('Review summary', 'summary', { type: 'textarea', required: true, full: true })));
		const submit = element('button', { type: 'submit', class: 'primary' }, 'Record independent review'); form.append(element('div', { class: 'form-actions' }, submit));
		form.addEventListener('submit', (event) => { event.preventDefault(); run(async () => { await command('/jobs/' + encodeURIComponent(job.id) + '/reviews', { ...data(form), jobId: job.id, ifVersion: job.version, candidateRevision: job.candidateRevision, requirementFingerprint: job.admission.ticketSHA256 }); await route(); }, submit); });
		result.push(panel('Review exact candidate', element('div', { class: 'panel-body' }, metadata([['Candidate', code(job.candidateRevision)], ['Requirements', code(job.admission.ticketSHA256)]]), element('hr'), form)));
	} else if (capabilities.includes('review') && !independent) result.push(element('div', { class: 'notice' }, 'This account coordinated or implemented the work. A different reviewer is required.'));
	if (capabilities.includes('accept') && job.candidateRevision && job.status === 'ready_for_acceptance') {
		const form = element('form', {}, field('Acceptance rationale', 'summary', { type: 'textarea', required: true }));
		const submit = element('button', { type: 'submit', class: 'primary' }, 'Accept exact candidate'); form.append(element('p', { class: 'fine' }, 'The service rechecks current requirements, independent passing evidence, review, and exact candidate identity before publishing a real Acceptance record.'), element('div', { class: 'form-actions' }, submit));
		form.addEventListener('submit', (event) => { event.preventDefault(); run(async () => { await command('/jobs/' + encodeURIComponent(job.id) + '/acceptance', { ...data(form), jobId: job.id, ifVersion: job.version, candidateRevision: job.candidateRevision, requirementFingerprint: job.admission.ticketSHA256 }); await route(); }, submit); });
		result.push(panel('Acceptance decision', element('div', { class: 'panel-body' }, metadata([['Exact candidate', code(job.candidateRevision)], ['Requirements', code(job.admission.ticketSHA256)]]), element('hr'), form)));
	} else if (capabilities.includes('accept') && !['accepted', 'cancelled'].includes(job.status)) {
		result.push(element('div', { class: 'notice' }, 'Acceptance is available after all required checks and an independent current approval pass. Current state: ' + human(job.status) + '.'));
	}
	return result;
}
async function adminPage(tab) {
	if (!state.me.installationAdmin) throw new Error('Installation administration access is required.');
	$('#breadcrumb').textContent = 'Administration';
	const head = [pageHead('Administration', 'Manage distinct identities, authorized project access, and fixed execution policy.'), tabs([['users', 'Users & tokens'], ['projects', 'Register project'], ['profiles', 'Executor & check profiles']], tab, (next) => navigate('admin/' + next))];
	if (tab === 'projects') return [...head, registerProjectForm()];
	if (tab === 'profiles') return [...head, ...(await profileForms())];
	const users = list(await api('/users'));
	const usersTable = table(['USER', 'KIND', 'ACCESS', 'STATUS', ''], users.map((user) => [element('div', {}, element('strong', {}, user.username), element('div', { class: 'subline mono' }, user.id)), user.kind, user.installationAdmin ? 'Installation admin' : 'Project grants', badge(user.disabled ? 'disabled' : 'enabled'), button(user.disabled ? 'Enable' : 'Disable', async () => { await command('/users/' + encodeURIComponent(user.id) + '/disabled', { userId: user.id, disabled: !user.disabled }, 'PUT'); await route(); })]));
	const create = element('form', {}, formGrid(field('Username', 'username', { required: true }), field('Account kind', 'kind', { choices: [{ value: 'human', label: 'Human user' }, { value: 'worker', label: 'Worker identity' }] }), field('Installation administrator', 'admin', { choices: [{ value: 'false', label: 'No' }, { value: 'true', label: 'Yes, installation administrator' }] })));
	const createButton = element('button', { type: 'submit', class: 'primary' }, 'Create user'); create.append(element('div', { class: 'form-actions' }, createButton));
	create.addEventListener('submit', (event) => { event.preventDefault(); run(async () => { const values = data(create); await command('/users', { username: values.username, kind: values.kind, installationAdmin: values.admin === 'true' }); await route(); }, createButton); });
	const token = element('form', {}, formGrid(field('User', 'userId', { choices: users.map((user) => ({ value: user.id, label: user.username })) }), field('Token label', 'label', { required: true, value: 'workspace' })));
	const tokenButton = element('button', { type: 'submit' }, 'Issue token'); const tokenOutput = element('div', { 'aria-live': 'polite' }); token.append(element('div', { class: 'form-actions' }, tokenButton), tokenOutput);
	token.addEventListener('submit', (event) => { event.preventDefault(); run(async () => { const values = data(token); const issued = await command('/users/' + encodeURIComponent(values.userId) + '/tokens', values); tokenOutput.replaceChildren(element('p', { class: 'fine' }, 'Save this token securely. Its plaintext is shown only in this response. Token ID: ' + issued.id), element('pre', { class: 'one-time-token' }, issued.token)); }, tokenButton); });
	const revoke = element('form', {}, field('Token ID to revoke', 'tokenId', { required: true })); const revokeButton = element('button', { type: 'submit', class: 'danger' }, 'Revoke token'); revoke.append(element('div', { class: 'form-actions' }, revokeButton));
	revoke.addEventListener('submit', (event) => { event.preventDefault(); run(async () => { const values = data(revoke); await command('/tokens/' + encodeURIComponent(values.tokenId) + '/revoke', values); revoke.reset(); notice('Token revoked.'); }, revokeButton); });
	return [...head, panel('Installation users', usersTable), element('div', { class: 'two-columns' }, panel('Create distinct user', element('div', { class: 'panel-body' }, create)), panel('Issue a personal or worker token', element('div', { class: 'panel-body' }, token))), panel('Revoke a token', element('div', { class: 'panel-body' }, revoke))];
}
async function grantsForm(project) {
	const grants = list(await api('/projects/' + encodeURIComponent(project.id) + '/grants'));
	const choices = state.me.installationAdmin ? list(await api('/users')).map((user) => ({ value: user.id, label: user.username })) : null;
	const form = element('form', {}, field('User ID', 'userId', choices ? { choices } : { required: true }), field('Capabilities, comma separated', 'capabilities', { value: 'read,coordinate', help: 'read, coordinate, verify, review, accept, project_admin, worker_implement, worker_verify. Leave empty to revoke all project capabilities.' }));
	const submit = element('button', { type: 'submit', class: 'primary' }, 'Save exact project grant'); form.append(element('p', { class: 'fine' }, 'Saving replaces this user’s capability set for this project.'), element('div', { class: 'form-actions' }, submit));
	form.addEventListener('submit', (event) => { event.preventDefault(); run(async () => { const values = data(form); await command('/projects/' + encodeURIComponent(project.id) + '/grants/' + encodeURIComponent(values.userId), { projectId: project.id, userId: values.userId, capabilities: split(values.capabilities) }, 'PUT'); await route(); }, submit); });
	return panel('Project access', element('div', {}, grants.length ? table(['USER ID', 'CAPABILITIES'], grants.map((grant) => [code(grant.userId), list(grant.capabilities).join(', ')])) : empty('No explicit grants', 'Installation administrators retain installation access.'), element('div', { class: 'panel-body' }, form)));
}
const split = (value) => String(value || '').split(',').map((item) => item.trim()).filter(Boolean);
function registerProjectForm() {
	const form = element('form', {}, formGrid(field('Existing OKF bundle path', 'bundlePath', { required: true, full: true, placeholder: '/absolute/path/to/records' }), field('Code repository path', 'repositoryPath', { required: true, full: true, placeholder: '/absolute/path/to/repository' }), field('Allowed source roots, one per line', 'allowedSourceRoots', { type: 'textarea', full: true, help: 'Supporting documents stay within these administrator-selected roots.' }), field('Required check profile keys', 'requiredChecks', { required: true, placeholder: 'go-test' }), field('Allowed executor profile keys', 'allowedExecutors', { required: true, placeholder: 'coding-agent' }), field('Maximum repairs', 'maxRepairAttempts', { type: 'number', min: 0, max: 10, value: 1, required: true }), field('Maximum duration, seconds', 'maxJobDurationSeconds', { type: 'number', min: 60, value: 3600, required: true }), field('Execution mode', 'fixture', { choices: [{ value: 'false', label: 'Ordinary project, real execution required' }, { value: 'true', label: 'Fixture test project, visibly simulated' }] }), field('Reader file limit', 'maxFiles', { type: 'number', min: 0, value: 0, help: '0 uses the bounded service default.' }), field('Reader byte limit', 'maxBytes', { type: 'number', min: 0, value: 0 })));
	const submit = element('button', { type: 'submit', class: 'primary' }, 'Register existing project'); form.append(element('div', { class: 'form-actions' }, submit), element('p', { class: 'fine' }, 'The service references the existing bundle and preserves Project and WorkItem identities. It does not copy a backlog.'));
	form.addEventListener('submit', (event) => { event.preventDefault(); run(async () => { const values = data(form); const project = await command('/projects', { bundlePath: values.bundlePath, repositoryPath: values.repositoryPath, allowedSourceRoots: values.allowedSourceRoots.split('\n').map((item) => item.trim()).filter(Boolean), readerLimits: { maxFiles: Number(values.maxFiles), maxBytes: Number(values.maxBytes) }, policy: { requiredChecks: split(values.requiredChecks), allowedExecutors: split(values.allowedExecutors), maxRepairAttempts: Number(values.maxRepairAttempts), maxJobDurationSeconds: Number(values.maxJobDurationSeconds), fixture: values.fixture === 'true' } }); await refreshProjects(); await navigate('project/' + encodeURIComponent(project.id)); }, submit); });
	return panel('Register a project', element('div', { class: 'panel-body' }, form));
}
async function profileForms() {
	const [executors, checks] = await Promise.all([api('/executors'), api('/checks')]);
	const result = [panel('Registered profiles', table(['TYPE', 'KEY', 'ARGUMENT ARRAY', 'MODE'], [...list(executors).map((profile) => ['Executor', profile.key, code(JSON.stringify(profile.command)), badge(profile.fixture ? 'fixture' : 'external')]), ...list(checks).map((profile) => ['Check', profile.key, code(JSON.stringify(profile.command)), badge(profile.fixture ? 'fixture' : 'external')])]))];
	for (const kind of ['executors', 'checks']) {
		const form = element('form', {}, formGrid(field('Profile key', 'key', { required: true }), field('Display name', 'displayName', { required: true }), field('Fixed command argument array as JSON', 'command', { type: 'textarea', full: true, required: true, value: '["/absolute/path/to/program"]', help: kind === 'checks' ? 'Use a literal JSON argument array, with no shell. For a fixture check: ["/absolute/path/to/ctx","factory","fixture-check"]. The executable must exist on the worker host.' : 'Use a literal JSON argument array, with no shell. For a fixture executor: ["/absolute/path/to/ctx","factory","fixture"]. Fixture mode does not rewrite the command.' }), field('Execution mode', 'fixture', { choices: [{ value: 'false', label: 'External execution' }, { value: 'true', label: 'Offline fixture' }] }), ...(kind === 'checks' ? [field('Environment', 'environment', { required: true, value: 'local' })] : [])));
		const submit = element('button', { type: 'submit', class: 'primary' }, kind === 'checks' ? 'Register required-check profile' : 'Register executor profile'); form.append(element('div', { class: 'form-actions' }, submit));
		form.addEventListener('submit', (event) => { event.preventDefault(); run(async () => { const values = data(form); const argv = JSON.parse(values.command); if (!Array.isArray(argv) || argv.some((arg) => typeof arg !== 'string')) throw new Error('Command must be a JSON array of strings.'); await command('/' + kind, { key: values.key, displayName: values.displayName, command: argv, fixture: values.fixture === 'true', ...(kind === 'checks' ? { environment: values.environment } : {}) }); await route(); }, submit); });
		result.push(panel(kind === 'checks' ? 'Register check profile' : 'Register executor profile', element('div', { class: 'panel-body' }, form)));
	}
	return result;
}

$('#signin-form').addEventListener('submit', (event) => { event.preventDefault(); const target = event.submitter; run(async () => { $('#signin-error').hidden = true; state.token = $('#token').value.trim(); try { await connect(); } catch (error) { signout(); $('#signin-error').textContent = error.message; $('#signin-error').hidden = false; } }, target); });
$('#signout').addEventListener('click', signout);
$('#nav-queue').addEventListener('click', () => navigate('queue'));
$('#nav-admin').addEventListener('click', () => navigate('admin/users'));
$('#refresh-projects').addEventListener('click', () => run(refreshProjects));
window.addEventListener('hashchange', () => run(route));
if (state.token) connect().catch((error) => { signout(); $('#signin-error').textContent = error.message; $('#signin-error').hidden = false; });
setInterval(async () => {
	if (!state.me || document.hidden || mutationsInFlight || /INPUT|TEXTAREA|SELECT/.test(document.activeElement?.tagName || '')) return;
	try {
		if (state.view === 'job' && state.tab === 'overview' && state.job) {
			const current = await api('/jobs/' + encodeURIComponent(state.job.id));
			if (current.version !== state.job.version || current.acceptanceStatus !== state.job.acceptanceStatus) await route();
		} else if (state.view === 'queue') {
			const jobs = list(await api('/jobs?limit=100'));
			if (JSON.stringify(jobs.map((job) => [job.id, job.version])) !== state.queueVersion) await route();
		}
	} catch (error) { notice(error.message, true); }
}, 5000);

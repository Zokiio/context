const $ = (selector) => document.querySelector(selector);
const list = (value) => Array.isArray(value) ? value : [];
const e = encodeURIComponent;
const state = { flash: '', token: '', me: null, projects: [], capabilities: new Map(), project: null, board: null, boardKey: '', view: '', generation: 0, drawer: null, drawerReturn: null, theme: 'auto' };
const pending = new Map();
let mutationsInFlight = 0;
try { state.token = sessionStorage.getItem('waymark-token') || ''; } catch {}
try { state.theme = localStorage.getItem('waymark-theme') || 'auto'; } catch {}

// Elements

function element(tag, attributes = {}, ...children) {
	const result = document.createElement(tag);
	for (const [name, value] of Object.entries(attributes)) {
		if (value === null || value === undefined) continue;
		if (name.startsWith('on')) result.addEventListener(name.slice(2), value);
		else if (name === 'class') result.className = value;
		else if (name.startsWith('aria-') || name.startsWith('data-')) result.setAttribute(name, value);
		else if (name in result) result[name] = value;
		else result.setAttribute(name, value);
	}
	for (const child of children.flat()) if (child !== null && child !== undefined && child !== false) result.append(child instanceof Node ? child : document.createTextNode(String(child)));
	return result;
}
const ICONS = {
	check: ['M20 6 9 17l-5-5'],
	close: ['M18 6 6 18M6 6l12 12'],
	moon: ['M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z'],
	sun: ['M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41', 'M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0z'],
	eye: ['M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z', 'M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0z'],
	arrow: ['M5 12h14M12 5l7 7-7 7'],
};
function icon(name, size = 18) {
	const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
	for (const [key, value] of Object.entries({ width: size, height: size, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': name === 'check' ? 2.25 : 1.75, 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'aria-hidden': 'true' })) svg.setAttribute(key, value);
	for (const d of ICONS[name]) {
		const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
		path.setAttribute('d', d);
		svg.append(path);
	}
	return svg;
}
const button = (text, action, kind = '') => element('button', { type: 'button', class: 'button ' + kind, onclick: (event) => run(action, event.currentTarget) }, text);
const link = (text, href, kind = '') => element('a', { href, class: 'button ' + kind }, text);
// rich renders `code` spans in agent-written text as inline code.
const rich = (text) => String(text || '').split(/(`[^`]+`)/).filter(Boolean).map((part) => part.length > 2 && part.startsWith('`') && part.endsWith('`') ? element('code', { class: 'inline' }, part.slice(1, -1)) : part);
const plural = (count, one, many) => count === 1 ? '1 ' + one : count + ' ' + many;
const sentence = (text) => { const value = String(text).trim(); return value && value[0].toUpperCase() + value.slice(1) + (/[.!?]$/.test(value) ? '' : '.'); };
const human = (value) => String(value || 'unknown').replaceAll('_', ' ');
const money = (value) => '$' + Number(value || 0).toFixed(2);
const when = (value) => value ? new Date(value).toLocaleString() : 'not recorded';
const short = (value) => value ? String(value).slice(0, 12) : 'none';
function clock(seconds) {
	seconds = Math.max(0, Math.floor(seconds));
	const hours = Math.floor(seconds / 3600), minutes = Math.floor((seconds % 3600) / 60), rest = seconds % 60;
	return hours ? hours + ' h ' + minutes + ' min' : minutes + ':' + String(rest).padStart(2, '0');
}
function ago(value) {
	if (!value) return '';
	const seconds = (Date.now() - new Date(value).getTime()) / 1000;
	if (seconds < 60) return 'just now';
	if (seconds < 3600) return Math.round(seconds / 60) + ' min ago';
	if (seconds < 86400) return Math.round(seconds / 3600) + ' h ago';
	return new Date(value).toLocaleDateString();
}
const elapsed = (since) => since ? element('span', { class: 'elapsed', 'data-since': since }, clock((Date.now() - Date.parse(since)) / 1000)) : null;
const page = (width, ...children) => element('div', { class: 'page ' + width }, ...children);
const surface = (...children) => element('section', { class: 'surface' }, ...children);
const fold = (summary, open, ...children) => element('details', { class: 'fold', open: !!open }, element('summary', {}, summary), ...children);
const check = (text, open = false, extra = null) => element('div', { class: 'check' + (open ? ' open' : '') }, open ? icon('eye') : icon('check'), element('div', { class: 'stack tight' }, element('span', {}, rich(text)), extra));
function empty(title, detail) { return element('div', { class: 'empty' }, element('strong', {}, title), element('span', {}, detail)); }
function table(headers, rows) {
	return element('div', { class: 'table-scroll' }, element('table', {}, element('thead', {}, element('tr', {}, headers.map((title) => element('th', {}, title)))), element('tbody', {}, rows.map((cells) => element('tr', {}, cells.map((cell) => element('td', {}, cell)))))));
}
function field(label, name, options = {}) {
	const { choices, help, full, ...attributes } = options;
	let input;
	if (choices) input = element('select', { name, ...attributes }, choices.map((choice) => element('option', { value: choice.value }, choice.label)));
	else if (attributes.type === 'textarea') { delete attributes.type; input = element('textarea', { name, ...attributes }); }
	else input = element('input', { type: 'text', name, ...attributes });
	input.id = 'field-' + crypto.randomUUID();
	return element('div', { class: 'field' + (full ? ' full' : '') }, element('label', { htmlFor: input.id }, label), input, help ? element('span', { class: 'fine' }, help) : null);
}
const formGrid = (...fields) => element('div', { class: 'form-grid' }, fields);
const data = (form) => Object.fromEntries(new FormData(form));
const split = (value) => String(value || '').split(',').map((item) => item.trim()).filter(Boolean);
const lines = (value) => String(value || '').split('\n').map((item) => item.trim()).filter(Boolean);

// Requests

let noticeTimer = null;
function notice(message, error = false) {
	const target = $('#notice');
	target.hidden = !message;
	target.textContent = message;
	target.className = error ? 'error' : '';
	clearTimeout(noticeTimer);
	// Confirmations fade; errors stay until the next action.
	if (message && !error) noticeTimer = setTimeout(() => { target.hidden = true; }, 8000);
}
async function run(action, target) {
	if (target) target.disabled = true;
	try { await action(); }
	catch (error) { notice(error.message || 'That did not work.', true); }
	finally { if (target && target.isConnected) target.disabled = false; }
}
async function api(path, options = {}) {
	const controller = new AbortController();
	const timeout = setTimeout(() => controller.abort(), 30000);
	try {
		const response = await fetch('/v1' + path, { method: options.method || 'GET', headers: { Authorization: 'Bearer ' + state.token, Accept: 'application/json', ...(options.body ? { 'Content-Type': 'application/json' } : {}) }, body: options.body ? JSON.stringify(options.body) : undefined, credentials: 'omit', redirect: 'error', signal: controller.signal });
		const value = response.status === 204 ? null : await response.json();
		if (!response.ok) {
			const error = new Error(value.message || 'Request failed');
			error.code = value.code; error.status = response.status;
			if (response.status === 401 && state.me) signout();
			throw error;
		}
		return value;
	} catch (error) {
		if (error.name === 'AbortError' || error instanceof TypeError) throw new Error('The connection was interrupted, and the outcome may be pending. Refresh before trying again; repeating the same request is safe.');
		throw error;
	} finally { clearTimeout(timeout); }
}
async function command(path, body, method = 'POST') {
	const identity = method + path + JSON.stringify(body);
	if (!pending.has(identity)) pending.set(identity, crypto.randomUUID());
	mutationsInFlight++;
	try {
		const result = await api(path, { method, body: { ...body, requestId: pending.get(identity) } });
		pending.delete(identity);
		notice('');
		return result;
	} catch (error) {
		if (error.status >= 400 && error.status < 500) pending.delete(identity);
		throw error;
	} finally { mutationsInFlight--; }
}
const can = (capability, projectId = state.project) => !!state.me?.installationAdmin || list(state.capabilities.get(projectId)).includes(capability);

// Theme

function resolvedTheme() {
	if (state.theme !== 'auto') return state.theme;
	return matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}
function applyTheme() {
	document.documentElement.dataset.theme = state.theme;
	const dark = resolvedTheme() === 'dark';
	const toggle = $('#theme-toggle');
	toggle.replaceChildren(icon(dark ? 'sun' : 'moon'));
	toggle.setAttribute('aria-label', dark ? 'Switch to light mode' : 'Switch to dark mode');
}

// Session and routing

function navigate(fragment) { if (location.hash === '#' + fragment) return route(); location.hash = fragment; }
function signout() {
	state.token = ''; state.me = null; state.board = null; state.generation++; pending.clear();
	try { sessionStorage.removeItem('waymark-token'); } catch {}
	closeDrawer(false);
	$('#workspace').hidden = true; $('#signin').hidden = false; $('#signin-error').hidden = true; $('#token').value = '';
}
async function connect() {
	state.me = await api('/me');
	try { sessionStorage.setItem('waymark-token', state.token); } catch {}
	$('#token').value = ''; $('#signin').hidden = true; $('#workspace').hidden = false;
	$('#user-initials').textContent = state.me.username.slice(0, 2).toUpperCase();
	$('#signout').setAttribute('aria-label', 'Sign out ' + state.me.username);
	$('#nav-admin').hidden = !state.me.installationAdmin;
	await refreshProjects();
	await route();
}
async function refreshProjects() {
	state.projects = list(await api('/projects'));
	state.capabilities = new Map(await Promise.all(state.projects.map(async (project) => [project.id, list(await api('/projects/' + e(project.id) + '/capabilities'))])));
	let saved = null;
	try { saved = localStorage.getItem('waymark-project'); } catch {}
	state.project = state.projects.find((project) => project.id === saved)?.id || state.projects[0]?.id || null;
	const select = $('#project-select');
	select.replaceChildren(...state.projects.map((project) => element('option', { value: project.id }, project.title)));
	select.hidden = !state.projects.length;
	if (state.project) select.value = state.project;
}
async function route(quiet = false) {
	if (!state.me) return;
	const generation = ++state.generation;
	const parts = location.hash.slice(1).split('/').map(decodeURIComponent);
	state.view = parts[0] || 'board';
	const scroll = window.scrollY;
	if (!quiet) { closeDrawer(false); notice(state.flash); state.flash = ''; }
	const area = { board: 'board', list: 'board', item: 'board', job: 'board', insights: 'insights', ask: 'ask' }[state.view] || '';
	for (const anchor of document.querySelectorAll('[data-nav]')) {
		if (anchor.dataset.nav === area) anchor.setAttribute('aria-current', 'page'); else anchor.removeAttribute('aria-current');
	}
	if (!quiet) $('#content').replaceChildren(element('p', { class: 'loading' }, 'Loading…'));
	try {
		let content;
		if (state.view === 'admin') content = await adminPage(parts[1] || 'users');
		else if (!state.project) content = noProjects();
		else if (state.view === 'list') content = await listPage();
		else if (state.view === 'item') content = await itemPage(parts[1] || '');
		else if (state.view === 'job') content = await jobPage(parts[1] || '');
		else if (state.view === 'insights') content = await insightsPage();
		else if (state.view === 'ask') content = await askPage();
		else if (state.view === 'access') content = await accessPage();
		else content = await boardPage();
		if (generation !== state.generation) return;
		$('#content').replaceChildren(content);
		if (quiet) window.scrollTo(0, scroll);
	} catch (error) {
		if (generation !== state.generation) return;
		$('#content').replaceChildren(page('narrow', element('h1', { class: 'title' }, 'This view could not open'), element('p', { class: 'alert bad', role: 'alert' }, error.message), element('div', { class: 'actions' }, button('Try again', () => route()))));
	}
}
function noProjects() {
	return page('narrow', empty('No projects yet', state.me.installationAdmin ? 'Register a project to put its work on the board.' : 'Ask an administrator for access to a project.'), state.me.installationAdmin ? element('div', { class: 'actions' }, link('Register a project', '#admin/projects', 'primary')) : null);
}

// Board

const COLUMNS = [['next', 'Up next'], ['plan', 'Plan'], ['build', 'Build'], ['approve', 'Approve'], ['merge', 'Merge'], ['done', 'Done']];
async function loadBoard() {
	state.board = await api('/projects/' + e(state.project) + '/board');
	state.boardKey = boardKey(state.board);
	return state.board;
}
function boardKey(board) {
	return JSON.stringify(list(board.cards).map((card) => [card.workItemId, card.column, card.state, card.status, card.line, card.jobVersion, card.refinementId, card.expectationId, card.progress]));
}
const cards = (board) => list(board.cards);
function needsYou(board) {
	return cards(board).filter((card) => card.state === 'needs_you').sort((a, b) => Date.parse(a.updatedAt || 0) - Date.parse(b.updatedAt || 0));
}
function autopilotLine(board) {
	if (!board.pilot) return 'the autopilot is off on this installation';
	return board.project.policy.ask?.autopilot ? 'ready work starts by itself' : 'ready work waits for you to start it';
}
function boardHead(board, view) {
	const needs = needsYou(board);
	const title = needs.length === 0 ? 'Nothing needs you' : needs.length === 1 ? '1 thing needs you' : needs.length + ' things need you';
	const line = [plural(board.running, 'agent working', 'agents working'), board.mergedToday + ' merged today', autopilotLine(board)].join(' · ');
	const views = element('nav', { class: 'segmented', 'aria-label': 'View' }, element('a', { href: '#board', 'aria-current': view === 'board' ? 'page' : null }, 'Board'), element('a', { href: '#list', 'aria-current': view === 'list' ? 'page' : null }, 'List'));
	return element('div', { class: 'page-head' }, element('div', {}, element('h1', { class: 'title' }, title), element('p', { class: 'subtitle' }, line)), element('div', { class: 'row' }, views, needs.length ? button('Next: ' + needs[0].title, () => openCard(needs[0]), 'primary') : null));
}
function who(key, board) {
	const ask = board.project.policy.ask || {};
	switch (key) {
	case 'next': return board.pilot && ask.autopilot ? ['Starts by itself', false] : ['You start it', true];
	case 'plan': return ['Agents, then you', true];
	case 'build': return ['Agents', false];
	case 'approve': return ['You', true];
	case 'merge': return ['Automatic', false];
	}
	return ['', false];
}
const ORDER = { needs_you: 0, running: 1, waiting: 2, done: 3 };
function columnElement(board, key, name) {
	let entries = cards(board).filter((card) => card.column === key).sort((a, b) => ORDER[a.state] - ORDER[b.state] || (key === 'done' ? Date.parse(b.since || 0) - Date.parse(a.since || 0) : Date.parse(a.updatedAt || 0) - Date.parse(b.updatedAt || 0)));
	if (key === 'done') entries = entries.slice(0, 8);
	const [label, you] = who(key, board);
	return element('section', { class: 'column', 'aria-label': name },
		element('div', { class: 'column-head' }, element('span', {}, element('h2', {}, name), element('span', { class: 'count' }, entries.length)), label ? element('span', { class: 'who' + (you ? ' you' : '') }, label) : null),
		entries.length ? entries.map(cardElement) : element('p', { class: 'column-empty' }, { next: 'Nothing waiting.', plan: 'Nothing being planned.', build: 'Nothing being built.', approve: 'Nothing to approve.', merge: 'Nothing merging.', done: 'Nothing merged yet.' }[key]));
}
function cardAction(card) {
	if (card.decision === 'plan') return card.questions ? 'Answer' : card.state === 'needs_you' ? 'Review the plan' : 'Start';
	if (card.decision === 'approve') return 'Review';
	return 'Open';
}
function cardElement(card) {
	const kind = { needs_you: 'need', running: 'running', waiting: 'waiting', done: 'done' }[card.state];
	const title = element('span', { class: 'card-title' }, card.title);
	const cost = card.costUsd ? element('span', { class: 'card-meta' }, money(card.costUsd)) : null;
	let body;
	if (card.state === 'needs_you') body = [element('span', { class: 'eyebrow warn' }, card.status), title, element('span', { class: 'card-line' }, card.line), element('span', { class: 'card-foot' }, element('span', { class: 'card-action' }, cardAction(card) + ' →'), cost)];
	else if (card.state === 'running') body = [element('span', { class: 'card-status' }, element('span', { class: 'dot', 'aria-hidden': 'true' }), element('span', {}, card.status), elapsed(card.since)), title, element('span', { class: 'card-line' }, card.line), card.progress ? element('span', { class: 'progress-note' }, card.progress) : null, cost];
	else if (card.state === 'waiting') body = [title, element('span', { class: 'eyebrow' }, card.status), element('span', { class: 'card-line' }, card.line), card.decision ? element('span', { class: 'card-action' }, cardAction(card) + ' →') : null];
	else body = [element('span', { class: 'card-status ok' }, icon('check', 14), card.status), title, element('span', { class: 'card-line' }, [card.line, card.since ? ago(card.since) : ''].filter(Boolean).join(' · '))];
	return element('button', { type: 'button', class: 'card ' + kind, onclick: () => openCard(card) }, body);
}
function openCard(card) {
	if (card.decision === 'plan' || (card.decision === 'approve' && card.jobId)) return openDrawer(card);
	if (card.jobId) return navigate('job/' + e(card.jobId));
	return navigate('item/' + e(card.workItemId));
}
async function boardPage() {
	const board = await loadBoard();
	return page('', boardHead(board, 'board'), element('div', { class: 'board-scroll' }, element('div', { class: 'board' }, COLUMNS.map(([key, name]) => columnElement(board, key, name)))));
}
async function listPage() {
	const board = await loadBoard();
	const needs = needsYou(board);
	const running = cards(board).filter((card) => card.state === 'running');
	const next = cards(board).filter((card) => card.state === 'waiting');
	const done = cards(board).filter((card) => card.state === 'done' && card.since && new Date(card.since).toDateString() === new Date().toDateString());
	const row = (card) => element('button', { type: 'button', class: 'decision-row', onclick: () => openCard(card) }, element('span', { class: 'grow' }, element('span', { class: 'eyebrow warn' }, card.decision === 'approve' ? 'Approve the change' : card.decision === 'plan' ? 'Agree what done means' : card.status), element('span', { class: 'lead' }, element('strong', {}, card.title)), element('span', { class: 'card-line' }, card.line)), element('span', { class: 'button primary' }, cardAction(card)));
	return page('narrow', boardHead(board, 'list'),
		needs.length ? element('section', { class: 'stack', 'aria-label': 'Needs you' }, needs.map(row)) : empty('Nothing needs you', 'Everything else runs by itself or waits for its turn.'),
		running.length ? element('section', { class: 'stack' }, element('h2', { class: 'section' }, 'Agents working'), element('div', { class: 'surface compact' }, running.map((card) => element('a', { class: 'list-row', href: card.jobId ? '#job/' + e(card.jobId) : '#item/' + e(card.workItemId) }, element('span', { class: 'dot', 'aria-hidden': 'true' }), element('span', { class: 'grow' }, element('strong', {}, card.title), element('span', { class: 'card-status' }, card.status)), elapsed(card.since))))) : null,
		element('div', { class: 'row' }, element('div', { class: 'surface compact stack tight' }, element('span', { class: 'eyebrow' }, 'Up next · ' + next.length), element('span', { class: 'fine' }, next.map((card) => card.title).join(', ') || 'Nothing waiting.')), element('div', { class: 'surface compact stack tight' }, element('span', { class: 'eyebrow ok' }, 'Merged today · ' + done.length), element('span', { class: 'fine' }, done.map((card) => card.title).join(', ') || 'Nothing yet.'))));
}

// The decision drawer

async function openDrawer(card) {
	closeDrawer(false);
	state.drawer = card;
	state.drawerReturn = document.activeElement;
	const close = element('button', { type: 'button', class: 'icon-button', 'aria-label': 'Close', onclick: () => closeDrawer(true) }, icon('close'));
	const body = element('div', { class: 'stack loose' }, element('p', { class: 'loading' }, 'Loading…'));
	const approve = card.decision === 'approve';
	const eyebrow = (card.state === 'needs_you' ? 'Needs you · ' : '') + (approve ? 'approve the change' : 'agree what done means');
	const pageHref = approve ? '#job/' + e(card.jobId) : '#item/' + e(card.workItemId);
	const drawer = element('aside', { class: 'drawer', role: 'dialog', 'aria-modal': 'true', 'aria-label': approve ? 'Approve the change' : 'Agree what done means' },
		element('div', { class: 'drawer-head' }, element('div', {}, element('span', { class: 'eyebrow warn' }, eyebrow), element('h2', {}, card.title), element('a', { href: pageHref, class: 'fine', onclick: () => closeDrawer(false) }, 'Open as a page')), close), body);
	$('#drawer-root').replaceChildren(element('button', { type: 'button', class: 'scrim', tabIndex: -1, 'aria-label': 'Close the panel', onclick: () => closeDrawer(true) }), drawer);
	close.focus();
	try { body.replaceChildren(...(approve ? await approveContent(card.jobId, 'drawer') : await planContent(card, 'drawer'))); }
	catch (error) { body.replaceChildren(element('p', { class: 'alert bad' }, error.message)); }
}
function closeDrawer(restore) {
	if (!state.drawer) return;
	state.drawer = null;
	$('#drawer-root').replaceChildren();
	if (restore && state.drawerReturn?.isConnected) state.drawerReturn.focus();
}
async function decided(message, fragment = 'board') {
	closeDrawer(false);
	state.flash = message;
	await navigate(fragment);
}

// Agreeing what done means

function runNote(run) {
	return run ? element('p', { class: 'fine' }, [run.model, run.turns ? plural(run.turns, 'turn', 'turns') : '', money(run.costUsd)].filter(Boolean).join(' · ')) : null;
}
function consentBox(board) {
	const input = element('input', { type: 'checkbox', checked: true });
	const ready = board.canTest ? 'the tests are ready' : 'the requirements are settled';
	const text = board.pilot ? 'Start the build as soon as ' + ready + ', unless something new comes up.' : 'Start the build as soon as ' + ready + '. The autopilot is off on this installation, so the build waits for you.';
	return { node: element('label', { class: 'consent' }, input, element('span', {}, text)), value: () => input.checked };
}
async function plan(card, body) {
	const result = await command('/projects/' + e(state.project) + '/plan', { workItemId: card.workItemId, ...body });
	await decided(result.job ? 'The build has started.' : (result.waiting || 'Saved.'));
}
async function planContent(card, mode) {
	const board = state.board || await loadBoard();
	const [refinement, expectation] = await Promise.all([card.refinementId ? api('/refinements/' + e(card.refinementId)) : null, card.expectationId ? api('/expectations/' + e(card.expectationId)) : null]);
	if (expectation?.status === 'proposed') return testsDecision(card, expectation, board, mode);
	if (expectation?.status === 'accepted') return startDecision(card, expectation);
	if (expectation && ['failed', 'rejected'].includes(expectation.status)) return [...retryDecision(card, expectation.error || 'The tests were rejected.', 'Write the tests again', () => plan(card, {})), expectation.notes ? fold('The author’s note', true, element('p', {}, rich(expectation.notes))) : null, runNote(expectation.run)].filter(Boolean);
	if (refinement?.status === 'proposed') return refineDecision(card, refinement, board, mode);
	if (refinement?.status === 'failed') return retryDecision(card, refinement.error || 'Sharpening failed.', 'Sharpen again', () => sharpen(card));
	return agreeDecision(card, board, refinement);
}
async function sharpen(card) {
	await command('/projects/' + e(state.project) + '/refinements', { workItemId: card.workItemId });
	await decided('An agent is sharpening the requirements.');
}
function questionElement(question, answer) {
	const options = list(question.options);
	const other = element('input', { type: 'text', placeholder: 'Your answer', 'aria-label': 'Your answer', hidden: options.length > 0, oninput: (event) => { answer.other = event.target.value; } });
	answer.choice = options.length ? 0 : -1;
	const buttons = [];
	let otherButton = null;
	const select = (index) => {
		answer.choice = index;
		buttons.forEach((entry, position) => entry.setAttribute('aria-pressed', String(position === index)));
		otherButton?.setAttribute('aria-pressed', String(index === -1));
		other.hidden = index !== -1;
		if (index === -1) other.focus();
	};
	options.forEach((text, index) => buttons.push(element('button', { type: 'button', class: 'option', 'aria-pressed': String(index === 0), onclick: () => select(index) }, element('span', { class: 'radio' }), element('span', { class: 'grow' }, text), index === 0 ? element('span', { class: 'tag' }, 'Suggested') : null)));
	if (options.length) otherButton = element('button', { type: 'button', class: 'option', 'aria-pressed': 'false', onclick: () => select(-1) }, element('span', { class: 'radio' }), element('span', { class: 'grow' }, 'Something else'));
	const id = 'question-' + crypto.randomUUID();
	return element('div', { class: 'question', role: 'group', 'aria-labelledby': id }, element('h3', { id, class: 'question-title' }, question.question), question.unblocks ? element('span', { class: 'fine' }, 'Decides: ' + question.unblocks) : null, buttons, otherButton, other);
}
function refineDecision(card, refinement, board, mode) {
	const proposal = refinement.proposal || {};
	const proposed = list(proposal.criteria), questions = list(proposal.questions);
	const answers = questions.map(() => ({ choice: 0, other: '' }));
	const editor = element('textarea', { rows: Math.max(4, proposed.length + 1), value: proposed.join('\n'), hidden: true, 'aria-label': 'Requirements, one per line' });
	const vague = new Set(list(refinement.after?.criteria).filter((criterion) => criterion.flagged).map((criterion) => criterion.text));
	const view = element('ol', { class: 'requirements' }, proposed.map((text) => element('li', {}, rich(text), vague.has(text) ? [' ', element('span', { class: 'tag warn' }, 'Still looks vague')] : null)));
	const consent = consentBox(board);
	const criteria = () => lines(editor.value);
	const given = () => questions.map((question, index) => ({ question: question.question, answer: answers[index].choice >= 0 ? list(question.options)[answers[index].choice] : answers[index].other.trim() })).filter((answer) => answer.answer);
	const actions = element('div', { class: 'actions' });
	if (questions.length) {
		actions.append(button('Send answers', () => {
			const answered = given();
			if (answered.length < questions.length) throw new Error('Answer every question, or accept the requirements without answering.');
			return plan(card, { refinementId: refinement.id, answers: answered, continue: consent.value() });
		}, 'primary'), button('Accept without answering', () => plan(card, { refinementId: refinement.id, criteria: criteria(), continue: consent.value() })));
	} else {
		actions.append(button('Agree and continue', () => plan(card, { refinementId: refinement.id, criteria: criteria(), continue: consent.value() }), 'primary'));
	}
	actions.append(button('Discard', async () => { await command('/refinements/' + e(refinement.id) + '/reject', {}); await decided('The sharpened requirements were discarded.'); }, 'danger'));
	const original = list(refinement.criteria);
	return [
		element('p', { class: 'lead' }, 'Your ' + plural(original.length, 'requirement was', 'requirements were') + ' too vague to test, so an agent rewrote them as ' + plural(proposed.length, 'requirement', 'requirements') + ' it can check.' + (questions.length ? ' Answer ' + plural(questions.length, 'question', 'questions') + ' and it carries on.' : '')),
		vague.size && !questions.length ? element('div', { class: 'callout warn' }, element('p', {}, 'Kev still rates ' + plural(vague.size, 'requirement', 'requirements') + ' as vague, so the go-ahead stopped here. Read ' + (vague.size === 1 ? 'it' : 'them') + ', edit if needed, and agree to carry on.')) : null,
		...questions.map((question, index) => questionElement(question, answers[index])),
		fold(element('span', {}, 'Read the ' + plural(proposed.length, 'new requirement', 'new requirements') + ' ', element('span', { class: 'muted' }, '· about 1 min')), mode === 'page' || vague.size > 0,
			element('p', { class: 'fine' }, 'Read the examples. Agents sometimes write one that breaks their own rule.'), view, editor,
			element('div', { class: 'actions' }, button('Edit the requirements', () => { editor.hidden = !editor.hidden; view.hidden = !editor.hidden; if (!editor.hidden) editor.focus(); }, 'small'))),
		fold('What you wrote', false, element('ul', { class: 'plain' }, original.map((text) => element('li', {}, rich(text))))),
		board.canTest ? element('p', { class: 'fine' }, 'A separate agent writes locked tests from these and flags any example that contradicts its own rule.') : null,
		consent.node,
		actions,
		runNote(refinement.run),
	].filter(Boolean);
}
function coverageOf(expectation) {
	const coverage = new Map(list(expectation?.coverage).map((entry) => [entry.criterion, list(entry.tests)]));
	return list(expectation?.criteria).map((text, index) => ({ text, tests: coverage.get(index + 1) || [] }));
}
function diffView(patch, max = 1500) {
	const rows = String(patch || '').split('\n').slice(0, max).map((line) => element('div', { class: line.startsWith('+') && !line.startsWith('+++') ? 'add' : line.startsWith('-') && !line.startsWith('---') ? 'del' : line.startsWith('@@') ? 'hunk' : /^(diff|index|\+\+\+|---|new file|deleted file)/.test(line) ? 'meta' : '' }, line || ' '));
	return element('pre', { class: 'code' }, rows);
}
function buildDefaults(project) {
	const policy = project.policy;
	const executor = list(policy.allowedExecutors)[0] || 'no executor allowed yet';
	return 'Builder: ' + executor + ' · up to ' + plural(Math.min(2, policy.maxRepairAttempts), 'fix', 'fixes') + ' · within ' + Math.round(Math.min(3600, policy.maxJobDurationSeconds) / 60) + ' min';
}
function testsDecision(card, expectation, board, mode) {
	const rows = coverageOf(expectation);
	const unchecked = rows.filter((row) => !row.tests.length).length;
	const concerns = list(expectation.concerns);
	return [
		element('p', { class: 'lead' }, 'A separate agent wrote locked tests for your ' + plural(rows.length, 'requirement', 'requirements') + '. The builder cannot change them.' + (unchecked ? ' ' + plural(unchecked, 'requirement has', 'requirements have') + ' no test, so you check ' + (unchecked === 1 ? 'it' : 'them') + ' after the build.' : rows.length ? ' Every requirement has a test.' : '')),
		concerns.length ? element('div', { class: 'callout warn' }, element('h3', { class: 'section' }, 'The test author found ' + plural(concerns.length, 'problem', 'problems')), element('ul', { class: 'plain' }, concerns.map((text) => element('li', {}, text)))) : null,
		rows.length ? surface(element('h3', { class: 'section' }, 'Each requirement and the test that checks it'), rows.map((row) => check(row.text, !row.tests.length, row.tests.length ? element('span', { class: 'chips' }, row.tests.map((test) => element('span', { class: 'chip' }, test))) : element('span', { class: 'tag warn' }, 'You check this')))) : null,
		fold('See the tests', mode === 'page', element('p', { class: 'fine mono' }, list(expectation.files).join(', ')), diffView(expectation.patch)),
		expectation.notes ? fold('The author’s note', false, element('p', {}, rich(expectation.notes))) : null,
		element('p', { class: 'fine' }, buildDefaults(board.project)),
		element('div', { class: 'actions' },
			button('Approve and build', () => plan(card, { expectationId: expectation.id }), 'primary'),
			button('Write them again', async () => { await command('/expectations/' + e(expectation.id) + '/reject', {}); await plan(card, {}); }),
			button('Reject', async () => { await command('/expectations/' + e(expectation.id) + '/reject', {}); await decided('The tests were rejected.'); }, 'danger')),
		runNote(expectation.run),
	].filter(Boolean);
}
function startDecision(card, expectation) {
	return [element('p', { class: 'lead' }, 'The locked tests are agreed.' + (card.status.startsWith('Waits for') ? ' ' + card.status + '. ' + card.line : '')), element('div', { class: 'actions' }, button('Start the build', () => plan(card, { expectationId: expectation.id }), 'primary'))];
}
function retryDecision(card, message, label, action) {
	return [element('div', { class: 'callout bad' }, element('p', {}, sentence(message))), element('div', { class: 'actions' }, button(label, action, 'primary'))];
}
async function agreeDecision(card, board, refinement) {
	const orientation = await api('/projects/' + e(state.project) + '/orientation');
	const signals = (orientation.criterionSignals || {})[card.workItemId];
	const criteria = list(signals?.criteria);
	const flagged = criteria.filter((criterion) => criterion.flagged).length;
	const consent = consentBox(board);
	const agreed = refinement?.status === 'accepted';
	const actions = element('div', { class: 'actions' });
	if (flagged && board.canRefine && !agreed) actions.append(button('Sharpen the requirements', () => sharpen(card), 'primary'), button('Agree and start', () => plan(card, { continue: consent.value() })));
	else actions.append(button(agreed ? 'Start the next step' : 'Agree and start', () => plan(card, { continue: consent.value() }), 'primary'));
	if (!agreed && !flagged && board.canRefine) actions.append(button('Sharpen first', () => sharpen(card), 'quiet'));
	return [
		element('p', { class: 'lead' }, agreed ? 'The sharpened requirements are agreed.' : 'Agree what done means to start this work.'),
		criteria.length ? surface(element('h3', { class: 'section' }, 'The requirements'), criteria.map((criterion) => check(criterion.text, criterion.flagged, criterion.flagged ? element('span', { class: 'tag warn' }, 'Looks vague') : null))) : element('p', { class: 'fine' }, 'Its requirements are in the work item in the project records.'),
		element('p', { class: 'fine' }, board.canTest ? 'A separate agent writes locked tests first. You see them before the build, unless you let it carry on.' : 'The build starts right away, because this project has no locked tests.'),
		board.canTest ? consent.node : null,
		actions,
	].filter(Boolean);
}

// Approving the change

function criteriaFromAdmission(job) {
	const sources = list(job.admission?.sources);
	const ticket = sources.find((source) => source.path === job.admission.ticketPath) || sources[0];
	const result = [];
	let inside = false;
	for (const line of String(ticket?.text || '').split('\n')) {
		if (/^##\s+/.test(line)) { inside = /^##\s+Acceptance criteria\s*$/i.test(line.trim()); continue; }
		const item = inside && line.match(/^\s*[-*]\s+(.*)$/);
		if (item) result.push(item[1].trim());
	}
	return result;
}
const FINDINGS = [['wrong_requirement', 'A requirement was wrong'], ['missing_test', 'A requirement had no test'], ['bug', 'A bug'], ['hard_to_read', 'Hard to read'], ['out_of_scope', 'Out of scope']];
async function approveContent(jobId, mode) {
	const job = await api('/jobs/' + e(jobId));
	const [change, tests] = await Promise.all([api('/jobs/' + e(jobId) + '/change').catch(() => null), job.admission.expectationsId ? api('/expectations/' + e(job.admission.expectationsId)).catch(() => null) : null]);
	const rows = tests && list(tests.criteria).length ? coverageOf(tests) : criteriaFromAdmission(job).map((text) => ({ text, tests: [] }));
	const open = rows.filter((row) => !row.tests.length), covered = rows.filter((row) => row.tests.length);
	const files = list(change?.files);
	const added = files.reduce((sum, file) => sum + file.added, 0), removed = files.reduce((sum, file) => sum + file.removed, 0);
	if (!['ready_for_review', 'ready_for_acceptance'].includes(job.status)) return [element('p', { class: 'lead' }, 'This change is no longer waiting for approval: ' + human(job.status) + '.'), link('Open the job', '#job/' + e(job.id))];
	const independent = job.implementerId !== state.me.userId && (job.coordinatorId !== state.me.userId || job.admission.allowCoordinatorReview);
	const findings = new Set();
	const chips = FINDINGS.map(([key, label]) => element('button', { type: 'button', class: 'toggle-chip', 'aria-pressed': 'false', onclick: (event) => { if (findings.has(key)) findings.delete(key); else findings.add(key); event.currentTarget.setAttribute('aria-pressed', String(findings.has(key))); } }, label));
	const mergeBox = element('input', { type: 'checkbox', checked: can('merge', job.projectId) });
	const approve = async () => {
		const merge = mergeBox.checked;
		await command('/jobs/' + e(job.id) + '/approve', { jobId: job.id, ifVersion: job.version, candidateRevision: job.candidateRevision, requirementFingerprint: job.admission.ticketSHA256, findings: [...findings], merge });
		if (mode === 'drawer') await decided(merge ? 'Approved. The checks run again on the combined version before main moves.' : 'Approved.');
		else { await route(); notice('Approved.'); }
	};
	const fixText = element('textarea', { rows: 3, placeholder: 'What should change?', 'aria-label': 'What should change' });
	const fixForm = element('div', { class: 'stack', hidden: true }, fixText, element('div', { class: 'actions' }, button('Send back for a fix', async () => {
		const summary = fixText.value.trim();
		if (!summary) throw new Error('Say what should change.');
		const reviewed = await command('/jobs/' + e(job.id) + '/reviews', { jobId: job.id, ifVersion: job.version, candidateRevision: job.candidateRevision, requirementFingerprint: job.admission.ticketSHA256, decision: 'changes_requested', summary, findings: [...findings] });
		if (list(reviewed.availableActions).includes('repair')) await command('/jobs/' + e(job.id) + '/actions', { jobId: job.id, ifVersion: reviewed.version, action: 'repair' });
		await decided('Sent back for a fix.', mode === 'drawer' ? 'board' : 'job/' + e(job.id));
	}, 'primary')));
	let confirmReject = false;
	const reject = button('Reject', async () => {
		if (!confirmReject) { confirmReject = true; reject.textContent = 'Reject: click again to stop this build'; return; }
		await command('/jobs/' + e(job.id) + '/actions', { jobId: job.id, ifVersion: job.version, action: 'cancel' });
		await decided('The build was stopped.');
	}, 'danger');
	return [
		element('p', { class: 'lead' }, (files.length ? 'Changes ' + plural(files.length, 'file', 'files') + ' (+' + added + ' −' + removed + '). ' : '') + 'Every required check passed in a fresh copy' + (tests ? ', and the locked tests were not touched.' : '.')),
		open.length ? element('div', { class: 'callout warn' }, element('h3', { class: 'section' }, 'Check this yourself'), element('p', { class: 'fine' }, tests ? 'No locked test checks these, so they are left for you.' : 'This change has no locked tests, so check every requirement yourself.'), element('ul', { class: 'plain' }, open.map((row) => element('li', {}, rich(row.text))))) : element('div', { class: 'callout ok' }, element('p', {}, 'A locked test checks every requirement. Read the change if you want to; the checks already passed.')),
		change ? fold('The change', mode === 'page' || open.length > 0, element('div', { class: 'files' }, files.map((file) => element('span', {}, file.path + '  ', element('span', { class: 'plus' }, '+' + file.added), ' ', element('span', { class: 'minus' }, '−' + file.removed)))), diffView(change.patch), change.truncated ? element('p', { class: 'fine' }, 'The change is longer than shown here.') : null) : null,
		covered.length || list(job.admission.requiredChecks).length ? fold(element('span', {}, icon('check', 16), ' Already checked'), false, covered.map((row) => check(row.text, false, element('span', { class: 'chips' }, row.tests.map((test) => element('span', { class: 'chip' }, test))))), list(job.admission.requiredChecks).map((key) => check(key + ' passed in a fresh copy.'))) : null,
		fold(element('span', {}, 'Anything the checks missed? ', element('span', { class: 'muted' }, 'Optional')), false, element('div', { class: 'chips', role: 'group', 'aria-label': 'What you caught' }, chips), element('p', { class: 'fine' }, 'Your answer helps the factory decide when it can ask you less.')),
		independent ? null : element('div', { class: 'callout bad' }, element('p', {}, 'You submitted or built this work, so someone else has to approve it.')),
		can('merge', job.projectId) ? element('label', { class: 'switch' }, mergeBox, element('span', {}, element('strong', {}, 'Merge after approving. '), element('span', { class: 'muted' }, 'The checks run again on the combined version, and main moves only if they pass.'))) : null,
		element('div', { class: 'actions' }, element('button', { type: 'button', class: 'button primary', disabled: !independent, onclick: (event) => run(approve, event.currentTarget) }, can('merge', job.projectId) ? 'Approve and merge' : 'Approve'), button('Ask for a fix', () => { fixForm.hidden = !fixForm.hidden; if (!fixForm.hidden) fixText.focus(); }), reject),
		fixForm,
	].filter(Boolean);
}

// Work item and job pages

function stagesElement(card) {
	const position = { next: 0, plan: 0, build: 1, approve: 2, merge: 3, done: 4 }[card?.column] ?? 0;
	const names = ['Plan', 'Build', 'Approve', 'Merged'];
	return element('ol', { class: 'stages', 'aria-label': 'Progress' }, names.map((name, index) => {
		if (index < position || position === 4) return element('li', { class: 'done' }, '✓ ' + name);
		if (index === position) {
			if (card?.state === 'needs_you') return element('li', { class: 'you' }, name + ' · needs you');
			if (card?.state === 'running') return element('li', { class: 'auto' }, name + ' · running');
		}
		return element('li', {}, name);
	}));
}
function pageHead(card, title) {
	return element('div', { class: 'stack tight' }, element('div', { class: 'crumbs' }, element('a', { href: '#board' }, 'Board'), ' / ', title), element('div', { class: 'page-head' }, element('div', {}, element('h1', { class: 'title' }, title)), stagesElement(card)));
}
function statusSurface(card) {
	const eyebrow = card.state === 'needs_you' ? ['warn', 'Needs you'] : card.state === 'running' ? ['info', 'Automatic · nothing needs you'] : card.state === 'done' ? ['ok', 'Done'] : ['', 'Waiting'];
	return surface(element('span', { class: 'eyebrow ' + eyebrow[0] }, eyebrow[1]), element('div', { class: 'row' }, element('strong', { class: 'lead' }, card.status), elapsed(card.since)), element('p', {}, card.line), card.progress ? element('p', { class: 'progress-note' }, card.progress) : null);
}
function howItWorks() {
	return surface(element('h3', { class: 'section' }, 'After you agree'), element('ol', { class: 'requirements' }, element('li', {}, 'A separate agent writes locked tests and flags contradictions.'), element('li', {}, 'The build starts in its own copy, and checks run in a fresh one.'), element('li', {}, 'You approve the change, and it merges.')), element('p', { class: 'fine' }, element('a', { href: '#ask' }, 'Change when the factory asks you')));
}
async function itemPage(workItemId) {
	const board = await loadBoard();
	const card = cards(board).find((entry) => entry.workItemId === workItemId);
	if (!card) throw new Error('This work item is not on the board.');
	if (card.jobId) return jobPage(card.jobId);
	const main = card.decision === 'plan' ? [element('span', { class: 'eyebrow warn' }, card.state === 'needs_you' ? 'Needs you · agree what done means' : 'Agree what done means'), ...(await planContent(card, 'page'))] : [statusSurface(card)];
	return page('wide', pageHead(card, card.title), element('div', { class: 'columns' }, element('div', { class: 'main' }, main), element('div', { class: 'side' }, howItWorks())));
}
async function jobPage(id) {
	const [job, board] = await Promise.all([api('/jobs/' + e(id)), loadBoard()]);
	const card = cards(board).find((entry) => entry.jobId === job.id) || cards(board).find((entry) => entry.workItemId === job.workItemId);
	const title = card?.title || job.workItemId;
	let main;
	if (['ready_for_review', 'ready_for_acceptance'].includes(job.status)) main = [element('span', { class: 'eyebrow warn' }, 'Needs you · approve the change'), ...(await approveContent(job.id, 'page'))];
	else if (job.status === 'accepted') main = mergeSection(job);
	else if (['verification_failed', 'failed', 'stale', 'implementation_unknown', 'cancellation_unknown'].includes(job.status)) main = fixSection(job);
	else main = card ? [statusSurface(card)] : [surface(element('p', {}, human(job.status)))];
	return page('wide', pageHead(card, title), element('div', { class: 'columns' }, element('div', { class: 'main' }, main), element('div', { class: 'side' }, jobSide(job, card))), technicalDetails(job));
}
function mergeSection(job) {
	const merge = job.merge;
	const step = (title, detail, kind, label) => element('li', {}, element('span', { class: 'step-dot ' + kind, 'aria-hidden': 'true' }, kind === 'done' ? '✓' : kind === 'failed' ? '!' : ''), element('div', { class: 'stack tight' }, element('div', { class: 'row spread' }, element('strong', {}, title), element('span', { class: 'eyebrow ' + (kind === 'done' ? 'ok' : kind === 'failed' ? 'bad' : kind === 'running' ? 'info' : '') }, label)), element('span', { class: 'muted' }, detail)));
	if (!merge) {
		return [surface(element('span', { class: 'eyebrow warn' }, 'Needs you'), element('p', { class: 'lead' }, 'Approved but not merged. Work that depends on it waits.'), can('merge', job.projectId) ? element('div', { class: 'actions' }, button('Merge now', () => mergeJob(job), 'primary')) : element('p', { class: 'fine' }, 'Someone with merge access has to merge it.'))];
	}
	if (merge.status === 'verifying') {
		return [surface(element('span', { class: 'eyebrow info' }, 'Automatic · nothing needs you'), element('p', { class: 'lead' }, 'The factory is checking your approved change on top of the latest ' + merge.branch + '. It moves only if every check passes.'), element('ol', { class: 'steps' }, step('Combine with the latest ' + merge.branch, 'Your approved change on top of it.', 'done', 'Done'), step('Run the checks on the combined version', list(merge.checks).join(', ') || 'Required checks', 'running', 'Running'), step('Move ' + merge.branch, 'Only after every check passes.', '', 'Waiting')))];
	}
	if (merge.status === 'failed') {
		return [surface(element('span', { class: 'eyebrow bad' }, 'Needs you · merge failed'), element('p', { class: 'lead' }, merge.failure || 'The merge failed.'), element('p', { class: 'fine' }, merge.branch + ' was left as it was. Your approval is kept.'), can('merge', job.projectId) ? element('div', { class: 'actions' }, button('Try the merge again', () => mergeJob(job, merge.branch), 'primary')) : null)];
	}
	const problem = element('textarea', { rows: 3, placeholder: 'What turned out wrong?', 'aria-label': 'What turned out wrong' });
	return [
		surface(element('span', { class: 'eyebrow ok' }, 'Done'), element('p', { class: 'lead' }, 'Merged into ' + merge.branch + (merge.mergedAt ? ' ' + ago(merge.mergedAt) : '') + '.'), delegatedApproval(job), element('ol', { class: 'steps' }, step('Combine with the latest ' + merge.branch, '', 'done', 'Done'), step('Run the checks on the combined version', merge.mode === 'contained' ? 'The branch already contained the change.' : 'Every check passed.', 'done', 'Passed'), step('Move ' + merge.branch, short(merge.revision), 'done', 'Done'))),
		can('coordinate', job.projectId) ? fold('Found a problem later?', false, element('p', { class: 'fine' }, 'Reporting it resets the trust count and makes the factory ask one level more often.'), problem, element('div', { class: 'actions' }, button('Report a problem', async () => {
			const summary = problem.value.trim();
			if (!summary) throw new Error('Describe the problem.');
			await command('/jobs/' + e(job.id) + '/problems', { jobId: job.id, summary });
			await route(); notice('Problem reported.');
		}))) : null,
	].filter(Boolean);
}
// delegatedApproval says when the autopilot approved the change on a person's behalf.
function delegatedApproval(job) {
	const reason = [...list(job.events)].reverse().find((event) => event.kind === 'review_recorded')?.data?.delegated;
	return reason ? element('p', { class: 'fine' }, sentence('The autopilot approved this for you: ' + reason.replace(/^autopilot: /, ''))) : null;
}
async function mergeJob(job, branch = '') {
	await command('/jobs/' + e(job.id) + '/merge', { jobId: job.id, ifVersion: job.version, candidateRevision: job.candidateRevision, targetBranch: branch });
	await route();
	notice('Merging. The checks run on the combined version first.');
}
function fixSection(job) {
	const failed = list(job.checks).filter((result) => result.candidateRevision === job.candidateRevision && result.status !== 'passed').map((result) => result.checkKey + ' ' + result.status);
	const reasons = { verification_failed: 'The checks failed' + (failed.length ? ': ' + [...new Set(failed)].join(', ') + '.' : '.'), failed: 'The build failed.', stale: 'The requirements changed after the build started.', implementation_unknown: 'The builder stopped reporting. Decide what happened before anything else runs.', cancellation_unknown: 'The build was asked to stop, and it is not yet clear that it did.' };
	const actions = element('div', { class: 'actions' });
	for (const kind of list(job.availableActions)) {
		if (kind === 'resolve_execution') continue;
		const label = { repair: 'Ask for a fix (' + job.repairCount + ' of ' + job.admission.repairLimit + ' used)', cancel: 'Stop and reject', reconcile: 'Check what happened' }[kind] || human(kind);
		actions.append(button(label, async () => { await command('/jobs/' + e(job.id) + '/actions', { jobId: job.id, ifVersion: job.version, action: kind }); await route(); }, kind === 'repair' ? 'primary' : kind === 'cancel' ? 'danger' : ''));
	}
	const result = [surface(element('span', { class: 'eyebrow warn' }, 'Needs you'), element('p', { class: 'lead' }, reasons[job.status] || human(job.status)), actions.children.length ? actions : element('p', { class: 'fine' }, 'No action is available to you.'))];
	if (list(job.availableActions).includes('resolve_execution')) {
		const form = element('form', { class: 'stack' }, field('Uncertain attempt', 'attemptId', { choices: list(job.attempts).filter((attempt) => /unknown|running|assigned|registered|acknowledged/.test(attempt.status)).map((attempt) => ({ value: attempt.id, label: attempt.id })) }), field('What you observed', 'resolution', { choices: ['still_running', 'not_started', 'stopped_failed', 'stopped_cancelled'].map((value) => ({ value, label: human(value) })) }), field('Evidence', 'evidence', { type: 'textarea', required: true }));
		const submit = element('button', { type: 'submit', class: 'button' }, 'Record what happened');
		form.append(element('div', { class: 'actions' }, submit));
		form.addEventListener('submit', (event) => { event.preventDefault(); run(async () => { await command('/jobs/' + e(job.id) + '/actions', { ...data(form), jobId: job.id, ifVersion: job.version, action: 'resolve_execution' }); await route(); }, submit); });
		result.push(fold('Resolve an uncertain run', true, form));
	}
	return result;
}
function jobSide(job, card) {
	const tests = job.admission.expectationsId;
	return [
		surface(element('h3', { class: 'section' }, 'The build'), element('p', { class: 'muted' }, job.admission.executorKey + ' · ' + plural(job.repairCount, 'fix', 'fixes') + ' of ' + job.admission.repairLimit + ' used'), element('p', { class: 'fine' }, 'Started ' + ago(job.createdAt) + ' · deadline ' + when(job.admission.deadline)), card?.costUsd ? element('p', { class: 'fine' }, 'Agents have spent ' + money(card.costUsd) + ' on this item so far, planning included.') : null),
		tests ? surface(element('h3', { class: 'section' }, 'Locked tests'), element('p', { class: 'muted' }, 'The build started from tests the builder could not change.')) : null,
		element('p', { class: 'fine' }, element('a', { href: '#ask' }, 'Change when the factory asks you')),
	].filter(Boolean);
}
function technicalDetails(job) {
	const facts = element('dl', { class: 'stack tight' }, [['Job', job.id + ' · version ' + job.version], ['Candidate', job.candidateRevision || 'none yet'], ['Started from', job.admission.baseRevision], ['Requirements', job.admission.ticketSHA256], ['Acceptance record', job.acceptanceRecordPath || 'none'], ['Acceptance', job.acceptanceStatus]].map(([label, value]) => element('div', {}, element('dt', { class: 'eyebrow' }, label), element('dd', { class: 'mono fine' }, value))));
	const attempts = table(['Stage', 'Status', 'Run', 'Finished'], list(job.attempts).map((attempt) => [human(attempt.purpose === 'merge' ? 'merge check' : attempt.stage) + (attempt.checkKey ? ' · ' + attempt.checkKey : ''), human(attempt.checkStatus || attempt.status), element('span', { class: 'mono' }, attempt.nativeRunId || 'waiting'), attempt.finishedAt ? when(attempt.finishedAt) : '']));
	const events = element('ol', { class: 'timeline' }, [...list(job.events)].reverse().slice(0, 40).map((event) => element('li', {}, element('strong', {}, human(event.kind)), element('time', {}, '#' + event.sequence + ' · ' + when(event.createdAt)))));
	return fold('Technical details', false, facts, element('h3', { class: 'section' }, 'Attempts'), attempts, element('h3', { class: 'section' }, 'History'), events);
}

// Insights and the ask policy

async function insightsPage() {
	const [metrics, board] = await Promise.all([api('/projects/' + e(state.project) + '/metrics'), loadBoard()]);
	const tile = (label, value, note) => element('div', { class: 'tile' }, element('span', { class: 'label' }, label), element('span', { class: 'value' }, value), element('span', { class: 'fine' }, note));
	const decisions = metrics.humanDecisions + metrics.delegatedDecisions;
	const meter = element('div', { class: 'meter', 'aria-hidden': 'true' }, element('span', { class: 'human' }), element('span', { class: 'delegated' }));
	meter.children[0].style.flex = String(metrics.humanDecisions || 0);
	meter.children[1].style.flex = String(metrics.delegatedDecisions || 0);
	const trust = element('div', { class: 'meter', 'aria-hidden': 'true' }, element('span'));
	trust.children[0].style.width = Math.min(100, metrics.trustedMerges * 10) + '%';
	const findings = FINDINGS.filter(([key]) => metrics.findings?.[key]);
	return page('mid',
		element('div', { class: 'stack tight' }, element('h1', { class: 'title' }, 'How it’s going'), element('p', { class: 'subtitle' }, 'From the factory’s own records. ' + plural(metrics.mergedJobs, 'change merged', 'changes merged') + (metrics.mergedJobs < 10 ? ' so far, so no trends yet.' : '.'))),
		element('div', { class: 'tiles' },
			tile('Passed the first time', metrics.acceptedJobs ? metrics.firstPassAccepted + ' of ' + metrics.acceptedJobs : '—', plural(metrics.repairs, 'fix', 'fixes') + ' needed'),
			tile('Problems found after merge', String(metrics.problemsReported), 'Report one from a merged job'),
			tile('Your decisions per change', metrics.mergedJobs ? metrics.humanDecisionsPerMerge.toFixed(1) : '—', metrics.delegatedDecisions ? 'The autopilot made ' + metrics.delegatedDecisions + ' more for you' : 'At most two with the new flow')),
		decisions ? surface(element('h2', { class: 'section' }, 'Who decided'), meter, element('div', { class: 'legend' }, element('span', {}, 'You · ' + metrics.humanDecisions), element('span', { class: 'delegated' }, 'Autopilot, for you · ' + metrics.delegatedDecisions))) : null,
		surface(element('h2', { class: 'section' }, 'What reviews caught'), findings.length ? element('ul', { class: 'plain' }, findings.map(([key, label]) => element('li', {}, label + ': ' + metrics.findings[key]))) : element('p', { class: 'fine' }, 'Nothing recorded yet. When you approve, you can tag what the checks missed.')),
		surface(element('div', { class: 'row spread' }, element('h2', { class: 'section' }, 'Earning trust'), element('span', { class: 'mono' }, metrics.trustedMerges + ' of 10')), trust, element('p', { class: 'muted' }, 'Merges with no reported problem. At 10 you can let the factory merge without asking. A reported problem resets the count.')),
		element('div', { class: 'row spread' }, element('span', { class: 'muted' }, 'Agent cost: ' + (metrics.mergedJobs ? money(metrics.costPerMergedJobUsd) + ' per change, ' : '') + money(metrics.totalCostUsd) + ' in total.'), element('a', { href: '#ask' }, 'Decide when the factory asks →')),
		element('p', { class: 'fine' }, 'Measured ' + when(metrics.generatedAt) + '. Board refreshed ' + ago(board.generatedAt) + '.'));
}
async function askPage() {
	const board = await loadBoard();
	const ask = board.project.policy.ask || {};
	const editable = can('project_admin');
	const choice = { before: ask.beforeBuild || 'always', after: ask.afterBuild || 'always' };
	const group = (key, when, title, what, options) => {
		const buttons = options.map(([value, label, note, suggested, locked]) => element('button', { type: 'button', class: 'option', disabled: !editable || locked, 'aria-pressed': String(choice[key] === value), onclick: () => { choice[key] = value; buttons.forEach((entry, index) => entry.setAttribute('aria-pressed', String(options[index][0] === value))); } }, element('span', { class: 'radio' }), element('span', { class: 'grow' }, element('strong', {}, label), note ? element('span', { class: 'fine' }, note) : null), suggested ? element('span', { class: 'tag' }, 'Suggested') : null));
		return surface(element('span', { class: 'eyebrow' }, when), element('h2', { class: 'section' }, title), element('p', { class: 'muted' }, what), element('div', { class: 'stack tight', role: 'group', 'aria-label': title }, buttons));
	};
	const trusted = board.trustedMerges;
	const autopilot = element('input', { type: 'checkbox', checked: !!ask.autopilot, disabled: !editable });
	const saved = ask.setBy ? 'Saved ' + ago(ask.setAt) + (ask.setBy === state.me.userId ? ' by you.' : '.') : 'Not saved yet, so the factory asks at every step and starts nothing by itself.';
	return page('mid',
		element('div', { class: 'stack tight' }, element('h1', { class: 'title' }, 'How much to ask you'), element('p', { class: 'subtitle' }, 'Every check still runs. These settings only decide when the factory stops and waits for you.')),
		board.pilot ? null : element('div', { class: 'callout info' }, element('p', {}, 'The autopilot does not run on this installation, so nothing is skipped yet. Start the service with --pilot, or use solo mode.')),
		group('before', 'Before the build', 'Agree what done means', 'You read the sharpened requirements and the locked tests, and answer any questions. Then the build starts.', [['always', 'Every time'], ['when_unclear', 'Only when requirements look vague or the agent has questions', '', true], ['never', 'Never. Start building straight away', 'Locked tests that raise a concern still stop for you.']]),
		group('after', 'After the build', 'Approve the change', 'You check what tests can’t, and approve. Merging follows by itself.', [['always', 'Every time'], ['when_untested', 'Only when a requirement has no locked test'], ['never', 'Never. Merge when every check passes', trusted >= 10 ? 'Earned: ' + trusted + ' merges with no reported problem.' : 'Unlocks after 10 merges with no reported problem. ' + trusted + ' so far.', false, trusted < 10]]),
		surface(element('label', { class: 'switch' }, autopilot, element('span', { class: 'stack tight' }, element('strong', {}, 'Run the autopilot'), element('span', { class: 'muted' }, 'Start ready work by itself, write locked tests, fix failed checks up to the limit, and retry a merge that failed without a failing check. It acts as you, and every step it takes is marked as delegated.')))),
		element('div', { class: 'row spread' }, element('span', { class: 'fine' }, editable ? saved : saved + ' Only a project administrator can change it.'), editable ? button('Save', async () => { await command('/projects/' + e(state.project) + '/ask', { beforeBuild: choice.before, afterBuild: choice.after, autopilot: autopilot.checked }, 'PUT'); await route(); notice('Saved.'); }, 'primary') : null));
}

// Administration

async function adminPage(tab) {
	if (!state.me.installationAdmin) throw new Error('Installation administration access is required.');
	const tabs = element('nav', { class: 'segmented', 'aria-label': 'Administration' }, [['users', 'Users and tokens'], ['projects', 'Register a project'], ['profiles', 'Builders and checks']].map(([key, label]) => element('a', { href: '#admin/' + key, 'aria-current': tab === key ? 'page' : null }, label)), state.project ? element('a', { href: '#access' }, 'Project access') : null);
	const head = element('div', { class: 'stack' }, element('h1', { class: 'title' }, 'Administration'), tabs);
	if (tab === 'projects') return page('mid', head, registerProjectForm());
	if (tab === 'profiles') return page('mid', head, ...(await profileForms()));
	const users = list(await api('/users'));
	const usersTable = table(['User', 'Kind', 'Access', 'Status', ''], users.map((user) => [element('div', { class: 'stack tight' }, element('strong', {}, user.username), element('span', { class: 'fine mono' }, user.id)), user.kind, user.installationAdmin ? 'Installation admin' : 'Project grants', user.disabled ? 'disabled' : 'enabled', button(user.disabled ? 'Enable' : 'Disable', async () => { await command('/users/' + e(user.id) + '/disabled', { userId: user.id, disabled: !user.disabled }, 'PUT'); await route(); }, 'small')]));
	const create = element('form', { class: 'stack' }, formGrid(field('Username', 'username', { required: true }), field('Account kind', 'kind', { choices: [{ value: 'human', label: 'Person' }, { value: 'worker', label: 'Worker' }] }), field('Installation administrator', 'admin', { choices: [{ value: 'false', label: 'No' }, { value: 'true', label: 'Yes' }] })));
	const createButton = element('button', { type: 'submit', class: 'button primary' }, 'Create user');
	create.append(element('div', { class: 'actions' }, createButton));
	create.addEventListener('submit', (event) => { event.preventDefault(); run(async () => { const values = data(create); await command('/users', { username: values.username, kind: values.kind, installationAdmin: values.admin === 'true' }); await route(); }, createButton); });
	const token = element('form', { class: 'stack' }, formGrid(field('User', 'userId', { choices: users.map((user) => ({ value: user.id, label: user.username })) }), field('Token label', 'label', { required: true, value: 'workspace' })));
	const tokenButton = element('button', { type: 'submit', class: 'button' }, 'Issue token');
	const tokenOutput = element('div', { 'aria-live': 'polite' });
	token.append(element('div', { class: 'actions' }, tokenButton), tokenOutput);
	token.addEventListener('submit', (event) => { event.preventDefault(); run(async () => { const values = data(token); const issued = await command('/users/' + e(values.userId) + '/tokens', values); tokenOutput.replaceChildren(element('p', { class: 'fine' }, 'Save this token now. It is shown only once. Token ID: ' + issued.id), element('pre', { class: 'secret mono' }, issued.token)); }, tokenButton); });
	const revoke = element('form', { class: 'stack' }, field('Token ID to revoke', 'tokenId', { required: true }));
	const revokeButton = element('button', { type: 'submit', class: 'button danger' }, 'Revoke token');
	revoke.append(element('div', { class: 'actions' }, revokeButton));
	revoke.addEventListener('submit', (event) => { event.preventDefault(); run(async () => { const values = data(revoke); await command('/tokens/' + e(values.tokenId) + '/revoke', values); revoke.reset(); notice('Token revoked.'); }, revokeButton); });
	return page('mid', head, surface(element('h2', { class: 'section' }, 'Users'), usersTable), surface(element('h2', { class: 'section' }, 'Create a user'), create), surface(element('h2', { class: 'section' }, 'Issue a token'), token), surface(element('h2', { class: 'section' }, 'Revoke a token'), revoke));
}
async function accessPage() {
	const project = state.projects.find((entry) => entry.id === state.project);
	if (!can('project_admin')) throw new Error('Project administration access is required.');
	const grants = list(await api('/projects/' + e(project.id) + '/grants'));
	const choices = state.me.installationAdmin ? list(await api('/users')).map((user) => ({ value: user.id, label: user.username })) : null;
	const form = element('form', { class: 'stack' }, field('User', 'userId', choices ? { choices } : { required: true }), field('Capabilities, comma separated', 'capabilities', { value: 'read,coordinate', help: 'read, coordinate, verify, review, accept, merge, project_admin, worker_implement, worker_verify. Leave empty to remove all.' }));
	const submit = element('button', { type: 'submit', class: 'button primary' }, 'Save access');
	form.append(element('p', { class: 'fine' }, 'Saving replaces this user’s capabilities for the project.'), element('div', { class: 'actions' }, submit));
	form.addEventListener('submit', (event) => { event.preventDefault(); run(async () => { const values = data(form); await command('/projects/' + e(project.id) + '/grants/' + e(values.userId), { projectId: project.id, userId: values.userId, capabilities: split(values.capabilities) }, 'PUT'); await route(); }, submit); });
	return page('mid', element('h1', { class: 'title' }, 'Access to ' + project.title), surface(grants.length ? table(['User', 'Capabilities'], grants.map((grant) => [element('span', { class: 'mono' }, grant.userId), list(grant.capabilities).join(', ')])) : empty('No explicit grants', 'Installation administrators keep access.')), surface(form));
}
function registerProjectForm() {
	const form = element('form', { class: 'stack' }, formGrid(
		field('Records folder', 'bundlePath', { required: true, full: true, placeholder: '/absolute/path/to/records' }),
		field('Code repository', 'repositoryPath', { required: true, full: true, placeholder: '/absolute/path/to/repository' }),
		field('Other allowed source folders, one per line', 'allowedSourceRoots', { type: 'textarea', full: true }),
		field('Required checks', 'requiredChecks', { required: true, placeholder: 'go-test' }),
		field('Allowed builders', 'allowedExecutors', { required: true, placeholder: 'claude-agent' }),
		field('Locked test paths, one per line', 'protectedPaths', { type: 'textarea', full: true, help: 'Tests the builder may not change, such as tests/ or **/*_test.go. Locked tests need these.' }),
		field('Who can approve', 'allowCoordinatorReview', { choices: [{ value: 'false', label: 'Someone other than the person who started the work' }, { value: 'true', label: 'Anyone, including the person who started it' }] }),
		field('Most fixes per change', 'maxRepairAttempts', { type: 'number', min: 0, max: 10, value: 2, required: true }),
		field('Longest build, seconds', 'maxJobDurationSeconds', { type: 'number', min: 60, value: 3600, required: true }),
		field('Mode', 'fixture', { choices: [{ value: 'false', label: 'Real project' }, { value: 'true', label: 'Fixture test project' }] })));
	const submit = element('button', { type: 'submit', class: 'button primary' }, 'Register the project');
	form.append(element('div', { class: 'actions' }, submit), element('p', { class: 'fine' }, 'The factory reads the existing records in place and keeps their identities.'));
	form.addEventListener('submit', (event) => { event.preventDefault(); run(async () => { const values = data(form); const project = await command('/projects', { bundlePath: values.bundlePath, repositoryPath: values.repositoryPath, allowedSourceRoots: lines(values.allowedSourceRoots), readerLimits: { maxFiles: 0, maxBytes: 0 }, policy: { requiredChecks: split(values.requiredChecks), allowedExecutors: split(values.allowedExecutors), maxRepairAttempts: Number(values.maxRepairAttempts), maxJobDurationSeconds: Number(values.maxJobDurationSeconds), fixture: values.fixture === 'true', protectedPaths: lines(values.protectedPaths), allowCoordinatorReview: values.allowCoordinatorReview === 'true' } }); try { localStorage.setItem('waymark-project', project.id); } catch {} await refreshProjects(); await navigate('board'); }, submit); });
	return surface(element('h2', { class: 'section' }, 'Register a project'), form);
}
async function profileForms() {
	const [executors, checks] = await Promise.all([api('/executors'), api('/checks')]);
	const result = [surface(element('h2', { class: 'section' }, 'Registered builders and checks'), table(['Type', 'Key', 'Command', 'Mode'], [...list(executors).map((profile) => ['Builder', profile.key, element('span', { class: 'mono' }, JSON.stringify(profile.command)), profile.fixture ? 'fixture' : 'real']), ...list(checks).map((profile) => ['Check', profile.key, element('span', { class: 'mono' }, JSON.stringify(profile.command)), profile.fixture ? 'fixture' : 'real'])]))];
	for (const kind of ['executors', 'checks']) {
		const form = element('form', { class: 'stack' }, formGrid(field('Key', 'key', { required: true }), field('Display name', 'displayName', { required: true }), field('Command as a JSON array', 'command', { type: 'textarea', full: true, required: true, value: '["/absolute/path/to/program"]', help: 'A literal argument array, run without a shell.' }), field('Mode', 'fixture', { choices: [{ value: 'false', label: 'Real' }, { value: 'true', label: 'Fixture' }] }), kind === 'checks' ? field('Environment', 'environment', { required: true, value: 'local' }) : null));
		const submit = element('button', { type: 'submit', class: 'button primary' }, kind === 'checks' ? 'Register the check' : 'Register the builder');
		form.append(element('div', { class: 'actions' }, submit));
		form.addEventListener('submit', (event) => { event.preventDefault(); run(async () => { const values = data(form); const argv = JSON.parse(values.command); if (!Array.isArray(argv) || argv.some((arg) => typeof arg !== 'string')) throw new Error('The command must be a JSON array of strings.'); await command('/' + kind, { key: values.key, displayName: values.displayName, command: argv, fixture: values.fixture === 'true', ...(kind === 'checks' ? { environment: values.environment } : {}) }); await route(); }, submit); });
		result.push(surface(element('h2', { class: 'section' }, kind === 'checks' ? 'Register a check' : 'Register a builder'), form));
	}
	return result;
}

// Wiring

applyTheme();
matchMedia('(prefers-color-scheme: dark)').addEventListener('change', applyTheme);
$('#theme-toggle').addEventListener('click', () => {
	state.theme = resolvedTheme() === 'dark' ? 'light' : 'dark';
	try { localStorage.setItem('waymark-theme', state.theme); } catch {}
	applyTheme();
});
$('#signin-form').addEventListener('submit', (event) => {
	event.preventDefault();
	run(async () => {
		$('#signin-error').hidden = true;
		state.token = $('#token').value.trim();
		try { await connect(); } catch (error) { signout(); $('#signin-error').textContent = error.message; $('#signin-error').hidden = false; }
	}, event.submitter);
});
$('#signout').addEventListener('click', signout);
$('#project-select').addEventListener('change', (event) => {
	state.project = event.target.value;
	try { localStorage.setItem('waymark-project', state.project); } catch {}
	navigate('board');
});
document.addEventListener('keydown', (event) => { if (event.key === 'Escape' && state.drawer) closeDrawer(true); });
window.addEventListener('hashchange', () => run(() => route()));
if (state.token) connect().catch((error) => { signout(); $('#signin-error').textContent = error.message; $('#signin-error').hidden = false; });
setInterval(() => {
	for (const node of document.querySelectorAll('[data-since]')) node.textContent = clock((Date.now() - Date.parse(node.dataset.since)) / 1000);
}, 1000);
setInterval(async () => {
	if (!state.me || !state.project || document.hidden || mutationsInFlight || state.drawer || /INPUT|TEXTAREA|SELECT/.test(document.activeElement?.tagName || '')) return;
	if (!['board', 'list', 'item', 'job', ''].includes(state.view)) return;
	try {
		const board = await api('/projects/' + e(state.project) + '/board');
		if (boardKey(board) !== state.boardKey) await route(true);
	} catch (error) { notice(error.message, true); }
}, 3000);

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
	for (const child of children.flat(Infinity)) if (child !== null && child !== undefined && child !== false) result.append(child instanceof Node ? child : document.createTextNode(String(child)));
	return result;
}
const ICONS = {
	check: ['M20 6 9 17l-5-5'],
	close: ['M18 6 6 18M6 6l12 12'],
	moon: ['M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z'],
	sun: ['M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41', 'M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0z'],
	eye: ['M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z', 'M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0z'],
	arrow: ['M5 12h14M12 5l7 7-7 7'],
	plus: ['M12 5v14M5 12h14'],
	spark: ['M12 3v4M12 17v4M3 12h4M17 12h4M6.3 6.3l2.5 2.5M15.2 15.2l2.5 2.5M17.7 6.3l-2.5 2.5M8.8 15.2l-2.5 2.5'],
	person: ['M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8z', 'M4 21a8 8 0 0 1 16 0'],
	terminal: ['m5 8 4 4-4 4', 'M12 16h7'],
	merge: ['M6 3v12', 'M18 9a3 3 0 1 0 0-6 3 3 0 0 0 0 6z', 'M6 21a3 3 0 1 0 0-6 3 3 0 0 0 0 6z', 'M18 9a9 9 0 0 1-9 9'],
	pencil: ['M12 20h9', 'M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z'],
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
	state.token = ''; state.me = null; state.board = null; state.workspace = null; state.generation++; pending.clear(); taskSpaces.clear();
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
	const parts = location.hash.slice(1).split('/').map(decodeURIComponent);
	const nextId = parts[0] === 'new' ? 'new' : ['item', 'run', 'history', 'decision'].includes(parts[0]) ? parts[1] : null;
	const sameTask = state.workspace && nextId === state.workspace.id && state.project === state.workspace.project;
	if (!quiet && state.workspace && !sameTask && !canLeaveWorkspace()) {
		history.replaceState(null, '', state.routeHash || '#board');
		return;
	}
	if (!quiet && ['board', 'list', 'graph'].includes(state.view) && nextId) state.boardReturn = { hash: state.routeHash || '#board', view: state.view, top: window.scrollY, left: document.querySelector('.board-scroll')?.scrollLeft || 0 };
	if (!sameTask) state.workspace = null;
	const generation = ++state.generation;
	state.view = parts[0] || 'board';
	state.routeHash = location.hash || '#board';
	const scroll = window.scrollY;
	const horizontal = document.querySelector('.board-scroll')?.scrollLeft || 0;
	if (!quiet) { closeDrawer(false); notice(state.flash); state.flash = ''; }
	const area = { board: 'board', list: 'board', graph: 'board', run: 'board', history: 'board', decision: 'board', new: 'board', item: 'board', job: 'board', insights: 'insights', ask: 'ask', agents: 'agents' }[state.view] || '';
	for (const anchor of document.querySelectorAll('[data-nav]')) {
		if (anchor.dataset.nav === area) anchor.setAttribute('aria-current', 'page'); else anchor.removeAttribute('aria-current');
	}
	if (!quiet && !sameTask) $('#content').replaceChildren(element('p', { class: 'loading' }, 'Loading…'));
	try {
		let content;
		if (state.view === 'admin') content = await adminPage(parts[1] || 'users');
		else if (!state.project) content = noProjects();
		else if (state.view === 'list') content = await listPage();
		else if (state.view === 'graph') content = await graphPage();
		else if (state.view === 'new') content = await taskWorkspace('new');
		else if (['run', 'history', 'decision'].includes(state.view)) content = await taskWorkspace(parts[1] || '', state.view);
		else if (state.view === 'item') content = await itemPage(parts[1] || '');
		else if (state.view === 'job') content = await jobPage(parts[1] || '');
		else if (state.view === 'insights') content = await insightsPage();
		else if (state.view === 'ask') content = await askPage();
		else if (state.view === 'agents') content = await agentsPage();
		else if (state.view === 'access') content = await accessPage();
		else content = await boardPage();
		if (generation !== state.generation) return;
		if ($('#content').firstChild !== content) $('#content').replaceChildren(content);
		if (nextId) state.workspace = taskSpaces.get(taskKey(state.project, nextId));
		if (quiet) { window.scrollTo(0, scroll); const boardScroll = document.querySelector('.board-scroll'); if (boardScroll) boardScroll.scrollLeft = horizontal; }
		else if (state.boardReturn?.hash === location.hash) { window.scrollTo(0, state.boardReturn.top); const boardScroll = document.querySelector('.board-scroll'); if (boardScroll) boardScroll.scrollLeft = state.boardReturn.left; }
		else if (!sameTask) window.scrollTo(0, 0);
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
	return JSON.stringify(list(board.cards).map((card) => [card.workItemId, card.column, card.state, card.status, card.line, card.jobVersion, card.refinementId, card.expectationId, card.progress, list(card.blockedBy).join()]));
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
	const views = element('nav', { class: 'segmented', 'aria-label': 'View' }, element('a', { href: '#board', 'aria-current': view === 'board' ? 'page' : null }, 'Board'), element('a', { href: '#list', 'aria-current': view === 'list' ? 'page' : null }, 'List'), element('a', { href: '#graph', 'aria-current': view === 'graph' ? 'page' : null }, 'Graph'));
	const add = can('coordinate') ? button('Add work', () => openWorkSheet()) : null;
	return element('div', { class: 'page-head' }, element('div', {}, element('h1', { class: 'title' }, title), element('p', { class: 'subtitle' }, line)), element('div', { class: 'row' }, views, add, board.publicationPending && can('coordinate') ? button('Resume pending save', async () => { await command('/projects/' + e(state.project) + '/work-items/reconcile', {}); await route(); }) : null, needs.length ? button('Next: ' + needs[0].title, () => openCard(needs[0]), 'primary') : null));
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
		entries.length ? entries.map(cardElement) : element('p', { class: 'column-empty' }, { next: 'Nothing waiting.', plan: 'Nothing being planned.', build: 'Nothing being built.', approve: 'Nothing to approve.', merge: 'Nothing merging.', done: 'Nothing merged yet.' }[key]),
		key === 'next' && can('coordinate') ? element('button', { type: 'button', class: 'add-work', onclick: () => openWorkSheet() }, icon('plus', 16), 'Add work') : null);
}
function cardAction(card) {
	if (card.decision === 'plan') return card.questions ? 'Answer' : card.state === 'needs_you' ? 'Review the plan' : 'Start';
	if (card.decision === 'approve') return 'Review';
	return 'Open';
}
const STEPS = ['Plan', 'Agree', 'Build', 'Checks', 'Approve', 'Merge'];
// cardElement shows a card as a few scannable parts: a state chip, the title, short
// facts, the six-step bar, and an action only when it needs you.
function cardElement(card) {
	const kind = { needs_you: 'need', running: 'running', waiting: 'waiting', done: 'done' }[card.state];
	const title = element('span', { class: 'card-title' }, card.title);
	if (card.state === 'done') {
		return element('button', { type: 'button', class: 'card done', onclick: () => openCard(card) }, title, element('span', { class: 'card-done' }, icon('check', 13), card.status, card.since ? element('span', { class: 'card-meta' }, '· ' + ago(card.since)) : null));
	}
	const blocked = card.state === 'waiting' && card.status.startsWith('Waits for');
	const chip = element('span', { class: 'state-chip ' + kind },
		card.state === 'running' ? element('span', { class: 'dot', 'aria-hidden': 'true' }) : card.state === 'needs_you' ? element('span', { class: 'chip-dot', 'aria-hidden': 'true' }) : null,
		card.state === 'needs_you' ? 'Needs you' : blocked ? 'Waiting' : card.status);
	const meta = card.state === 'running' ? elapsed(card.since) : card.costUsd ? element('span', { class: 'card-meta' }, money(card.costUsd)) : null;
	const facts = list(card.facts).length ? element('span', { class: 'card-facts' }, list(card.facts).map((item) => element('span', { class: 'card-fact' + (item.tone ? ' ' + item.tone : '') }, item.text))) : null;
	const step = Math.min(card.step || 0, STEPS.length - 1);
	const bar = element('span', { class: 'step-bar', role: 'img', 'aria-label': 'Step ' + (step + 1) + ' of ' + STEPS.length + ': ' + STEPS[step] }, STEPS.map((name, index) => element('span', { class: 'step-seg' + (index < step ? ' done' : index === step ? ' ' + kind : '') })));
	const action = card.state === 'needs_you' ? card.status : card.decision ? cardAction(card) : '';
	return element('button', { type: 'button', class: 'card ' + kind, onclick: () => openCard(card) },
		element('span', { class: 'card-top' }, chip, meta), title, facts,
		card.state === 'running' && card.progress ? element('span', { class: 'progress-note' }, card.progress) : null,
		bar, action ? element('span', { class: 'card-action' }, action, icon('arrow', 16)) : null);
}
function openCard(card) { return navigate('item/' + e(card.workItemId)); }
async function boardPage() {
	const board = await loadBoard();
	return page('', boardHead(board, 'board'), element('div', { class: 'board-scroll' }, element('div', { class: 'board' }, COLUMNS.map(([key, name]) => columnElement(board, key, name)))), await taskInventory());
}
async function taskInventory() {
	const records = list(await api('/projects/' + e(state.project) + '/work-items'));
	const rows = element('div', { class: 'stack tight' });
	const filter = element('select', { 'aria-label': 'Task filter', onchange: () => { state.taskFilter = filter.value; draw(); } }, [['drafts', 'Unqueued drafts'], ['archived', 'Archived'], ['all', 'All tasks']].map(([value, label]) => element('option', { value }, label)));
	filter.value = state.taskFilter || 'drafts';
	const draw = () => {
		const selected = records.filter((record) => filter.value === 'all' || (filter.value === 'archived' ? record.archived : !record.queued && !record.archived));
		rows.replaceChildren(...(selected.length ? selected.map((record) => element('a', { class: 'list-row', href: '#item/' + e(record.workItemId) }, element('strong', { class: 'grow' }, record.draft.title), element('span', { class: 'tag' }, record.archived ? 'Archived' : record.queued ? 'Queued' : 'Draft'))) : [element('p', { class: 'fine' }, 'No tasks match this filter.')]));
	};
	draw();
	return element('section', { class: 'surface stack task-inventory' }, element('div', { class: 'row spread' }, element('h2', { class: 'section' }, 'Task library'), filter), rows);
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

// Task editing and workspace

// Markdown is built entirely with text nodes. Raw HTML remains visible text, and
// links accept only HTTP(S), mailto, fragments and relative references.
function markdownInline(text) {
	const result = [];
	const pattern = /(`[^`\n]+`|\[[^\]\n]+\]\([^\s)]+\)|\*\*[^*\n]+\*\*|\*[^*\n]+\*)/g;
	let cursor = 0;
	for (const match of String(text).matchAll(pattern)) {
		result.push(text.slice(cursor, match.index));
		const part = match[0];
		if (part.startsWith('`')) result.push(element('code', {}, part.slice(1, -1)));
		else if (part.startsWith('[')) {
			const link = part.match(/^\[([^\]]+)\]\(([^)]+)\)$/);
			const destination = link[2];
			const safe = !/[\u0000-\u0020\u007f]/.test(destination) && !destination.startsWith('//') && !destination.includes('\\') && (!/^[\w+.-]+:/.test(destination) || /^(https?:|mailto:)/i.test(destination));
			result.push(safe ? element('a', { href: destination, target: destination.startsWith('#') ? null : '_blank', rel: 'noopener noreferrer' }, link[1]) : link[1]);
		} else result.push(element(part.startsWith('**') ? 'strong' : 'em', {}, part.slice(part.startsWith('**') ? 2 : 1, part.startsWith('**') ? -2 : -1)));
		cursor = match.index + part.length;
	}
	result.push(text.slice(cursor));
	return result;
}
function markdown(text) {
	const root = element('div', { class: 'markdown' });
	const source = String(text || '').replaceAll('\r\n', '\n').split('\n');
	let index = 0;
	while (index < source.length) {
		const line = source[index];
		if (!line.trim()) { index++; continue; }
		const fence = line.match(/^\s*(```+|~~~+)(.*)$/);
		if (fence) {
			const body = []; index++;
			while (index < source.length && !source[index].trim().startsWith(fence[1])) body.push(source[index++]);
			index++;
			root.append(element('pre', {}, element('code', {}, body.join('\n'))));
			continue;
		}
		const heading = line.match(/^(#{1,6})\s+(.*)$/);
		if (heading) { root.append(element('h' + Math.min(6, heading[1].length + 1), {}, markdownInline(heading[2]))); index++; continue; }
		if (/^\s*(?:[-*+] |\d+[.)] )/.test(line)) {
			const ordered = /^\s*\d/.test(line), items = element(ordered ? 'ol' : 'ul');
			while (index < source.length && (ordered ? /^\s*\d+[.)] / : /^\s*[-*+] /).test(source[index])) {
				items.append(element('li', {}, markdownInline(source[index++].replace(/^\s*(?:[-*+]|\d+[.)])\s+/, ''))));
			}
			root.append(items); continue;
		}
		if (/^>\s?/.test(line)) {
			const quote = []; while (index < source.length && /^>\s?/.test(source[index])) quote.push(source[index++].replace(/^>\s?/, ''));
			root.append(element('blockquote', {}, markdown(quote.join('\n')))); continue;
		}
		const paragraph = [line]; index++;
		while (index < source.length && source[index].trim() && !/^(#{1,6}\s|\s*```|\s*~~~|>\s?|\s*[-*+] |\s*\d+[.)] )/.test(source[index])) paragraph.push(source[index++]);
		root.append(element('p', {}, markdownInline(paragraph.join('\n'))));
	}
	return root;
}
function normalizedDraft(draft) {
	return { title: draft.title || '', description: draft.description || '', criteria: list(draft.criteria).filter((text) => text.trim()), blockedBy: [...list(draft.blockedBy)], spec: [...list(draft.spec)], context: [...list(draft.context)] };
}
function workForm(board, draft, selfId, onChange = () => {}) {
	const projectId = state.project, values = normalizedDraft(draft);
	if (!values.criteria.length) values.criteria.push('');
	let timer = 0, scoreGeneration = 0, disposed = false;
	const changed = () => onChange(normalizedDraft(values));
	const title = element('input', { type: 'text', value: values.title, maxLength: 200, required: true, 'aria-label': 'Title', oninput: (event) => { values.title = event.target.value; changed(); score(); } });
	const description = element('textarea', { rows: 12, value: values.description, 'aria-label': 'What and why', oninput: (event) => { values.description = event.target.value; changed(); preview.replaceChildren(markdown(values.description)); } });
	const preview = element('div', { class: 'description-preview', hidden: true }, markdown(values.description));
	const writing = element('button', { type: 'button', 'aria-pressed': 'true', onclick: () => { description.hidden = false; preview.hidden = true; writing.setAttribute('aria-pressed', 'true'); reading.setAttribute('aria-pressed', 'false'); description.focus(); } }, 'Write');
	const reading = element('button', { type: 'button', 'aria-pressed': 'false', onclick: () => { description.hidden = true; preview.hidden = false; reading.setAttribute('aria-pressed', 'true'); writing.setAttribute('aria-pressed', 'false'); } }, 'Preview');
	const rows = element('div', { class: 'stack tight' }), vague = element('p', { class: 'fine', role: 'status' });
	const score = () => {
		clearTimeout(timer); const generation = ++scoreGeneration;
		timer = setTimeout(async () => {
			const criteria = values.criteria.filter((text) => text.trim());
			if (!criteria.length) { vague.textContent = ''; return; }
			let signals;
			try { signals = await api('/projects/' + e(projectId) + '/criteria/score', { method: 'POST', body: { title: values.title, criteria } }); } catch { signals = null; }
			if (disposed || generation !== scoreGeneration) return;
			let position = 0;
			for (const [index, row] of [...rows.children].entries()) {
				const signal = values.criteria[index]?.trim() ? list(signals?.criteria)[position++] : null;
				const badge = row.querySelector('[data-score]');
				badge.textContent = signals?.status === 'observed' && signal ? signal.flagged ? 'Looks vague' : 'Checkable' : '';
				badge.className = 'card-fact ' + (signal?.flagged ? 'warn' : 'ok');
			}
			const flagged = list(signals?.criteria).filter((item) => item.flagged).length;
			vague.textContent = flagged ? plural(flagged, 'requirement looks', 'requirements look') + ' vague. You can save and refine it before starting.' : '';
		}, 600);
	};
	const addRow = (text) => {
		const textarea = element('textarea', { rows: 2, value: text, placeholder: 'A result someone could check', oninput: () => { values.criteria[[...rows.children].indexOf(row)] = textarea.value; changed(); score(); } });
		const row = element('div', { class: 'requirement-row' }, element('span', { class: 'requirement-number', 'aria-hidden': 'true' }), textarea, element('span', { class: 'requirement-side' }, element('span', { 'data-score': '' }), element('button', { type: 'button', class: 'button quiet', onclick: () => { const index = [...rows.children].indexOf(row); values.criteria.splice(index, 1); row.remove(); numberRows(); changed(); score(); } }, 'Remove')));
		rows.append(row); numberRows(); return textarea;
	};
	const numberRows = () => [...rows.children].forEach((row, index) => { row.querySelector('.requirement-number').textContent = index + 1; row.querySelector('textarea').setAttribute('aria-label', 'Requirement ' + (index + 1)); row.querySelector('button').setAttribute('aria-label', 'Remove requirement ' + (index + 1)); });
	values.criteria.forEach(addRow);
	const deps = element('div', { class: 'dep-row' });
	const others = cards(board).filter((card) => card.workItemId !== selfId);
	const drawDeps = () => {
		deps.replaceChildren(...values.blockedBy.map((id) => element('span', { class: 'dep-chip' }, others.find((card) => card.workItemId === id)?.title || id, element('button', { type: 'button', 'aria-label': 'Remove dependency ' + id, onclick: () => { values.blockedBy = values.blockedBy.filter((value) => value !== id); drawDeps(); changed(); } }, icon('close', 14)))), element('select', { 'aria-label': 'Add a dependency', onchange: (event) => { if (event.target.value) { values.blockedBy.push(event.target.value); drawDeps(); changed(); } } }, element('option', { value: '' }, 'Add a dependency'), others.filter((card) => !values.blockedBy.includes(card.workItemId)).map((card) => element('option', { value: card.workItemId }, card.title))));
	};
	drawDeps(); score();
	const links = (key, label) => field(label, key, { type: 'textarea', rows: 3, value: values[key].join('\n'), help: 'One document path per line, relative to the task record.', oninput: (event) => { values[key] = lines(event.target.value); changed(); } });
	const form = element('div', { class: 'stack loose work-form' }, element('label', { class: 'field' }, element('span', { class: 'field-label' }, 'Title'), title), element('div', { class: 'field' }, element('div', { class: 'row spread' }, element('span', { class: 'field-label' }, 'What and why'), element('div', { class: 'segmented', 'aria-label': 'Description mode' }, writing, reading)), description, preview, element('span', { class: 'fine' }, 'Markdown supports headings, lists, links and code blocks.')), element('fieldset', { class: 'stack' }, element('legend', {}, 'Done when'), rows, button('Add a requirement', () => { values.criteria.push(''); addRow('').focus(); changed(); score(); }, 'quiet'), vague), element('div', { class: 'stack tight' }, element('span', { class: 'field-label' }, 'Depends on'), deps), formGrid(links('spec', 'Specification links'), links('context', 'Context links')));
	return { form, draft: () => normalizedDraft(values), dispose: () => { disposed = true; clearTimeout(timer); } };
}
function openWorkSheet() { return navigate('new'); }

const taskSpaces = new Map();
const RUN_KIND = { agent: 'Agent', you: 'You', check: 'Check', automatic: 'Automatic' };
const RUN_STATUS = { done: 'Done', running: 'Running', needs_you: 'Needs you', next: 'Next', idle: 'Idle', skipped: 'Skipped', failed: 'Failed' };
const taskKey = (project, id) => project + '/' + id;
const taskURL = (space) => '/projects/' + e(space.project) + '/work-items/' + e(space.id);
const latestJob = (space) => list(space.activity?.jobs)[0];
const independentReview = (job) => job.implementerId !== state.me.userId && (job.coordinatorId !== state.me.userId || job.admission.allowCoordinatorReview);
const canReviewJob = (job) => !!job && job.status === 'ready_for_review' && state.me?.kind === 'human' && can('review', job.projectId) && independentReview(job);
const canApproveJob = (job) => !!job && state.me?.kind === 'human' && can('accept', job.projectId) && (job.status === 'ready_for_acceptance' || canReviewJob(job));
const canRejectJob = (job) => !!job && can('coordinate', job.projectId) && list(job.availableActions).includes('cancel');
const canPlan = (projectId = state.project) => state.me?.kind === 'human' && can('coordinate', projectId);
const canOpenResultDecision = (job) => canApproveJob(job) || canReviewJob(job) || canRejectJob(job);

function workspaceHasInput(space) { return !!(space?.dirty || space?.decisionDirty || space?.disputeDraft?.dirty || [...(space?.disputeReviews?.values() || [])].some((review) => review.dirty)); }
function dirtyWorkspace() { return workspaceHasInput(state.workspace); }
function canLeaveWorkspace() { return !dirtyWorkspace() || confirm('Leave this task with unsaved changes? Your input stays in this tab.'); }
function itemTabs(workItemId, current) {
	return element('nav', { class: 'segmented task-tabs', 'aria-label': 'Task workspace' }, ['task', 'run', 'history'].map((tab) => element('a', { href: '#' + (tab === 'task' ? 'item' : tab) + '/' + e(workItemId), 'aria-current': tab === current ? 'page' : null }, tab[0].toUpperCase() + tab.slice(1))));
}
function taskDescription(record) {
	const draft = record.draft || {};
	return element('div', { class: 'stack loose' }, draft.description ? markdown(draft.description) : element('p', { class: 'muted' }, 'No description yet.'), element('section', { class: 'stack' }, element('h2', { class: 'section' }, 'Done when'), element('ol', { class: 'requirements' }, list(draft.criteria).map((text) => element('li', {}, markdown(text))))), list(draft.blockedBy).length ? element('section', { class: 'stack tight' }, element('h2', { class: 'section' }, 'Dependencies'), list(draft.blockedBy).map((id) => element('a', { href: '#item/' + e(id) }, cards(state.board || {}).find((card) => card.workItemId === id)?.title || id))) : null, ['spec', 'context'].map((key) => list(draft[key]).length ? element('section', { class: 'stack tight' }, element('h2', { class: 'section' }, key === 'spec' ? 'Specification links' : 'Context links'), list(draft[key]).map((path) => element('code', { class: 'reference-path' }, path))) : null));
}
function patchRegion(target, identity, build) {
	if (target.dataset.snapshot === identity) return;
	const opened = new Map([...target.querySelectorAll('details[data-key]')].map((node) => [node.dataset.key, node.open]));
	const scrolls = [...target.querySelectorAll('[data-scroll]')].map((node) => [node.dataset.scroll, node.scrollTop, node.scrollLeft]);
	target.replaceChildren(...[build()].flat(Infinity).filter(Boolean)); target.dataset.snapshot = identity;
	for (const node of target.querySelectorAll('details[data-key]')) if (opened.has(node.dataset.key)) node.open = opened.get(node.dataset.key);
	for (const [key, top, left] of scrolls) { const node = [...target.querySelectorAll('[data-scroll]')].find((item) => item.dataset.scroll === key); if (node) { node.scrollTop = top; node.scrollLeft = left; } }
}
async function taskWorkspace(id, tab = 'task') {
	const key = taskKey(state.project, id), project = state.project;
	let space = taskSpaces.get(key);
	if (!space) {
		const board = state.board?.project.id === project ? state.board : await loadBoard();
		const [record, inventory] = await Promise.all([id === 'new' ? { draft: normalizedDraft({}), workItemId: 'new', actions: [] } : api('/projects/' + e(project) + '/work-items/' + e(id)), api('/projects/' + e(project) + '/work-items')]);
		space = { id, project, record, board, inventory: list(inventory), tab, follow: true, selected: '', dirty: false, editor: null, artifactNodes: new Map(), lastSuccess: null, error: '', refreshSequence: 0 };
		taskSpaces.set(key, space); buildWorkspace(space);
		if (id === 'new') editTask(space);
	}
	setWorkspaceTab(space, tab);
	if (id !== 'new') refreshWorkspace(space);
	return space.root;
}
function buildWorkspace(space) {
	space.title = element('h1', { class: 'title' }, space.id === 'new' ? 'New task' : space.record.draft.title);
	space.status = element('span', { class: 'state-chip' }); space.dirtyLabel = element('span', { class: 'fine' });
	space.freshness = element('span', { class: 'freshness', role: 'status' });
	space.tabs = space.id === 'new' ? null : itemTabs(space.id, space.tab);
	space.task = element('section', { class: 'task-panel stack loose', 'aria-label': 'Task' });
	space.run = element('section', { class: 'stack loose', hidden: true, 'aria-label': 'Run' });
	space.history = element('section', { class: 'stack loose', hidden: true, 'aria-label': 'History' });
	space.decisionNotice = element('div', { class: 'callout warn stack', hidden: true, role: 'status' });
	space.decisionContent = element('div', { class: 'stack loose' });
	space.decisionContent.addEventListener('input', () => { space.decisionDirty = true; });
	space.decisionContent.addEventListener('change', () => { space.decisionDirty = true; });
	space.decisionContent.addEventListener('click', (event) => { if (event.target.closest('button[aria-pressed]')) space.decisionDirty = true; });
	space.decision = element('section', { class: 'surface stack loose', hidden: true, 'aria-label': 'Decision' }, space.decisionNotice, space.decisionContent);
	space.actions = element('div', { class: 'actions task-actions' });
	space.root = page('task-workspace', element('div', { class: 'workspace-header stack' }, element('div', { class: 'crumbs' }, element('a', { href: state.boardReturn?.hash || '#board' }, 'Back to ' + (state.boardReturn?.view || 'board')), element('span', {}, '/'), element('span', {}, space.id === 'new' ? 'New task' : 'Task')), element('div', { class: 'page-head' }, element('div', { class: 'stack tight' }, space.title, element('div', { class: 'row' }, space.status, space.dirtyLabel)), space.actions), element('div', { class: 'row spread workspace-navigation' }, space.tabs, space.freshness)), space.task, space.run, space.history, space.decision);
	space.stagePath = element('div', { class: 'stage-path', 'aria-label': 'Run stages' });
	space.stageDetails = element('section', { class: 'surface stack stage-details' });
	space.live = element('section', { class: 'surface stack live-activity', 'aria-label': 'Activity' });
	space.runSummary = element('div', { class: 'run-summary-line' });
	space.disputes = element('section', { class: 'stack loose dispute-list', 'aria-label': 'Disputes', hidden: true });
	space.disputeForm = element('section', { class: 'surface stack loose dispute-form', 'aria-label': 'Report a dispute', hidden: true });
	space.disputeReviews = new Map();
	const follow = element('input', { type: 'checkbox', checked: true, onchange: () => { space.follow = follow.checked; renderRun(space); } });
	space.followControl = follow;
	space.run.append(element('div', { class: 'row spread' }, space.runSummary, element('label', { class: 'switch' }, follow, 'Follow active step')), space.disputeForm, space.stagePath, space.disputes, element('div', { class: 'run-columns' }, space.stageDetails, space.live));
	space.taskContent = element('section', { class: 'surface stack loose task-document' });
	space.task.append(space.taskContent);
	renderTask(space); renderWorkspaceHead(space);
}
function setWorkspaceTab(space, tab) {
	space.tab = tab;
	for (const [name, node] of [['task', space.task], ['run', space.run], ['history', space.history], ['decision', space.decision]]) node.hidden = name !== tab;
	for (const anchor of space.tabs?.querySelectorAll('a') || []) anchor.setAttribute('aria-current', anchor.hash.startsWith('#' + (tab === 'task' ? 'item' : tab) + '/') ? 'page' : 'false');
	if (tab === 'decision' && (!space.decisionLoaded || (!space.decisionDirty && !space.dirty))) loadDecision(space);
	else if (tab === 'decision') checkDecisionFreshness(space);
}
function renderTask(space) {
	if (space.editor) return;
	const edit = can('coordinate', space.project) ? button('Edit task', () => editTask(space)) : null;
	if (edit) edit.disabled = !!space.record.frozen || !!space.record.archived;
	space.taskContent.replaceChildren(...[element('div', { class: 'row spread' }, element('h2', { class: 'section' }, 'Task'), edit), space.record.frozen ? element('p', { class: 'callout warn' }, space.record.frozenReason || 'Requirements are frozen while execution is active or uncertain.') : null, taskDescription(space.record), element('p', { class: 'fine mono' }, space.record.path || 'Save this draft before queueing it.'), list(space.record.dependents).length ? element('section', { class: 'stack tight' }, element('h3', { class: 'section' }, 'Tasks depending on this'), list(space.record.dependents).map((item) => element('a', { href: '#item/' + e(item.id) }, item.title))) : null, space.record.deleteReason ? element('p', { class: 'fine' }, 'Permanent deletion is unavailable: ' + space.record.deleteReason) : null].filter(Boolean));
}
function editTask(space) {
	if (space.editor) return;
	space.baseSHA = space.record.ticketSHA256; space.baseDraft = JSON.stringify(normalizedDraft(space.record.draft));
	const changed = (draft) => { space.dirty = JSON.stringify(draft) !== space.baseDraft; renderWorkspaceHead(space); };
	const dependencyCards = new Map(cards(space.board).map((card) => [card.workItemId, card]));
	for (const record of space.inventory) if (!record.archived) dependencyCards.set(record.workItemId, { workItemId: record.workItemId, title: record.draft.title });
	space.editor = workForm({ ...space.board, cards: [...dependencyCards.values()] }, space.record.draft, space.id, changed);
	space.editorProblem = element('div', { class: 'stack', role: 'status' });
	space.saveButton = button(space.id === 'new' ? 'Save draft' : 'Save changes', () => saveTask(space), 'primary');
	space.taskContent.replaceChildren(element('div', { class: 'row spread' }, element('h2', { class: 'section' }, space.id === 'new' ? 'Write a task' : 'Edit task'), element('span', { class: 'fine' }, 'Changes are saved only when you choose Save.')), space.editor.form, space.editorProblem, element('div', { class: 'actions' }, space.saveButton, button('Discard changes', () => { if (space.dirty && !confirm('Discard your unsaved changes?')) return; space.editor.dispose(); space.editor = null; space.dirty = false; if (space.id === 'new') { taskSpaces.delete(taskKey(space.project, 'new')); navigate('board'); } else { renderTask(space); renderWorkspaceHead(space); } }, 'quiet')));
	space.editor.form.querySelector('input')?.focus();
}
async function saveTask(space) {
	space.editorProblem.replaceChildren();
	space.refreshSequence++;
	const draft = space.editor.draft();
	if (!draft.title.trim() || !draft.criteria.length) { space.editorProblem.append(element('p', { class: 'alert bad' }, 'Add a title and at least one requirement.')); return; }
	try {
		const saved = space.id === 'new' ? await command('/projects/' + e(space.project) + '/work-items', { draft, queue: false }) : await command(taskURL(space), { workItemId: space.id, ifTicketSHA256: space.baseSHA, draft }, 'PUT');
		space.editor.dispose(); space.editor = null; space.dirty = false; space.record = saved;
		if (space.id === 'new') { taskSpaces.delete(taskKey(space.project, 'new')); state.flash = 'Draft saved. Queue it when it is ready to run.'; await navigate('item/' + e(saved.workItemId)); }
		else { renderTask(space); renderWorkspaceHead(space); notice('Task saved.'); await refreshWorkspace(space); }
	} catch (error) {
		space.editorProblem.append(element('p', { class: 'alert bad' }, error.message + ' Your draft is still here.'));
		if (error.status === 409 || /stale|conflict/i.test(error.code || '')) {
			try {
				const latest = await api(taskURL(space));
				space.editorProblem.append(element('div', { class: 'stale-comparison stack' }, element('h3', {}, 'Latest saved task'), taskDescription(latest), button('Keep my draft and use this revision', () => { space.baseSHA = latest.ticketSHA256; space.record = latest; space.editorProblem.replaceChildren(element('p', { class: 'callout info' }, 'Your writing is unchanged. Review it against the latest record, then save to apply it.')); renderWorkspaceHead(space); })));
			} catch (latestError) { space.editorProblem.append(element('p', { class: 'fine' }, 'Could not load the latest record. ' + latestError.message)); }
		}
	}
}
function renderWorkspaceHead(space) {
	space.title.textContent = space.id === 'new' ? 'New task' : space.record.draft.title;
	space.dirtyLabel.textContent = space.dirty ? 'Unsaved changes' : space.editor ? 'Editing' : '';
	const record = space.record, job = latestJob(space);
	space.status.textContent = space.id === 'new' ? 'Draft' : record.archived ? 'Archived' : record.queued ? 'Queued' : 'Unqueued';
	if (job?.status === 'cancel_requested') space.status.textContent = 'Stop requested';
	if (job?.status === 'cancellation_unknown') space.status.textContent = 'Stop outcome unknown';
	if (job?.status === 'cancelled' && (record.stopped || space.activity?.held)) space.status.textContent = 'Stopped';
	if (record.frozen && space.editor) space.dirtyLabel.textContent += ' · ' + (record.frozenReason || 'Requirements are frozen during execution');
	if (space.saveButton) space.saveButton.disabled = !!record.frozen || !!record.archived;
	const identity = JSON.stringify([record.actions, record.ticketSHA256, record.queued, record.archived, space.dirty, space.card?.decision, space.card?.state, job?.version]);
	patchRegion(space.actions, identity, () => {
		const actions = [];
		if (space.id === 'new') return actions;
		if (list(record.actions).includes('reconcile')) actions.push(button('Resume pending save', async () => { await command('/projects/' + e(space.project) + '/work-items/reconcile', {}); await refreshWorkspace(space); }));
		for (const action of list(record.actions)) {
			if (!['queue', 'unqueue', 'archive', 'restore', 'delete'].includes(action)) continue;
			const node = button({ queue: 'Queue task', unqueue: 'Remove from queue', archive: 'Archive', restore: 'Restore as draft', delete: 'Delete draft' }[action], () => disposition(space, action), action === 'queue' ? 'primary' : action === 'delete' ? 'danger' : 'quiet');
			node.disabled = space.dirty; actions.push(node);
		}
		if (job?.status === 'disputed') actions.unshift(link('Review dispute', '#run/' + e(space.id), 'primary'));
		if (job?.status !== 'disputed' && space.card?.decision && record.queued && !record.archived && !record.stopped && space.card.state !== 'running' && (space.card.decision === 'approve' || space.card.state === 'needs_you' || list(record.actions).includes('start')) && (space.card.decision === 'approve' ? canOpenResultDecision(latestJob(space)) : canPlan(space.project))) actions.unshift(link(space.card.decision === 'approve' ? 'Review result' : space.card.state === 'needs_you' ? 'Review next step' : 'Start task', '#decision/' + e(space.id), 'primary'));
		return actions;
	});
}
async function disposition(space, action) {
	const dependents = list(space.record.dependents).map((item) => item.title || item.id);
	const impact = dependents.length ? '\nReferenced by: ' + dependents.join(', ') + '. References and history remain.' : '';
	const explanation = { queue: 'Queue this task? It becomes a current commitment. ' + sentence(autopilotLine(space.board)) + ' The project ask policy still applies.', unqueue: 'Remove this task from the queue? It will not be picked up automatically.', archive: 'Archive this task? It leaves the queue and remains available in history.', restore: 'Restore this task as an unqueued draft?', delete: 'Permanently delete this never-run draft? This cannot be undone.' }[action];
	if (!confirm(explanation + impact)) return;
	space.refreshSequence++;
	const saved = await command(taskURL(space) + '/disposition', { workItemId: space.id, ifTicketSHA256: space.record.ticketSHA256, action });
	if (saved.deleted) { taskSpaces.delete(taskKey(space.project, space.id)); state.workspace = null; state.flash = 'Draft deleted.'; return navigate('board'); }
	space.record = saved; renderTask(space); renderWorkspaceHead(space); await refreshWorkspace(space);
}
async function refreshWorkspace(space) {
	if (space.id === 'new') return;
	if (space.refreshPromise) return space.refreshPromise;
	space.refreshPromise = (async () => {
		const sequence = ++space.refreshSequence;
		const results = await Promise.allSettled([api(taskURL(space)), api(taskURL(space) + '/run'), api(taskURL(space) + '/activity'), api('/projects/' + e(space.project) + '/board')]);
		if (sequence !== space.refreshSequence) return;
		const errors = [];
		for (const [index, result] of results.entries()) {
			if (result.status === 'rejected') { errors.push(result.reason.message); continue; }
			if (index === 0) {
				const changed = result.value.ticketSHA256 !== space.record.ticketSHA256 || result.value.frozen !== space.record.frozen || result.value.archived !== space.record.archived;
				space.record = result.value;
				if (changed && !space.editor) renderTask(space);
				if (space.editor && space.baseSHA !== space.record.ticketSHA256 && !space.editorProblem.children.length) space.editorProblem.append(element('p', { class: 'callout warn' }, 'The saved task changed while you were writing. Your draft is preserved. Saving will show the latest record for comparison.'));
			} else if (index === 1) space.graph = result.value;
			else if (index === 2) space.activity = result.value;
			else { space.board = result.value; space.card = cards(result.value).find((card) => card.workItemId === space.id); }
		}
		space.error = errors[0] || ''; if (!errors.length) space.lastSuccess = new Date().toISOString();
		renderWorkspaceHead(space); renderFreshness(space); renderRun(space); renderHistory(space); checkDecisionFreshness(space);
	})().finally(() => { space.refreshPromise = null; });
	return space.refreshPromise;
}
function renderFreshness(space) {
	if (space.id === 'new') { space.freshness.textContent = 'Draft save starts no agents'; return; }
	const old = space.lastSuccess && Date.now() - Date.parse(space.lastSuccess) > 15000;
	space.freshness.className = 'freshness' + (space.error || old ? ' disconnected' : '');
	space.freshness.textContent = space.error ? 'Disconnected · last received ' + (space.lastSuccess ? when(space.lastSuccess) : 'never') + ' · retrying' : space.lastSuccess ? (old ? 'Updates delayed' : 'Connected') + ' · updated ' + ago(space.lastSuccess) : 'Connecting…';
	space.freshness.title = space.error;
}
async function stopRun(space, jobId) {
	for (let attempt = 0; attempt < 2; attempt++) {
		const current = await api('/jobs/' + e(jobId));
		if (['cancel_requested', 'cancellation_unknown', 'cancelled'].includes(current.status)) { await refreshWorkspace(space); return; }
		if (!list(current.availableActions).includes('cancel')) { await refreshWorkspace(space); throw new Error('This run changed and can no longer be stopped. Its current state is ' + human(current.status) + '.'); }
		try {
			const stopped = await command('/jobs/' + e(jobId) + '/actions', { jobId, ifVersion: current.version, action: 'cancel' });
			space.refreshSequence++;
			if (space.activity) space.activity.jobs = list(space.activity.jobs).map((job) => job.id === jobId ? stopped : job);
			renderWorkspaceHead(space); renderRun(space); renderHistory(space);
			await refreshWorkspace(space);
			return;
		} catch (error) {
			if (attempt === 0 && error.status === 409) continue;
			await refreshWorkspace(space);
			throw error;
		}
	}
}
function runControls(space, job) {
	const controls = element('div', { class: 'actions' });
	if (list(job?.availableActions).includes('report_dispute')) controls.append(button('Report a dispute', () => openDispute(space, job)));
	for (const action of list(job?.availableActions)) {
		if (!['cancel', 'repair', 'reconcile'].includes(action)) continue;
		controls.append(button({ cancel: 'Stop run', repair: 'Request repair', reconcile: 'Check what happened' }[action], async () => {
			if (action === 'cancel' && !confirm('Request this run to stop? The factory will keep checking until workers are confirmed stopped. Automatic pickup remains off until you queue the task again.')) return;
			if (action === 'cancel') return stopRun(space, job.id);
			await command('/jobs/' + e(job.id) + '/actions', { jobId: job.id, ifVersion: job.version, action });
			await refreshWorkspace(space);
		}, action === 'cancel' ? 'danger' : ''));
	}
	if (job && list(job.availableActions).includes('resolve_execution')) controls.append(link('Resolve uncertain execution', '#job/' + e(job.id)));
	if (job?.status === 'accepted' && (!job.merge || job.merge.status === 'failed') && can('merge', space.project)) controls.append(button(job.merge ? 'Retry merge' : 'Merge', async () => { await command('/jobs/' + e(job.id) + '/merge', { jobId: job.id, ifVersion: job.version, candidateRevision: job.candidateRevision, targetBranch: '' }); await refreshWorkspace(space); }));
	if (job?.status !== 'disputed' && space.card?.decision && space.record.queued && !space.record.archived && !space.record.stopped && space.card.state !== 'running' && (space.card.decision === 'approve' || space.card.state === 'needs_you' || list(space.record.actions).includes('start')) && (space.card.decision === 'approve' ? canOpenResultDecision(latestJob(space)) : canPlan(space.project))) controls.append(link('Open decision', '#decision/' + e(space.id), 'primary'));
	return controls;
}
const DISPUTE_OWNER = { judge: 'Judge', test_author: 'Test author', you: 'You', implementer: 'Builder', none: 'No further handoff' };
const DISPUTE_STATUS = { judging: 'Judge is reviewing the dispute', needs_human: 'Waiting for your decision', authoring_tests: 'Test author is preparing a proposal', tests_proposed: 'Test proposal is ready for approval', routing: 'Controller is arranging the handoff', resolved: 'Handoff completed', stopped: 'Dispute stopped' };
const DISPUTE_KIND = { repair_implementation: 'Repair the implementation', revise_tests: 'Correct the tests', add_tests: 'Add missing tests', needs_human: 'Human decision required' };
const DISPUTE_ACTION = { retry_judge: 'Ask the judge again', repair_implementation: 'Send to builder', revise_tests: 'Ask test author to correct tests', add_tests: 'Ask test author to add tests', accept_tests: 'Approve tests and start replacement run', dismiss: 'Dismiss dispute and stop run', return_to_planning: 'Return task to planning' };
function referenceList(title, values) {
	return list(values).length ? element('section', { class: 'stack tight' }, element('h3', { class: 'section' }, title), element('ul', { class: 'reference-list' }, values.map((value) => element('li', {}, element('code', {}, String(value)))))) : null;
}
function disputeDetails(space, dispute) {
	const decision = dispute.decision, input = dispute.input || {};
	return [element('div', { class: 'row spread' }, element('h2', { class: 'section' }, 'Judge decision'), element('span', { class: 'state-chip' }, human(dispute.status))),
		element('div', { class: 'dispute-handoff' }, element('div', { class: 'stack tight' }, element('span', { class: 'eyebrow' }, dispute.status === 'resolved' ? 'Handed to' : 'Next up'), element('strong', { class: 'next-owner' }, DISPUTE_OWNER[dispute.nextOwner] || 'Not recorded')), element('p', {}, DISPUTE_STATUS[dispute.status] || human(dispute.status))),
		dispute.error ? element('p', { class: 'callout warn' }, dispute.error) : null,
		dispute.resolution ? element('section', { class: 'stack tight' }, element('h3', { class: 'section' }, 'Recorded human decision'), element('p', {}, (DISPUTE_ACTION[dispute.resolution] || human(dispute.resolution)) + ' · ' + actorName(space, dispute.resolvedBy)), markdown(dispute.rationale)) : null,
		decision ? element('section', { class: 'stack' }, element('h3', { class: 'section' }, DISPUTE_KIND[decision.kind] || human(decision.kind)), markdown(decision.rationale), decision.directive ? element('div', { class: 'callout stack' }, element('strong', {}, 'Proposed next work'), markdown(decision.directive)) : null, element('p', { class: 'fine' }, 'Preserves intended behavior: ' + (decision.preservesIntent === true ? 'yes' : decision.preservesIntent === false ? 'no' : 'not recorded') + ' · Weakens acceptance: ' + (decision.weakensAcceptance === true ? 'yes' : decision.weakensAcceptance === false ? 'no' : 'not recorded')), referenceList('Judge evidence', decision.evidenceRefs)) : element('p', { class: 'fine' }, 'No judge verdict has been recorded yet.'),
		element('details', { class: 'fold', 'data-key': 'dispute-report-' + dispute.id }, element('summary', {}, 'Dispute report'), element('div', { class: 'stack' }, markdown(input.summary), list(input.criterionRefs).length ? element('p', {}, 'Affected requirements: ' + input.criterionRefs.join(', ')) : null, referenceList('Affected tests', input.testPaths), markdown(input.observedFailure), referenceList('Reported evidence', input.evidenceRefs), input.proposedChange ? element('section', { class: 'stack tight' }, element('h3', { class: 'section' }, 'Proposed correction'), markdown(input.proposedChange)) : null)),
		dispute.run ? runNote(dispute.run) : null, dispute.log ? element('details', { class: 'fold', 'data-key': 'judge-output-' + dispute.id }, element('summary', {}, 'Judge output'), element('pre', { class: 'code output', 'data-scroll': 'judge-output-' + dispute.id }, dispute.log)) : null,
		element('div', { class: 'actions' }, link('Original run ' + short(dispute.jobId), '#job/' + e(dispute.jobId), 'quiet'), dispute.replacementJobId ? link('Replacement run ' + short(dispute.replacementJobId), '#job/' + e(dispute.replacementJobId), 'quiet') : null), element('p', { class: 'fine mono' }, 'Decision ' + short(dispute.id) + ' · version ' + dispute.version + ' · updated ' + when(dispute.updatedAt))];
}
function openDispute(space, job) {
	if (space.disputeDraft) { space.disputeForm.hidden = false; space.disputeForm.scrollIntoView({ block: 'start', behavior: 'smooth' }); return; }
	const draft = { jobId: job.id, version: job.version, dirty: false, stale: false };
	space.disputeDraft = draft;
	const form = element('form', { class: 'stack loose', oninput: () => { draft.dirty = true; }, onsubmit: (event) => { event.preventDefault(); run(() => submitDispute(space), draft.submit).finally(() => { if (space.disputeDraft === draft) draft.submit.disabled = draft.stale; }); } }, field('Dispute summary', 'summary', { required: true, maxLength: 4000 }), formGrid(field('Affected requirement numbers', 'criterionRefs', { placeholder: '1, 3', help: 'Use the numbers in this run’s approved requirements.' }), field('Affected test paths', 'testPaths', { type: 'textarea', rows: 2, help: 'One repository path per line.' })), field('Observed failure', 'observedFailure', { type: 'textarea', rows: 4, required: true }), field('Evidence references', 'evidenceRefs', { type: 'textarea', rows: 3, required: true, help: 'One check, attempt, artifact or file reference per line.' }), field('Proposed correction', 'proposedChange', { type: 'textarea', rows: 3, help: 'Optional. Describe a possible correction without changing the approved requirements.' }));
	draft.form = form; draft.notice = element('div', { class: 'callout warn stack', hidden: true, role: 'status' });
	draft.submit = element('button', { type: 'submit', class: 'button primary' }, 'Send dispute to judge');
	form.append(draft.notice, element('div', { class: 'actions' }, draft.submit, button('Close report', () => { if (draft.dirty && !confirm('Discard this unsent dispute?')) return; space.disputeDraft = null; space.disputeForm.replaceChildren(); space.disputeForm.hidden = true; }, 'quiet')));
	space.disputeForm.replaceChildren(element('h2', { class: 'section' }, 'Report a dispute'), element('p', {}, 'Challenge a requirement or protected test with evidence. The judge will decide whether the builder, test author or you should act next.'), element('p', { class: 'fine' }, 'Original run ' + short(job.id) + ' · run version ' + job.version + '. The run keeps its original requirements and test evidence.'), form);
	space.disputeForm.hidden = false; space.disputeForm.scrollIntoView({ block: 'start', behavior: 'smooth' }); form.querySelector('input').focus();
}
function checkDisputeDraft(space) {
	const draft = space.disputeDraft; if (!draft) return;
	const current = list(space.activity?.jobs).find((job) => job.id === draft.jobId);
	const allowed = list(current?.availableActions).includes('report_dispute');
	if (!draft.stale && current?.version === draft.version && allowed) return;
	draft.stale = true; draft.submit.disabled = true; draft.notice.hidden = false;
	draft.notice.replaceChildren(element('strong', {}, 'Run changed; review before sending'), element('p', {}, 'Your report is preserved. The current run is ' + human(current?.status) + (current ? ', version ' + current.version : '') + '.'), link('Inspect current run evidence', '#job/' + e(draft.jobId), 'quiet'), allowed ? button('Use current run version', () => { draft.version = current.version; draft.stale = false; draft.notice.hidden = true; draft.submit.disabled = false; }) : element('p', {}, 'This run no longer allows a new dispute.'));
}
async function submitDispute(space) {
	const draft = space.disputeDraft;
	if (!draft || draft.stale || !draft.form.reportValidity()) return;
	const values = data(draft.form), criterionRefs = split(values.criterionRefs).map(Number), testPaths = lines(values.testPaths);
	if (criterionRefs.some((value) => !Number.isInteger(value) || value < 1)) throw new Error('Requirement numbers must be positive whole numbers separated by commas.');
	if (!criterionRefs.length && !testPaths.length) throw new Error('Identify at least one affected requirement or test path.');
	try {
		await command('/jobs/' + e(draft.jobId) + '/disputes', { jobId: draft.jobId, ifVersion: draft.version, dispute: { summary: values.summary.trim(), criterionRefs, testPaths, observedFailure: values.observedFailure.trim(), evidenceRefs: lines(values.evidenceRefs), proposedChange: values.proposedChange.trim() } });
		space.disputeDraft = null; space.disputeForm.replaceChildren(); space.disputeForm.hidden = true; await refreshWorkspace(space);
	} catch (error) { if (error.status === 409) { draft.stale = true; await refreshWorkspace(space); checkDisputeDraft(space); } throw error; }
}
function disputeReview(space, dispute) {
	let review = space.disputeReviews.get(dispute.id);
	if (!review) {
		review = { dirty: false, snapshot: dispute, actionsKey: '' };
		review.notice = element('div', { class: 'callout warn stack', hidden: true, role: 'status' });
		review.rationale = field('Reason for your decision', 'rationale', { type: 'textarea', rows: 3, required: true, oninput: () => { review.dirty = true; } });
		review.controls = element('div', { class: 'actions' });
		review.node = element('section', { class: 'stack dispute-review', 'aria-label': 'Resolve dispute' }, element('h3', { class: 'section' }, 'Your decision'), review.notice, review.rationale, review.controls);
		space.disputeReviews.set(dispute.id, review);
	}
	const changed = dispute.version !== review.snapshot.version || JSON.stringify(dispute.availableActions) !== JSON.stringify(review.snapshot.availableActions);
	if ((changed && review.dirty) || review.stale) {
		review.stale = true; review.notice.hidden = false;
		review.notice.replaceChildren(element('strong', {}, 'Judge decision changed; review the latest state'), element('p', {}, 'Your reason is preserved. Review the updated decision above before taking action.'), button('Review latest judge decision', () => { review.snapshot = dispute; review.stale = false; review.notice.hidden = true; review.actionsKey = ''; updateDisputeControls(space, review); }));
	} else if (changed) { review.snapshot = dispute; review.stale = false; review.notice.hidden = true; }
	review.node.hidden = !list(dispute.availableActions).length && !review.dirty;
	updateDisputeControls(space, review);
	return review.node;
}
function updateDisputeControls(space, review) {
	const identity = JSON.stringify([review.snapshot.version, review.snapshot.availableActions, review.stale]);
	if (review.actionsKey === identity) return;
	review.actionsKey = identity;
	review.controls.replaceChildren(...list(review.snapshot.availableActions).filter((action) => DISPUTE_ACTION[action]).map((action) => {
		const control = button(DISPUTE_ACTION[action], async () => {
			const input = review.rationale.querySelector('textarea');
			if (review.stale || !input.reportValidity()) return;
			const snapshot = review.snapshot;
			try {
				await command('/disputes/' + e(snapshot.id) + '/resolve', { disputeId: snapshot.id, ifVersion: snapshot.version, action, rationale: input.value.trim() });
				review.dirty = false; input.value = ''; await refreshWorkspace(space);
			} catch (error) { if (error.status === 409) { review.stale = true; review.actionsKey = ''; await refreshWorkspace(space); updateDisputeControls(space, review); } throw error; }
		}, action === 'dismiss' ? 'quiet' : '');
		control.disabled = !!review.stale; return control;
	}));
}
function testProposal(space, dispute) {
	if (!dispute.expectationId) return null;
	const proposal = list(space.activity?.expectations).find((entry) => entry.id === dispute.expectationId);
	const content = element('div', { class: 'stack' });
	const show = (value) => content.replaceChildren(...[markdown(value.notes || ''), value.patch ? diffView(value.patch) : element('p', { class: 'fine' }, 'No test patch has been reported.'), referenceList('Test author concerns', value.concerns)].filter(Boolean));
	if (proposal) show(proposal);
	else { content.append(button('Load test proposal', async () => show(await api('/expectations/' + e(dispute.expectationId))))); }
	return element('details', { class: 'fold', 'data-key': 'dispute-tests-' + dispute.id + '-' + (proposal?.status || dispute.status), open: dispute.status === 'tests_proposed' }, element('summary', {}, 'Test author proposal ' + short(dispute.expectationId)), content);
}
function renderDisputes(space) {
	const disputes = list(space.activity?.disputes);
	space.disputes.hidden = !disputes.length;
	for (const dispute of disputes) {
		let entry = [...space.disputes.children].find((node) => node.dataset.dispute === dispute.id);
		if (!entry) { entry = element('article', { class: 'surface stack loose dispute-card', 'data-dispute': dispute.id }, element('div', { class: 'stack loose dispute-status' })); space.disputes.append(entry); }
		patchRegion(entry.firstElementChild, JSON.stringify([dispute, list(space.activity?.expectations).find((entry) => entry.id === dispute.expectationId)]), () => [disputeDetails(space, dispute), testProposal(space, dispute)]);
		const review = disputeReview(space, dispute); if (review.parentNode !== entry) entry.append(review);
	}
	for (const entry of [...space.disputes.children]) if (!disputes.some((dispute) => dispute.id === entry.dataset.dispute)) entry.remove();
	checkDisputeDraft(space);
}
function artifactView(space, job, artifact, context) {
	const key = context + '/' + job.id + '/' + artifact.id;
	if (space.artifactNodes.has(key)) return space.artifactNodes.get(key);
	const content = element('div', { class: 'stack tight' });
	let loaded = false;
	const details = element('details', { class: 'fold artifact', 'data-key': 'artifact-' + key, ontoggle: async () => {
		if (!details.open || loaded) return;
		loaded = true; content.replaceChildren(element('p', { class: 'fine' }, 'Loading output…'));
		try { const value = await api('/jobs/' + e(job.id) + '/artifacts/' + e(artifact.id)); content.replaceChildren(element('pre', { class: 'code output', 'data-scroll': key }, value.content || 'This artifact is empty.')); }
		catch (error) { loaded = false; content.replaceChildren(element('p', { class: 'alert bad' }, error.message)); }
	} }, element('summary', {}, artifact.name + ' · ' + artifact.bytes + ' bytes'), content);
	space.artifactNodes.set(key, details); return details;
}
function attemptView(space, job, attempt, context = 'run') {
	const timing = attempt.startedAt ? (attempt.finishedAt ? clock((Date.parse(attempt.finishedAt) - Date.parse(attempt.startedAt)) / 1000) : elapsed(attempt.startedAt)) : 'Not started';
	return element('article', { class: 'attempt stack', 'data-key': attempt.id }, element('div', { class: 'row spread' }, element('strong', {}, human(attempt.purpose === 'merge' ? 'Merge check' : attempt.stage) + (attempt.checkKey ? ' · ' + attempt.checkKey : '') + ' · attempt ' + attempt.ordinal), element('span', { class: 'tag' }, human(attempt.checkStatus || attempt.status))), element('div', { class: 'run-metadata' }, element('span', {}, timing), element('span', {}, attempt.lastHeartbeatAt ? 'Last contact ' + when(attempt.lastHeartbeatAt) : 'No heartbeat recorded')), attempt.progress ? markdown(attempt.progress) : null, attempt.summary ? markdown(attempt.summary) : null, element('p', { class: 'fine mono' }, 'Input ' + short(attempt.inputRevision) + ' · candidate ' + short(attempt.candidateRevision)), list(attempt.artifacts).map((artifact) => artifactView(space, job, artifact, context)), !list(attempt.artifacts).length ? element('p', { class: 'fine' }, 'Full output is available after the worker reports its result. Heartbeat notes appear above when reported.') : null);
}
function renderRun(space) {
	renderDisputes(space);
	const steps = list(space.graph?.steps), job = latestJob(space);
	const stopped = job && ['cancel_requested', 'cancellation_unknown', 'cancelled'].includes(job.status);
	const active = steps.find((step) => step.status === 'needs_you') || steps.find((step) => step.status === 'running') || [...steps].reverse().find((step) => step.status === 'done') || steps[0];
	if ((space.follow || !space.selected) && active) space.selected = active.key;
	const summary = stopped ? { cancel_requested: 'Stop requested. Waiting for workers to confirm termination.', cancellation_unknown: 'Stop outcome unknown. Reconcile the run before starting anything else.', cancelled: 'Run stopped. Queue this task explicitly to allow another run.' }[job.status] : job ? human(job.status) : 'No implementation run yet';
	patchRegion(space.runSummary, JSON.stringify([job, space.graph?.costUsd, space.card?.decision, space.card?.state]), () => [element('strong', {}, summary), element('p', { class: 'fine' }, space.graph?.costUsd ? money(space.graph.costUsd) + ' reported by completed agent calls. Live usage is unavailable.' : 'No reported cost yet. Live usage is unavailable.'), runControls(space, job)]);
	patchRegion(space.stagePath, JSON.stringify([steps.map((step) => [step.key, step.title, step.status]), space.selected, stopped]), () => steps.length ? steps.map((step) => element('button', { type: 'button', class: 'stage-step ' + step.status, 'aria-pressed': String(step.key === space.selected), onclick: () => { space.selected = step.key; space.follow = false; space.followControl.checked = false; renderRun(space); } }, element('span', { class: 'fine' }, RUN_KIND[step.kind] || step.kind), element('strong', {}, step.title), element('span', { class: 'stage-status' }, stopped && step.status === 'running' ? 'Last observed: running' : RUN_STATUS[step.status] || human(step.status)))) : empty('No run yet', 'Queue the task, then start it when its requirements are ready.'));
	const selected = steps.find((step) => step.key === space.selected);
	patchRegion(space.stageDetails, JSON.stringify([selected, job?.version]), () => selected ? [element('span', { class: 'eyebrow' }, (RUN_KIND[selected.kind] || '') + ' · ' + (RUN_STATUS[selected.status] || human(selected.status))), element('h2', { class: 'section' }, selected.title), markdown(selected.summary), list(selected.notes).map(markdown), element('dl', { class: 'run-facts' }, [['Model', selected.model], ['Reported turns', selected.turns], ['Reported duration', selected.seconds ? Math.round(selected.seconds) + ' s' : ''], ['Reported cost', selected.costUsd ? money(selected.costUsd) : ''], ['Decision by', selected.by]].filter(([, value]) => value).map(([label, value]) => [element('dt', {}, label), element('dd', {}, String(value))]))] : [empty('Select a stage', 'Stage details appear here.')]);
	patchRegion(space.live, JSON.stringify([job, space.activity?.refinements, space.activity?.expectations]), () => [element('h2', { class: 'section' }, 'Activity'), job ? [element('p', { class: 'fine' }, 'Job ' + short(job.id) + ' · updated ' + when(job.updatedAt)), [...list(job.attempts)].reverse().map((attempt) => attemptView(space, job, attempt)), checkResults(job), eventTimeline(space, job)] : planningActivity(space)]);
}
function checkResults(job) {
	if (!list(job.checks).length) return null;
	return element('section', { class: 'stack' }, element('h3', { class: 'section' }, 'Check results'), table(['Check', 'Result', 'Tested revision', 'Requirements'], list(job.checks).map((check) => [check.checkKey, human(check.status), element('code', { title: check.candidateRevision }, short(check.candidateRevision)), element('code', { title: check.requirementFingerprint }, short(check.requirementFingerprint))])));
}
function planningActivity(space) {
	const entries = [['Refinement', list(space.activity?.refinements)[0]], ['Test authoring', list(space.activity?.expectations)[0]]].filter(([, entry]) => entry);
	return entries.length ? entries.map(([name, entry]) => element('article', { class: 'attempt stack' }, element('strong', {}, name + ' · ' + human(entry.status)), element('p', { class: 'fine' }, 'Started ' + when(entry.createdAt)), entry.error ? element('p', { class: 'alert bad' }, entry.error) : null, entry.notes ? markdown(entry.notes) : null, entry.run ? runNote(entry.run) : element('p', { class: 'fine' }, 'No worker output or live usage has been reported for this stage.'))) : empty('No worker activity yet', 'Activity appears when planning or a run starts.');
}
function actorName(space, id) { return list(space.activity?.actors).find((actor) => actor.id === id)?.username || id || 'Controller'; }
function eventTimeline(space, job) {
	return element('ol', { class: 'timeline' }, [...list(job.events)].reverse().map((event) => element('li', {}, element('strong', {}, human(event.kind)), element('time', {}, '#' + event.sequence + ' · ' + when(event.createdAt) + ' · ' + actorName(space, event.actorId)), event.data?.delegated ? element('p', { class: 'fine' }, 'Delegated authority: ' + event.data.delegated) : null, Object.keys(event.data || {}).length ? element('details', { 'data-key': 'event-' + job.id + '-' + event.sequence }, element('summary', {}, 'Decision and event details'), element('pre', { class: 'code' }, JSON.stringify(event.data, null, 2))) : null)));
}
function renderHistory(space) {
	const { generatedAt, ...snapshot } = space.activity || {};
	patchRegion(space.history, JSON.stringify(snapshot), () => {
		const entries = [];
		for (const job of list(space.activity?.jobs)) entries.push({ at: job.createdAt, node: element('details', { class: 'surface history-entry', 'data-key': 'job-' + job.id, open: job.id === latestJob(space)?.id }, element('summary', {}, 'Implementation · ' + human(job.status) + ' · ' + when(job.createdAt)), element('div', { class: 'stack loose' }, element('dl', { class: 'history-identities' }, [['Job', job.id], ['Admitted requirements', job.admission.ticketSHA256], ['Started from', job.admission.baseRevision], ['Candidate', job.candidateRevision || 'None'], ['Merge revision', job.merge?.revision || 'None']].map(([label, value]) => [element('dt', {}, label), element('dd', {}, element('code', {}, value))])), element('details', { class: 'fold', 'data-key': 'requirements-' + job.id }, element('summary', {}, 'Original requirements and context'), list(job.admission.sources).map((source) => element('section', { class: 'stack' }, element('h3', { class: 'section mono' }, source.path), element('pre', { class: 'code' }, source.text)))), list(job.attempts).map((attempt) => attemptView(space, job, attempt, 'history')), checkResults(job), list(job.reviews).map((review) => element('section', { class: 'stack' }, element('strong', {}, 'Review · ' + human(review.decision)), element('p', { class: 'fine' }, 'By ' + actorName(space, review.reviewerId) + ' · ' + when(review.createdAt)), markdown(review.summary), element('p', { class: 'fine mono' }, 'Candidate ' + review.candidateRevision + ' · requirements ' + review.requirementFingerprint))), eventTimeline(space, job))) });
		for (const [name, values] of [['Refinement', space.activity?.refinements], ['Test proposal', space.activity?.expectations]]) for (const proposal of list(values)) entries.push({ at: proposal.createdAt, node: element('details', { class: 'surface history-entry', 'data-key': name + '-' + proposal.id }, element('summary', {}, name + ' · ' + human(proposal.status) + ' · ' + when(proposal.createdAt)), element('div', { class: 'stack' }, proposal.error ? element('p', { class: 'alert bad' }, proposal.error) : null, markdown(proposal.notes || proposal.proposal?.summary || ''), list(proposal.criteria).length ? element('section', {}, element('h3', { class: 'section' }, 'Original requirements'), element('ol', { class: 'requirements' }, list(proposal.criteria).map((text) => element('li', {}, markdown(text))))) : null, proposal.proposal?.criteria ? element('section', {}, element('h3', { class: 'section' }, 'Proposed requirements'), element('ol', { class: 'requirements' }, list(proposal.proposal.criteria).map((text) => element('li', {}, markdown(text))))) : null, proposal.patch ? diffView(proposal.patch) : null, element('details', { 'data-key': 'record-' + proposal.id }, element('summary', {}, 'Record, answers and execution details'), element('pre', { class: 'code' }, JSON.stringify(proposal, null, 2))))) });
		for (const dispute of list(space.activity?.disputes)) entries.push({ at: dispute.createdAt, node: element('details', { class: 'surface history-entry', 'data-key': 'dispute-' + dispute.id }, element('summary', {}, 'Judge · ' + (DISPUTE_KIND[dispute.decision?.kind] || human(dispute.status)) + ' · ' + when(dispute.createdAt)), element('div', { class: 'stack loose' }, disputeDetails(space, dispute), testProposal(space, dispute))) });
		entries.sort((a, b) => Date.parse(b.at) - Date.parse(a.at));
		return [element('h2', { class: 'section' }, 'History'), space.activity?.truncated ? element('p', { class: 'callout warn' }, 'This view shows the newest 100 entries of each kind. Older records remain in the service.') : null, entries.length ? entries.map((entry) => entry.node) : empty('No execution history yet', 'Jobs, attempts, decisions and proposals remain here after the task changes.')];
	});
}
function decisionIdentity(space) {
	const job = latestJob(space), card = space.card;
	return JSON.stringify([space.record.ticketSHA256, space.record.queued, space.record.archived, space.record.stopped, card?.decision, card?.refinementId, card?.expectationId, card?.jobId, job?.id, job?.version, job?.status, job?.candidateRevision, list(space.activity?.refinements)[0], list(space.activity?.expectations)[0]]);
}
function disableDecisionActions(space) {
	space.decisionDisabled ||= new Map();
	for (const control of space.decisionContent.querySelectorAll('button')) {
		if (!space.decisionDisabled.has(control)) space.decisionDisabled.set(control, control.disabled);
		control.disabled = true;
	}
}
function checkDecisionFreshness(space) {
	if (!space.decisionLoaded || space.decisionLoading) return;
	if (space.dirty) {
		space.decisionTaskBlocked = true; space.decisionNotice.hidden = false;
		space.decisionNotice.replaceChildren(element('p', {}, 'Save or discard your task edits before deciding. Your decision input is preserved.'), link('Return to task edits', '#item/' + e(space.id)));
		disableDecisionActions(space); return;
	}
	if (space.decisionTaskBlocked) {
		space.decisionTaskBlocked = false;
		if (!space.decisionStale) {
			space.decisionNotice.hidden = true;
			for (const [control, disabled] of space.decisionDisabled || []) control.disabled = disabled;
			space.decisionDisabled = null;
		} else space.decisionStale = false;
	}
	if (space.decisionStale || space.decisionIdentity === decisionIdentity(space)) return;
	space.decisionStale = true;
	space.decisionNotice.hidden = false;
	space.decisionNotice.replaceChildren(element('strong', {}, 'Decision changed; reload to continue'), element('p', {}, 'The task or run changed after you opened this decision. Your input is still here. Reload to review the latest state before deciding.'), button('Reload decision', () => loadDecision(space, true)));
	disableDecisionActions(space);
}
async function loadDecision(space, reload = false) {
	if (space.decisionLoading) return;
	if (reload && space.decisionDirty && !confirm('Reload this decision and discard its unsaved input?')) return;
	if (space.dirty) { space.decisionContent.replaceChildren(element('p', { class: 'callout warn' }, 'Save or discard your task edits before making a decision. The run uses the saved requirements.'), link('Return to your edits', '#item/' + e(space.id))); return; }
	space.decisionLoading = true;
	space.decisionContent.replaceChildren(element('p', { class: 'fine' }, 'Loading decision…'));
	space.decisionNotice.hidden = true;
	try {
		await refreshWorkspace(space);
		const identity = decisionIdentity(space), card = space.card;
		let content;
		if (latestJob(space)?.status === 'disputed') content = [element('p', {}, 'Review the judge’s decision, test proposal and next handoff in Run.'), link('Review dispute', '#run/' + e(space.id), 'primary')];
		else if (card?.decision === 'approve' && card.jobId) content = await approveContent(card.jobId, 'page');
		else if (card?.decision === 'plan' && space.record.queued && !space.record.stopped && !space.record.archived) content = canPlan(space.project) ? await planContent(card, 'page', space.board) : [element('p', {}, 'You can read this task, but making planning decisions requires coordinate access and a human account.')];
		else content = [element('p', {}, 'No decision is waiting for you.'), link('Open run', '#run/' + e(space.id))];
		space.decisionContent.replaceChildren(element('h2', { class: 'section' }, latestJob(space)?.status === 'disputed' ? 'Resolve the dispute' : card?.decision === 'approve' ? 'Review the result' : 'Agree what done means'), ...content);
		space.decisionIdentity = identity; space.decisionLoaded = true; space.decisionDirty = false; space.decisionStale = false; space.decisionDisabled = null; space.decisionTaskBlocked = false;
	} catch (error) { space.decisionContent.replaceChildren(element('p', { class: 'alert bad' }, error.message), button('Try again', () => loadDecision(space))); }
	finally { space.decisionLoading = false; checkDecisionFreshness(space); }
}

// The work graph

async function graphPage() {
	const board = await loadBoard();
	if (!cards(board).length) return page('', boardHead(board, 'graph'), empty('Nothing on the board yet', 'Work items appear here once the project has some.'));
	return page('', boardHead(board, 'graph'), workGraph(board), element('p', { class: 'fine' }, 'Each arrow points from work that must finish first to the work that waits for it. Dashed arrows are still open.'));
}
// workGraph draws each group of connected work as a row, left to right in dependency
// order, and the work that depends on nothing as a group of its own.
function workGraph(board) {
	const byId = new Map(cards(board).map((card) => [card.workItemId, card]));
	const deps = (card) => list(card.blockedBy).filter((id) => byId.has(id) && id !== card.workItemId);
	const depth = new Map();
	const depthOf = (card, visiting = new Set()) => {
		if (depth.has(card.workItemId)) return depth.get(card.workItemId);
		if (visiting.has(card.workItemId)) return 0;
		visiting.add(card.workItemId);
		const value = deps(card).reduce((most, id) => Math.max(most, depthOf(byId.get(id), visiting) + 1), 0);
		visiting.delete(card.workItemId);
		depth.set(card.workItemId, value);
		return value;
	};
	const group = new Map([...byId.keys()].map((id) => [id, id]));
	const root = (id) => group.get(id) === id ? id : root(group.get(id));
	for (const card of byId.values()) for (const id of deps(card)) group.set(root(id), root(card.workItemId));
	const groups = new Map();
	for (const card of byId.values()) {
		const key = root(card.workItemId);
		if (!groups.has(key)) groups.set(key, []);
		groups.get(key).push(card);
	}
	const rank = (members) => Math.min(...members.map((card) => ORDER[card.state]));
	const connected = [...groups.values()].filter((members) => members.length > 1).sort((a, b) => rank(a) - rank(b));
	const loose = [...groups.values()].filter((members) => members.length === 1).flat().sort((a, b) => ORDER[a.state] - ORDER[b.state] || a.title.localeCompare(b.title));
	const nodes = new Map();
	const node = (card) => {
		const kind = { needs_you: 'need', running: 'running', waiting: 'waiting', done: 'done' }[card.state];
		const status = card.state === 'running' ? element('span', { class: 'card-status' }, element('span', { class: 'dot', 'aria-hidden': 'true' }), card.status) : element('span', { class: 'eyebrow' + (card.state === 'needs_you' ? ' warn' : card.state === 'done' ? ' ok' : '') }, card.status);
		const item = element('button', { type: 'button', class: 'graph-node ' + kind, onclick: () => openCard(card) }, status, element('span', { class: 'graph-title' }, card.title));
		nodes.set(card.workItemId, item);
		return element('div', { class: 'graph-item', role: 'listitem' }, item);
	};
	const row = (members) => {
		const layers = [];
		for (const card of members) (layers[depthOf(card)] ||= []).push(card);
		const position = new Map();
		const pull = (card) => { const ids = deps(card).filter((id) => position.has(id)); return ids.length ? ids.reduce((sum, id) => sum + position.get(id), 0) / ids.length : Infinity; };
		return element('div', { class: 'graph-row', role: 'list', 'aria-label': 'Connected work' }, layers.filter(Boolean).map((layer, index) => {
			layer.sort((a, b) => (index ? pull(a) - pull(b) : 0) || ORDER[a.state] - ORDER[b.state] || a.title.localeCompare(b.title));
			layer.forEach((card, place) => position.set(card.workItemId, place));
			return element('div', { class: 'graph-layer', role: 'presentation' }, layer.map(node));
		}));
	};
	const edges = svg('svg', { class: 'graph-edges', 'aria-hidden': 'true' });
	const graph = element('div', { class: 'graph' }, edges, connected.map(row),
		loose.length ? element('section', { class: 'stack tight' }, element('h2', { class: 'section' }, connected.length ? 'Not connected to other work' : 'No item depends on another yet'), element('div', { class: 'graph-loose', role: 'list' }, loose.map(node))) : null);
	const draw = () => {
		if (!graph.isConnected) return;
		const origin = graph.getBoundingClientRect();
		const paths = [svg('defs', {}, ['met', 'open'].map((kind) => svg('marker', { id: 'arrow-' + kind, viewBox: '0 0 8 8', refX: '7', refY: '4', markerWidth: '8', markerHeight: '8', orient: 'auto' }, svg('path', { d: 'M0,0 L8,4 L0,8 z', class: 'arrow ' + kind }))))];
		for (const card of byId.values()) {
			for (const id of deps(card)) {
				const from = nodes.get(id).getBoundingClientRect(), to = nodes.get(card.workItemId).getBoundingClientRect();
				const x1 = from.right - origin.left, y1 = from.top + from.height / 2 - origin.top, x2 = to.left - origin.left - 2, y2 = to.top + to.height / 2 - origin.top, bend = Math.max(24, (x2 - x1) / 2);
				const kind = byId.get(id).state === 'done' ? 'met' : 'open';
				paths.push(svg('path', { class: 'edge ' + kind, d: `M${x1},${y1} C${x1 + bend},${y1} ${x2 - bend},${y2} ${x2},${y2}`, 'marker-end': `url(#arrow-${kind})` }));
			}
		}
		edges.replaceChildren(...paths);
	};
	new ResizeObserver(draw).observe(graph);
	return element('div', { class: 'board-scroll' }, graph);
}
function svg(tag, attributes = {}, ...children) {
	const result = document.createElementNS('http://www.w3.org/2000/svg', tag);
	for (const [name, value] of Object.entries(attributes)) result.setAttribute(name, value);
	result.append(...children.flat().filter(Boolean));
	return result;
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
async function decided(message, fragment = '') {
	if (state.workspace) state.workspace.decisionDirty = false;
	if (!fragment) fragment = state.workspace && state.workspace.id !== 'new' ? 'run/' + e(state.workspace.id) : 'board';
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
async function planContent(card, mode, currentBoard = null) {
	if (!canPlan()) return [element('p', {}, 'Making planning decisions requires coordinate access and a human account.')];
	const board = currentBoard || state.board || await loadBoard();
	const [refinement, expectation] = await Promise.all([card.refinementId ? api('/refinements/' + e(card.refinementId)) : null, card.expectationId ? api('/expectations/' + e(card.expectationId)) : null]);
	if (expectation?.status === 'proposed') return testsDecision(card, expectation, board, mode);
	if (expectation?.status === 'accepted') return startDecision(card, expectation);
	if (expectation && ['failed', 'rejected'].includes(expectation.status)) return [...retryDecision(card, expectation.error || 'The tests were rejected.', 'Write the tests again', () => plan(card, {})), expectation.notes ? fold('The author’s note', true, markdown(expectation.notes)) : null, runNote(expectation.run)].filter(Boolean);
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
		expectation.notes ? fold('The author’s note', false, markdown(expectation.notes)) : null,
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
	const independent = independentReview(job), mayApprove = canApproveJob(job), mayReview = canReviewJob(job);
	const findings = new Set();
	const chips = FINDINGS.map(([key, label]) => element('button', { type: 'button', class: 'toggle-chip', 'aria-pressed': 'false', onclick: (event) => { if (findings.has(key)) findings.delete(key); else findings.add(key); event.currentTarget.setAttribute('aria-pressed', String(findings.has(key))); } }, label));
	const mergeBox = element('input', { type: 'checkbox', checked: can('merge', job.projectId) });
	const approve = async () => {
		const merge = mergeBox.checked;
		await command('/jobs/' + e(job.id) + '/approve', { jobId: job.id, ifVersion: job.version, candidateRevision: job.candidateRevision, requirementFingerprint: job.admission.ticketSHA256, findings: [...findings], merge });
		if (mode === 'drawer') await decided(merge ? 'Approved. The checks run again on the combined version before main moves.' : 'Approved.');
		else { if (state.workspace) state.workspace.decisionDirty = false; await navigate('run/' + e(job.workItemId)); notice('Approved.'); }
	};
	const fixText = element('textarea', { rows: 3, placeholder: 'What should change?', 'aria-label': 'What should change' });
	const fixForm = element('div', { class: 'stack', hidden: true }, fixText, element('div', { class: 'actions' }, button('Send back for a fix', async () => {
		const summary = fixText.value.trim();
		if (!summary) throw new Error('Say what should change.');
		const reviewed = await command('/jobs/' + e(job.id) + '/reviews', { jobId: job.id, ifVersion: job.version, candidateRevision: job.candidateRevision, requirementFingerprint: job.admission.ticketSHA256, decision: 'changes_requested', summary, findings: [...findings] });
		if (can('coordinate', job.projectId) && list(reviewed.availableActions).includes('repair')) await command('/jobs/' + e(job.id) + '/actions', { jobId: job.id, ifVersion: reviewed.version, action: 'repair' });
		await decided('Sent back for a fix.', mode === 'drawer' ? 'board' : 'run/' + e(job.workItemId));
	}, 'primary')));
	let confirmReject = false;
	const reject = button('Reject', async () => {
		if (!confirmReject) { confirmReject = true; reject.textContent = 'Reject: click again to stop this build'; return; }
		await command('/jobs/' + e(job.id) + '/actions', { jobId: job.id, ifVersion: job.version, action: 'cancel' });
		await decided('Stop requested. Check the run for confirmed termination.', 'run/' + e(job.workItemId));
	}, 'danger');
	return [
		element('p', { class: 'lead' }, (files.length ? 'Changes ' + plural(files.length, 'file', 'files') + ' (+' + added + ' −' + removed + '). ' : '') + 'Every required check passed in a fresh copy' + (tests ? ', and the locked tests were not touched.' : '.')),
		open.length ? element('div', { class: 'callout warn' }, element('h3', { class: 'section' }, 'Check this yourself'), element('p', { class: 'fine' }, tests ? 'No locked test checks these, so they are left for you.' : 'This change has no locked tests, so check every requirement yourself.'), element('ul', { class: 'plain' }, open.map((row) => element('li', {}, rich(row.text))))) : element('div', { class: 'callout ok' }, element('p', {}, 'A locked test checks every requirement. Read the change if you want to; the checks already passed.')),
		change ? fold('The change', mode === 'page' || open.length > 0, element('div', { class: 'files' }, files.map((file) => element('span', {}, file.path + '  ', element('span', { class: 'plus' }, '+' + file.added), ' ', element('span', { class: 'minus' }, '−' + file.removed)))), diffView(change.patch), change.truncated ? element('p', { class: 'fine' }, 'The change is longer than shown here.') : null) : null,
		covered.length || list(job.admission.requiredChecks).length ? fold(element('span', {}, icon('check', 16), ' Already checked'), false, covered.map((row) => check(row.text, false, element('span', { class: 'chips' }, row.tests.map((test) => element('span', { class: 'chip' }, test))))), list(job.admission.requiredChecks).map((key) => check(key + ' passed in a fresh copy.'))) : null,
		fold(element('span', {}, 'Anything the checks missed? ', element('span', { class: 'muted' }, 'Optional')), false, element('div', { class: 'chips', role: 'group', 'aria-label': 'What you caught' }, chips), element('p', { class: 'fine' }, 'Your answer helps the factory decide when it can ask you less.')),
		job.status !== 'ready_for_review' || independent ? null : element('div', { class: 'callout bad' }, element('p', {}, 'You submitted or built this work, so someone else has to approve it.')),
		mayApprove && can('merge', job.projectId) ? element('label', { class: 'switch' }, mergeBox, element('span', {}, element('strong', {}, 'Merge after approving. '), element('span', { class: 'muted' }, 'The checks run again on the combined version, and main moves only if they pass.'))) : null,
		element('div', { class: 'actions' }, mayApprove ? element('button', { type: 'button', class: 'button primary', onclick: (event) => run(approve, event.currentTarget) }, job.status === 'ready_for_acceptance' ? (can('merge', job.projectId) ? 'Accept and merge' : 'Accept result') : (can('merge', job.projectId) ? 'Approve and merge' : 'Approve')) : null, mayReview ? button('Ask for a fix', () => { fixForm.hidden = !fixForm.hidden; if (!fixForm.hidden) fixText.focus(); }) : null, canRejectJob(job) ? reject : null),
		!mayApprove ? element('p', { class: 'fine' }, job.status === 'ready_for_acceptance' ? 'Accepting this reviewed result requires accept access and a human account.' : 'Approving requires review and accept access, and an independent human reviewer.') : null,
		mayReview ? fixForm : null,
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
	return element('div', { class: 'stack tight' }, element('div', { class: 'crumbs' }, element('a', { href: '#board' }, 'Board'), ' / ', title), element('div', { class: 'page-head' }, element('div', {}, element('h1', { class: 'title' }, title)), element('div', { class: 'row' }, card ? itemTabs(card.workItemId, 'task') : null, stagesElement(card))));
}
function statusSurface(card) {
	const eyebrow = card.state === 'needs_you' ? ['warn', 'Needs you'] : card.state === 'running' ? ['info', 'Automatic · nothing needs you'] : card.state === 'done' ? ['ok', 'Done'] : ['', 'Waiting'];
	return surface(element('span', { class: 'eyebrow ' + eyebrow[0] }, eyebrow[1]), element('div', { class: 'row' }, element('strong', { class: 'lead' }, card.status), elapsed(card.since)), element('p', {}, card.line), card.progress ? element('p', { class: 'progress-note' }, card.progress) : null);
}
function howItWorks() {
	return surface(element('h3', { class: 'section' }, 'After you agree'), element('ol', { class: 'requirements' }, element('li', {}, 'A separate agent writes locked tests and flags contradictions.'), element('li', {}, 'The build starts in its own copy, and checks run in a fresh one.'), element('li', {}, 'You approve the change, and it merges.')), element('p', { class: 'fine' }, element('a', { href: '#ask' }, 'Change when the factory asks you')));
}
async function itemPage(workItemId) { return taskWorkspace(workItemId, 'task'); }
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
	const events = element('ol', { class: 'timeline' }, [...list(job.events)].reverse().slice(0, 40).map((event) => element('li', {}, element('strong', {}, human(event.kind)), element('time', {}, '#' + event.sequence + ' · ' + when(event.createdAt) + ' · ' + actorName(space, event.actorId)))));
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

// The agents and what they are told

// AGENTS summarizes each agent's built-in rules; the full prompts live in the worker.
const AGENTS = [
	{ key: 'refiner', when: 'Before the build', title: 'Sharpens requirements', what: 'Runs when requirements look vague. It reads the repository but cannot change anything.', example: 'For example: name the Go package and the exact function signature in every requirement.', rules: [
		'Rewrite each requirement so it names a result someone can check: a command and its outcome, an output, a behaviour, or a file.',
		'Keep the work item’s intent and scope. Never add features, drop a requirement, or weaken one.',
		'Ask at most three questions, each with likely answers, and only when the answer would change the requirements or the build.',
		'Rely on the protected tests as they are.'] },
	{ key: 'testAuthor', when: 'Before the build', title: 'Writes locked tests', what: 'Writes the tests the builder is held to, in a copy of the repository the builder never sees.', example: 'For example: use table-driven tests, and name each test after its requirement.', rules: [
		'Change files only under the protected paths. Any other change rejects the whole suite.',
		'Follow the repository’s test conventions, with one focused test per requirement a test can check.',
		'Derive expected values from the requirements, and report an example that contradicts its own rule instead of encoding it.',
		'Report which test checks each requirement, and any problem found in the requirements.'] },
	{ key: 'builder', when: 'The build', title: 'Builds the change', what: 'Writes the change in its own copy of the repository, starting from the locked tests.', example: 'For example: keep functions short and give every exported name a doc comment.', rules: [
		'Satisfy every requirement. The requirements are frozen, and the project’s records are off limits.',
		'Never commit. Waymark commits what the builder leaves as the candidate.',
		'Never touch the protected paths. A candidate that does is rejected.',
		'Stop and explain when a requirement cannot be met without changing the requirements or the tests.'] },
];
async function agentsPage() {
	const board = await loadBoard();
	const instructions = board.project.policy.instructions || {};
	const editable = can('project_admin');
	const inputs = {};
	const agent = ({ key, when, title, what, example, rules }) => {
		const id = 'instructions-' + key;
		inputs[key] = element('textarea', { id, rows: 4, maxLength: 4000, readOnly: !editable, placeholder: editable ? example : 'None.', value: instructions[key] || '' });
		return surface(element('span', { class: 'eyebrow' }, when), element('h2', { class: 'section' }, title), element('p', { class: 'muted' }, what),
			fold('Its built-in rules', false, element('ul', { class: 'plain' }, rules.map((rule) => element('li', {}, rule)))),
			element('div', { class: 'stack tight' }, element('label', { class: 'field-label', htmlFor: id }, 'Your instructions'), inputs[key]));
	};
	const saved = instructions.setBy ? 'Saved ' + ago(instructions.setAt) + (instructions.setBy === state.me.userId ? ' by you.' : '.') : 'No instructions yet, so each agent follows only its built-in rules.';
	return page('mid',
		element('div', { class: 'stack tight' }, element('h1', { class: 'title' }, 'Agents'), element('p', { class: 'subtitle' }, 'What each agent is told on this project. Your instructions come after its built-in rules, which win where the two disagree. A run keeps the instructions it started with.')),
		AGENTS.map(agent),
		element('div', { class: 'row spread' }, element('span', { class: 'fine' }, editable ? saved : saved + ' Only a project administrator can change them.'), editable ? button('Save', async () => { await command('/projects/' + e(state.project) + '/instructions', { refiner: inputs.refiner.value, testAuthor: inputs.testAuthor.value, builder: inputs.builder.value }, 'PUT'); await route(); notice('Saved. Runs that start from now on get these instructions.'); }, 'primary') : null));
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
$('#signout').addEventListener('click', () => { if (![...taskSpaces.values()].some(workspaceHasInput) || confirm('Sign out and discard all unsaved task and decision input?')) signout(); });
$('#project-select').addEventListener('change', (event) => {
	if (!canLeaveWorkspace()) { event.target.value = state.project; return; }
	state.workspace = null; state.board = null; state.boardReturn = null;
	state.project = event.target.value;
	try { localStorage.setItem('waymark-project', state.project); } catch {}
	navigate('board');
});
document.addEventListener('keydown', (event) => { if (event.key === 'Escape' && state.drawer) closeDrawer(true); });
window.addEventListener('hashchange', () => run(() => route()));
window.addEventListener('beforeunload', (event) => { if ([...taskSpaces.values()].some(workspaceHasInput)) { event.preventDefault(); event.returnValue = ''; } });
if (state.token) connect().catch((error) => { signout(); $('#signin-error').textContent = error.message; $('#signin-error').hidden = false; });
setInterval(() => {
	for (const node of document.querySelectorAll('[data-since]')) node.textContent = clock((Date.now() - Date.parse(node.dataset.since)) / 1000);
	if (state.workspace) renderFreshness(state.workspace);
}, 1000);
setInterval(async () => {
	if (!state.me || !state.project || document.hidden || mutationsInFlight) return;
	if (state.workspace) { await refreshWorkspace(state.workspace); return; }
	if (state.drawer || /INPUT|TEXTAREA|SELECT/.test(document.activeElement?.tagName || '')) return;
	if (!['board', 'list', 'graph', 'job', ''].includes(state.view)) return;
	try {
		const board = await api('/projects/' + e(state.project) + '/board');
		if (boardKey(board) !== state.boardKey) await route(true);
	} catch (error) { notice(error.message, true); }
}, 3000);

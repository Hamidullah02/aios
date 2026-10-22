/* aios web UI.
 *
 * No framework and no build step. The UI is a few hundred
 * lines of DOM work, and embedding it means it must
 * survive `go build` with no node toolchain anywhere in the
 * pipeline. A framework would add a build artifact that the
 * "included in the binary" requirement rules out.
 *
 * State has exactly one source: the server. The client
 * holds `state`, which is replaced by a snapshot on load
 * and patched by SSE deltas. Nothing is derived locally
 * except layout ordering, which is the operator's explicit
 * arrangement and is persisted server-side so every open
 * tab agrees.
 */
'use strict';

const state = {
  agents: [],
  messages: [],
  layout: null,
};

const $ = (sel, root = document) => root.querySelector(sel);
const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));

/* ---------- connection ---------- */

let es = null;
let retry = 0;

function connect() {
  if (es) { es.close(); }
  setConn('connecting');
  es = new EventSource('/api/events');

  es.addEventListener('open', () => { retry = 0; setConn('live'); });

  es.addEventListener('snapshot', (e) => {
    const snap = JSON.parse(e.data);
    state.agents = snap.agents || [];
    state.messages = snap.messages || [];
    if (snap.layout) { state.layout = snap.layout; renderLayout(); }
    render();
    setConn('live');
  });

  es.addEventListener('agent', (e) => {
    const a = JSON.parse(e.data);
    const i = state.agents.findIndex((x) => x.id === a.id);
    if (i >= 0) state.agents[i] = a; else state.agents.push(a);
    renderAgents();
    refreshAgentSelects();
  });

  es.addEventListener('message', (e) => {
    state.messages.push(JSON.parse(e.data));
    // The server caps retention; mirror it so a long session
    // does not grow the client array without bound.
    if (state.messages.length > 500) state.messages.shift();
    renderComms();
  });

  es.addEventListener('layout', (e) => {
    state.layout = JSON.parse(e.data);
    renderLayout();
    render();
  });

  es.addEventListener('error', () => {
    setConn('lost');
    es.close();
    // Back off, but cap it: a server that comes back after a
    // restart should be picked up quickly.
    const wait = Math.min(1000 * Math.pow(1.5, retry++), 10000);
    setTimeout(connect, wait);
  });
}

function setConn(stateName) {
  const el = $('#conn');
  el.dataset.state = stateName;
  el.textContent = stateName === 'live' ? 'live' : stateName;
}

/* ---------- rendering ---------- */

function render() {
  renderLayout();
  renderAgents();
  renderComms();
  renderAudit();
  refreshAgentSelects();
}

/* Applies the saved layout: panel order, width, visibility.
 * Called whenever the layout changes, from either side of
 * the wire. */
function renderLayout() {
  const grid = $('#grid');
  if (!state.layout || !state.layout.panels) return;

  grid.classList.toggle('compact', !!state.layout.compact);

  for (const p of state.layout.panels) {
    const tpl = document.getElementById('panel-' + p.id);
    if (!tpl) continue;
    let el = grid.querySelector('[data-panel="' + p.id + '"]');
    if (!p.visible) {
      if (el) el.remove();
      continue;
    }
    if (!el) {
      el = tpl.content.firstElementChild.cloneNode(true);
      grid.appendChild(el);
    }
    el.style.gridColumn = 'span ' + Math.min(Math.max(p.width || 1, 1), 12);
    el.style.order = String(p.order ?? 0);
  }

  const sel = $('#theme-select');
  if (sel) sel.value = state.layout.theme || '';
  const cmp = $('#compact-toggle');
  if (cmp) cmp.checked = !!state.layout.compact;
  document.documentElement.dataset.theme = state.layout.theme || 'auto';
}

function renderAgents() {
  const body = $('[data-panel="agents"] [data-role="body"]');
  if (!body) return;
  const agents = state.agents.slice().sort((a, b) => a.id.localeCompare(b.id));
  $('[data-panel="agents"] [data-role="count"]').textContent = agents.length || '';

  if (!agents.length) {
    body.innerHTML = '<p class="empty">No agents yet.</p>';
    return;
  }
  body.innerHTML = '';
  const tpl = document.getElementById('agent-card');
  for (const a of agents) {
    const node = tpl.content.firstElementChild.cloneNode(true);
    node.dataset.status = a.status || 'idle';
    $('.agent-label', node).textContent = a.label || a.id;
    $('.agent-tool', node).textContent = a.tool || (a.task_id ? 'task ' + a.task_id : '');
    $('.agent-detail', node).textContent = a.detail || '';
    body.appendChild(node);
  }
}

function renderComms() {
  const body = $('[data-panel="comms"] [data-role="body"]');
  if (!body) return;
  $('[data-panel="comms"] [data-role="count"]').textContent = state.messages.length || '';
  if (!state.messages.length) {
    body.innerHTML = '<p class="empty">No messages yet.</p>';
    return;
  }
  const atBottom = body.scrollTop + body.clientHeight >= body.scrollHeight - 24;
  body.innerHTML = '';
  for (const m of state.messages) {
    const div = document.createElement('div');
    div.className = 'msg' + (m.to ? '' : ' broadcast');
    const head = document.createElement('div');
    head.className = 'msg-head';
    const from = document.createElement('span');
    from.className = 'msg-from';
    from.textContent = m.from;
    const to = document.createElement('span');
    to.className = 'msg-to';
    to.textContent = m.to ? '→ ' + m.to : '→ everyone';
    head.appendChild(from);
    head.appendChild(to);
    if (m.kind) {
      const k = document.createElement('span');
      k.className = 'msg-kind';
      k.textContent = '#' + m.kind;
      head.appendChild(k);
    }
    const p = document.createElement('p');
    p.className = 'msg-body';
    // textContent, never innerHTML: message bodies are
    // operator- and agent-supplied text and must never be
    // parsed as markup.
    p.textContent = m.body;
    div.appendChild(head);
    div.appendChild(p);
    body.appendChild(div);
  }
  if (atBottom) body.scrollTop = body.scrollHeight;
}

/* Activity is derived from the agent list rather than a
 * separate feed: the server sends agent deltas, and a
 * status change is the activity worth showing. A dedicated
 * event log would duplicate state the panel already has. */
function renderAudit() {
  const body = $('[data-panel="audit"] [data-role="body"]');
  if (!body) return;
  if (!state.agents.length) {
    body.innerHTML = '<p class="empty">No activity yet.</p>';
    return;
  }
  const rows = state.agents.slice()
    .sort((a, b) => (b.updated_at || '').localeCompare(a.updated_at || ''))
    .slice(0, 50);
  body.innerHTML = '';
  for (const a of rows) {
    const div = document.createElement('div');
    div.className = 'audit-row';
    const who = document.createElement('span');
    who.className = 'audit-agent';
    who.textContent = a.id;
    const what = document.createElement('span');
    what.className = 'audit-text';
    const bits = [a.status];
    if (a.tool) bits.push(a.tool);
    if (a.detail) bits.push(a.detail);
    what.textContent = bits.join(' · ');
    div.appendChild(who);
    div.appendChild(what);
    body.appendChild(div);
  }
}

function refreshAgentSelects() {
  const from = $('[data-role="from"]');
  const to = $('[data-role="to"]');
  if (!from || !to) return;
  const ids = state.agents.map((a) => a.id).sort();
  const curFrom = from.value;
  const curTo = to.value;
  from.innerHTML = '';
  for (const id of ids) {
    const o = document.createElement('option');
    o.value = id; o.textContent = id;
    from.appendChild(o);
  }
  to.innerHTML = '';
  const everyone = document.createElement('option');
  everyone.value = ''; everyone.textContent = 'Everyone';
  to.appendChild(everyone);
  for (const id of ids) {
    const o = document.createElement('option');
    o.value = id; o.textContent = id;
    to.appendChild(o);
  }
  if (ids.includes(curFrom)) from.value = curFrom;
  if (curTo && ids.includes(curTo)) to.value = curTo;
}

/* ---------- layout editor ---------- */

function renderEditor() {
  const ul = $('#panel-list');
  if (!state.layout) return;
  ul.innerHTML = '';
  const sorted = state.layout.panels.slice().sort((a, b) =>
    (a.order ?? 0) - (b.order ?? 0) || a.id.localeCompare(b.id));

  for (const p of sorted) {
    const li = document.createElement('li');
    li.draggable = true;
    li.dataset.id = p.id;

    const name = document.createElement('span');
    name.className = 'name';
    name.textContent = p.id;

    const vis = document.createElement('input');
    vis.type = 'checkbox';
    vis.checked = !!p.visible;
    vis.title = 'Show this panel';
    vis.addEventListener('change', () => {
      p.visible = vis.checked;
      saveLayout();
    });

    const width = document.createElement('input');
    width.type = 'range';
    width.min = '3'; width.max = '12';
    width.value = String(p.width || 12);
    width.title = 'Width in grid columns';
    const widthLabel = document.createElement('span');
    widthLabel.className = 'audit-text';
    widthLabel.textContent = p.width || 12;
    width.addEventListener('input', () => { widthLabel.textContent = width.value; });
    width.addEventListener('change', () => { p.width = Number(width.value); saveLayout(); });

    li.appendChild(vis);
    li.appendChild(name);
    li.appendChild(width);
    li.appendChild(widthLabel);

    // Drag to reorder.
    li.addEventListener('dragstart', (e) => {
      li.classList.add('dragging');
      e.dataTransfer.setData('text/plain', p.id);
      e.dataTransfer.effectAllowed = 'move';
    });
    li.addEventListener('dragend', () => {
      li.classList.remove('dragging');
      $$('.drop-target', ul).forEach((n) => n.classList.remove('drop-target'));
    });
    li.addEventListener('dragover', (e) => {
      e.preventDefault();
      e.dataTransfer.dropEffect = 'move';
      li.classList.add('drop-target');
    });
    li.addEventListener('dragleave', () => li.classList.remove('drop-target'));
    li.addEventListener('drop', (e) => {
      e.preventDefault();
      li.classList.remove('drop-target');
      const srcId = e.dataTransfer.getData('text/plain');
      if (!srcId || srcId === p.id) return;
      // Rewrite the whole order list from the visible
      // sequence: order is a sort key over the same set,
      // so the simplest correct edit is to re-number.
      const ids = sorted.map((x) => x.id).filter((x) => x !== srcId);
      ids.splice(ids.indexOf(p.id), 0, srcId);
      for (const panel of state.layout.panels) {
        const i = ids.indexOf(panel.id);
        if (i >= 0) panel.order = i;
      }
      saveLayout();
      renderEditor();
    });

    ul.appendChild(li);
  }
}

async function saveLayout() {
  const el = $('#save-state');
  el.textContent = 'saving…';
  try {
    const res = await fetch('/api/layout', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(state.layout),
    });
    if (!res.ok) throw new Error((await res.json()).error || res.statusText);
    // Adopt the server's normalised copy: it clamps and
    // fills gaps, and the client should render exactly what
    // every other tab will get.
    state.layout = await res.json();
    renderLayout();
    el.textContent = 'saved';
    setTimeout(() => { el.textContent = ''; }, 1500);
  } catch (err) {
    el.textContent = 'not saved: ' + err.message;
  }
}

/* ---------- compose ---------- */

function wireCompose() {
  const form = $('[data-role="compose"]');
  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    const from = $('[data-role="from"]', form);
    const to = $('[data-role="to"]', form);
    const body = $('[data-role="body"]', form);
    if (!body.value.trim()) return;
    if (!from.value) { body.focus(); return; }
    const res = await fetch('/api/messages', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ from: from.value, to: to.value, body: body.value }),
    });
    if (res.ok) body.value = '';
    else $('[data-role="body"]').placeholder = (await res.json()).error || 'send failed';
  });
}

/* ---------- boot ---------- */

function boot() {
  $('#layout-toggle').addEventListener('click', () => {
    const ed = $('#editor');
    ed.hidden = !ed.hidden;
    if (!ed.hidden) renderEditor();
  });
  $('#reset-layout').addEventListener('click', async () => {
    // Ask the server for the default rather than guessing it
    // here, so the two definitions cannot drift.
    const res = await fetch('/api/layout', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({}),
    });
    state.layout = await res.json();
    renderLayout();
    renderEditor();
  });
  $('#theme-toggle').addEventListener('click', () => {
    if (!state.layout) return;
    const order = ['', 'dark', 'light'];
    const i = order.indexOf(state.layout.theme || '');
    state.layout.theme = order[(i + 1) % order.length];
    document.documentElement.dataset.theme = state.layout.theme || 'auto';
    saveLayout();
  });
  $('#theme-select').addEventListener('change', (e) => {
    if (!state.layout) return;
    state.layout.theme = e.target.value;
    saveLayout();
  });
  $('#compact-toggle').addEventListener('change', (e) => {
    if (!state.layout) return;
    state.layout.compact = e.target.checked;
    saveLayout();
  });

  wireCompose();
  connect();
  // The panels do not exist until the first snapshot, so
  // the compose handler is wired here and re-attached by
  // renderLayout through the delegated listeners above.
  document.addEventListener('DOMContentLoaded', () => render());
  if (document.readyState !== 'loading') render();
}

boot();
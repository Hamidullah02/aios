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

// OPERATOR_SENDER is the reserved sender id for the operator's
// own messages. A message from an agent is a note between
// agents; a message from the operator is a request for work, and
// the server turns it into a plan.
const OPERATOR_SENDER = 'operator';

const state = {
  agents: [],
  messages: [],
  layout: null,
  // tasks are the things an operator asked for from the
  // browser, and what became of them. This is the thing the
  // UI is actually for: the agent panels below are the
  // machinery, these are the jobs.
  tasks: [],
  // canSubmit says whether THIS server can start work. When it
  // is false the composer is replaced with an explanation,
  // because a Start button that cannot work is worse than no
  // Start button at all.
  canSubmit: false,
  // askBusy is true while a task request is in flight. The
  // Start button stays disabled for the duration, so a second
  // click cannot queue a second run. It is separate from
  // canSubmit because the two disable the button for different
  // reasons: one is the server's capability, the other is the
  // operator's own request already in flight.
  askBusy: false,
  // approvals are the requests waiting on a human decision.
  // canResolve says whether THIS server can apply one; when it
  // is false (`aios web` as a separate process) the buttons are
  // hidden and the panel points at the terminal or the daemon,
  // because an approve button that cannot work is worse than
  // no button at all.
  approvals: [],
  canResolveApprovals: false,
};

const $ = (sel, root = document) => root.querySelector(sel);
const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));

/* ---------- session ---------- */

/* The UI is behind a session token. The browser holds it in an
 * HttpOnly cookie, so the page script cannot read it -- which is
 * the point: a cross-site script must not be able to lift the
 * session out of the cookie jar.
 *
 * That leaves one gap: the EventSource that carries the live feed
 * cannot set a header, and it relies on the cookie. The cookie is
 * SameSite=Lax, so it rides a top-level navigation (the operator
 * clicking the link) but not a cross-site fetch. The Origin check
 * on the server refuses a state-changing request that did not come
 * from this origin, so the combination is what protects the API.
 *
 * If the cookie is missing -- the operator opened the page
 * directly rather than through the session link -- the API calls
 * come back 401. The UI says so instead of showing an empty
 * dashboard that looks like an idle system. */

async function apiFetch(path, options) {
  const opts = options || {};
  opts.credentials = 'same-origin';
  opts.headers = opts.headers || {};
  return fetch(path, opts);
}

/* stripSessionToken removes the token from the address bar.
 *
 * The server already redirects to the clean URL after the
 * handshake, so this is a belt-and-braces measure for a browser
 * that restores the original URL from history. */

function stripSessionToken() {
  if (!window.location.search) return;
  const url = new URL(window.location.href);
  if (!url.searchParams.has('token')) return;
  url.searchParams.delete('token');
  window.history.replaceState({}, '', url.pathname + url.search);
}

/* ---------- connection ---------- */

let es = null;
let retry = 0;
/* sessionChecked guards the one-off 401 probe below, so a
 * reconnect loop does not hammer the API for a session that is
 * not coming back. */
let sessionChecked = false;

function connect() {
  if (es) { es.close(); }
  setConn('connecting');
  es = new EventSource('/api/events');

  es.addEventListener('open', () => { retry = 0; setConn('live'); });

  es.addEventListener('snapshot', (e) => {
    const snap = JSON.parse(e.data);
    state.agents = snap.agents || [];
    state.messages = snap.messages || [];
    state.tasks = snap.tasks || [];
    state.canSubmit = !!snap.can_submit_tasks;
    state.approvals = snap.approvals || [];
    state.canResolveApprovals = !!snap.can_resolve_approvals;
    if (snap.layout) { state.layout = snap.layout; renderLayout(); }
    render();
    setConn('live');
  });

  // The approval list arrives as a whole list, not a delta: a
  // grant and a deny can land in the same tick, and merging
  // per-id updates on the client is where a stale "approve"
  // button survives its own decision.
  es.addEventListener('approvals', (e) => {
    state.approvals = JSON.parse(e.data) || [];
    renderApprovals();
  });

  // A task update arrives as one task, not the whole list: a task
  // changes status many times a second while it runs, and
  // replacing the whole list on each of those would make every
  // open tab re-render everything for one card.
  es.addEventListener('task', (e) => {
    const t = JSON.parse(e.data);
    const i = state.tasks.findIndex((x) => x.id === t.id);
    if (i >= 0) state.tasks[i] = t; else state.tasks.push(t);
    renderTasks();
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
    // A refused stream is the shape a missing session takes:
    // the server answers 401 and closes rather than holding the
    // connection open. Say so plainly instead of reconnecting
    // forever against a server that will keep refusing.
    if (es.readyState === 2 /* CLOSED */ && !sessionChecked) {
      sessionChecked = true;
      apiFetch('/api/state')
        .then((r) => { if (r.status === 401) showSessionExpired(); })
        .catch(() => {});
    }
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

/* showSessionExpired tells the operator the session is gone.
 *
 * The UI sits behind a session token. If the cookie is missing or
 * stale -- the operator opened the page without going through the
 * session link, or the server restarted and minted a new token --
 * every API call comes back 401. Without this, the dashboard
 * would sit there empty and look exactly like an idle system,
 * which is the worst possible reading of "nothing is happening".
 *
 * The remedy is in the message: the link the server printed. */
function showSessionExpired() {
  setConn('lost');
  const el = $('#conn');
  el.textContent = 'session expired';
  el.title = 'Open the link the aios command printed to start a new session.';
}

/* ---------- rendering ---------- */

function render() {
  renderLayout();
  renderTasks();
  renderAgents();
  renderApprovals();
  renderComms();
  renderAudit();
  refreshAgentSelects();
  // The composer's enabled state comes from the server's
  // capability flag, so it is applied wherever the snapshot is
  // applied rather than once at load. Without this the button
  // was left enabled on a server that cannot start work.
  setAskEnabled();
}

/* ---------- tasks ---------- */

/* renderTasks draws the jobs the operator asked for.
 *
 * The card leads with the operator's own words, because the
 * request is the thing they will look for when they come back to
 * the page. Everything else -- the steps, the status, the
 * outcome -- hangs off that. */
function renderTasks() {
  const body = $('[data-panel="tasks"] [data-role="body"]');
  if (!body) return;

  const count = $('[data-panel="tasks"] [data-role="count"]');
  if (count) count.textContent = state.tasks.length || '';

  if (!state.tasks.length) {
    body.innerHTML = '<p class="empty">Nothing asked for yet.</p>';
    return;
  }

  body.innerHTML = '';
  const tpl = document.getElementById('task-card');
  // Newest first: the task the operator just submitted is the
  // one they are looking at.
  const tasks = state.tasks.slice().reverse();
  for (const t of tasks) {
    const node = tpl.content.firstElementChild.cloneNode(true);
    node.dataset.status = t.status || 'planning';
    $('.task-goal', node).textContent = t.goal || '';

    const steps = $('.task-steps', node);
    if (t.plan_summary && t.plan_summary.length) {
      for (const line of t.plan_summary) {
        const li = document.createElement('li');
        // textContent, never innerHTML: the plan came from a
        // model and must never be parsed as markup.
        li.textContent = line;
        steps.appendChild(li);
      }
    } else {
      steps.hidden = true;
    }

    const detail = $('.task-detail', node);
    detail.textContent = t.detail || '';
    detail.hidden = !t.detail;

    const result = $('.task-result', node);
    result.textContent = t.result || '';
    result.hidden = !t.result;

    body.appendChild(node);
  }
}

/* ---------- the composer ---------- */

/* wireAsk wires the one field that matters.
 *
 * The operator types what they want done. There is no plan to
 * fill in, no tool to choose, no model to pick, and no flag to
 * understand. Whatever they type is the request; the server
 * works out the rest.
 *
 * The button is disabled while a request is in flight and the
 * note says why. Silently ignoring a second click would leave
 * the operator wondering whether their second request was
 * accepted. */
function wireAsk() {
  const form = $('#ask-form');
  const input = $('#ask-input');
  const note = $('#ask-note');
  const button = $('#ask-submit');

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    const goal = input.value.trim();
    if (!goal) { input.focus(); return; }

    state.askBusy = true;
    button.disabled = true;
    note.dataset.kind = '';
    note.textContent = 'starting…';
    try {
      const res = await apiFetch('/api/tasks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ goal: goal }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        // The server says what went wrong in words. Show that
        // rather than a generic failure: the difference between
        // "no AI model key is configured" and "something went
        // wrong" is the difference between a fixable problem
        // and a dead end.
        note.dataset.kind = 'error';
        note.textContent = data.error || 'could not start that';
        state.askBusy = false;
        setAskEnabled();
        return;
      }
      input.value = '';
      note.textContent = 'started.';
      // The task card arrives over the event stream; showing
      // it immediately here would mean two sources of truth for
      // the same card.
      setTimeout(() => { if (note.textContent === 'started.') note.textContent = ''; }, 2500);
      // The request landed, so the composer is free again. The
      // server may refuse the next one while this run is in
      // flight, and it says so in words when it does.
      state.askBusy = false;
      setAskEnabled();
    } catch (err) {
      note.dataset.kind = 'error';
      note.textContent = 'could not start that: ' + err.message;
      state.askBusy = false;
      setAskEnabled();
    }
  });
}

/* setAskEnabled turns the composer on or off.
 *
 * A server that cannot start tasks must not show a Start button.
 * It shows the field disabled and the reason, so the operator
 * knows the product is working and this particular deployment
 * is the one that cannot take work. */
function setAskEnabled() {
  const input = $('#ask-input');
  const button = $('#ask-submit');
  const note = $('#ask-note');
  if (!input || !button) return;
  if (state.askBusy) {
    // A request is in flight. The button stays off until it
    // lands, and the note keeps saying what is happening.
    input.disabled = true;
    button.disabled = true;
    return;
  }
  if (state.canSubmit) {
    input.disabled = false;
    button.disabled = false;
    if (note.dataset.kind === 'unavailable') {
      note.textContent = '';
      note.dataset.kind = '';
    }
    return;
  }
  input.disabled = true;
  button.disabled = true;
  note.dataset.kind = 'unavailable';
  note.textContent = 'This server cannot start new tasks. It can only show work that is already running.';
}

/* ---------- approvals ---------- */

/* renderApprovals draws the queue of decisions waiting on a
 * human.
 *
 * The card leads with the target, not the id. An operator is
 * being asked to authorize a specific command or path, and
 * "allow apr-3f9c…" asks them to consent to a string. The id
 * is still shown (it is the handle for `aios approve`), but it
 * is the least important thing on the card. */
function renderApprovals() {
  const body = $('[data-panel="approvals"] [data-role="body"]');
  if (!body) return;

  const count = $('[data-panel="approvals"] [data-role="count"]');
  if (count) count.textContent = state.approvals.length || '';

  if (!state.approvals.length) {
    body.innerHTML = '<p class="empty">Nothing waiting for you.</p>';
    return;
  }

  body.innerHTML = '';
  for (const a of state.approvals) {
    const node = $('#approval-card').content.firstElementChild.cloneNode(true);
    $('.approval-action', node).textContent = a.action || 'action';
    $('.approval-task', node).textContent = a.task_id ? ('step ' + a.task_id) : (a.id || '');
    $('.approval-target', node).textContent = a.target || '(no target)';
    const reason = $('.approval-reason', node);
    reason.textContent = a.reason || '';
    reason.hidden = !a.reason;

    const note = $('.approval-note', node);
    if (state.canResolveApprovals) {
      const yes = $('[data-role="approve"]', node);
      const no = $('[data-role="deny"]', node);
      yes.addEventListener('click', () => decide(a.id, true, node));
      no.addEventListener('click', () => decide(a.id, false, node));
    } else {
      // This process does not own the approvals. Say where to
      // go instead of offering a button that will fail.
      $('.approval-actions', node).remove();
      note.textContent = 'Decide with `aios approve ' + (a.id || '<id>') +
        '`, or at the run terminal.';
    }
    body.appendChild(node);
  }
}

async function decide(id, approve, node) {
  const note = $('.approval-note', node);
  // Disable both buttons immediately: a double-click on "Allow
  // once" would otherwise POST twice, and the second POST is a
  // 409 on an approval that is already granted.
  $('.approval-actions button', node).forEach((b) => { b.disabled = true; });
  note.textContent = approve ? 'allowing…' : 'denying…';
  try {
    const res = await apiFetch('/api/approvals/' + encodeURIComponent(id), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ approve: approve }),
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      note.textContent = 'failed: ' + (data.error || res.statusText);
      $('.approval-actions button', node).forEach((b) => { b.disabled = false; });
      return;
    }
    // Drop it locally too: the SSE event will confirm, but
    // waiting for a round trip leaves a decided approval on
    // screen inviting a second click.
    state.approvals = state.approvals.filter((x) => x.id !== id);
    renderApprovals();
  } catch (err) {
    note.textContent = 'failed: ' + err.message;
    $('.approval-actions button', node).forEach((b) => { b.disabled = false; });
  }
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
    body.innerHTML = '<p class="empty">Nothing running yet.</p>';
    return;
  }
  body.innerHTML = '';
  const tpl = document.getElementById('agent-card');
  for (const a of agents) {
    const node = tpl.content.firstElementChild.cloneNode(true);
    node.dataset.status = a.status || 'idle';
    $('.agent-label', node).textContent = a.label || a.id;
    $('.agent-tool', node).textContent = a.tool || (a.task_id ? 'step ' + a.task_id : '');
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
  // The operator is a sender like any other, and a message from
  // the operator is a request for work. Without this option the
  // From list only offered agents, so the one sender whose
  // messages actually start something was the one the browser
  // would not let you pick.
  const op = document.createElement('option');
  op.value = OPERATOR_SENDER; op.textContent = 'Me';
  from.appendChild(op);
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
  else if (curFrom === OPERATOR_SENDER) from.value = curFrom;
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
    const res = await apiFetch('/api/layout', {
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

/* ---------- compose (agent messages) ---------- */

/* The agent message box is for talking to the agents, not for
 * giving the system work. The task composer at the top of the
 * page is the entry point for work; this box is for the
 * operator talking to a running agent.
 *
 * The one exception is the operator's own name in the From
 * list. A message from the operator is a request for work: the
 * server turns it into a plan and runs it. So the operator can
 * start a task from either box, and both paths end in the same
 * place. */
function wireCompose() {
  const form = $('[data-role="compose"]');
  if (!form) return;
  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    const from = $('[data-role="from"]', form);
    const to = $('[data-role="to"]', form);
    const body = $('[data-role="body"]', form);
    if (!body.value.trim()) return;
    if (!from.value) { body.focus(); return; }
    const res = await apiFetch('/api/messages', {
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
  stripSessionToken();
  $('#layout-toggle').addEventListener('click', () => {
    const ed = $('#editor');
    ed.hidden = !ed.hidden;
    if (!ed.hidden) renderEditor();
  });
  $('#reset-layout').addEventListener('click', async () => {
    // Ask the server for the default rather than guessing it
    // here, so the two definitions cannot drift.
    const res = await apiFetch('/api/layout', {
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

  wireAsk();
  wireCompose();
  connect();
  // The panels do not exist until the first snapshot, so
  // the compose handler is wired here and re-attached by
  // renderLayout through the delegated listeners above.
  document.addEventListener('DOMContentLoaded', () => render());
  if (document.readyState !== 'loading') render();
}

boot();

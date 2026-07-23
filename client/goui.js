/**
 * GoUI Client Runtime — vanilla JS, no dependencies.
 * Connects via WebSocket, applies server patches, delegates DOM events.
 */

const SESSION_KEY = 'goui.sessionId';
const EVENT_ATTRS = {
  click: 'g-click',
  change: 'g-change',
  submit: 'g-submit',
  input: 'g-input',
};

export class GoUIClient {
  constructor(wsUrl, componentName, opts = {}) {
    this.wsUrl = wsUrl;
    this.componentName = componentName;
    this.locale = opts.locale || 'tr';
    this.mount = typeof opts.mount === 'string'
      ? document.querySelector(opts.mount)
      : (opts.mount || document.body);
    this.onPush = opts.onPush || ((msg) => console.log('[goui push]', msg));
    this.onError = opts.onError || ((msg) => console.error('[goui error]', msg));
    this.onConnect = opts.onConnect || (() => {});

    this.ws = null;
    this.sessionId = sessionStorage.getItem(SESSION_KEY) || '';
    this.componentRoots = new Map();
    this.reconnectAttempt = 0;
    this.reconnectTimer = null;
    this.inputTimers = new WeakMap();
    this.intentionalClose = false;
    this._ssrHydrated = false;

    this._onClick = this._onClick.bind(this);
    this._onChange = this._onChange.bind(this);
    this._onSubmit = this._onSubmit.bind(this);
    this._onInput = this._onInput.bind(this);
  }

  connect() {
    this.intentionalClose = false;
    this._bindDelegation();
    this._openSocket();
  }

  disconnect() {
    this.intentionalClose = true;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this._unbindDelegation();
  }

  sendEvent(componentId, eventName, payload = {}) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return;
    }
    this.ws.send(JSON.stringify({
      type: 'event',
      component: componentId,
      event: eventName,
      payload,
    }));
  }

  sendPrefetch(componentName) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN || !componentName) {
      return;
    }
    this.ws.send(JSON.stringify({
      type: 'prefetch',
      component: componentName,
    }));
  }

  sendActivate(componentName) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN || !componentName) {
      return;
    }
    this.ws.send(JSON.stringify({
      type: 'activate',
      component: componentName,
    }));
  }

  _bindDelegation() {
    document.addEventListener('click', this._onClick);
    document.addEventListener('change', this._onChange);
    document.addEventListener('submit', this._onSubmit);
    document.addEventListener('input', this._onInput);
  }

  _unbindDelegation() {
    document.removeEventListener('click', this._onClick);
    document.removeEventListener('change', this._onChange);
    document.removeEventListener('submit', this._onSubmit);
    document.removeEventListener('input', this._onInput);
  }

  _openSocket() {
    const url = this._buildUrl();
    this.ws = new WebSocket(url);

    this.ws.onopen = () => {
      this.reconnectAttempt = 0;
      this.onConnect();
    };

    this.ws.onmessage = (ev) => {
      let frame;
      try {
        frame = JSON.parse(ev.data);
      } catch {
        this.onError('invalid frame');
        return;
      }
      this._handleFrame(frame);
    };

    this.ws.onclose = () => {
      this.ws = null;
      if (!this.intentionalClose) {
        this._scheduleReconnect();
      }
    };
  }

  _buildUrl() {
    const base = this.wsUrl.startsWith('ws')
      ? this.wsUrl
      : `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}${this.wsUrl}`;
    const url = new URL(base);
    if (this.sessionId) {
      url.searchParams.set('session', this.sessionId);
    } else {
      url.searchParams.set('component', this.componentName);
    }
    if (this.locale) {
      url.searchParams.set('locale', this.locale);
    }
    return url.toString();
  }

  _scheduleReconnect() {
    const delay = Math.min(500 * (2 ** this.reconnectAttempt), 10000);
    this.reconnectAttempt += 1;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this._openSocket();
    }, delay);
  }

  _handleFrame(frame) {
    switch (frame.type) {
      case 'session':
        this._handleSession(frame.payload);
        break;
      case 'render':
        this._handleRender(frame.component, frame.payload);
        break;
      case 'push':
        this.onPush(frame.payload);
        break;
      case 'error': {
        const message = frame.payload?.message || 'unknown error';
        if (message === 'session not found' || message.includes('session not found')) {
          this.sessionId = '';
          sessionStorage.removeItem(SESSION_KEY);
          this.componentRoots.clear();
          if (this.ws) {
            this.ws.close();
          }
          this.onError(message + ' — reconnecting fresh');
          return;
        }
        this.onError(message);
        break;
      }
      default:
        break;
    }
  }

  _handleSession(payload) {
    const id = typeof payload === 'string' ? payload : payload?.id;
    if (!id) {
      return;
    }
    this.sessionId = id;
    sessionStorage.setItem(SESSION_KEY, id);
  }

  _handleRender(componentId, payload) {
    const patches = Array.isArray(payload) ? payload : JSON.parse(payload);
    if (!Array.isArray(patches)) {
      return;
    }

    let root = this.componentRoots.get(componentId);

    for (const patch of patches) {
      if (patch.op === 'replace' && (!patch.path || patch.path.length === 0)) {
        this._applyFullRender(componentId, patch.html || '');
        root = this.componentRoots.get(componentId);
        continue;
      }

      if (!root) {
        root = this.mount.querySelector(`[data-goui-component="${componentId}"]`) || this.mount;
      }
      applyPatch(root, patch);
    }

    const updated = this.mount.querySelector(`[data-goui-component="${componentId}"]`);
    if (updated) {
      this.componentRoots.set(componentId, updated);
    }
  }

  _applyFullRender(componentId, html) {
    const existing = this.componentRoots.get(componentId);
    if (existing && existing.parentNode) {
      const tpl = document.createElement('template');
      tpl.innerHTML = html.trim();
      const next = tpl.content.firstElementChild;
      if (next) {
        existing.replaceWith(next);
        this.componentRoots.set(componentId, next);
        return;
      }
    }

    // ModeSEO: adopt server-rendered DOM on first connect (avoid flash).
    if (this.mount && !this._ssrHydrated) {
      const ssr = this.mount.querySelector('[data-goui-ssr]');
      if (ssr) {
        ssr.setAttribute('data-goui-component', componentId);
        ssr.removeAttribute('data-goui-ssr');
        this.componentRoots.set(componentId, ssr);
        this._ssrHydrated = true;
        return;
      }
    }

    if (this.mount) {
      const tpl = document.createElement('template');
      tpl.innerHTML = html.trim();
      this.mount.replaceChildren(...tpl.content.childNodes);
      const root = this.mount.querySelector(`[data-goui-component="${componentId}"]`);
      if (root) {
        this.componentRoots.set(componentId, root);
      }
    }
  }

  _onClick(ev) {
    const el = ev.target.closest(`[${EVENT_ATTRS.click}]`);
    if (!el) {
      return;
    }
    const eventName = el.getAttribute(EVENT_ATTRS.click);
    const componentId = this._componentId(el);
    if (!eventName || !componentId) {
      return;
    }
    this.sendEvent(componentId, eventName, collectPayload(el));
  }

  _onChange(ev) {
    const el = ev.target.closest(`[${EVENT_ATTRS.change}]`);
    if (!el) {
      return;
    }
    const eventName = el.getAttribute(EVENT_ATTRS.change);
    const componentId = this._componentId(el);
    if (!eventName || !componentId) {
      return;
    }
    this.sendEvent(componentId, eventName, collectPayload(el));
  }

  _onSubmit(ev) {
    const form = ev.target.closest(`[${EVENT_ATTRS.submit}]`);
    if (!form) {
      return;
    }
    ev.preventDefault();
    const eventName = form.getAttribute(EVENT_ATTRS.submit);
    const componentId = this._componentId(form);
    if (!eventName || !componentId) {
      return;
    }
    this.sendEvent(componentId, eventName, collectFormPayload(form));
  }

  _onInput(ev) {
    const el = ev.target.closest(`[${EVENT_ATTRS.input}]`);
    if (!el) {
      return;
    }
    const eventName = el.getAttribute(EVENT_ATTRS.input);
    const componentId = this._componentId(el);
    if (!eventName || !componentId) {
      return;
    }
    const debounce = parseInt(el.getAttribute('g-debounce') || '300', 10);
    if (this.inputTimers.has(el)) {
      clearTimeout(this.inputTimers.get(el));
    }
    this.inputTimers.set(el, setTimeout(() => {
      this.sendEvent(componentId, eventName, collectPayload(el));
    }, debounce));
  }

  _componentId(el) {
    const root = el.closest('[data-goui-component]');
    return root ? root.getAttribute('data-goui-component') : null;
  }
}

function meaningfulChildren(node) {
  return Array.from(node.childNodes).filter((child) => {
    if (child.nodeType === Node.TEXT_NODE) {
      return child.textContent.trim() !== '';
    }
    return child.nodeType === Node.ELEMENT_NODE;
  });
}

function resolvePath(rootEl, path) {
  let current = rootEl;
  for (const index of path || []) {
    const children = meaningfulChildren(current);
    current = children[index];
    if (!current) {
      return null;
    }
  }
  return current;
}

function isGoUIIgnored(node) {
  if (!node || node.nodeType !== Node.ELEMENT_NODE) {
    return false;
  }
  return node.hasAttribute('data-goui-ignore') || !!node.closest('[data-goui-ignore]');
}

function parseHTML(html) {
  const tpl = document.createElement('template');
  tpl.innerHTML = html.trim();
  return Array.from(tpl.content.childNodes);
}

function applyPatch(rootEl, patch) {
  const path = patch.path || [];

  // Client-owned subtrees (Quill / CodeMirror) must not be reconciled.
  if (path.length > 0) {
    const probe = resolvePath(rootEl, path);
    if (isGoUIIgnored(probe)) {
      return;
    }
    if (patch.op === 'insert' || patch.op === 'remove') {
      const parentPath = path.slice(0, -1);
      const parent = parentPath.length ? resolvePath(rootEl, parentPath) : rootEl;
      if (isGoUIIgnored(parent)) {
        return;
      }
    }
  }

  switch (patch.op) {
    case 'replace': {
      if (path.length === 0) {
        const tpl = document.createElement('template');
        tpl.innerHTML = patch.html || '';
        const next = tpl.content.firstElementChild;
        if (next) {
          rootEl.replaceWith(next);
        } else {
          rootEl.innerHTML = patch.html || '';
        }
        return;
      }
      const target = resolvePath(rootEl, path);
      if (!target) {
        return;
      }
      const tpl = document.createElement('template');
      tpl.innerHTML = patch.html || '';
      const next = tpl.content.firstElementChild;
      if (next) {
        target.replaceWith(next);
      }
      break;
    }
    case 'update_text': {
      const target = resolvePath(rootEl, path);
      if (!target) {
        return;
      }
      if (target.nodeType === Node.TEXT_NODE) {
        target.textContent = patch.text || '';
      } else {
        const children = meaningfulChildren(target);
        if (children[0] && children[0].nodeType === Node.TEXT_NODE) {
          children[0].textContent = patch.text || '';
        } else {
          target.textContent = patch.text || '';
        }
      }
      break;
    }
    case 'set_attr': {
      const target = resolvePath(rootEl, path);
      if (target && target.nodeType === Node.ELEMENT_NODE) {
        const name = patch.attr;
        const value = patch.value ?? '';
        target.setAttribute(name, value);
        // Boolean DOM properties must stay in sync for form controls after patch.
        if (name === 'checked' || name === 'selected' || name === 'disabled' || name === 'readOnly' || name === 'readonly') {
          const prop = name === 'readonly' ? 'readOnly' : name;
          target[prop] = true;
        }
        if (name === 'value' && 'value' in target) {
          target.value = value;
        }
      }
      break;
    }
    case 'remove_attr': {
      const target = resolvePath(rootEl, path);
      if (target && target.nodeType === Node.ELEMENT_NODE) {
        const name = patch.attr;
        target.removeAttribute(name);
        if (name === 'checked' || name === 'selected' || name === 'disabled' || name === 'readOnly' || name === 'readonly') {
          const prop = name === 'readonly' ? 'readOnly' : name;
          target[prop] = false;
        }
      }
      break;
    }
    case 'insert': {
      const parentPath = path.slice(0, -1);
      const index = path[path.length - 1];
      const parent = parentPath.length ? resolvePath(rootEl, parentPath) : rootEl;
      if (!parent) {
        return;
      }
      const nodes = parseHTML(patch.html || '');
      const children = meaningfulChildren(parent);
      const ref = children[index] || null;
      for (const node of nodes) {
        parent.insertBefore(node, ref);
      }
      break;
    }
    case 'remove': {
      const target = resolvePath(rootEl, path);
      if (target) {
        target.remove();
      }
      break;
    }
    case 'move': {
      const parent = resolvePath(rootEl, path);
      if (!parent) {
        return;
      }
      const children = meaningfulChildren(parent);
      const node = children[patch.from_idx];
      if (!node) {
        return;
      }
      let ref = null;
      if (patch.to_idx >= children.length) {
        ref = null;
      } else if (patch.to_idx > patch.from_idx) {
        ref = children[patch.to_idx + 1] || null;
      } else {
        ref = children[patch.to_idx];
      }
      parent.insertBefore(node, ref);
      break;
    }
    default:
      break;
  }
}

function collectPayload(el) {
  const payload = {};
  const dataValue = el.getAttribute && el.getAttribute('data-goui-value');
  if (dataValue != null && dataValue !== '') {
    payload.value = dataValue;
  }
  const dataLevel = el.getAttribute && el.getAttribute('data-goui-level');
  if (dataLevel != null && dataLevel !== '') {
    payload.level = dataLevel;
  }
  const dataIndex = el.getAttribute && el.getAttribute('data-goui-index');
  if (dataIndex != null && dataIndex !== '') {
    payload.index = dataIndex;
  }
  const dataId = el.getAttribute && el.getAttribute('data-goui-id');
  if (dataId != null && dataId !== '') {
    payload.id = dataId;
  }
  const dataName = el.getAttribute && el.getAttribute('data-goui-name');
  if (dataName != null && dataName !== '') {
    payload.name = dataName;
  }
  const dataUrl = el.getAttribute && el.getAttribute('data-goui-url');
  if (dataUrl != null && dataUrl !== '') {
    payload.url = dataUrl;
  }
  const dataSize = el.getAttribute && el.getAttribute('data-goui-size');
  if (dataSize != null && dataSize !== '') {
    payload.size = dataSize;
  }
  const dataCT = el.getAttribute && el.getAttribute('data-goui-content-type');
  if (dataCT != null && dataCT !== '') {
    payload.contentType = dataCT;
  }
  if (el instanceof HTMLInputElement) {
    if (el.type === 'checkbox' || el.type === 'radio') {
      payload.checked = el.checked;
      payload.value = el.value;
      return payload;
    }
    payload.value = el.value;
    return payload;
  }
  if (el instanceof HTMLSelectElement || el instanceof HTMLTextAreaElement) {
    payload.value = el.value;
    return payload;
  }
  return payload;
}

function collectFormPayload(form) {
  const fields = {};
  const data = new FormData(form);
  for (const [key, value] of data.entries()) {
    if (fields[key] !== undefined) {
      if (!Array.isArray(fields[key])) {
        fields[key] = [fields[key]];
      }
      fields[key].push(value);
    } else {
      fields[key] = value;
    }
  }
  return { fields };
}

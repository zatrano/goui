/**
 * Toast UI for WS push frames.
 * Kind: success | error | warning | info
 */

const DEFAULT_MS = 5000;
const ERROR_MS = 8000;

let host = null;

export function enhanceToast(root = document) {
  ensureHost(root === document ? document.body : root);
  return showToast;
}

export function showToast(payload) {
  ensureHost(document.body);
  const kind = normalizeKind(payload?.kind);
  const text = payload?.text || '';
  if (!text) {
    return;
  }

  const el = document.createElement('div');
  el.className = `goui-toast goui-toast-${kind}`;
  el.setAttribute('role', kind === 'error' ? 'alert' : 'status');

  const msg = document.createElement('span');
  msg.className = 'goui-toast-text';
  msg.textContent = text;

  const close = document.createElement('button');
  close.type = 'button';
  close.className = 'goui-toast-close';
  close.setAttribute('aria-label', 'Kapat');
  close.textContent = '×';

  el.append(msg, close);
  // Newest on top
  host.prepend(el);

  const ttl = kind === 'error' ? ERROR_MS : DEFAULT_MS;
  const timer = setTimeout(() => dismiss(el), ttl);

  close.addEventListener('click', () => {
    clearTimeout(timer);
    dismiss(el);
  });
}

function dismiss(el) {
  if (!el || !el.isConnected) {
    return;
  }
  el.classList.add('is-leaving');
  const done = () => el.remove();
  el.addEventListener('transitionend', done, { once: true });
  setTimeout(done, 280);
}

function ensureHost(parent) {
  if (host && host.isConnected) {
    return;
  }
  host = parent.querySelector('.goui-toast-host');
  if (!host) {
    host = document.createElement('div');
    host.className = 'goui-toast-host';
    host.setAttribute('aria-live', 'polite');
    parent.appendChild(host);
  }
}

function normalizeKind(kind) {
  switch (kind) {
    case 'success':
    case 'error':
    case 'warning':
    case 'info':
      return kind;
    default:
      return 'info';
  }
}

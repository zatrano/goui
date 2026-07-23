/**
 * Lightweight prefetch: hover (~100ms) or viewport entry → silent server Mount.
 * Usage: <a href="#" data-goui-prefetch="contact">…</a>
 * Optional activate on click: data-goui-activate="contact"
 */

const HOVER_MS = 100;
const ATTR_PREFETCH = 'data-goui-prefetch';
const ATTR_ACTIVATE = 'data-goui-activate';

/**
 * @param {import('../goui.js').GoUIClient} client
 * @param {ParentNode} [root]
 */
export function enhancePrefetch(client, root = document) {
  const requested = new Set();
  const hoverTimers = new WeakMap();

  function requestPrefetch(name) {
    if (!name || requested.has(name)) {
      return;
    }
    requested.add(name);
    if (typeof client.sendPrefetch === 'function') {
      client.sendPrefetch(name);
    }
  }

  function onMouseOver(ev) {
    const el = ev.target.closest?.(`[${ATTR_PREFETCH}]`);
    if (!el || !root.contains(el)) {
      return;
    }
    const name = el.getAttribute(ATTR_PREFETCH);
    if (!name || requested.has(name)) {
      return;
    }
    if (hoverTimers.has(el)) {
      clearTimeout(hoverTimers.get(el));
    }
    hoverTimers.set(el, setTimeout(() => {
      hoverTimers.delete(el);
      requestPrefetch(name);
    }, HOVER_MS));
  }

  function onMouseOut(ev) {
    const el = ev.target.closest?.(`[${ATTR_PREFETCH}]`);
    if (!el || !root.contains(el)) {
      return;
    }
    const related = ev.relatedTarget;
    if (related && el.contains(related)) {
      return;
    }
    if (hoverTimers.has(el)) {
      clearTimeout(hoverTimers.get(el));
      hoverTimers.delete(el);
    }
  }

  function onClick(ev) {
    const el = ev.target.closest?.(`[${ATTR_ACTIVATE}]`);
    if (!el || !root.contains(el)) {
      return;
    }
    const name = el.getAttribute(ATTR_ACTIVATE);
    if (!name) {
      return;
    }
    ev.preventDefault();
    requestPrefetch(name);
    if (typeof client.sendActivate === 'function') {
      client.sendActivate(name);
    }
  }

  root.addEventListener('mouseover', onMouseOver);
  root.addEventListener('mouseout', onMouseOut);
  root.addEventListener('click', onClick);

  const observed = new WeakSet();
  const io = typeof IntersectionObserver !== 'undefined'
    ? new IntersectionObserver((entries) => {
      for (const entry of entries) {
        if (!entry.isIntersecting) {
          continue;
        }
        const name = entry.target.getAttribute?.(ATTR_PREFETCH);
        if (name) {
          requestPrefetch(name);
        }
      }
    }, { rootMargin: '80px', threshold: 0.01 })
    : null;

  function observeEl(el) {
    if (!io || !el || observed.has(el)) {
      return;
    }
    observed.add(el);
    io.observe(el);
  }

  function scan() {
    const scope = root.querySelectorAll ? root : document;
    scope.querySelectorAll(`[${ATTR_PREFETCH}]`).forEach(observeEl);
  }

  scan();

  const mo = typeof MutationObserver !== 'undefined'
    ? new MutationObserver(() => scan())
    : null;
  if (mo && root.nodeType === Node.ELEMENT_NODE) {
    mo.observe(root, { childList: true, subtree: true });
  } else if (mo && root === document) {
    mo.observe(document.documentElement, { childList: true, subtree: true });
  }

  return {
    dispose() {
      root.removeEventListener('mouseover', onMouseOver);
      root.removeEventListener('mouseout', onMouseOut);
      root.removeEventListener('click', onClick);
      if (io) {
        io.disconnect();
      }
      if (mo) {
        mo.disconnect();
      }
    },
  };
}

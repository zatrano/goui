/**
 * CodeMirror 5 (CDN) bridge for CodeEditor.
 * Editor chrome is client-owned (data-goui-ignore).
 */
export function enhanceCodeEditor(root = document) {
  const scan = () => {
    root.querySelectorAll('[data-goui-cm-mount]:not([data-goui-cm-ready])').forEach(mountCM);
  };
  scan();
  const obs = new MutationObserver(scan);
  obs.observe(root === document ? document.body : root, { childList: true, subtree: true });
  return () => obs.disconnect();
}

function mountCM(seed) {
  if (typeof window.CodeMirror === 'undefined') {
    return;
  }
  seed.setAttribute('data-goui-cm-ready', '1');
  const wrap = seed.closest('[data-goui-code]');
  const sync = wrap && wrap.querySelector('textarea.goui-editor-sync');
  const mode = (wrap && wrap.getAttribute('data-mode')) || 'javascript';
  const initial = (wrap && wrap.getAttribute('data-initial')) || '';

  const cm = window.CodeMirror.fromTextArea(seed, {
    lineNumbers: true,
    mode,
    theme: 'default',
    indentUnit: 2,
    tabSize: 2,
  });

  let hydrating = true;
  cm.on('change', () => {
    if (hydrating || !sync) return;
    sync.value = cm.getValue();
    sync.dispatchEvent(new Event('input', { bubbles: true }));
  });

  if (initial) {
    cm.setValue(initial);
  }
  hydrating = false;
  if (sync) {
    sync.value = cm.getValue();
  }
}

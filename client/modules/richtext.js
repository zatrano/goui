/**
 * Quill (CDN) bridge for RichTextEditor.
 * Editor chrome is client-owned (data-goui-ignore); HTML syncs via empty textarea g-input.
 */
export function enhanceRichText(root = document) {
  const scan = () => {
    root.querySelectorAll('[data-goui-quill-mount]:not([data-goui-quill-ready])').forEach(mountQuill);
  };
  scan();
  const obs = new MutationObserver(scan);
  obs.observe(root === document ? document.body : root, { childList: true, subtree: true });
  return () => obs.disconnect();
}

function mountQuill(mount) {
  if (typeof window.Quill === 'undefined') {
    return;
  }
  mount.setAttribute('data-goui-quill-ready', '1');
  const wrap = mount.closest('[data-goui-richtext]');
  const sync = wrap && wrap.querySelector('textarea.goui-editor-sync');
  const initial = (wrap && wrap.getAttribute('data-initial')) || '';

  const quill = new window.Quill(mount, {
    theme: 'snow',
    modules: {
      toolbar: [
        [{ header: [1, 2, false] }],
        ['bold', 'italic', 'underline'],
        [{ list: 'ordered' }, { list: 'bullet' }],
        ['link'],
        ['clean'],
      ],
    },
  });

  let hydrating = true;
  quill.on('text-change', () => {
    if (hydrating || !sync) return;
    sync.value = quill.root.innerHTML;
    sync.dispatchEvent(new Event('input', { bubbles: true }));
  });

  if (initial) {
    quill.clipboard.dangerouslyPasteHTML(initial);
  }
  hydrating = false;
  if (sync) {
    sync.value = quill.root.innerHTML;
  }
}

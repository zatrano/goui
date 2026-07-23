/**
 * Drag-drop / file input → POST /goui/upload → notify server via carrier g-click.
 */
export function enhanceUpload(root = document) {
  root.addEventListener('dragover', onDragOver);
  root.addEventListener('drop', onDrop);
  root.addEventListener('change', onChange);
}

function onDragOver(ev) {
  const zone = ev.target.closest('[data-goui-upload]');
  if (!zone) return;
  ev.preventDefault();
  zone.classList.add('is-dragover');
}

function onDrop(ev) {
  const zone = ev.target.closest('[data-goui-upload]');
  if (!zone) return;
  ev.preventDefault();
  zone.classList.remove('is-dragover');
  const files = [...(ev.dataTransfer?.files || [])];
  handleFiles(zone, files);
}

function onChange(ev) {
  const input = ev.target;
  if (!(input instanceof HTMLInputElement) || !input.classList.contains('goui-upload-input')) {
    return;
  }
  const zone = input.closest('[data-goui-upload]');
  if (!zone) return;
  handleFiles(zone, [...(input.files || [])]);
  input.value = '';
}

async function handleFiles(zone, files) {
  if (!files.length) return;
  const multiple = zone.getAttribute('data-multiple') === '1';
  const accept = zone.getAttribute('data-accept') || '';
  const url = zone.getAttribute('data-upload-url') || '/goui/upload';
  const list = multiple ? files : files.slice(0, 1);
  for (const file of list) {
    if (accept && !fileMatchesAccept(file, accept)) {
      continue;
    }
    try {
      const meta = await postFile(url, file);
      notifyUploaded(zone, meta);
    } catch (err) {
      console.error('[goui upload]', err);
    }
  }
}

function fileMatchesAccept(file, accept) {
  const parts = accept.split(',').map((s) => s.trim()).filter(Boolean);
  if (!parts.length) return true;
  return parts.some((p) => {
    if (p.endsWith('/*')) {
      return file.type.startsWith(p.slice(0, -1));
    }
    if (p.startsWith('.')) {
      return file.name.toLowerCase().endsWith(p.toLowerCase());
    }
    return file.type === p;
  });
}

export async function postFile(url, file) {
  const fd = new FormData();
  fd.append('file', file, file.name);
  const res = await fetch(url, { method: 'POST', body: fd });
  const data = await res.json();
  if (!res.ok) {
    throw new Error(data.error || res.statusText);
  }
  return data;
}

export function notifyUploaded(zone, meta) {
  const carrier = zone.querySelector('.goui-upload-carrier');
  if (!carrier) return;
  carrier.setAttribute('data-goui-value', meta.id || '');
  carrier.setAttribute('data-goui-id', meta.id || '');
  carrier.setAttribute('data-goui-name', meta.name || '');
  carrier.setAttribute('data-goui-url', meta.url || '');
  carrier.setAttribute('data-goui-size', String(meta.size || 0));
  carrier.setAttribute('data-goui-content-type', meta.contentType || '');
  carrier.click();
}

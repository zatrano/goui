/**
 * Canvas signature → PNG blob → POST /goui/upload → carrier notify.
 */
import { postFile, notifyUploaded } from './upload.js';

export function enhanceSignature(root = document) {
  const scan = () => {
    root.querySelectorAll('[data-goui-signature]:not([data-goui-sig-ready])').forEach(mountPad);
  };
  scan();
  const obs = new MutationObserver(scan);
  obs.observe(root === document ? document.body : root, { childList: true, subtree: true });
  root.addEventListener('click', onClick);
  return () => obs.disconnect();
}

function mountPad(wrap) {
  wrap.setAttribute('data-goui-sig-ready', '1');
  const canvas = wrap.querySelector('.goui-signature-canvas');
  if (!canvas) return;
  const ctx = canvas.getContext('2d');
  ctx.strokeStyle = '#111';
  ctx.lineWidth = 2;
  ctx.lineCap = 'round';
  let drawing = false;

  const pos = (e) => {
    const r = canvas.getBoundingClientRect();
    const x = (e.clientX - r.left) * (canvas.width / r.width);
    const y = (e.clientY - r.top) * (canvas.height / r.height);
    return { x, y };
  };

  canvas.addEventListener('pointerdown', (e) => {
    drawing = true;
    canvas.setPointerCapture(e.pointerId);
    const p = pos(e);
    ctx.beginPath();
    ctx.moveTo(p.x, p.y);
  });
  canvas.addEventListener('pointermove', (e) => {
    if (!drawing) return;
    const p = pos(e);
    ctx.lineTo(p.x, p.y);
    ctx.stroke();
  });
  canvas.addEventListener('pointerup', () => { drawing = false; });
  wrap._sigCtx = ctx;
  wrap._sigCanvas = canvas;
}

function onClick(ev) {
  const save = ev.target.closest('.goui-signature-save');
  const clear = ev.target.closest('.goui-signature-clear-local');
  if (save) {
    ev.preventDefault();
    const wrap = save.closest('[data-goui-signature]');
    savePad(wrap);
  } else if (clear) {
    ev.preventDefault();
    const wrap = clear.closest('[data-goui-signature]');
    const canvas = wrap && wrap.querySelector('.goui-signature-canvas');
    const ctx = canvas && canvas.getContext('2d');
    if (ctx && canvas) {
      ctx.clearRect(0, 0, canvas.width, canvas.height);
    }
  }
}

async function savePad(wrap) {
  if (!wrap) return;
  const canvas = wrap.querySelector('.goui-signature-canvas');
  const url = wrap.getAttribute('data-upload-url') || '/goui/upload';
  await new Promise((resolve) => {
    canvas.toBlob(async (blob) => {
      try {
        const file = new File([blob], 'signature.png', { type: 'image/png' });
        const meta = await postFile(url, file);
        notifyUploaded(wrap, meta);
      } catch (err) {
        console.error('[goui signature]', err);
      } finally {
        resolve();
      }
    }, 'image/png');
  });
}

/**
 * Minimal 1:1 avatar crop — select image, drag to pan, crop square, upload.
 */
import { postFile, notifyUploaded } from './upload.js';

export function enhanceAvatar(root = document) {
  root.addEventListener('change', onPick);
  root.addEventListener('click', onClick);
}

function onPick(ev) {
  const input = ev.target;
  if (!(input instanceof HTMLInputElement) || !input.classList.contains('goui-avatar-input')) {
    return;
  }
  const wrap = input.closest('[data-goui-avatar]');
  const file = input.files && input.files[0];
  if (!wrap || !file) return;
  input.value = '';
  openCrop(wrap, file);
}

function onClick(ev) {
  const apply = ev.target.closest('.goui-crop-apply');
  const cancel = ev.target.closest('.goui-crop-cancel');
  if (apply) {
    ev.preventDefault();
    const wrap = apply.closest('[data-goui-avatar]');
    finishCrop(wrap);
  } else if (cancel) {
    ev.preventDefault();
    const wrap = cancel.closest('[data-goui-avatar]');
    closeCrop(wrap);
  }
}

function openCrop(wrap, file) {
  const overlay = wrap.querySelector('.goui-crop-overlay');
  const canvas = wrap.querySelector('.goui-crop-canvas');
  if (!overlay || !canvas) return;
  const ctx = canvas.getContext('2d');
  const img = new Image();
  const url = URL.createObjectURL(file);
  img.onload = () => {
    wrap._crop = { img, url, offsetX: 0, offsetY: 0, dragging: false, lastX: 0, lastY: 0 };
    overlay.hidden = false;
    drawCrop(wrap);
    canvas.onpointerdown = (e) => {
      wrap._crop.dragging = true;
      wrap._crop.lastX = e.clientX;
      wrap._crop.lastY = e.clientY;
      canvas.setPointerCapture(e.pointerId);
    };
    canvas.onpointermove = (e) => {
      if (!wrap._crop?.dragging) return;
      wrap._crop.offsetX += e.clientX - wrap._crop.lastX;
      wrap._crop.offsetY += e.clientY - wrap._crop.lastY;
      wrap._crop.lastX = e.clientX;
      wrap._crop.lastY = e.clientY;
      drawCrop(wrap);
    };
    canvas.onpointerup = () => { if (wrap._crop) wrap._crop.dragging = false; };
  };
  img.src = url;
}

function drawCrop(wrap) {
  const canvas = wrap.querySelector('.goui-crop-canvas');
  const { img, offsetX, offsetY } = wrap._crop;
  const ctx = canvas.getContext('2d');
  const size = canvas.width;
  const scale = Math.max(size / img.width, size / img.height);
  const w = img.width * scale;
  const h = img.height * scale;
  ctx.fillStyle = '#222';
  ctx.fillRect(0, 0, size, size);
  ctx.drawImage(img, (size - w) / 2 + offsetX, (size - h) / 2 + offsetY, w, h);
  ctx.strokeStyle = 'rgba(255,255,255,0.8)';
  ctx.lineWidth = 2;
  ctx.strokeRect(1, 1, size - 2, size - 2);
}

async function finishCrop(wrap) {
  if (!wrap?._crop) return;
  const canvas = wrap.querySelector('.goui-crop-canvas');
  const url = wrap.getAttribute('data-upload-url') || '/goui/upload';
  await new Promise((resolve) => {
    canvas.toBlob(async (blob) => {
      try {
        const file = new File([blob], 'avatar.png', { type: 'image/png' });
        const meta = await postFile(url, file);
        notifyUploaded(wrap, meta);
      } catch (err) {
        console.error('[goui avatar]', err);
      } finally {
        closeCrop(wrap);
        resolve();
      }
    }, 'image/png');
  });
}

function closeCrop(wrap) {
  const overlay = wrap.querySelector('.goui-crop-overlay');
  if (overlay) overlay.hidden = true;
  if (wrap._crop?.url) URL.revokeObjectURL(wrap._crop.url);
  wrap._crop = null;
}

/**
 * OTP/PIN cell helpers: auto-advance, backspace, paste.
 * Values still flow to the server via g-input / commit events.
 */
export function enhanceOTP(root = document) {
  root.addEventListener('input', onInput);
  root.addEventListener('keydown', onKeydown);
  root.addEventListener('paste', onPaste);
}

function cellsOf(el) {
  const wrap = el.closest('[data-goui-otp]');
  if (!wrap) return null;
  return [...wrap.querySelectorAll('.goui-otp-cell')];
}

function onInput(ev) {
  const el = ev.target;
  if (!(el instanceof HTMLInputElement) || !el.classList.contains('goui-otp-cell')) {
    return;
  }
  const cells = cellsOf(el);
  if (!cells) return;
  const idx = cells.indexOf(el);
  if (el.value.length > 1) {
    el.value = el.value.slice(-1);
  }
  if (el.value && idx >= 0 && idx < cells.length - 1) {
    cells[idx + 1].focus();
    cells[idx + 1].select();
  }
}

function onKeydown(ev) {
  const el = ev.target;
  if (!(el instanceof HTMLInputElement) || !el.classList.contains('goui-otp-cell')) {
    return;
  }
  const cells = cellsOf(el);
  if (!cells) return;
  const idx = cells.indexOf(el);
  if (ev.key === 'Backspace' && !el.value && idx > 0) {
    cells[idx - 1].focus();
    cells[idx - 1].select();
  } else if (ev.key === 'ArrowLeft' && idx > 0) {
    ev.preventDefault();
    cells[idx - 1].focus();
  } else if (ev.key === 'ArrowRight' && idx < cells.length - 1) {
    ev.preventDefault();
    cells[idx + 1].focus();
  }
}

function onPaste(ev) {
  const el = ev.target;
  if (!(el instanceof HTMLInputElement) || !el.classList.contains('goui-otp-cell')) {
    return;
  }
  const wrap = el.closest('[data-goui-otp]');
  const cells = cellsOf(el);
  if (!wrap || !cells) return;
  const text = (ev.clipboardData || window.clipboardData).getData('text') || '';
  const chars = text.replace(/\s+/g, '').split('').slice(0, cells.length);
  if (!chars.length) return;
  ev.preventDefault();
  chars.forEach((ch, i) => {
    cells[i].value = ch;
  });
  const commit = wrap.getAttribute('data-goui-otp-commit');
  if (commit) {
    // Trigger a synthetic change on a temp carrier via last cell g-input path:
    // fire input on each cell so server gets digits, then focus last.
    cells.forEach((cell) => {
      cell.dispatchEvent(new Event('input', { bubbles: true }));
    });
  }
  const focusIdx = Math.min(chars.length, cells.length) - 1;
  if (focusIdx >= 0) {
    cells[focusIdx].focus();
  }
}

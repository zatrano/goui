/**
 * Optional Tier-2 selectable helpers.
 * Does NOT filter options client-side — filtering is server-driven via g-input.
 * Provides keyboard highlight within an already-rendered list.
 */
export function enhanceSelectable(root = document) {
  root.addEventListener('keydown', (ev) => {
    const panel = ev.target.closest('.goui-searchable-panel');
    if (!panel) {
      return;
    }
    const options = [...panel.querySelectorAll('.goui-searchable-option:not(.is-disabled)')];
    if (!options.length) {
      return;
    }
    let idx = options.findIndex((el) => el.classList.contains('is-active'));
    if (ev.key === 'ArrowDown') {
      ev.preventDefault();
      idx = Math.min(options.length - 1, idx + 1);
      setActive(options, idx);
    } else if (ev.key === 'ArrowUp') {
      ev.preventDefault();
      idx = Math.max(0, idx < 0 ? 0 : idx - 1);
      setActive(options, idx);
    } else if (ev.key === 'Enter' && idx >= 0) {
      ev.preventDefault();
      options[idx].click();
    }
  });
}

function setActive(options, idx) {
  options.forEach((el, i) => {
    el.classList.toggle('is-active', i === idx);
  });
  options[idx]?.scrollIntoView({ block: 'nearest' });
}

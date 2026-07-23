/**
 * Visual calendar for CalendarDatePicker.
 * Month/year navigation is client-only; day selection uses g-click + data-goui-value
 * so the existing GoUIClient delegation sends the final date to the server.
 */
const WEEKDAYS = ['Pt', 'Sa', 'Ça', 'Pe', 'Cu', 'Ct', 'Pz'];
const MONTHS = [
  'Ocak', 'Şubat', 'Mart', 'Nisan', 'Mayıs', 'Haziran',
  'Temmuz', 'Ağustos', 'Eylül', 'Ekim', 'Kasım', 'Aralık',
];

export function enhanceCalendar(root = document) {
  const scan = () => {
    root.querySelectorAll('[data-goui-calendar-mount]:not([data-goui-cal-ready])').forEach(mount);
  };
  scan();
  const obs = new MutationObserver(scan);
  obs.observe(root === document ? document.body : root, { childList: true, subtree: true });
  return () => obs.disconnect();
}

function mount(panel) {
  panel.setAttribute('data-goui-cal-ready', '1');
  const selected = panel.getAttribute('data-selected') || '';
  const min = panel.getAttribute('data-min') || '';
  const max = panel.getAttribute('data-max') || '';
  const selectEvent = panel.getAttribute('data-select-event') || 'select';

  let view = parseYMD(selected) || new Date();
  view = new Date(view.getFullYear(), view.getMonth(), 1);

  const render = () => {
    panel.innerHTML = '';
    panel.appendChild(buildHeader(view, (next) => {
      view = next;
      render();
    }));
    panel.appendChild(buildGrid(view, selected, min, max, selectEvent));
  };
  render();
}

function buildHeader(view, setView) {
  const header = document.createElement('div');
  header.className = 'goui-calendar-header';

  const prev = document.createElement('button');
  prev.type = 'button';
  prev.className = 'goui-calendar-nav';
  prev.textContent = '‹';
  prev.addEventListener('click', (ev) => {
    ev.preventDefault();
    ev.stopPropagation();
    setView(new Date(view.getFullYear(), view.getMonth() - 1, 1));
  });

  const label = document.createElement('div');
  label.className = 'goui-calendar-month';
  label.textContent = `${MONTHS[view.getMonth()]} ${view.getFullYear()}`;

  const next = document.createElement('button');
  next.type = 'button';
  next.className = 'goui-calendar-nav';
  next.textContent = '›';
  next.addEventListener('click', (ev) => {
    ev.preventDefault();
    ev.stopPropagation();
    setView(new Date(view.getFullYear(), view.getMonth() + 1, 1));
  });

  header.append(prev, label, next);
  return header;
}

function buildGrid(view, selected, min, max, selectEvent) {
  const wrap = document.createElement('div');
  wrap.className = 'goui-calendar-grid';

  WEEKDAYS.forEach((d) => {
    const el = document.createElement('div');
    el.className = 'goui-calendar-dow';
    el.textContent = d;
    wrap.appendChild(el);
  });

  const year = view.getFullYear();
  const month = view.getMonth();
  const firstDow = (new Date(year, month, 1).getDay() + 6) % 7; // Monday=0
  const daysInMonth = new Date(year, month + 1, 0).getDate();

  for (let i = 0; i < firstDow; i++) {
    const empty = document.createElement('div');
    empty.className = 'goui-calendar-day is-empty';
    wrap.appendChild(empty);
  }

  for (let day = 1; day <= daysInMonth; day++) {
    const ymd = formatYMD(year, month, day);
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'goui-calendar-day';
    btn.textContent = String(day);
    btn.setAttribute('g-click', selectEvent);
    btn.setAttribute('data-goui-value', ymd);

    if (ymd === selected) {
      btn.classList.add('is-selected');
    }
    if ((min && ymd < min) || (max && ymd > max)) {
      btn.disabled = true;
      btn.classList.add('is-disabled');
      btn.removeAttribute('g-click');
    }
    const today = formatYMD(new Date().getFullYear(), new Date().getMonth(), new Date().getDate());
    if (ymd === today) {
      btn.classList.add('is-today');
    }
    wrap.appendChild(btn);
  }
  return wrap;
}

function parseYMD(s) {
  if (!s || !/^\d{4}-\d{2}-\d{2}$/.test(s)) {
    return null;
  }
  const [y, m, d] = s.split('-').map(Number);
  return new Date(y, m - 1, d);
}

function formatYMD(y, m0, d) {
  const mm = String(m0 + 1).padStart(2, '0');
  const dd = String(d).padStart(2, '0');
  return `${y}-${mm}-${dd}`;
}

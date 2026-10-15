// select, then open (product/spec.md, interaction): a task is a link marked
// data-task. a click selects it; a click on the selected one follows the link,
// so a double click opens too. a click elsewhere, or Esc, deselects.
document.addEventListener("click", (e) => {
  const task = e.target.closest("[data-task]");
  const selected = document.querySelector("[data-task][data-selected]");
  if (task && task === selected) return;
  selected?.removeAttribute("data-selected");
  if (task) {
    e.preventDefault();
    task.setAttribute("data-selected", "");
  }
});

document.addEventListener("keydown", (e) => {
  if (e.key === "Escape") document.querySelector("[data-task][data-selected]")?.removeAttribute("data-selected");
});

// an info toast goes after ~5 s; errors stay until the next one. the undo
// after a delete is one (the + comes back with it, css), and the url forgets
// ?deleted= so a reload or back shows no undo. settings swaps toasts in.
function timeToast() {
  const info = document.querySelector('#toast [role="status"]:not([data-timed])');
  if (!info) return;
  info.setAttribute("data-timed", "");
  if (info.hasAttribute("data-undo")) history.replaceState(null, "", location.pathname);
  setTimeout(() => info.parentElement.remove(), 5000);
}
timeToast();
document.addEventListener("htmx:oobAfterSwap", timeToast);

// "+ Add time" in the task form: one more box in the weekly schedule, copied
// from its <template> with the next free index.
document.addEventListener("click", (e) => {
  const add = e.target.closest("[data-add-time]");
  if (!add) return;
  const places = add.closest("[data-places]");
  const used = [...places.querySelectorAll('input[name="place"]')].map((i) => Number(i.value));
  const next = used.length ? Math.max(...used) + 1 : 0;
  const html = places.querySelector("template[data-place]").innerHTML.replaceAll("__i__", next);
  add.insertAdjacentHTML("beforebegin", html);
  pickOnTouch(add.previousElementSibling);
});

// the week opens scrolled to now, and the now-line moves on every minute,
// counted from the minute the server drew it. a tab bar link swaps the page in
// (hx-boost), so a new page starts it again.
function startNow() {
  const now = document.querySelector("[data-now]");
  if (!now) return;
  now.dataset.loaded = Date.now();
  now.scrollIntoView({ block: "center" });
}
startNow();
document.addEventListener("htmx:afterSettle", startNow);
setInterval(() => {
  const now = document.querySelector("[data-now]");
  if (!now) return;
  const timeline = now.closest("[data-timeline]");
  const from = Number(timeline.dataset.from) * 60;
  const span = Number(timeline.dataset.span) * 60;
  const minute = Number(now.dataset.now) + Math.floor((Date.now() - Number(now.dataset.loaded)) / 60000);
  if (minute > from + span) now.remove();
  else now.style.top = ((minute - from) / span) * 100 + "%";
}, 60000);

// the week on the phone: one day, picked in the day picker or by a swipe.
// the day's head, chips and column carry data-shown; past either end of the
// week, the week before or after opens. the url keeps the day for a reload.
function showDay(week, i) {
  const heads = week.querySelectorAll("a[data-day]");
  if (i < 0) return location.assign(week.dataset.before);
  if (i >= heads.length) return location.assign(week.dataset.after);
  for (const el of week.querySelectorAll("[data-day]")) el.toggleAttribute("data-shown", el.dataset.day === String(i));
  week.querySelector("[data-day-name]").textContent = heads[i].dataset.name;
  history.replaceState(null, "", heads[i].href);
}

document.addEventListener("click", (e) => {
  const head = e.target.closest("[data-week] a[data-day]");
  if (!head) return;
  e.preventDefault();
  showDay(head.closest("[data-week]"), Number(head.dataset.day));
});

let swipe = null; // where a one-finger touch started
document.addEventListener("touchstart", (e) => {
  swipe = e.touches.length === 1 ? { x: e.touches[0].clientX, y: e.touches[0].clientY } : null;
}, { passive: true });
document.addEventListener("touchend", (e) => {
  const week = e.target.closest("[data-week]");
  if (!swipe || !week || matchMedia("(min-width: 768px)").matches) return;
  const dx = e.changedTouches[0].clientX - swipe.x;
  const dy = e.changedTouches[0].clientY - swipe.y;
  swipe = null;
  if (Math.abs(dx) < 50 || Math.abs(dx) < 2 * Math.abs(dy)) return;
  const on = week.querySelector("a[data-day][data-shown]");
  showDay(week, Number(on.dataset.day) + (dx < 0 ? 1 : -1));
});

// settings' emoji: a face picked in the open box shows in it, and the box
// closes; a click elsewhere or Esc closes it too.
document.addEventListener("change", (e) => {
  const box = e.target.closest("details[data-emoji]");
  if (!box) return;
  box.querySelector("summary").textContent = e.target.value;
  box.open = false;
});
document.addEventListener("click", (e) => {
  for (const box of document.querySelectorAll("details[data-emoji][open]")) if (!box.contains(e.target)) box.open = false;
});
document.addEventListener("keydown", (e) => {
  if (e.key === "Escape") for (const box of document.querySelectorAll("details[data-emoji][open]")) box.open = false;
});

// the phone's ‹ (settings): back where the person came from, if it was here
// (a boosted link leaves htmx's mark on the history entry); else its link.
document.addEventListener("click", (e) => {
  if (!e.target.closest("[data-back]")) return;
  if (history.length > 1 && (history.state?.htmx || document.referrer.startsWith(location.origin))) {
    e.preventDefault();
    history.back();
  }
});

// a time field: what was typed (1830, 18.30, 8) becomes HH:MM on leaving it,
// as the server reads it (clock.ReadTime). anything else stays for the server
// to refuse.
document.addEventListener("focusout", (e) => {
  if (!e.target.matches("[data-time]")) return;
  const s = e.target.value.trim().replace(/[.hH]/, ":");
  let m = s.match(/^(\d{1,2}):(\d{2})?$/) || s.match(/^(\d{1,2})$/);
  if (!m) m = s.match(/^(\d{1,2})(\d{2})$/);
  if (!m) return;
  const [h, min] = [Number(m[1]), Number(m[2] || 0)];
  if (h < 24 && min < 60 || h === 24 && min === 0) {
    e.target.value = String(h).padStart(2, "0") + ":" + String(min).padStart(2, "0");
  }
});

// the pickers (figma: Date picker, Time picker): a month under a date field, the
// 24 hours and then minutes in 5s under a time field. they only fill the
// field; the server reads it as it reads typing (clock.ReadDate, ReadTime). on
// a touch screen the field opens its picker instead of the keyboard.
const months = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
const longMonths = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"];
const weekdays = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
const dayMs = 86400000;
const midnight = (d) => new Date(d.getFullYear(), d.getMonth(), d.getDate());
const days = (a, b) => Math.round((midnight(b) - midnight(a)) / dayMs);

// readDate reads a field as clock.ReadDate does: 2026-10-15, 15/10(/2026),
// Oct 15 (2026), Thu, Oct 15; no year is the nearest one. null if not a date.
function readDate(s) {
  const today = midnight(new Date());
  let m = s.trim().match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if (m) return new Date(+m[1], m[2] - 1, +m[3]);
  const words = s.toLowerCase().split(/[\s/.,-]+/).filter(Boolean);
  let month = -1;
  const nums = [];
  for (const w of words) {
    const i = months.findIndex((x) => w.startsWith(x.toLowerCase()));
    if (w.length >= 3 && i >= 0) month = i;
    else if (/^\d+$/.test(w)) nums.push(+w);
    else if (!(w.length >= 3 && weekdays.some((x) => w.startsWith(x.toLowerCase())))) return null;
  }
  let day, year;
  if (month >= 0 && nums.length >= 1 && nums.length <= 2) [day, year] = nums;
  else if (month < 0 && nums.length >= 2 && nums.length <= 3) [day, month, year] = [nums[0], nums[1] - 1, nums[2]];
  else return null;
  if (year !== undefined && year < 100) year += 2000;
  if (year === undefined) {
    year = [-1, 0, 1].map((k) => today.getFullYear() + k)
      .sort((a, b) => Math.abs(days(today, new Date(a, month, day))) - Math.abs(days(today, new Date(b, month, day))))[0];
  }
  const d = new Date(year, month, day);
  return d.getMonth() === month && d.getDate() === day ? d : null;
}

// shortDate writes a date as Date.Short does: "Thu, Oct 15", with the year
// past ~6 months.
function shortDate(d) {
  let s = `${weekdays[d.getDay()]}, ${months[d.getMonth()]} ${d.getDate()}`;
  if (Math.abs(days(new Date(), d)) > 182) s += `, ${d.getFullYear()}`;
  return s;
}

let picker = null; // the one open picker: { el, field }

function closePicker() {
  picker?.el.remove();
  picker = null;
}

const cell = "flex h-9 cursor-pointer items-center justify-center text-body text-text";
const picked = " border-2 border-border bg-accent";

function openPicker(field) {
  if (picker?.field === field) return;
  closePicker();
  const el = document.createElement("div");
  el.className = "absolute left-0 top-full z-10 mt-1 flex w-[280px] flex-col gap-2 border-2 border-border bg-surface p-3";
  el.addEventListener("mousedown", (e) => e.preventDefault()); // keep the field focused
  picker = { el, field };
  field.parentElement.append(el);
  if (field.matches("[data-date]")) {
    const d = readDate(field.value) || new Date();
    drawMonth(new Date(d.getFullYear(), d.getMonth(), 1));
  } else drawTime();
  place(el, field);
}

// place flips the picker above its field when its scroller has no room below,
// and to the field's right edge when it would run off the right.
function place(el, field) {
  const box = field.closest(".overflow-y-auto")?.getBoundingClientRect() ?? { top: 0, bottom: innerHeight, right: innerWidth };
  const f = field.getBoundingClientRect();
  if (f.bottom + 4 + el.offsetHeight > Math.min(box.bottom, innerHeight) && f.top - 4 - el.offsetHeight > box.top) {
    el.classList.replace("top-full", "bottom-[48px]");
    el.classList.replace("mt-1", "mb-1");
  }
  if (f.left + el.offsetWidth > Math.min(box.right ?? innerWidth, innerWidth)) {
    el.classList.replace("left-0", "right-0");
  }
}

function drawMonth(first) {
  const { el, field } = picker;
  const chosen = readDate(field.value);
  const today = midnight(new Date());
  const start = new Date(first.getFullYear(), first.getMonth(), 1 - ((first.getDay() + 6) % 7)); // monday first
  const nav = "inline-flex size-9 cursor-pointer items-center justify-center border-2 border-border bg-surface text-strong text-text";
  let html = `<div class="flex items-center justify-between"><span class="text-strong text-text">${longMonths[first.getMonth()]} ${first.getFullYear()}</span>
    <span class="flex gap-2"><button type="button" class="${nav}" data-month="-1">‹</button><button type="button" class="${nav}" data-month="1">›</button></span></div>
    <div class="grid grid-cols-[repeat(7,36px)]">${["M", "T", "W", "T", "F", "S", "S"].map((d) => `<span class="flex h-5 items-center justify-center text-label text-text-muted uppercase">${d}</span>`).join("")}</div>
    <div class="grid grid-cols-[repeat(7,36px)]">`;
  for (let i = 0; i < 42; i++) {
    const d = new Date(start.getFullYear(), start.getMonth(), start.getDate() + i);
    let cls = cell;
    if (d.getMonth() !== first.getMonth()) cls = cls.replace("text-text", "text-text-muted");
    if (chosen && days(chosen, d) === 0) cls += picked;
    else if (days(today, d) === 0) cls += " border-2 border-border";
    html += `<button type="button" class="${cls}" data-day="${d.getFullYear()}-${d.getMonth()}-${d.getDate()}">${d.getDate()}</button>`;
  }
  el.innerHTML = html + "</div>";
  el.onclick = (e) => {
    const b = e.target.closest("button");
    if (!b) return;
    if (b.dataset.month) return drawMonth(new Date(first.getFullYear(), first.getMonth() + Number(b.dataset.month), 1));
    const [y, m, d] = b.dataset.day.split("-").map(Number);
    field.value = shortDate(new Date(y, m, d));
    closePicker();
  };
}

function drawTime() {
  const { el, field } = picker;
  const [h, m] = (field.value.match(/^(\d{2}):(\d{2})$/) || []).slice(1);
  const grid = (name, values, on) =>
    `<span class="text-label text-text-muted uppercase">${name}</span><div class="grid grid-cols-[repeat(6,42px)]">${values
      .map((v) => `<button type="button" class="${cell}${v === on ? picked : ""}" data-${name.toLowerCase()}="${v}">${v}</button>`)
      .join("")}</div>`;
  const two = (n) => String(n).padStart(2, "0");
  el.innerHTML = grid("Hour", Array.from({ length: 24 }, (_, i) => two(i)), h) + grid("Minute", Array.from({ length: 12 }, (_, i) => two(i * 5)), m);
  el.onclick = (e) => {
    const b = e.target.closest("button");
    if (!b) return;
    if (b.dataset.hour) {
      field.value = `${b.dataset.hour}:${m ?? "00"}`;
      return drawTime(); // now the minutes
    }
    field.value = `${h ?? "00"}:${b.dataset.minute}`;
    closePicker();
  };
}

document.addEventListener("focusin", (e) => {
  if (e.target.matches("[data-date], [data-time]")) openPicker(e.target);
});
document.addEventListener("click", (e) => {
  // the path as dispatched: a redraw may have replaced the button clicked
  if (!picker || e.composedPath().includes(picker.el) || e.target === picker.field) return;
  if (e.target.matches?.("[data-date], [data-time]")) return openPicker(e.target);
  closePicker();
});
document.addEventListener("keydown", (e) => {
  if (e.key === "Escape") closePicker();
  if (e.key === "Tab") closePicker();
});
// a touch screen picks: its date and time fields open no keyboard. a keyboard
// may still type.
function pickOnTouch(root) {
  if (!matchMedia("(pointer: coarse)").matches) return;
  for (const f of root.querySelectorAll("[data-date], [data-time]")) f.setAttribute("inputmode", "none");
}
pickOnTouch(document);

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

// the undo toast after a delete: it goes after ~5 s (the + comes back with
// it, css), and the url forgets ?deleted= so a reload or back shows no undo.
const undo = document.querySelector("[data-undo]");
if (undo) {
  history.replaceState(null, "", location.pathname);
  setTimeout(() => undo.parentElement.remove(), 5000);
}

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
});

// the week opens scrolled to now, and the now-line moves on every minute,
// counted from the minute the server drew it.
const now = document.querySelector("[data-now]");
if (now) {
  const timeline = now.closest("[data-timeline]");
  const from = Number(timeline.dataset.from) * 60;
  const span = Number(timeline.dataset.span) * 60;
  const drawn = Number(now.dataset.now);
  const loaded = Date.now();
  now.scrollIntoView({ block: "center" });
  setInterval(() => {
    const minute = drawn + Math.floor((Date.now() - loaded) / 60000);
    if (minute > from + span) now.remove();
    else now.style.top = ((minute - from) / span) * 100 + "%";
  }, 60000);
}

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

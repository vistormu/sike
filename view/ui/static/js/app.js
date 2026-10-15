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

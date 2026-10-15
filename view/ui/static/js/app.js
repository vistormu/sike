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

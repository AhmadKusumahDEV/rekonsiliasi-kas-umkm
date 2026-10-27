// Motion never owns app state, delays an action, or hides content without JS.
const reduced = matchMedia("(prefers-reduced-motion: reduce)");
const active = new Set();
let keyboard = false;
let lastView = "";
const enter = "cubic-bezier(0.22, 1, 0.36, 1)";
function settle() {
  for (const animation of active) {
    try {
      animation.finish();
    } catch {
      animation.cancel();
    }
  }
  active.clear();
}
document.addEventListener(
  "keydown",
  () => {
    keyboard = true;
    document.body.dataset.input = "keyboard";
    settle();
  },
  { capture: true },
);
document.addEventListener(
  "pointerdown",
  () => {
    keyboard = false;
    document.body.dataset.input = "pointer";
  },
  { capture: true, passive: true },
);
reduced.addEventListener("change", () => {
  if (reduced.matches) settle();
});
function play(element, frames, duration, delay = 0) {
  if (!element || reduced.matches || keyboard || !element.animate) return;
  const animation = element.animate(frames, {
    duration,
    delay,
    easing: enter,
    fill: "backwards",
  });
  active.add(animation);
  const done = () => active.delete(animation);
  animation.addEventListener("finish", done, { once: true });
  animation.addEventListener("cancel", done, { once: true });
}
export function presentView(container, view) {
  if (document.body.dataset.kind !== "umkm" || view === lastView) return;
  const order = [
    "overview",
    "entries",
    "reconcile",
    "imports",
    "history",
    "reports",
    "settings",
  ];
  const direction = order.indexOf(view) < order.indexOf(lastView) ? -1 : 1;
  lastView = view;
  settle();
  [...container.children].slice(0, 6).forEach((element, index) => {
    play(
      element,
      [
        {
          opacity: 0,
          transform:
            "translate3d(" + (index ? 0 : direction * 12) + "px,10px,0)",
        },
        { opacity: 1, transform: "translate3d(0,0,0)" },
      ],
      320,
      Math.min(index * 35, 140),
    );
  });
}
export function presentAuth(container, register) {
  if (document.body.dataset.kind !== "umkm") return;
  lastView = "";
  settle();
  const photo = container.querySelector(".auth-story > img");
  play(
    photo,
    [
      { opacity: 0.5, transform: "scale(1.025)" },
      { opacity: 1, transform: "scale(1)" },
    ],
    600,
  );
  const parts = container.querySelectorAll(
    ".auth-copy > *, .auth-heading, .auth-inner > form > label, .auth-inner > form > #owner-fields, .auth-inner > form > button, .auth-switch",
  );
  [...parts].forEach((element, index) => {
    play(
      element,
      [
        {
          opacity: 0,
          transform: "translate3d(" + (register ? 8 : -8) + "px,8px,0)",
        },
        { opacity: 1, transform: "translate3d(0,0,0)" },
      ],
      340,
      Math.min(index * 30, 180),
    );
  });
}

// The in-match pause menu (backlog item 021): Continue, Restart and
// Character Change, named and ordered as Ikemen GO's default [Pause Menu]
// (resources/defaultMotif.ini, commit 54d4148a). Ikemen GO's Exit quits the
// program; a browser tab has no program to quit, so the web build leaves it
// out. Plain DOM built once and toggled in place, like `result-screen.ts`.
import { onLocaleChange, t } from "../i18n/i18n.ts";

/** A menu input, from the keyboard or a gamepad. */
export type PauseNav = "up" | "down" | "confirm" | "cancel";

export interface PauseMenuOptions {
  onContinue: () => void;
  onRestart: () => void;
  onCharacterChange: () => void;
}

export interface PauseMenu {
  /** The menu's root element, hidden until `open()`. */
  element: HTMLElement;
  /** Shows the menu with Continue highlighted and focused. */
  open(): void;
  /** Hides the menu without choosing anything. */
  close(): void;
  isOpen(): boolean;
  /** Moves the highlight, chooses the highlighted entry, or cancels (= Continue). */
  navigate(nav: PauseNav): void;
  /** Releases the locale subscription. */
  dispose(): void;
}

interface Entry {
  key: string;
  label: string;
  choose: () => void;
}

export function createPauseMenu(options: PauseMenuOptions): PauseMenu {
  const entries: Entry[] = [
    { key: "pause.continue", label: "Continue", choose: options.onContinue },
    { key: "pause.restart", label: "Restart", choose: options.onRestart },
    {
      key: "pause.characterChange",
      label: "Character Change",
      choose: options.onCharacterChange,
    },
  ];

  const element = document.createElement("div");
  element.className = "pause-menu";
  element.setAttribute("role", "dialog");
  element.setAttribute("aria-modal", "true");
  element.setAttribute("hidden", "");

  const panel = document.createElement("wuik-panel");
  panel.className = "pause-menu__panel";
  element.appendChild(panel);

  const title = document.createElement("h2");
  title.className = "pause-menu__title";
  title.id = "pause-menu-title";
  element.setAttribute("aria-labelledby", title.id);
  panel.appendChild(title);

  const buttons = entries.map((_entry, index) => {
    const button = document.createElement("wuik-button");
    button.className = "pause-menu__item";
    button.setAttribute("tabindex", "0");
    button.addEventListener("click", () => choose(index));
    button.addEventListener("focus", () => highlight(index));
    panel.appendChild(button);
    return button;
  });

  let cursor = 0;
  let shown = false;

  function render(): void {
    title.textContent = t("pause.title", "Pause");
    entries.forEach((entry, index) => {
      buttons[index].textContent = t(entry.key, entry.label);
    });
  }

  function highlight(index: number): void {
    cursor = index;
    buttons.forEach((button, i) => {
      button.toggleAttribute("data-active", i === index);
    });
  }

  function move(delta: number): void {
    const next = (cursor + delta + entries.length) % entries.length;
    highlight(next);
    buttons[next].focus();
  }

  function close(): void {
    shown = false;
    element.setAttribute("hidden", "");
  }

  function choose(index: number): void {
    close();
    entries[index].choose();
  }

  element.addEventListener("keydown", (event) => {
    if (event.key === "ArrowUp") {
      event.preventDefault();
      move(-1);
    } else if (event.key === "ArrowDown") {
      event.preventDefault();
      move(1);
    }
  });

  render();
  const unsubscribe = onLocaleChange(render);

  return {
    element,
    open(): void {
      shown = true;
      element.removeAttribute("hidden");
      highlight(0);
      buttons[0].focus();
    },
    close,
    isOpen: () => shown,
    navigate(nav: PauseNav): void {
      if (!shown) return;
      if (nav === "up") move(-1);
      if (nav === "down") move(1);
      if (nav === "confirm") choose(cursor);
      if (nav === "cancel") choose(0);
    },
    dispose(): void {
      unsubscribe();
    },
  };
}

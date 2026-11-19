import { afterEach, describe, expect, it, vi } from "vitest";
import { initAppI18n } from "../i18n/i18n.ts";
import { createPauseMenu } from "./pause-menu.ts";

function build() {
  const handlers = {
    onContinue: vi.fn(),
    onRestart: vi.fn(),
    onCharacterChange: vi.fn(),
  };
  const menu = createPauseMenu(handlers);
  document.body.appendChild(menu.element);
  return { menu, ...handlers };
}

function items(menu: { element: HTMLElement }): HTMLElement[] {
  return Array.from(menu.element.querySelectorAll(".pause-menu__item"));
}

afterEach(() => {
  document.body.replaceChildren();
});

describe("createPauseMenu", () => {
  it("is a hidden dialog listing Ikemen GO's entries in order", () => {
    const { menu } = build();
    expect(menu.element.hasAttribute("hidden")).toBe(true);
    expect(menu.element.getAttribute("role")).toBe("dialog");
    expect(menu.element.textContent).toContain("Pause");
    expect(items(menu).map((b) => b.textContent)).toEqual([
      "Continue",
      "Restart",
      "Character Change",
    ]);
  });

  it("opens on Continue, focused, and runs the chosen entry once", () => {
    const { menu, onContinue, onRestart } = build();
    menu.open();
    expect(menu.isOpen()).toBe(true);
    expect(document.activeElement).toBe(items(menu)[0]);
    expect(items(menu)[0].hasAttribute("data-active")).toBe(true);
    menu.navigate("down");
    menu.navigate("confirm");
    expect(onRestart).toHaveBeenCalledTimes(1);
    expect(onContinue).not.toHaveBeenCalled();
    expect(menu.isOpen()).toBe(false);
    expect(menu.element.hasAttribute("hidden")).toBe(true);
  });

  it("wraps around with arrow keys and cancel means Continue", () => {
    const { menu, onCharacterChange, onContinue } = build();
    menu.open();
    menu.element.dispatchEvent(
      new KeyboardEvent("keydown", { key: "ArrowUp", bubbles: true }),
    );
    expect(document.activeElement).toBe(items(menu)[2]);
    items(menu)[2].click();
    expect(onCharacterChange).toHaveBeenCalledTimes(1);
    menu.open();
    menu.navigate("cancel");
    expect(onContinue).toHaveBeenCalledTimes(1);
  });

  it("ignores navigation while closed", () => {
    const { menu, onContinue } = build();
    menu.navigate("confirm");
    expect(onContinue).not.toHaveBeenCalled();
  });

  it("re-translates in place when the locale changes", async () => {
    const instance = await initAppI18n();
    await instance.changeLanguage("en");
    const { menu } = build();
    await instance.changeLanguage("fr");
    expect(items(menu)[0].textContent).toBe("Continuer");
    menu.dispose();
    await instance.changeLanguage("en");
    expect(items(menu)[0].textContent).toBe("Continuer");
    window.localStorage.clear();
  });
});

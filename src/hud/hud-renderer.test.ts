import { afterEach, describe, expect, it, vi } from "vitest";
import { initAppI18n } from "../i18n/i18n.ts";
import type { MatchSnapshot } from "../wasm/core-types.ts";
import { createHud } from "./hud-renderer.ts";

interface FighterValues {
  health: number;
  power: number;
  lifeMax?: number;
  powerMax?: number;
}

// Each fighter carries its character's maxima; 1000 life and 3000 power
// unless given.
function snapshot(
  round: number,
  fighters: [FighterValues, FighterValues],
  wins: [number, number] = [0, 0],
) {
  const withMaxima = (f: FighterValues) => ({
    lifeMax: 1000,
    powerMax: 3000,
    ...f,
  });
  return {
    round,
    wins,
    fighters: [withMaxima(fighters[0]), withMaxima(fighters[1])] as [
      MatchSnapshot["fighters"][0],
      MatchSnapshot["fighters"][1],
    ],
  };
}

describe("createHud", () => {
  it("renders both players' health/power bars with the real placeholder ranges on the first update", () => {
    const hud = createHud({ bestOf: 3 });

    hud.update(
      snapshot(
        1,
        [
          { health: 800, power: 450 },
          { health: 300, power: 1500 },
        ],
        [0, 0],
      ),
    );

    const p1Health = hud.element.querySelector(
      '[aria-label="Player 1 health"]',
    );
    const p2Health = hud.element.querySelector(
      '[aria-label="Player 2 health"]',
    );
    const p1Power = hud.element.querySelector('[aria-label="Player 1 power"]');
    const p2Power = hud.element.querySelector('[aria-label="Player 2 power"]');

    expect(p1Health?.getAttribute("role")).toBe("progressbar");
    expect(p1Health?.getAttribute("aria-valuemin")).toBe("0");
    expect(p1Health?.getAttribute("aria-valuemax")).toBe("1000");
    expect(p1Health?.getAttribute("aria-valuenow")).toBe("800");
    expect(p1Health?.getAttribute("aria-valuetext")).toBe("800 / 1000");

    expect(p2Health?.getAttribute("aria-valuenow")).toBe("300");
    expect(p1Power?.getAttribute("aria-valuemax")).toBe("3000");
    expect(p1Power?.getAttribute("aria-valuenow")).toBe("450");
    expect(p2Power?.getAttribute("aria-valuenow")).toBe("1500");
  });

  it("uses each fighter's own max health/power from the snapshot, never a hardcoded 100", () => {
    const hud = createHud({ bestOf: 1 });

    hud.update(
      snapshot(
        1,
        [
          { health: 250, power: 100, lifeMax: 500, powerMax: 200 },
          { health: 500, power: 200, lifeMax: 500, powerMax: 200 },
        ],
        [0, 0],
      ),
    );

    const p1Health = hud.element.querySelector(
      '[aria-label="Player 1 health"]',
    );
    expect(p1Health?.getAttribute("aria-valuemax")).toBe("500");
    const fill = hud.element.querySelector(
      ".hud__health-fill--1",
    ) as HTMLElement;
    expect(fill.style.width).toBe("50%");
  });

  it("reads a full bar at round start for a character whose life is not 1000, and fills power against its own power", () => {
    const hud = createHud({ bestOf: 1 });

    hud.update(
      snapshot(1, [
        { health: 1200, power: 3000, lifeMax: 1200, powerMax: 5000 },
        { health: 800, power: 0, lifeMax: 800, powerMax: 3000 },
      ]),
    );

    const fill = (selector: string) =>
      (hud.element.querySelector(selector) as HTMLElement).style.width;
    expect(fill(".hud__health-fill--1")).toBe("100%");
    expect(fill(".hud__health-fill--2")).toBe("100%");
    expect(fill(".hud__power-fill--1")).toBe("60%");
    expect(
      hud.element
        .querySelector('[aria-label="Player 1 health"]')
        ?.getAttribute("aria-valuemax"),
    ).toBe("1200");
    expect(
      hud.element
        .querySelector('[aria-label="Player 1 power"]')
        ?.getAttribute("aria-valuemax"),
    ).toBe("5000");
  });

  it("shows the current round number, each player's win count, and the best-of setting", () => {
    const hud = createHud({ bestOf: 3 });

    hud.update(
      snapshot(
        2,
        [
          { health: 1000, power: 0 },
          { health: 1000, power: 0 },
        ],
        [1, 0],
      ),
    );

    const roundBlock = hud.element.querySelector(".hud__round");
    expect(roundBlock?.textContent).toContain("2");
    expect(roundBlock?.textContent).toContain("3");
    const wins = hud.element.querySelectorAll(".hud__wins");
    expect(wins[0].textContent).toContain("1");
    expect(wins[1].textContent).toContain("0");
  });

  it("announces a round/win change once via aria-live, but never on an ordinary health/power-only update", () => {
    const hud = createHud({ bestOf: 3 });
    const announcement = hud.element.querySelector(
      "[aria-live]",
    ) as HTMLElement;
    let setCount = 0;
    const descriptor = Object.getOwnPropertyDescriptor(
      Node.prototype,
      "textContent",
    );
    if (!descriptor?.get || !descriptor?.set) {
      throw new Error("expected Node.prototype.textContent accessors");
    }
    Object.defineProperty(announcement, "textContent", {
      get() {
        return descriptor.get?.call(announcement);
      },
      set(value: string | null) {
        setCount++;
        descriptor.set?.call(announcement, value);
      },
    });

    hud.update(
      snapshot(
        1,
        [
          { health: 1000, power: 0 },
          { health: 1000, power: 0 },
        ],
        [0, 0],
      ),
    );
    expect(setCount).toBe(1);

    // Health-only change, same round/wins: no new announcement.
    hud.update(
      snapshot(
        1,
        [
          { health: 900, power: 0 },
          { health: 1000, power: 0 },
        ],
        [0, 0],
      ),
    );
    expect(setCount).toBe(1);

    // Wins change: a new announcement fires.
    hud.update(
      snapshot(
        2,
        [
          { health: 1000, power: 0 },
          { health: 1000, power: 0 },
        ],
        [1, 0],
      ),
    );
    expect(setCount).toBe(2);
  });

  it("fills a bar to 100% at most and 0% at least, whatever value the snapshot carries", () => {
    const hud = createHud({ bestOf: 3 });

    hud.update(
      snapshot(1, [
        { health: 1500, power: 9000 },
        { health: 0, power: 0 },
      ]),
    );

    const fill = (selector: string) =>
      (hud.element.querySelector(selector) as HTMLElement).style.width;
    expect(fill(".hud__health-fill--1")).toBe("100%");
    expect(fill(".hud__power-fill--1")).toBe("100%");
    expect(fill(".hud__health-fill--2")).toBe("0%");
  });

  it("fills a bar in proportion to the value over the max", () => {
    const hud = createHud({ bestOf: 3 });

    hud.update(
      snapshot(1, [
        { health: 250, power: 750 },
        { health: 1000, power: 3000 },
      ]),
    );

    const fill = (selector: string) =>
      (hud.element.querySelector(selector) as HTMLElement).style.width;
    expect(fill(".hud__health-fill--1")).toBe("25%");
    expect(fill(".hud__power-fill--1")).toBe("25%");
    expect(fill(".hud__health-fill--2")).toBe("100%");
  });

  describe("live locale switching (backlog item 009)", () => {
    afterEach(async () => {
      const instance = await initAppI18n();
      await instance.changeLanguage("en");
      window.localStorage.clear();
    });

    it("re-translates the round label in place without losing bar values", async () => {
      const instance = await initAppI18n();
      await instance.changeLanguage("en");
      const hud = createHud({ bestOf: 3 });
      hud.update(
        snapshot(
          2,
          [
            { health: 700, power: 0 },
            { health: 900, power: 0 },
          ],
          [1, 0],
        ),
      );
      const englishLabel =
        hud.element.querySelector(".hud__round")?.textContent;

      await instance.changeLanguage("fr");
      await vi.waitFor(() => {
        expect(hud.element.querySelector(".hud__round")?.textContent).not.toBe(
          englishLabel,
        );
      });
      const frenchLabel = hud.element.querySelector(".hud__round")?.textContent;
      expect(frenchLabel).toContain("2");
      expect(frenchLabel).toContain("3");
      const p1Health = hud.element.querySelector(".hud__health--1");
      expect(p1Health?.getAttribute("aria-valuenow")).toBe("700");
      hud.dispose();
    });

    it("stops re-translating after dispose()", async () => {
      const instance = await initAppI18n();
      await instance.changeLanguage("en");
      const hud = createHud({ bestOf: 1 });
      hud.update(
        snapshot(
          1,
          [
            { health: 1000, power: 0 },
            { health: 1000, power: 0 },
          ],
          [0, 0],
        ),
      );
      const before = hud.element.querySelector(".hud__round")?.textContent;

      hud.dispose();
      await instance.changeLanguage("fr");
      await new Promise((resolve) => setTimeout(resolve, 10));

      expect(hud.element.querySelector(".hud__round")?.textContent).toBe(
        before,
      );
    });
  });
});

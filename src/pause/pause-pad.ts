// Reads the pause-related gamepad buttons once per frame, on press only:
// Start (standard-mapping button 9) opens and closes the pause menu, as
// Ikemen GO's menu.cancel.key; the d-pad, A (0) and B (1) drive the menu.
import type { PauseNav } from "./pause-menu.ts";

const START = 9;
const NAV_BUTTONS: readonly [number, PauseNav][] = [
  [12, "up"],
  [13, "down"],
  [0, "confirm"],
  [1, "cancel"],
];
const WATCHED = [START, ...NAV_BUTTONS.map(([index]) => index)];

export interface PausePadEvents {
  start: boolean;
  nav: PauseNav | null;
}

export interface PausePad {
  /** What was newly pressed on any connected pad since the previous poll. */
  poll(): PausePadEvents;
}

function defaultGetGamepads(): (Gamepad | null)[] {
  const fn = (navigator as Partial<Navigator>).getGamepads;
  if (typeof fn !== "function") return [];
  const result = fn.call(navigator);
  if (!result) return [];
  return Array.from(result);
}

export function createPausePad(
  getGamepads: () => (Gamepad | null)[] = defaultGetGamepads,
): PausePad {
  const held = new Map<string, boolean>();

  function pressedNow(pad: Gamepad, index: number): boolean {
    const key = `${pad.index}:${index}`;
    const down = pad.buttons[index]?.pressed === true;
    const was = held.get(key) === true;
    held.set(key, down);
    return down && !was;
  }

  return {
    poll(): PausePadEvents {
      const events: PausePadEvents = { start: false, nav: null };
      for (const pad of getGamepads()) {
        if (!pad || pad.connected === false) continue;
        const fresh = new Set(
          WATCHED.filter((index) => pressedNow(pad, index)),
        );
        if (fresh.has(START)) events.start = true;
        for (const [index, nav] of NAV_BUTTONS) {
          if (fresh.has(index) && events.nav === null) events.nav = nav;
        }
      }
      return events;
    },
  };
}

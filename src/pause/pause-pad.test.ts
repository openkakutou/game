import { describe, expect, it } from "vitest";
import { createPausePad } from "./pause-pad.ts";

function pad(pressed: number[], index = 0): Gamepad {
  const buttons = Array.from({ length: 17 }, (_, i) => ({
    pressed: pressed.includes(i),
  }));
  return { index, connected: true, buttons } as unknown as Gamepad;
}

describe("createPausePad", () => {
  it("reports Start on press only, not while held", () => {
    let pads: (Gamepad | null)[] = [pad([9])];
    const p = createPausePad(() => pads);
    expect(p.poll().start).toBe(true);
    expect(p.poll().start).toBe(false);
    pads = [pad([])];
    p.poll();
    pads = [pad([9])];
    expect(p.poll().start).toBe(true);
  });

  it("maps the d-pad, A and B to menu inputs from any pad", () => {
    let pads: (Gamepad | null)[] = [null, pad([13], 1)];
    const p = createPausePad(() => pads);
    expect(p.poll().nav).toBe("down");
    pads = [pad([0]), pad([13], 1)];
    expect(p.poll().nav).toBe("confirm");
    pads = [pad([1]), pad([], 1)];
    expect(p.poll().nav).toBe("cancel");
  });

  it("reports nothing without pads", () => {
    const p = createPausePad(() => []);
    expect(p.poll()).toEqual({ start: false, nav: null });
  });
});

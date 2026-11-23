import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  closeMatch,
  mountFiles,
  nextRound,
  resetCoreBridgeForTests,
  startMatch,
  tick,
} from "./core-bridge.ts";
import type { StartMatchRequest } from "./core-types.ts";

// A fake core: the Go runtime glue is replaced by a tiny script whose
// `Go.run` publishes `OpenKakutouCore`, so the bridge's own loading,
// envelope parsing and validation run for real against replies the test
// controls. The real module is exercised by `core-bridge.real.test.ts`.
const emptyWasmModule = new Uint8Array([0, 0x61, 0x73, 0x6d, 1, 0, 0, 0]);

const fakeGlue = `
globalThis.Go = class {
  constructor() { this.importObject = {}; }
  async run() { globalThis.OpenKakutouCore = globalThis.__fakeCore; }
};`;

interface FakeCore {
  mount: ReturnType<typeof vi.fn>;
  startMatch: ReturnType<typeof vi.fn>;
  tick: ReturnType<typeof vi.fn>;
  nextRound: ReturnType<typeof vi.fn>;
  closeMatch: ReturnType<typeof vi.fn>;
}

function installFakeCore(): FakeCore {
  const fake: FakeCore = {
    mount: vi.fn(() => '{"ok":true}'),
    startMatch: vi.fn(),
    tick: vi.fn(),
    nextRound: vi.fn(),
    closeMatch: vi.fn(() => '{"ok":true}'),
  };
  (globalThis as unknown as { __fakeCore: FakeCore }).__fakeCore = fake;
  return fake;
}

const loadOptions = {
  fetchWasmExecSource: async () => fakeGlue,
  fetchWasmBytes: async () => emptyWasmModule,
};

function snapshotJSON(overrides: Record<string, unknown> = {}): string {
  return JSON.stringify({
    ok: true,
    tick: 3,
    round: 1,
    timer: 5997,
    roundEnded: false,
    outcome: "none",
    over: false,
    winner: -1,
    roundWinner: -1,
    wins: [0, 0],
    camX: 0,
    fighters: [
      {
        x: -70,
        y: 0,
        facing: 1,
        anim: 0,
        animTime: 3,
        animOwner: 0,
        stateNo: 0,
        health: 1000,
        power: 0,
      },
      {
        x: 70,
        y: 0,
        facing: -1,
        anim: 0,
        animTime: 3,
        animOwner: 1,
        stateNo: 0,
        health: 900,
        power: 150,
      },
    ],
    cues: [{ side: 1, group: 5, sample: 0 }],
    ...overrides,
  });
}

const request: StartMatchRequest = {
  p1: "chars/a/a.def",
  p2: "chars/b/b.def",
  stage: "stages/s.def",
  setup: { rounds: 3, timeLimitSeconds: 99 },
  p2cpu: false,
  viewWidth: 400,
};

let core: FakeCore;
beforeEach(() => {
  resetCoreBridgeForTests();
  core = installFakeCore();
});

describe("startMatch", () => {
  it("sends the request as JSON and returns the id, the HUD values and the first snapshot", async () => {
    core.startMatch.mockReturnValue(
      snapshotJSON({ id: 7, roundTimerTicks: 5940, roundsToWin: 2 }),
    );

    const result = await startMatch(request, loadOptions);

    expect(JSON.parse(core.startMatch.mock.calls[0][0])).toEqual(request);
    expect(result.ok).toBe(true);
    if (!result.ok) return;
    expect(result.data.id).toBe(7);
    expect(result.data.roundsToWin).toBe(2);
    expect(result.data.roundTimerTicks).toBe(5940);
    expect(result.data.fighters[1].health).toBe(900);
    expect(result.data.cues).toEqual([{ side: 1, group: 5, sample: 0 }]);
  });

  it("returns the core's reason as the error when the core refuses the match", async () => {
    core.startMatch.mockReturnValue(
      '{"ok":false,"error":{"reason":"cannot load character \\"x\\": missing"}}',
    );

    const result = await startMatch(request, loadOptions);

    expect(result).toEqual({
      ok: false,
      error: 'cannot load character "x": missing',
    });
  });

  it("returns an error naming the missing field when the start reply carries no match id", async () => {
    core.startMatch.mockReturnValue(
      snapshotJSON({ roundTimerTicks: 5940, roundsToWin: 2 }),
    );

    const result = await startMatch(request, loadOptions);

    expect(result.ok).toBe(false);
    if (result.ok) return;
    expect(result.error).toContain("id");
  });
});

describe("tick", () => {
  it("sends the match id and both players' inputs and returns the snapshot", async () => {
    core.tick.mockReturnValue(snapshotJSON());

    const result = await tick(
      4,
      { p1: { right: true }, p2: { buttons: { a: true } } },
      loadOptions,
    );

    expect(core.tick.mock.calls[0][0]).toBe(4);
    expect(JSON.parse(core.tick.mock.calls[0][1])).toEqual({
      p1: { right: true },
      p2: { buttons: { a: true } },
    });
    expect(result.ok).toBe(true);
    if (!result.ok) return;
    expect(result.data.tick).toBe(3);
    expect(result.data.fighters[0].x).toBe(-70);
    expect(result.data.fighters[1].facing).toBe(-1);
  });

  it("reports an ended round with its outcome, winner and the match result", async () => {
    core.tick.mockReturnValue(
      snapshotJSON({
        roundEnded: true,
        outcome: "ko",
        roundWinner: 0,
        over: true,
        winner: 0,
        wins: [2, 1],
      }),
    );

    const result = await tick(4, { p1: {}, p2: {} }, loadOptions);

    expect(result.ok).toBe(true);
    if (!result.ok) return;
    expect(result.data.outcome).toBe("ko");
    expect(result.data.roundWinner).toBe(0);
    expect(result.data.over).toBe(true);
    expect(result.data.winner).toBe(0);
    expect(result.data.wins).toEqual([2, 1]);
  });

  it.each([
    ["a fighter's health is not a number", { fighters: [{ x: 0 }, { x: 0 }] }],
    ["there is only one fighter", { fighters: [] }],
    ["the outcome is not one the core defines", { outcome: "flawless" }],
    ["the wins are not a pair of numbers", { wins: [1] }],
    ["the camera position is missing", { camX: undefined }],
    [
      "a cue names a side that does not exist",
      { cues: [{ side: 2, group: 0, sample: 0 }] },
    ],
    ["the winner is not a side or -1", { winner: 5 }],
  ])(
    "returns an error instead of a snapshot when %s",
    async (_name, overrides) => {
      core.tick.mockReturnValue(snapshotJSON(overrides));

      const result = await tick(4, { p1: {}, p2: {} }, loadOptions);

      expect(result.ok).toBe(false);
    },
  );

  it("returns the side each fighter's animation is drawn from, and refuses a snapshot without one of 0 or 1", async () => {
    const valid = JSON.parse(snapshotJSON());
    valid.fighters[1].animOwner = 0;
    core.tick.mockReturnValue(JSON.stringify(valid));
    const thrown = await tick(4, { p1: {}, p2: {} }, loadOptions);
    expect(thrown.ok).toBe(true);
    if (!thrown.ok) return;
    expect(thrown.data.fighters[1].animOwner).toBe(0);

    for (const animOwner of [undefined, 2, -1]) {
      const bad = JSON.parse(snapshotJSON());
      bad.fighters[1].animOwner = animOwner;
      core.tick.mockReturnValue(JSON.stringify(bad));
      const refused = await tick(4, { p1: {}, p2: {} }, loadOptions);
      expect(refused.ok).toBe(false);
      if (refused.ok) return;
      expect(refused.error).toContain("animOwner");
    }
  });

  it("returns an error when the core replies with something that is not JSON", async () => {
    core.tick.mockReturnValue("<html>");

    const result = await tick(4, { p1: {}, p2: {} }, loadOptions);

    expect(result.ok).toBe(false);
    if (result.ok) return;
    expect(result.error).toContain("malformed JSON");
  });

  it("returns an error instead of throwing when the core call itself throws", async () => {
    core.tick.mockImplementation(() => {
      throw new Error("runtime exited");
    });

    const result = await tick(4, { p1: {}, p2: {} }, loadOptions);

    expect(result).toEqual({
      ok: false,
      error: "game core tick failed: runtime exited",
    });
  });
});

describe("nextRound and closeMatch", () => {
  it("nextRound returns the first snapshot of the next round", async () => {
    core.nextRound.mockReturnValue(snapshotJSON({ round: 2, tick: 0 }));

    const result = await nextRound(4, loadOptions);

    expect(core.nextRound).toHaveBeenCalledWith(4);
    expect(result.ok).toBe(true);
    if (!result.ok) return;
    expect(result.data.round).toBe(2);
  });

  it("nextRound returns the core's reason when the round is not over", async () => {
    core.nextRound.mockReturnValue(
      '{"ok":false,"error":{"reason":"the round is not over"}}',
    );

    const result = await nextRound(4, loadOptions);

    expect(result).toEqual({ ok: false, error: "the round is not over" });
  });

  it("closeMatch asks the core to forget the match", async () => {
    const result = await closeMatch(4, loadOptions);

    expect(core.closeMatch).toHaveBeenCalledWith(4);
    expect(result.ok).toBe(true);
  });
});

describe("mountFiles", () => {
  it("mounts each file under its name with its bytes", async () => {
    const bytes = new Uint8Array([1, 2, 3]);

    const result = await mountFiles(
      [{ name: "chars/a/a.def", bytes }],
      loadOptions,
    );

    expect(result.ok).toBe(true);
    expect(core.mount).toHaveBeenCalledWith("chars/a/a.def", bytes);
  });

  it("mounts a name only once per page, even when asked again", async () => {
    const file = { name: "chars/a/a.sff", bytes: new Uint8Array([9]) };

    await mountFiles([file], loadOptions);
    await mountFiles(
      [file, { name: "chars/a/a.air", bytes: file.bytes }],
      loadOptions,
    );

    expect(core.mount.mock.calls.map((c) => c[0])).toEqual([
      "chars/a/a.sff",
      "chars/a/a.air",
    ]);
  });

  it("stops at the first file the core refuses and returns its reason", async () => {
    core.mount.mockReturnValueOnce(
      '{"ok":false,"error":{"reason":"invalid file name"}}',
    );

    const result = await mountFiles(
      [
        { name: "../x", bytes: new Uint8Array() },
        { name: "ok.def", bytes: new Uint8Array() },
      ],
      loadOptions,
    );

    expect(result).toEqual({ ok: false, error: "invalid file name" });
    expect(core.mount).toHaveBeenCalledTimes(1);
  });

  it("mounts a refused name again on a later call, since it never reached the core's file system", async () => {
    core.mount.mockReturnValueOnce('{"ok":false,"error":{"reason":"nope"}}');
    const file = { name: "a.def", bytes: new Uint8Array() };

    await mountFiles([file], loadOptions);
    const second = await mountFiles([file], loadOptions);

    expect(second.ok).toBe(true);
    expect(core.mount).toHaveBeenCalledTimes(2);
  });
});

describe("loading the core", () => {
  it("returns a clear error instead of rejecting when the WASM assets cannot be fetched", async () => {
    const result = await tick(
      1,
      { p1: {}, p2: {} },
      {
        fetchWasmExecSource: async () => {
          throw new Error(
            "failed to fetch ./wasm/core-wasm_exec.js: 404 Not Found",
          );
        },
        fetchWasmBytes: async () => emptyWasmModule,
      },
    );

    expect(result).toEqual({
      ok: false,
      error:
        "game core WASM module failed to load: failed to fetch ./wasm/core-wasm_exec.js: 404 Not Found",
    });
  });

  it("loads the module once for any number of calls", async () => {
    core.tick.mockReturnValue(snapshotJSON());
    const fetchWasmBytes = vi.fn(async () => emptyWasmModule);

    await tick(1, { p1: {}, p2: {} }, { ...loadOptions, fetchWasmBytes });
    await tick(1, { p1: {}, p2: {} }, { ...loadOptions, fetchWasmBytes });

    expect(fetchWasmBytes).toHaveBeenCalledTimes(1);
  });
});

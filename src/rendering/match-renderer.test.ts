import { beforeEach, describe, expect, it, vi } from "vitest";
import type { TickInputPair } from "../input/tick-input-source.ts";
import type { PausePadEvents } from "../pause/pause-pad.ts";
import type { ResultOverlayOptions } from "../result/result-screen.ts";
import type {
  CoreResult,
  MatchSnapshot,
  StartMatchData,
} from "../wasm/core-types.ts";
import type { StageSummary } from "../wasm/stage-types.ts";
import type { CharacterSummary } from "../wasm/types.ts";
import { renderMatch } from "./match-renderer.ts";
import type {
  MatchRendererInput,
  MatchRendererOptions,
} from "./match-renderer.ts";
import type { DrawCommand } from "./scene-composition.ts";

function character(): CharacterSummary {
  return {
    name: "Fighter",
    animations: [
      {
        number: 0,
        frames: [
          {
            group: 0,
            image: 0,
            x: 0,
            y: 0,
            time: -1,
            flip: "",
            blend: "",
            clsn1: [],
            clsn2: [],
          },
        ],
        loopStart: 0,
      },
    ],
    sprites: [
      {
        index: 0,
        sprites: [
          {
            group: 0,
            image: 0,
            width: 40,
            height: 80,
            axisX: 20,
            axisY: 79,
            palette: 0,
          },
        ],
      },
    ],
    stateDefs: [{ number: 0, type: "S" }],
    sounds: [],
  };
}

function stage(): StageSummary {
  return {
    name: "Stage",
    bgDef: {
      spriteFile: "",
      localCoordWidth: 400,
      localCoordHeight: 240,
      zOffset: 200,
      zoomOut: 1,
      zoomIn: 1,
      modelFile: "",
      xScale: 1,
      yScale: 1,
    },
    elements: [],
    animations: {},
    stageBoundaries: { left: -200, right: 200, topBound: 0, bottomBound: 0 },
    musicFile: "",
  };
}

function snapshot(overrides: Partial<MatchSnapshot> = {}): MatchSnapshot {
  return {
    tick: 0,
    round: 1,
    timer: 5940,
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
        animTime: 0,
        stateNo: 0,
        health: 1000,
        power: 0,
      },
      {
        x: 70,
        y: 0,
        facing: -1,
        anim: 0,
        animTime: 0,
        stateNo: 0,
        health: 1000,
        power: 0,
      },
    ],
    cues: [],
    ...overrides,
  };
}

function startResponse(
  id = 1,
  overrides: Partial<MatchSnapshot> = {},
): CoreResult<StartMatchData> {
  return {
    ok: true,
    data: { id, roundTimerTicks: 5940, roundsToWin: 2, ...snapshot(overrides) },
  };
}

function tickResponse(
  overrides: Partial<MatchSnapshot> = {},
): CoreResult<MatchSnapshot> {
  return { ok: true, data: snapshot({ tick: 1, ...overrides }) };
}

/** A tick that ends the round: `winner` took it. */
function roundEndResponse(
  winner: 0 | 1,
  overrides: Partial<MatchSnapshot> = {},
): CoreResult<MatchSnapshot> {
  return tickResponse({
    roundEnded: true,
    outcome: "ko",
    roundWinner: winner,
    ...overrides,
  });
}

function nextRoundResponse(): CoreResult<MatchSnapshot> {
  return { ok: true, data: snapshot({ round: 2 }) };
}

function baseInput(
  overrides: Partial<MatchRendererInput> = {},
): MatchRendererInput {
  return {
    player1: {
      character: character(),
      sffBytes: new Uint8Array([1]),
      defName: "chars/a/a.def",
    },
    player2: {
      character: character(),
      sffBytes: new Uint8Array([2]),
      defName: "chars/b/b.def",
    },
    stage: {
      stage: stage(),
      sffBytes: new Uint8Array([3]),
      defName: "stages/s.def",
    },
    config: { rounds: 3, timeLimit: { seconds: 99 } },
    onBackToSelect: vi.fn(),
    ...overrides,
  };
}

/**
 * A fake result overlay double: instead of a real DOM/timer-driven overlay
 * (`result/result-screen.ts`), this exposes the exact `ResultOverlayOptions`
 * `renderMatch` constructed it with, so a test can drive
 * `onRoundResultDone`/`onRematch`/`onBackToSelect` directly and assert on
 * `showRoundResult`/`showMatchResult`'s own call arguments -- without
 * depending on the real overlay's countdown timer at all (already covered
 * by `result/result-screen.test.ts`).
 */
function fakeResultOverlay() {
  const overlay = {
    element: document.createElement("div"),
    showRoundResult: vi.fn(),
    showMatchResult: vi.fn(),
    hide: vi.fn(),
    dispose: vi.fn(),
  };
  let capturedOptions: ResultOverlayOptions | null = null;
  const createResultOverlay = vi.fn((options: ResultOverlayOptions) => {
    capturedOptions = options;
    return overlay;
  });
  return {
    overlay,
    createResultOverlay,
    options: () => {
      if (!capturedOptions) throw new Error("result overlay not created yet");
      return capturedOptions;
    },
  };
}

/**
 * A fake match audio double: instead of a real `AudioContext`/Web Audio API
 * driver (`audio/match-audio.ts`), this exposes plain `vi.fn()` spies for
 * `playMusic`/`playSound`/`stop` -- already covered for real by
 * `audio/match-audio.test.ts` -- so this file only has to assert on *what*
 * `renderMatch` asked the audio controller to do, never the underlying
 * `AudioContext` wiring itself.
 */
function fakeMatchAudio() {
  const audio = {
    playMusic: vi.fn(async () => {}),
    playSound: vi.fn(),
    setPaused: vi.fn(),
    stop: vi.fn(),
  };
  const createAudio = vi.fn(() => audio);
  return { audio, createAudio };
}

function baseOptions(
  overrides: Partial<MatchRendererOptions> = {},
): MatchRendererOptions & {
  rafCallbacks: ((ts: number) => void)[];
  resultOverlay: ReturnType<typeof fakeResultOverlay>;
  matchAudio: ReturnType<typeof fakeMatchAudio>;
  keys: EventTarget;
  padEvents: PausePadEvents[];
} {
  const rafCallbacks: ((ts: number) => void)[] = [];
  const resultOverlay = fakeResultOverlay();
  const matchAudio = fakeMatchAudio();
  const keys = new EventTarget();
  const padEvents: PausePadEvents[] = [];
  return {
    keyTarget: keys,
    createPausePad: () => ({
      poll: () => padEvents.shift() ?? { start: false, nav: null },
    }),
    keys,
    padEvents,
    startMatch: vi.fn(async () => startResponse()),
    tick: vi.fn(async () => tickResponse()),
    nextRound: vi.fn(async () => nextRoundResponse()),
    closeMatch: vi.fn(async () => ({ ok: true as const, data: {} })),
    createResultOverlay: resultOverlay.createResultOverlay,
    createAudio: matchAudio.createAudio,
    resolveCharacterSprites: vi.fn(async (_sff, requests) =>
      requests.map(() => ({
        ok: true as const,
        pixels: new Uint8Array(40 * 80 * 4),
        width: 40,
        height: 80,
      })),
    ),
    resolveStageSprites: vi.fn(async () => []),
    resolveAnimationFrames: vi.fn(async () => ({
      ok: true as const,
      sprites: [],
    })),
    drawScene: vi.fn(),
    requestAnimationFrame: vi.fn((cb: (ts: number) => void) => {
      rafCallbacks.push(cb);
      return rafCallbacks.length;
    }),
    cancelAnimationFrame: vi.fn(),
    now: vi.fn(() => 0),
    rafCallbacks,
    resultOverlay,
    matchAudio,
    ...overrides,
  };
}

beforeEach(() => {
  vi.restoreAllMocks();
});

describe("renderMatch", () => {
  it("shows a preparing status synchronously, before any async work resolves", () => {
    const root = document.createElement("div");
    const options = baseOptions();

    void renderMatch(root, baseInput(), options);

    expect(root.textContent).toContain("Preparing match");
  });

  it("mounts a canvas sized to the stage's local coordinate dimensions once ready, and announces the match has started", async () => {
    const root = document.createElement("div");
    const options = baseOptions();

    await renderMatch(root, baseInput(), options);

    const canvas = root.querySelector("canvas");
    expect(canvas).not.toBeNull();
    expect(canvas?.width).toBe(400);
    expect(canvas?.height).toBe(240);
    const liveRegion = root.querySelector(".match-renderer__announcement");
    expect(liveRegion?.textContent).toContain("Match started");
  });

  it("draws the initial frame before the tick loop starts", async () => {
    const root = document.createElement("div");
    const options = baseOptions();

    await renderMatch(root, baseInput(), options);

    expect(options.drawScene).toHaveBeenCalledTimes(1);
  });

  it("shows a clear error status instead of a blank/broken screen when the core refuses to start the match", async () => {
    const root = document.createElement("div");
    const options = baseOptions({
      startMatch: vi.fn(async () => ({ ok: false as const, error: "boom" })),
    });

    await renderMatch(root, baseInput(), options);

    expect(root.querySelector("canvas")).toBeNull();
    expect(root.textContent).toContain("boom");
  });

  it("degrades to a placeholder draw command instead of crashing when a fighter's sprite fails to resolve", async () => {
    const root = document.createElement("div");
    const options = baseOptions({
      resolveCharacterSprites: vi.fn(async (_sff, requests) =>
        requests.map(() => ({ ok: false as const, error: "sprite not found" })),
      ),
    });

    await renderMatch(root, baseInput(), options);

    const plan = (options.drawScene as ReturnType<typeof vi.fn>).mock
      .calls[0][1] as DrawCommand[];
    expect(plan.some((c) => c.kind === "placeholder")).toBe(true);
  });

  it("advances the simulation and redraws once a full tick interval has elapsed", async () => {
    const root = document.createElement("div");
    let currentTime = 0;
    const options = baseOptions({ now: vi.fn(() => currentTime) });

    const handle = await renderMatch(root, baseInput(), options);
    expect(options.rafCallbacks).toHaveLength(1);

    currentTime = 1000 / 60 + 1; // just over one tick interval
    options.rafCallbacks[0](currentTime);
    await vi.waitFor(() => {
      expect(options.drawScene).toHaveBeenCalledTimes(2); // initial + this tick
    });
    expect(options.tick).toHaveBeenCalledTimes(1);
    handle.stop();
  });

  it("stop() cancels the loop and releases the core match", async () => {
    const root = document.createElement("div");
    const options = baseOptions();

    const handle = await renderMatch(root, baseInput(), options);
    handle.stop();

    expect(options.cancelAnimationFrame).toHaveBeenCalled();
    await vi.waitFor(() => {
      expect(options.closeMatch).toHaveBeenCalledWith(1);
    });
  });

  it("a stray already-scheduled frame does nothing after stop()", async () => {
    const root = document.createElement("div");
    let currentTime = 0;
    const options = baseOptions({ now: vi.fn(() => currentTime) });

    const handle = await renderMatch(root, baseInput(), options);
    handle.stop();
    (options.tick as ReturnType<typeof vi.fn>).mockClear();

    currentTime = 1000; // plenty of elapsed time
    options.rafCallbacks[0]?.(currentTime);
    await Promise.resolve();

    expect(options.tick).not.toHaveBeenCalled();
  });

  it("stops a previous render loop on the same root before starting a new one", async () => {
    const root = document.createElement("div");
    const first = baseOptions();
    const firstHandle = await renderMatch(root, baseInput(), first);
    void firstHandle;

    const second = baseOptions();
    await renderMatch(root, baseInput(), second);

    expect(first.cancelAnimationFrame).toHaveBeenCalled();
  });

  it("starts the core match from each player's own .def, the stage's .def, the setup and the stage width as the camera's view", async () => {
    const root = document.createElement("div");
    const options = baseOptions();

    await renderMatch(root, baseInput(), options);

    expect(options.startMatch).toHaveBeenCalledWith({
      p1: "chars/a/a.def",
      p2: "chars/b/b.def",
      stage: "stages/s.def",
      setup: { rounds: 3, timeLimitSeconds: 99 },
      p2cpu: false,
      viewWidth: 400,
    });
  });

  it("asks for no time limit (0 seconds) when the setup chose unlimited", async () => {
    const root = document.createElement("div");
    const options = baseOptions();

    await renderMatch(
      root,
      baseInput({ config: { rounds: 5, timeLimit: "unlimited" } }),
      options,
    );

    const request = (options.startMatch as ReturnType<typeof vi.fn>).mock
      .calls[0][0];
    expect(request.setup).toEqual({ rounds: 5, timeLimitSeconds: 0 });
  });

  function fakeInputSource(inputs: TickInputPair) {
    return {
      read: vi.fn(() => inputs),
      dispose: vi.fn(),
    };
  }

  it("reads live input once per frame and threads it into every tick() call run that frame, even across a multi-tick catch-up burst", async () => {
    const root = document.createElement("div");
    let currentTime = 0;
    const inputs: TickInputPair = [
      {
        up: true,
        down: false,
        left: false,
        right: false,
        buttons: { a: true, b: false, c: false, x: false, y: false, z: false },
      },
      {
        up: false,
        down: false,
        left: false,
        right: false,
        buttons: { a: false, b: false, c: false, x: false, y: false, z: false },
      },
    ];
    const inputSource = fakeInputSource(inputs);
    const options = baseOptions({
      now: vi.fn(() => currentTime),
      createInputSource: vi.fn(() => inputSource),
    });

    await renderMatch(root, baseInput(), options);
    // 3 full tick intervals elapsed at once -> a multi-tick catch-up burst.
    currentTime = (1000 / 60) * 3 + 1;
    options.rafCallbacks[0](currentTime);
    await vi.waitFor(() => {
      expect(options.tick).toHaveBeenCalledTimes(3);
    });

    expect(inputSource.read).toHaveBeenCalledTimes(1);
    for (const call of (options.tick as ReturnType<typeof vi.fn>).mock.calls) {
      expect(call[0]).toBe(1);
      expect(call[1].p1).toBe(inputs[0]);
      expect(call[1].p2).toBe(inputs[1]);
    }
  });

  it("starts the stage's background music once the match begins (backlog item 013)", async () => {
    const root = document.createElement("div");
    const musicBytes = new Uint8Array([1, 2, 3]);
    const options = baseOptions();
    const input = baseInput();
    input.stage.musicBytes = musicBytes;

    await renderMatch(root, input, options);

    expect(options.matchAudio.audio.playMusic).toHaveBeenCalledWith(musicBytes);
  });

  it("starts the match with no background music when the stage declares none", async () => {
    const root = document.createElement("div");
    const options = baseOptions();

    await renderMatch(root, baseInput(), options);

    expect(options.matchAudio.audio.playMusic).toHaveBeenCalledWith(null);
  });

  it("plays a fighter's decoded sound the moment the core reports its cue this tick (backlog item 013)", async () => {
    const root = document.createElement("div");
    const hitSound = {
      group: 1,
      sample: 0,
      sampleRate: 11025,
      channels: 1,
      bitsPerSample: 16,
      pcm: [0, 0],
    };
    const input = baseInput();
    input.player1.character = {
      ...character(),
      sounds: [{ index: 1, sounds: [hitSound] }],
    };
    let currentTime = 0;
    const options = baseOptions({
      now: vi.fn(() => currentTime),
      tick: vi.fn(async () =>
        tickResponse({ cues: [{ side: 0, group: 1, sample: 0 }] }),
      ),
    });

    await renderMatch(root, input, options);
    currentTime = 1000 / 60 + 1;
    options.rafCallbacks[0](currentTime);

    await vi.waitFor(() => {
      expect(options.matchAudio.audio.playSound).toHaveBeenCalledWith(hitSound);
    });
  });

  it("plays a cue with the sounds of the side that triggered it, never the other fighter's", async () => {
    const root = document.createElement("div");
    const sound = (pcm: number[]) => ({
      group: 1,
      sample: 0,
      sampleRate: 11025,
      channels: 1,
      bitsPerSample: 16,
      pcm,
    });
    const p1Sound = sound([1, 1]);
    const p2Sound = sound([2, 2]);
    const input = baseInput();
    input.player1.character = {
      ...character(),
      sounds: [{ index: 1, sounds: [p1Sound] }],
    };
    input.player2.character = {
      ...character(),
      sounds: [{ index: 1, sounds: [p2Sound] }],
    };
    let currentTime = 0;
    const options = baseOptions({
      now: vi.fn(() => currentTime),
      tick: vi.fn(async () =>
        tickResponse({ cues: [{ side: 1, group: 1, sample: 0 }] }),
      ),
    });

    await renderMatch(root, input, options);
    currentTime = 1000 / 60 + 1;
    options.rafCallbacks[0](currentTime);

    await vi.waitFor(() => {
      expect(options.matchAudio.audio.playSound).toHaveBeenCalledTimes(1);
    });
    expect(options.matchAudio.audio.playSound).toHaveBeenCalledWith(p2Sound);
  });

  it("skips a cue silently when no decoded sample matches its (group, sample)", async () => {
    const root = document.createElement("div");
    let currentTime = 0;
    const options = baseOptions({
      now: vi.fn(() => currentTime),
      tick: vi.fn(async () =>
        tickResponse({ cues: [{ side: 0, group: 9, sample: 9 }] }),
      ),
    });

    await renderMatch(root, baseInput(), options);
    currentTime = 1000 / 60 + 1;
    options.rafCallbacks[0](currentTime);
    await vi.waitFor(() => {
      expect(options.tick).toHaveBeenCalledTimes(1);
    });

    expect(options.matchAudio.audio.playSound).not.toHaveBeenCalled();
  });

  it("plays every tick's own cues across a multi-tick catch-up burst, not just the last tick's", async () => {
    const root = document.createElement("div");
    const hitSound = {
      group: 1,
      sample: 0,
      sampleRate: 11025,
      channels: 1,
      bitsPerSample: 16,
      pcm: [0, 0],
    };
    const input = baseInput();
    input.player1.character = {
      ...character(),
      sounds: [{ index: 1, sounds: [hitSound] }],
    };
    let currentTime = 0;
    const options = baseOptions({ now: vi.fn(() => currentTime) });
    (options.tick as ReturnType<typeof vi.fn>)
      .mockResolvedValueOnce(
        tickResponse({ cues: [{ side: 0, group: 1, sample: 0 }] }),
      )
      .mockResolvedValueOnce(tickResponse({ cues: [] }))
      .mockResolvedValueOnce(
        tickResponse({ cues: [{ side: 0, group: 1, sample: 0 }] }),
      );

    await renderMatch(root, input, options);
    currentTime = (1000 / 60) * 3 + 1;
    options.rafCallbacks[0](currentTime);
    await vi.waitFor(() => {
      expect(options.tick).toHaveBeenCalledTimes(3);
    });

    expect(options.matchAudio.audio.playSound).toHaveBeenCalledTimes(2);
  });

  it("stops the match audio controller when stop() is called", async () => {
    const root = document.createElement("div");
    const options = baseOptions();

    const handle = await renderMatch(root, baseInput(), options);
    handle.stop();

    expect(options.matchAudio.audio.stop).toHaveBeenCalled();
  });

  it("disposes the input source when stop() is called", async () => {
    const root = document.createElement("div");
    const inputSource = fakeInputSource([
      {
        up: false,
        down: false,
        left: false,
        right: false,
        buttons: { a: false, b: false, c: false, x: false, y: false, z: false },
      },
      {
        up: false,
        down: false,
        left: false,
        right: false,
        buttons: { a: false, b: false, c: false, x: false, y: false, z: false },
      },
    ]);
    const options = baseOptions({
      createInputSource: vi.fn(() => inputSource),
    });

    const handle = await renderMatch(root, baseInput(), options);
    handle.stop();

    expect(inputSource.dispose).toHaveBeenCalled();
  });

  it("shows a live status line reflecting each player's active input source, updating when a source changes", async () => {
    const root = document.createElement("div");
    let onSourceChange:
      | ((playerIndex: 0 | 1, source: "keyboard" | "gamepad") => void)
      | undefined;
    const inputSource = fakeInputSource([
      {
        up: false,
        down: false,
        left: false,
        right: false,
        buttons: { a: false, b: false, c: false, x: false, y: false, z: false },
      },
      {
        up: false,
        down: false,
        left: false,
        right: false,
        buttons: { a: false, b: false, c: false, x: false, y: false, z: false },
      },
    ]);
    const options = baseOptions({
      createInputSource: vi.fn((onChange) => {
        onSourceChange = onChange;
        return inputSource;
      }),
    });

    await renderMatch(root, baseInput(), options);

    const status = root.querySelector(".match-renderer__input-status");
    expect(status).not.toBeNull();
    expect(status?.textContent).toContain("Keyboard");

    onSourceChange?.(0, "gamepad");

    expect(status?.textContent).toContain("Gamepad");
  });

  describe("in-match HUD (backlog item 004)", () => {
    it("mounts the HUD before the canvas, already populated with the match's starting health/round/wins", async () => {
      const root = document.createElement("div");
      const options = baseOptions();

      await renderMatch(root, baseInput(), options);

      const hud = root.querySelector(".hud");
      if (!hud) throw new Error("expected the HUD to be mounted");
      const canvas = root.querySelector("canvas");
      if (!canvas) throw new Error("expected the canvas to be mounted");
      // The HUD must precede the canvas in the DOM (see .vibe/decisions/009).
      expect(
        hud.compareDocumentPosition(canvas) & Node.DOCUMENT_POSITION_FOLLOWING,
      ).toBeTruthy();
      const p1Health = root.querySelector(".hud__health--1");
      expect(p1Health?.getAttribute("aria-valuenow")).toBe("1000");
      const roundBlock = root.querySelector(".hud__round");
      expect(roundBlock?.textContent).toContain("1");
    });

    it("updates the HUD once per simulated frame, in sync with the tick loop", async () => {
      const root = document.createElement("div");
      let currentTime = 0;
      const damagedTick = vi.fn(async () => {
        const base = tickResponse();
        if (!base.ok) throw new Error("expected an ok result");
        return {
          ok: true as const,
          data: {
            ...base.data,
            fighters: [
              { ...base.data.fighters[0], health: 640 },
              base.data.fighters[1],
            ] as MatchSnapshot["fighters"],
          },
        };
      });
      const options = baseOptions({
        now: vi.fn(() => currentTime),
        tick: damagedTick,
      });

      await renderMatch(root, baseInput(), options);
      currentTime = 1000 / 60 + 1;
      options.rafCallbacks[0](currentTime);

      await vi.waitFor(() => {
        expect(
          root.querySelector(".hud__health--1")?.getAttribute("aria-valuenow"),
        ).toBe("640");
      });
    });

    it("shows the best-of from the setup and the wins from the core's snapshot", async () => {
      const root = document.createElement("div");
      const options = baseOptions({
        startMatch: vi.fn(async () => startResponse(1, { wins: [1, 0] })),
      });

      await renderMatch(
        root,
        baseInput({ config: { rounds: 5, timeLimit: { seconds: 99 } } }),
        options,
      );

      expect(root.querySelector(".hud__round")?.textContent).toContain("5");
      expect(root.querySelectorAll(".hud__wins")[0].textContent).toContain("1");
    });
  });

  describe("core failures", () => {
    it("shows the core's error, stops the loop and releases the match when a tick fails, instead of freezing silently", async () => {
      const root = document.createElement("div");
      let currentTime = 0;
      const options = baseOptions({
        now: vi.fn(() => currentTime),
        tick: vi.fn(async () => ({
          ok: false as const,
          error: "match snapshot has no numeric camX",
        })),
      });

      await renderMatch(root, baseInput(), options);
      currentTime = 1000 / 60 + 1;
      options.rafCallbacks[0](currentTime);

      await vi.waitFor(() => {
        expect(root.textContent).toContain(
          "Match rendering stopped: match snapshot has no numeric camX",
        );
      });
      expect(options.cancelAnimationFrame).toHaveBeenCalled();
      expect(options.closeMatch).toHaveBeenCalledWith(1);
      expect(options.requestAnimationFrame).toHaveBeenCalledTimes(1);
    });
  });

  describe("drawing the core's state", () => {
    function plan(options: ReturnType<typeof baseOptions>, call: number) {
      return (options.drawScene as ReturnType<typeof vi.fn>).mock.calls[
        call
      ][1] as DrawCommand[];
    }

    it("draws each fighter where the core says it is, relative to the core's camera, mirrored when it faces left", async () => {
      const root = document.createElement("div");
      const options = baseOptions({
        startMatch: vi.fn(async () => startResponse(1, { camX: 30 })),
      });

      await renderMatch(root, baseInput(), options);

      const [p1, p2] = plan(options, 0);
      expect(p1.canvasX).toBe(200 - 70 - 30); // stage centre + x - camX
      expect(p2.canvasX).toBe(200 + 70 - 30);
      expect(p1.flipH).toBe(false);
      expect(p2.flipH).toBe(true);
    });

    it("redraws from the next snapshot's positions, not from the start", async () => {
      const root = document.createElement("div");
      let currentTime = 0;
      const moved = snapshot({ tick: 1, camX: 10 });
      moved.fighters[0] = { ...moved.fighters[0], x: -20 };
      const options = baseOptions({
        now: vi.fn(() => currentTime),
        tick: vi.fn(async () => ({ ok: true as const, data: moved })),
      });

      await renderMatch(root, baseInput(), options);
      currentTime = 1000 / 60 + 1;
      options.rafCallbacks[0](currentTime);
      await vi.waitFor(() => {
        expect(options.drawScene).toHaveBeenCalledTimes(2);
      });

      expect(plan(options, 1)[0].canvasX).toBe(200 - 20 - 10);
    });

    it("plays the stage's background animations at the snapshot's tick count", async () => {
      const root = document.createElement("div");
      let currentTime = 0;
      const input = baseInput();
      input.stage.stage.elements = [
        {
          name: "water",
          type: "anim",
          sprite: { group: 0, image: 0 },
          actionNumber: 7,
          layerNo: 0,
          startX: 0,
          startY: 0,
          deltaX: 1,
          deltaY: 1,
          tileX: 0,
          tileY: 0,
          tileSpacingX: 0,
          tileSpacingY: 0,
        },
      ];
      const options = baseOptions({
        now: vi.fn(() => currentTime),
        tick: vi.fn(async () => tickResponse({ tick: 42 })),
      });

      await renderMatch(root, input, options);
      currentTime = 1000 / 60 + 1;
      options.rafCallbacks[0](currentTime);
      await vi.waitFor(() => {
        expect(options.resolveAnimationFrames).toHaveBeenCalledTimes(2);
      });

      const calls = (options.resolveAnimationFrames as ReturnType<typeof vi.fn>)
        .mock.calls;
      expect(calls[0][0][0].elapsedTicks).toBe(0);
      expect(calls[1][0][0].elapsedTicks).toBe(42);
    });
  });

  describe("round/match result flow (backlog item 007)", () => {
    it("stops calling tick() the instant a round is decided, even mid-burst, and shows the round result", async () => {
      const root = document.createElement("div");
      let currentTime = 0;
      const tickMock = vi
        .fn()
        .mockResolvedValueOnce(tickResponse())
        .mockResolvedValueOnce(roundEndResponse(0))
        .mockResolvedValueOnce(tickResponse());
      const options = baseOptions({
        now: vi.fn(() => currentTime),
        tick: tickMock,
      });

      await renderMatch(root, baseInput(), options);
      // 3 full tick intervals elapsed at once -> a would-be 3-tick burst.
      currentTime = (1000 / 60) * 3 + 1;
      options.rafCallbacks[0](currentTime);

      await vi.waitFor(() => {
        expect(
          options.resultOverlay.overlay.showRoundResult,
        ).toHaveBeenCalled();
      });
      // Only 2 of the 3 ticks ran -- the 3rd would have double-counted an
      // already-decided round.
      expect(tickMock).toHaveBeenCalledTimes(2);
      expect(
        options.resultOverlay.overlay.showRoundResult,
      ).toHaveBeenCalledWith({ round: 1, winner: "p1" });
      // The loop is frozen, not rescheduled -- requestAnimationFrame was
      // only ever called once, at match start.
      expect(options.requestAnimationFrame).toHaveBeenCalledTimes(1);
    });

    it("shows the match result instead of the round result when the match is also decided this tick", async () => {
      const root = document.createElement("div");
      let currentTime = 0;
      const options = baseOptions({
        now: vi.fn(() => currentTime),
        tick: vi.fn(async () => roundEndResponse(1, { over: true, winner: 1 })),
      });

      await renderMatch(root, baseInput(), options);
      currentTime = 1000 / 60 + 1;
      options.rafCallbacks[0](currentTime);

      await vi.waitFor(() => {
        expect(
          options.resultOverlay.overlay.showMatchResult,
        ).toHaveBeenCalled();
      });
      expect(
        options.resultOverlay.overlay.showMatchResult,
      ).toHaveBeenCalledWith({ winner: "p2" });
      expect(
        options.resultOverlay.overlay.showRoundResult,
      ).not.toHaveBeenCalled();
    });

    it("resolves a double KO to a draw round result, not to either side", async () => {
      const root = document.createElement("div");
      let currentTime = 0;
      const options = baseOptions({
        now: vi.fn(() => currentTime),
        tick: vi.fn(async () =>
          tickResponse({
            roundEnded: true,
            outcome: "doubleKo",
            roundWinner: -1,
          }),
        ),
      });

      await renderMatch(root, baseInput(), options);
      currentTime = 1000 / 60 + 1;
      options.rafCallbacks[0](currentTime);

      await vi.waitFor(() => {
        expect(
          options.resultOverlay.overlay.showRoundResult,
        ).toHaveBeenCalledWith({ round: 1, winner: "draw" });
      });
    });

    it("starts the next round in the core and resumes the tick loop once the round result's auto-advance fires", async () => {
      const root = document.createElement("div");
      let currentTime = 0;
      const options = baseOptions({
        now: vi.fn(() => currentTime),
        tick: vi.fn(async () => roundEndResponse(0)),
      });

      await renderMatch(root, baseInput(), options);
      currentTime = 1000 / 60 + 1;
      options.rafCallbacks[0](currentTime);
      await vi.waitFor(() => {
        expect(
          options.resultOverlay.overlay.showRoundResult,
        ).toHaveBeenCalled();
      });

      options.resultOverlay.options().onRoundResultDone();

      await vi.waitFor(() => {
        expect(options.nextRound).toHaveBeenCalledWith(1);
      });
      // The loop actually resumed: a second requestAnimationFrame was
      // scheduled once the reset's own async sprite/HUD work settles, and a
      // further elapsed frame ticks again.
      await vi.waitFor(() => {
        expect(options.requestAnimationFrame).toHaveBeenCalledTimes(2);
      });
      expect(options.resultOverlay.overlay.hide).toHaveBeenCalled();
      // The HUD shows the new round the core started.
      expect(root.querySelector(".hud__round")?.textContent).toContain(
        "Round 2",
      );

      currentTime += 1000 / 60 + 1;
      options.rafCallbacks[1](currentTime);
      await vi.waitFor(() => {
        expect(options.tick).toHaveBeenCalledTimes(2);
      });
    });

    it("shows a clear error and stops the match instead of getting stuck when nextRound fails", async () => {
      const root = document.createElement("div");
      let currentTime = 0;
      const options = baseOptions({
        now: vi.fn(() => currentTime),
        tick: vi.fn(async () => roundEndResponse(0)),
        nextRound: vi.fn(async () => ({ ok: false as const, error: "boom" })),
      });

      await renderMatch(root, baseInput(), options);
      currentTime = 1000 / 60 + 1;
      options.rafCallbacks[0](currentTime);
      await vi.waitFor(() => {
        expect(
          options.resultOverlay.overlay.showRoundResult,
        ).toHaveBeenCalled();
      });

      options.resultOverlay.options().onRoundResultDone();

      await vi.waitFor(() => {
        expect(options.cancelAnimationFrame).toHaveBeenCalled();
      });
      expect(root.textContent).toContain("boom");
      expect(options.resultOverlay.overlay.hide).toHaveBeenCalled();
    });

    it("rematch restarts a fresh match with the same characters/stage/config, without any selection-screen involvement", async () => {
      const root = document.createElement("div");
      let currentTime = 0;
      const options = baseOptions({
        now: vi.fn(() => currentTime),
        tick: vi.fn(async () => roundEndResponse(0, { over: true, winner: 0 })),
      });
      const input = baseInput();

      await renderMatch(root, input, options);
      currentTime = 1000 / 60 + 1;
      options.rafCallbacks[0](currentTime);
      await vi.waitFor(() => {
        expect(
          options.resultOverlay.overlay.showMatchResult,
        ).toHaveBeenCalled();
      });

      options.resultOverlay.options().onRematch();

      await vi.waitFor(() => {
        expect(options.startMatch).toHaveBeenCalledTimes(2);
      });
      const [firstCall, secondCall] = (
        options.startMatch as ReturnType<typeof vi.fn>
      ).mock.calls;
      expect(secondCall[0]).toEqual(firstCall[0]);
      expect(input.onBackToSelect).not.toHaveBeenCalled();
    });

    it("back to select stops the match (releasing the core match and input listeners) before invoking the caller's callback", async () => {
      const root = document.createElement("div");
      let currentTime = 0;
      const options = baseOptions({
        now: vi.fn(() => currentTime),
        tick: vi.fn(async () => roundEndResponse(0, { over: true, winner: 0 })),
      });
      const input = baseInput();

      await renderMatch(root, input, options);
      currentTime = 1000 / 60 + 1;
      options.rafCallbacks[0](currentTime);
      await vi.waitFor(() => {
        expect(
          options.resultOverlay.overlay.showMatchResult,
        ).toHaveBeenCalled();
      });

      options.resultOverlay.options().onBackToSelect();

      expect(options.cancelAnimationFrame).toHaveBeenCalled();
      await vi.waitFor(() => {
        expect(options.closeMatch).toHaveBeenCalledWith(1);
      });
      expect(input.onBackToSelect).toHaveBeenCalledTimes(1);
    });
  });

  describe("pause menu (backlog item 021)", () => {
    function pressEscape(options: { keys: EventTarget }): void {
      options.keys.dispatchEvent(
        new KeyboardEvent("keydown", { key: "Escape" }),
      );
    }

    it("Escape freezes the simulation and the audio until Continue", async () => {
      const root = document.createElement("div");
      let currentTime = 0;
      const options = baseOptions({ now: vi.fn(() => currentTime) });
      await renderMatch(root, baseInput(), options);

      pressEscape(options);
      const menu = root.querySelector(".pause-menu");
      expect(menu?.hasAttribute("hidden")).toBe(false);
      expect(options.matchAudio.audio.setPaused).toHaveBeenLastCalledWith(true);

      currentTime = 1000;
      await options.rafCallbacks[0](currentTime);
      await options.rafCallbacks[1](currentTime + 500);
      expect(options.tick).not.toHaveBeenCalled();
      expect(options.rafCallbacks.length).toBe(3);

      currentTime = 2000;
      pressEscape(options); // Escape on the open menu = Continue
      expect(menu?.hasAttribute("hidden")).toBe(true);
      expect(options.matchAudio.audio.setPaused).toHaveBeenLastCalledWith(
        false,
      );
      await options.rafCallbacks[2](2000 + 1000 / 60 + 1);
      await vi.waitFor(() => {
        expect(options.tick).toHaveBeenCalledTimes(1);
      });
    });

    it("a gamepad's Start opens it and its d-pad and A choose Restart", async () => {
      const root = document.createElement("div");
      const options = baseOptions();
      const input = baseInput();
      await renderMatch(root, input, options);

      options.padEvents.push({ start: true, nav: null });
      await options.rafCallbacks[0](1);
      expect(root.querySelector(".pause-menu")?.hasAttribute("hidden")).toBe(
        false,
      );
      options.padEvents.push({ start: false, nav: "down" });
      await options.rafCallbacks[1](2);
      options.padEvents.push({ start: false, nav: "confirm" });
      await options.rafCallbacks[2](3);

      await vi.waitFor(() => {
        expect(options.startMatch).toHaveBeenCalledTimes(2);
      });
      expect(input.onBackToSelect).not.toHaveBeenCalled();
    });

    it("Character Change stops the match and goes back to selection", async () => {
      const root = document.createElement("div");
      const options = baseOptions();
      const input = baseInput();
      await renderMatch(root, input, options);

      pressEscape(options);
      const items = root.querySelectorAll<HTMLElement>(".pause-menu__item");
      items[2].click();

      expect(options.cancelAnimationFrame).toHaveBeenCalled();
      await vi.waitFor(() => {
        expect(options.closeMatch).toHaveBeenCalledWith(1);
      });
      expect(input.onBackToSelect).toHaveBeenCalledTimes(1);
      pressEscape(options);
      expect(root.querySelector(".pause-menu")?.hasAttribute("hidden")).toBe(
        true,
      );
    });

    it("does not open over a round or match result", async () => {
      const root = document.createElement("div");
      let currentTime = 0;
      const options = baseOptions({
        now: vi.fn(() => currentTime),
        tick: vi.fn(async () => roundEndResponse(0, { over: true, winner: 0 })),
      });
      await renderMatch(root, baseInput(), options);
      currentTime = 1000 / 60 + 1;
      options.rafCallbacks[0](currentTime);
      await vi.waitFor(() => {
        expect(
          options.resultOverlay.overlay.showMatchResult,
        ).toHaveBeenCalled();
      });
      pressEscape(options);
      expect(root.querySelector(".pause-menu")?.hasAttribute("hidden")).toBe(
        true,
      );
    });
  });

  describe("CPU opponent (backlog item 007)", () => {
    it("tells the core player 2 is the CPU when player2Control is 'cpu', since the core decides its inputs", async () => {
      const root = document.createElement("div");
      const options = baseOptions();

      await renderMatch(root, baseInput({ player2Control: "cpu" }), options);

      const request = (options.startMatch as ReturnType<typeof vi.fn>).mock
        .calls[0][0];
      expect(request.p2cpu).toBe(true);
    });

    it("leaves player 2 to the real input source (keyboard/gamepad) when player2Control is 'human' or omitted", async () => {
      const root = document.createElement("div");
      let currentTime = 0;
      const neutralButtons = {
        a: false,
        b: false,
        c: false,
        x: false,
        y: false,
        z: false,
      };
      const inputs: TickInputPair = [
        {
          up: false,
          down: false,
          left: false,
          right: false,
          buttons: neutralButtons,
        },
        {
          up: false,
          down: false,
          left: false,
          right: true,
          buttons: neutralButtons,
        },
      ];
      const inputSource = fakeInputSource(inputs);
      const options = baseOptions({
        now: vi.fn(() => currentTime),
        createInputSource: vi.fn(() => inputSource),
      });

      await renderMatch(root, baseInput(), options); // no player2Control set.
      currentTime = 1000 / 60 + 1;
      options.rafCallbacks[0](currentTime);

      await vi.waitFor(() => {
        expect(options.tick).toHaveBeenCalledTimes(1);
      });
      const request = (options.startMatch as ReturnType<typeof vi.fn>).mock
        .calls[0][0];
      expect(request.p2cpu).toBe(false);
      const tickInputs = (options.tick as ReturnType<typeof vi.fn>).mock
        .calls[0][1];
      expect(tickInputs.p2.right).toBe(true);
    });
  });
});

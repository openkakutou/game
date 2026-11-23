import { type MatchAudio, createMatchAudio } from "../audio/match-audio.ts";
import { buildSoundLookup, findSound } from "../audio/sound-lookup.ts";
import { type Hud, createHud } from "../hud/hud-renderer.ts";
import {
  type InputSourceKind,
  type TickInputSource,
  createTickInputSource,
} from "../input/tick-input-source.ts";
import {
  type PauseMenu,
  type PauseMenuOptions,
  createPauseMenu,
} from "../pause/pause-menu.ts";
import { type PausePad, createPausePad } from "../pause/pause-pad.ts";
import { deriveMatchOutcome, deriveRoundOutcome } from "../result/outcome.ts";
import {
  type ResultOverlay,
  type ResultOverlayOptions,
  createResultOverlay,
} from "../result/result-screen.ts";
import type { TimeLimitOption } from "../setup/setup-screen.ts";
import { resolveSprites as defaultResolveCharacterSprites } from "../wasm/bridge.ts";
import {
  closeMatch as defaultCloseMatch,
  nextRound as defaultNextRound,
  startMatch as defaultStartMatch,
  tick as defaultTick,
} from "../wasm/core-bridge.ts";
import type { MatchSnapshot } from "../wasm/core-types.ts";
import {
  resolveAnimationFrames as defaultResolveAnimationFrames,
  resolveSprites as defaultResolveStageSprites,
} from "../wasm/stage-bridge.ts";
import type {
  BGElement,
  SpriteRef,
  StageSummary,
} from "../wasm/stage-types.ts";
import type { CharacterSummary, Frame, Sound, Sprite } from "../wasm/types.ts";
// The match scene's canvas/timer orchestration: starts a match in the game
// core (`wasm/core-bridge.ts`, the same code the native game plays with),
// steps it with the live inputs at the simulation rate, and draws the
// snapshot it returns: the fighters and the camera come from the core, the
// sprites to draw them come from the `character`/`stage` bridges (cached,
// see `sprite-pixel-cache.ts`; backlog item 067 moves them to the core).
// This is the DOM/canvas/timer glue on top of the pure `scene-composition.ts`/
// `animation-resolution.ts`/`tick-scheduling.ts` logic -- see
// `.vibe/decisions/004-match-rendering-architecture.md` for the design
// decisions (timing policy, placeholder policy, transitional UX).
import {
  findAnimationByNumber,
  resolveCurrentFrame,
} from "./animation-resolution.ts";
import {
  type DrawCommand,
  type FighterRenderInput,
  buildMatchDrawPlan,
} from "./scene-composition.ts";
import {
  type ResolveSpritesFn,
  createSpritePixelCache,
} from "./sprite-pixel-cache.ts";
import { computeTicksToRun } from "./tick-scheduling.ts";

/** Simulation ticks per second: the core steps one tick per 1/60 s. */
const TICK_INTERVAL_MS = 1000 / 60;

/** Maximum simulation ticks run per `requestAnimationFrame` callback before the backlog is dropped (see `tick-scheduling.ts`). */
const DEFAULT_MAX_TICKS_PER_FRAME = 5;

/** Canvas size fallback when a stage's own `.def` leaves `[StageInfo] localcoord` unset. */
const FALLBACK_CANVAS_WIDTH = 400;
const FALLBACK_CANVAS_HEIGHT = 240;

export interface MatchRendererCharacterInput {
  character: CharacterSummary;
  sffBytes: Uint8Array;
  /** The name of this character's `.def` file in the core's file system (mounted before the match, see `main.ts`). */
  defName: string;
}

export interface MatchRendererStageInput {
  stage: StageSummary;
  sffBytes: Uint8Array;
  /** The name of this stage's `.def` file in the core's file system. */
  defName: string;
  /** This stage's background music file bytes, if it declared one and it fetched successfully -- `null`/`undefined` means no music plays for this match (backlog item 013). */
  musicBytes?: Uint8Array | null;
}

/** What the match setup screen chose, as far as the core needs it. */
export interface MatchRendererConfig {
  rounds: number;
  timeLimit: TimeLimitOption;
}

export interface MatchRendererInput {
  player1: MatchRendererCharacterInput;
  player2: MatchRendererCharacterInput;
  stage: MatchRendererStageInput;
  config: MatchRendererConfig;
  /** Who drives player 2 (backlog item 007): the core decides its inputs when `"cpu"`. Defaults to `"human"` when omitted. */
  player2Control?: "human" | "cpu";
  /**
   * Called when the player activates "Back to select" on the match result
   * screen -- the rendering module has no business knowing about roster/
   * stage/selection screens, so this is the caller's own escape hatch
   * (`main.ts` wires it to re-show character selection). `renderMatch`
   * always calls its own `stop()` first, releasing the core's match and
   * input listeners, before invoking this. See
   * `.vibe/decisions/010-round-match-result-and-cpu-opponent-design.md`.
   */
  onBackToSelect: () => void;
}

export interface MatchRendererHandle {
  /** Stops the tick loop and releases the core's match. Idempotent. */
  stop(): void;
}

type StartMatchFn = typeof defaultStartMatch;
type TickFn = typeof defaultTick;
type NextRoundFn = typeof defaultNextRound;
type CloseMatchFn = typeof defaultCloseMatch;
type ResolveAnimationFramesFn = typeof defaultResolveAnimationFrames;

export interface MatchRendererOptions {
  startMatch?: StartMatchFn;
  tick?: TickFn;
  /** Starts the next round once a round result's auto-advance countdown completes. Defaults to the real bridge's `nextRound`. */
  nextRound?: NextRoundFn;
  closeMatch?: CloseMatchFn;
  resolveCharacterSprites?: ResolveSpritesFn;
  resolveStageSprites?: ResolveSpritesFn;
  resolveAnimationFrames?: ResolveAnimationFramesFn;
  drawScene?: (canvas: HTMLCanvasElement, commands: DrawCommand[]) => void;
  requestAnimationFrame?: (callback: (timestamp: number) => void) => number;
  cancelAnimationFrame?: (handle: number) => void;
  now?: () => number;
  maxTicksPerFrame?: number;
  /**
   * Overrides the live tick input source entirely -- bypassing real
   * keyboard/gamepad wiring, the same rationale every other real-effect
   * option here follows. `onSourceChange` is threaded through so the
   * caller's fake can still exercise the input-status indicator. Defaults
   * to a real `createTickInputSource({ onSourceChange })`.
   */
  createInputSource?: (
    onSourceChange: (playerIndex: 0 | 1, source: InputSourceKind) => void,
  ) => TickInputSource;
  /** Overrides the round/match result overlay entirely -- for testing. Defaults to a real `createResultOverlay(options)`. */
  createResultOverlay?: (options: ResultOverlayOptions) => ResultOverlay;
  /** Overrides the match audio controller entirely -- bypassing the real `AudioContext`/Web Audio API wiring, same rationale as every other real-effect option here (backlog item 013). Defaults to a real `createMatchAudio()`. */
  createAudio?: () => MatchAudio;
  /** Overrides the pause menu entirely -- for testing. Defaults to a real `createPauseMenu(options)` (backlog item 021). */
  createPauseMenu?: (options: PauseMenuOptions) => PauseMenu;
  /** Overrides the gamepad reader for Start and menu buttons -- for testing. Defaults to a real `createPausePad()`. */
  createPausePad?: () => PausePad;
  /** Where Escape is listened for. Defaults to `document`. */
  keyTarget?: Pick<Document, "addEventListener" | "removeEventListener">;
}

/** A previous call's stop function, per root element, so a new call on the same root cleans up its predecessor's loop before starting its own -- mirrors `stage-viewer-web`'s own established convention. */
const activeLoopByRoot = new WeakMap<HTMLElement, () => void>();

function resolveOptions(options: MatchRendererOptions) {
  return {
    startMatch: options.startMatch ?? defaultStartMatch,
    tick: options.tick ?? defaultTick,
    nextRound: options.nextRound ?? defaultNextRound,
    closeMatch: options.closeMatch ?? defaultCloseMatch,
    resolveCharacterSprites:
      options.resolveCharacterSprites ?? defaultResolveCharacterSprites,
    resolveStageSprites:
      options.resolveStageSprites ?? defaultResolveStageSprites,
    resolveAnimationFrames:
      options.resolveAnimationFrames ?? defaultResolveAnimationFrames,
    drawScene: options.drawScene ?? createCanvasSceneDrawer(),
    requestAnimationFrame:
      options.requestAnimationFrame ??
      globalThis.requestAnimationFrame?.bind(globalThis),
    cancelAnimationFrame:
      options.cancelAnimationFrame ??
      globalThis.cancelAnimationFrame?.bind(globalThis),
    now: options.now ?? (() => performance.now()),
    maxTicksPerFrame: options.maxTicksPerFrame ?? DEFAULT_MAX_TICKS_PER_FRAME,
    createInputSource:
      options.createInputSource ??
      ((
        onSourceChange: (playerIndex: 0 | 1, source: InputSourceKind) => void,
      ) => createTickInputSource({ onSourceChange })),
    createResultOverlay: options.createResultOverlay ?? createResultOverlay,
    createAudio: options.createAudio ?? (() => createMatchAudio()),
    createPauseMenu: options.createPauseMenu ?? createPauseMenu,
    createPausePad: options.createPausePad ?? (() => createPausePad()),
    keyTarget: options.keyTarget ?? document,
  };
}

const noopHandle: MatchRendererHandle = { stop() {} };

/** The core's time limit in seconds, 0 meaning no limit. */
function timeLimitSeconds(timeLimit: TimeLimitOption): number {
  if (timeLimit === "unlimited") return 0;
  return timeLimit.seconds;
}

/**
 * Starts and renders a real match into `root`, replacing its previous
 * content (and stopping any previous loop already running on it). Shows a
 * brief "Preparing match…" status while the match starts and initial
 * sprites resolve, then a canvas kept in sync with the core's own
 * simulation rate -- see the ADR referenced above for the full design.
 */
export async function renderMatch(
  root: HTMLElement,
  input: MatchRendererInput,
  options: MatchRendererOptions = {},
): Promise<MatchRendererHandle> {
  activeLoopByRoot.get(root)?.();

  const deps = resolveOptions(options);

  root.replaceChildren();
  const status = document.createElement("p");
  status.className = "match-renderer__status";
  status.textContent = "Preparing match…";
  root.appendChild(status);

  const bgDef = input.stage.stage.bgDef;
  const canvasWidth =
    bgDef.localCoordWidth > 0 ? bgDef.localCoordWidth : FALLBACK_CANVAS_WIDTH;
  const canvasHeight =
    bgDef.localCoordHeight > 0
      ? bgDef.localCoordHeight
      : FALLBACK_CANVAS_HEIGHT;

  const created = await deps.startMatch({
    p1: input.player1.defName,
    p2: input.player2.defName,
    stage: input.stage.defName,
    setup: {
      rounds: input.config.rounds,
      timeLimitSeconds: timeLimitSeconds(input.config.timeLimit),
    },
    p2cpu: input.player2Control === "cpu",
    viewWidth: canvasWidth,
  });
  if (!created.ok) {
    status.textContent = `Could not start the match: ${created.error}`;
    return noopHandle;
  }
  const matchId = created.data.id;

  const canvas = document.createElement("canvas");
  canvas.className = "match-renderer__canvas";
  canvas.width = canvasWidth;
  canvas.height = canvasHeight;
  canvas.tabIndex = -1;

  const liveRegion = document.createElement("p");
  liveRegion.className = "match-renderer__announcement";
  liveRegion.setAttribute("aria-live", "polite");

  const characterCaches = [
    createSpritePixelCache(),
    createSpritePixelCache(),
  ] as const;
  const stageCache = createSpritePixelCache();

  const elements = input.stage.stage.elements;
  const animAnimations = input.stage.stage.animations;
  let resolvedAnimSpriteByElementIndex = new Map<number, SpriteRef | null>();

  const characters = [
    input.player1.character,
    input.player2.character,
  ] as const;
  const sffBytesBySide = [
    input.player1.sffBytes,
    input.player2.sffBytes,
  ] as const;
  // Computed once, not per draw: a character's own sprite list never
  // changes mid-match. Kept per-side, never merged into one shared map --
  // see scene-composition.ts's BuildMatchDrawPlanInput doc comment for why.
  const fighterSpriteMetaByKey: [Map<string, Sprite>, Map<string, Sprite>] = [
    buildSpriteMetaByKey(characters[0]),
    buildSpriteMetaByKey(characters[1]),
  ];
  // Computed once, not per tick: same "resolve once, reuse the lookup"
  // convention as fighterSpriteMetaByKey above (backlog item 013).
  const fighterSoundLookupByKey: [Map<string, Sound>, Map<string, Sound>] = [
    buildSoundLookup(characters[0]),
    buildSoundLookup(characters[1]),
  ];

  // Started fire-and-forget: `playMusic` never throws/rejects (see
  // `audio/match-audio.ts`), and a stage's background music has no bearing
  // on whether the match can start or simulate correctly, so this is never
  // awaited before the match proceeds.
  const audio = deps.createAudio();
  void audio.playMusic(input.stage.musicBytes ?? null);

  // The latest state the core returned: what the next frame draws.
  let latest: MatchSnapshot = created.data;

  // A thrown fighter may show its thrower's animation (ChangeAnim2): its
  // frame, sprite sheet, cache and sprite metadata are the owner's.
  async function resolveFighterSprites(): Promise<[Frame, Frame]> {
    const frames = latest.fighters.map((fighter) => {
      const animation = findAnimationByNumber(
        characters[fighter.animOwner].animations,
        fighter.anim,
      );
      return animation
        ? resolveCurrentFrame(
            animation.frames,
            animation.loopStart,
            fighter.animTime,
          )
        : resolveCurrentFrame([], 0, 0);
    }) as [Frame, Frame];

    await Promise.all(
      frames.map((frame, side) => {
        if (frame.group < 0 || frame.image < 0) return Promise.resolve();
        const owner = latest.fighters[side].animOwner;
        return characterCaches[owner].resolveMissing(
          sffBytesBySide[owner],
          [[frame.group, frame.image]],
          deps.resolveCharacterSprites,
        );
      }),
    );

    return frames;
  }

  async function resolveStageSprites(): Promise<void> {
    const requests: [number, number][] = [];
    for (const element of elements) {
      if (element.type === "normal" || element.type === "parallax") {
        requests.push([element.sprite.group, element.sprite.image]);
      }
    }
    if (requests.length > 0) {
      await stageCache.resolveMissing(
        input.stage.sffBytes,
        requests,
        deps.resolveStageSprites,
      );
    }

    const animElements: { element: BGElement; index: number }[] = [];
    elements.forEach((element, index) => {
      if (element.type === "anim") animElements.push({ element, index });
    });
    if (animElements.length === 0) return;

    // The background animations play with the round's own tick count.
    const animResult = await deps.resolveAnimationFrames(
      animElements.map(({ element }) => ({
        animation: animAnimations[String(element.actionNumber)] ?? null,
        elapsedTicks: latest.tick,
      })),
    );
    const nextResolved = new Map<number, SpriteRef | null>();
    if (animResult.ok) {
      animElements.forEach(({ index }, i) => {
        nextResolved.set(index, animResult.sprites[i] ?? null);
      });
      const animRequests = animResult.sprites
        .filter((ref) => ref.group >= 0 && ref.image >= 0)
        .map((ref): [number, number] => [ref.group, ref.image]);
      if (animRequests.length > 0) {
        await stageCache.resolveMissing(
          input.stage.sffBytes,
          animRequests,
          deps.resolveStageSprites,
        );
      }
    } else {
      for (const { index } of animElements) nextResolved.set(index, null);
    }
    resolvedAnimSpriteByElementIndex = nextResolved;
  }

  function drawCurrentState(frames: readonly [Frame, Frame]): void {
    const fighters: [FighterRenderInput, FighterRenderInput] = [
      {
        position: { x: latest.fighters[0].x, y: latest.fighters[0].y },
        facing: latest.fighters[0].facing,
        frame: frames[0],
        owner: latest.fighters[0].animOwner,
      },
      {
        position: { x: latest.fighters[1].x, y: latest.fighters[1].y },
        facing: latest.fighters[1].facing,
        frame: frames[1],
        owner: latest.fighters[1].animOwner,
      },
    ];
    const plan = buildMatchDrawPlan({
      elements,
      resolvedAnimSpriteByElementIndex,
      fighters,
      fighterSpriteMetaByKey,
      fighterPixelsByKey: [
        characterCaches[0].resolved,
        characterCaches[1].resolved,
      ],
      stagePixelsByKey: stageCache.resolved,
      zOffset: bgDef.zOffset,
      localCoordWidth: canvasWidth,
      cameraX: latest.camX,
    });
    deps.drawScene(canvas, plan);
  }

  // Created and populated before the first drawn frame, so the HUD never
  // shows blank/zero values even for one frame -- see `.vibe/decisions/009`.
  const hud: Hud = createHud({ bestOf: input.config.rounds });
  hud.update(latest);

  const initialFrames = await resolveFighterSprites();
  await resolveStageSprites();
  drawCurrentState(initialFrames);

  status.remove();

  const inputStatus = document.createElement("p");
  inputStatus.className = "match-renderer__input-status";
  inputStatus.setAttribute("aria-live", "polite");
  // A player's input source (keyboard vs. their assigned gamepad) is a
  // real, mid-match-changeable fact -- a disconnected/reconnected gamepad
  // falls that player back to keyboard, or picks it back up -- so this
  // gets its own visible, announced status line separate from the
  // one-shot "Match started" announcement in `liveRegion` above.
  const activeSources: [InputSourceKind, InputSourceKind] = [
    "keyboard",
    "keyboard",
  ];
  function renderInputStatus(): void {
    const label = (source: InputSourceKind) =>
      source === "gamepad" ? "Gamepad" : "Keyboard";
    inputStatus.textContent = `Player 1: ${label(activeSources[0])} · Player 2: ${label(activeSources[1])}`;
  }
  // The core ignores player 2's inputs when the CPU plays it, so the device
  // read for player 2 is sent as is either way.
  const inputSource = deps.createInputSource((playerIndex, source) => {
    activeSources[playerIndex] = source;
    renderInputStatus();
  });
  renderInputStatus();

  const resultOverlay = deps.createResultOverlay({
    onRoundResultDone: () => {
      void continueToNextRound();
    },
    onRematch: () => {
      // A fresh `renderMatch` call on the same `root` stops this instance
      // first (the existing `activeLoopByRoot` convention below), which
      // closes the old core match and disposes this overlay along with the
      // input source/HUD -- restarting from round 1 with the exact same
      // characters/stage/config, no selection-screen detour.
      void renderMatch(root, input, options);
    },
    onBackToSelect: () => {
      stop();
      input.onBackToSelect();
    },
  });

  // Opened by Escape or a gamepad's Start while the match runs; the tick
  // loop keeps scheduling frames but runs no tick until it closes (backlog
  // item 021).
  const pauseMenu = deps.createPauseMenu({
    onContinue: () => resume(),
    onRestart: () => {
      void renderMatch(root, input, options);
    },
    onCharacterChange: () => {
      stop();
      input.onBackToSelect();
    },
  });
  const pausePad = deps.createPausePad();

  root.append(
    hud.element,
    canvas,
    liveRegion,
    inputStatus,
    resultOverlay.element,
    pauseMenu.element,
  );
  canvas.focus();
  liveRegion.textContent = "Match started";

  let stopped = false;
  let rafHandle: number | null = null;
  // Seeded from `now()` (not the first rAF callback's own timestamp) so
  // the very first callback already has a real elapsed-time delta to work
  // with, rather than needing a "first call just establishes a baseline,
  // ticks nothing" throwaway frame.
  let lastTimestamp: number = deps.now();
  let accumulatorMs = 0;
  let paused = false;
  // Set while a round or match result is up: the loop is frozen behind it,
  // so there is nothing to pause.
  let resultShown = false;

  function pause(): void {
    if (stopped || paused || resultShown) return;
    paused = true;
    audio.setPaused(true);
    pauseMenu.open();
  }

  function resume(): void {
    if (!paused) return;
    paused = false;
    audio.setPaused(false);
    lastTimestamp = deps.now();
    accumulatorMs = 0;
    canvas.focus();
  }

  function handleKeydown(event: Event): void {
    if ((event as KeyboardEvent).key !== "Escape") return;
    if (pauseMenu.isOpen()) {
      pauseMenu.navigate("cancel");
      return;
    }
    pause();
  }
  deps.keyTarget.addEventListener("keydown", handleKeydown);

  function stop(): void {
    if (stopped) return;
    stopped = true;
    deps.keyTarget.removeEventListener("keydown", handleKeydown);
    pauseMenu.dispose();
    if (rafHandle !== null) deps.cancelAnimationFrame(rafHandle);
    inputSource.dispose();
    hud.dispose();
    resultOverlay.dispose();
    audio.stop();
    deps.closeMatch(matchId).catch(() => {
      // A failure to release the match is not user-visible -- it simply
      // stays resident for the life of the WASM instance, same degraded-
      // but-not-crashing behavior the bridge itself documents.
    });
  }

  /** Shows a core failure in place of the match and releases everything. */
  function failWith(error: string): void {
    status.textContent = `Match rendering stopped: ${error}`;
    root.append(status);
    stop();
  }

  /**
   * Runs up to `count` simulation ticks, stopping immediately -- mid-burst,
   * before this frame's remaining catch-up ticks -- the instant a tick ends
   * the round: the core refuses a tick after that until `nextRound`.
   */
  async function runTicks(
    count: number,
    inputs: ReturnType<TickInputSource["read"]>,
  ): Promise<boolean> {
    for (let i = 0; i < count; i++) {
      if (stopped) return false;
      const result = await deps.tick(matchId, { p1: inputs[0], p2: inputs[1] });
      if (!result.ok) {
        failWith(result.error);
        return false;
      }
      latest = result.data;
      // Played immediately, this exact tick -- never deferred to after the
      // burst or attached only to its final iteration, since a dropped
      // intermediate tick's own cues would otherwise be lost (backlog item
      // 013).
      for (const cue of latest.cues) {
        const sound = findSound(fighterSoundLookupByKey[cue.side], cue);
        if (sound) audio.playSound(sound);
      }
      if (latest.roundEnded) return true;
    }
    return true;
  }

  /**
   * Shows the round or match result once the round has ended, favoring the
   * match result when the match is also over (a round-then-match double
   * announcement would flash the round result screen for one frame before
   * immediately replacing it -- see the UX consultation).
   */
  function showRoundOrMatchEnd(): void {
    resultShown = true;
    const matchEnd = deriveMatchOutcome(latest);
    if (matchEnd) {
      resultOverlay.showMatchResult(matchEnd);
      return;
    }
    const roundEnd = deriveRoundOutcome(latest);
    if (!roundEnd) return; // Defensive: unreachable, the caller checked roundEnded.
    resultOverlay.showRoundResult(roundEnd);
  }

  /**
   * Advances to the next round once a round result's auto-advance countdown
   * completes: starts it in the core, hides the overlay, and resumes the
   * tick loop -- the accumulator/timestamp are reset the same way match
   * start seeds them, so the first new frame doesn't replay a stale
   * accumulator as an immediate catch-up burst against the fresh round.
   */
  async function continueToNextRound(): Promise<void> {
    const result = await deps.nextRound(matchId);
    if (!result.ok) {
      resultOverlay.hide();
      failWith(result.error);
      return;
    }

    latest = result.data;
    resultOverlay.hide();
    resultShown = false;

    lastTimestamp = deps.now();
    accumulatorMs = 0;

    const frames = await resolveFighterSprites();
    await resolveStageSprites();
    if (stopped) return;
    drawCurrentState(frames);
    hud.update(latest);

    rafHandle = deps.requestAnimationFrame((ts) => void frame(ts));
  }

  async function frame(timestamp: number): Promise<void> {
    if (stopped) return;

    const pad = pausePad.poll();
    if (pad.start && paused) {
      pauseMenu.navigate("cancel");
    } else if (pad.start) {
      pause();
    } else if (paused && pad.nav !== null) {
      pauseMenu.navigate(pad.nav);
    }
    if (stopped) return;
    if (paused) {
      lastTimestamp = timestamp;
      rafHandle = deps.requestAnimationFrame((ts) => void frame(ts));
      return;
    }

    const deltaMs = timestamp - lastTimestamp;
    lastTimestamp = timestamp;
    accumulatorMs += deltaMs;

    const { ticksToRun, remainderMs } = computeTicksToRun(
      accumulatorMs,
      TICK_INTERVAL_MS,
      deps.maxTicksPerFrame,
    );
    accumulatorMs = remainderMs;

    if (ticksToRun > 0) {
      // Read once per rendered frame, not once per tick: this same
      // snapshot is reused for every tick in this frame's catch-up burst --
      // see `.vibe/decisions/005-input-routing-design.md`.
      const inputs = inputSource.read();
      const ok = await runTicks(ticksToRun, inputs);
      if (!ok) return;
      const frames = await resolveFighterSprites();
      await resolveStageSprites();
      if (stopped) return;
      drawCurrentState(frames);
      // Same cadence as drawCurrentState above -- once per rendered frame
      // that actually simulated, never once per tick during a catch-up
      // burst (see .vibe/decisions/009).
      hud.update(latest);

      if (latest.roundEnded) {
        // The round/match just ended on this exact frame's final tick: this
        // frame's own sprite resolve/draw/HUD update above already show the
        // state that decided it (the KO pose), so freezing here -- not
        // rescheduling another `requestAnimationFrame` -- is what actually
        // keeps that frame on screen behind the result overlay. See
        // `.vibe/decisions/010`.
        showRoundOrMatchEnd();
        return;
      }
    }

    if (!stopped) {
      rafHandle = deps.requestAnimationFrame((ts) => void frame(ts));
    }
  }

  rafHandle = deps.requestAnimationFrame((ts) => void frame(ts));

  const handle: MatchRendererHandle = { stop };
  activeLoopByRoot.set(root, stop);
  return handle;
}

/** One character's own sprite metadata, keyed by `(group, image)` — never merged with another character's, since sprite numbering is only unique within one `.sff` sheet. */
function buildSpriteMetaByKey(
  character: CharacterSummary,
): Map<string, Sprite> {
  const map = new Map<string, Sprite>();
  for (const group of character.sprites) {
    for (const sprite of group.sprites) {
      map.set(`${sprite.group},${sprite.image}`, sprite);
    }
  }
  return map;
}

/**
 * The real canvas-drawing default: bakes each uniquely-resolved sprite's
 * pixels into its own offscreen canvas once (memoized by the `pixels`
 * array's own identity — the sprite cache always returns the same array
 * reference for an already-resolved sprite, never a fresh copy), then
 * `drawImage`s it with a translate/scale transform for facing/frame-flip
 * mirroring — never `putImageData` on the main canvas directly, and never
 * re-baked on a cache hit (see the ADR's realtime-rendering requirement).
 * Real pixel output is verified via a real-browser pass, not jsdom (which
 * has no `getContext("2d")` without the separate `canvas` npm package this
 * project doesn't depend on) — same convention every OpenKakutou canvas
 * feature already follows.
 */
export function createCanvasSceneDrawer(): (
  canvas: HTMLCanvasElement,
  commands: DrawCommand[],
) => void {
  const blitCache = new WeakMap<Uint8Array, HTMLCanvasElement>();

  function getOrBakeSpriteCanvas(cmd: DrawCommand): HTMLCanvasElement | null {
    if (!cmd.pixels) return null;
    const cached = blitCache.get(cmd.pixels);
    if (cached) return cached;

    const off = document.createElement("canvas");
    off.width = cmd.width;
    off.height = cmd.height;
    const offCtx = off.getContext("2d");
    if (!offCtx) return null;
    const clamped = new Uint8ClampedArray(cmd.pixels);
    offCtx.putImageData(new ImageData(clamped, cmd.width, cmd.height), 0, 0);
    blitCache.set(cmd.pixels, off);
    return off;
  }

  return (canvas, commands) => {
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    for (const cmd of commands) {
      ctx.save();
      ctx.translate(cmd.canvasX, cmd.canvasY);
      ctx.scale(cmd.flipH ? -1 : 1, cmd.flipV ? -1 : 1);
      const drawX = -(cmd.flipH ? cmd.width - cmd.axisX : cmd.axisX);
      const drawY = -(cmd.flipV ? cmd.height - cmd.axisY : cmd.axisY);

      if (cmd.kind === "sprite") {
        const sprite = getOrBakeSpriteCanvas(cmd);
        if (sprite) ctx.drawImage(sprite, drawX, drawY);
      } else {
        ctx.fillStyle = "rgba(255, 0, 200, 0.65)";
        ctx.fillRect(drawX, drawY, cmd.width, cmd.height);
        ctx.strokeStyle = "rgba(255, 255, 255, 0.9)";
        ctx.lineWidth = 2;
        ctx.strokeRect(drawX, drawY, cmd.width, cmd.height);
      }
      ctx.restore();
    }
  };
}

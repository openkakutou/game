// Typed mirror of the JSON contract of the game core WASM module
// (`native/cmd/core-wasm`, `native/internal/core`): the request shapes the
// bridge sends and the state snapshot each call returns. Field names match
// the Go side's JSON exactly.

/** Which way a fighter faces: 1 right, -1 left. */
export type Facing = 1 | -1;

/** Which side a value belongs to: 0 player 1, 1 player 2. */
export type Side = 0 | 1;

/** One tick's raw directional/button input for one fighter. A field left unset reads as not held. */
export interface TickInput {
  up?: boolean;
  down?: boolean;
  left?: boolean;
  right?: boolean;
  /** Currently-held button names (a, b, c, x, y, z), true when held. */
  buttons?: Record<string, boolean>;
}

/** How the round just played ended; "none" while it is still being fought. */
export type RoundOutcome = "none" | "ko" | "doubleKo" | "timeout" | "draw";

/** One fighter's state to draw: position in stage pixels (y up, 0 is the ground), animation to play, its elapsed ticks and the side whose animations and sprites it comes from, health and power. */
export interface FighterSnapshot {
  x: number;
  y: number;
  facing: Facing;
  anim: number;
  animTime: number;
  /** The fighter's own side, or its opponent's when it shows the thrower's animation (a thrown fighter, ChangeAnim2). */
  animOwner: Side;
  stateNo: number;
  health: number;
  power: number;
}

/** A sound a fighter triggered this tick (`PlaySnd`): the same (group, sample) reference its decoded sounds are keyed by. */
export interface SoundCue {
  side: Side;
  group: number;
  sample: number;
}

/** The state to draw after a start, a tick or a round change. */
export interface MatchSnapshot {
  tick: number;
  round: number;
  /** Ticks left on the round timer. */
  timer: number;
  roundEnded: boolean;
  outcome: RoundOutcome;
  /** The match is decided; no round follows. */
  over: boolean;
  /** The match winner once `over`, else -1. */
  winner: -1 | Side;
  /** Who won the round that just ended: -1 for a draw or while fighting. */
  roundWinner: -1 | Side;
  wins: [number, number];
  /** Where the camera looks, in stage pixels (the middle of the view). */
  camX: number;
  fighters: [FighterSnapshot, FighterSnapshot];
  cues: SoundCue[];
}

/** What `startMatch` returns: the first snapshot plus the match id and the values the HUD needs once. */
export interface StartMatchData extends MatchSnapshot {
  id: number;
  roundTimerTicks: number;
  roundsToWin: number;
}

export interface StartMatchRequest {
  /** Names of the mounted `.def` files. */
  p1: string;
  p2: string;
  stage: string;
  setup: { rounds: number; timeLimitSeconds: number };
  /** The core decides player 2's inputs. */
  p2cpu: boolean;
  /** The width the camera frames, in pixels. */
  viewWidth: number;
}

/** Result of a bridge call: exactly one of `data`/`error` is meaningful. */
export type CoreResult<T> =
  | { ok: true; data: T }
  | { ok: false; error: string };

// Bridge to the game core WASM module (`native/cmd/core-wasm`): loads
// `core-wasm_exec.js`, instantiates `core.wasm` and wraps the global
// `OpenKakutouCore` calls (`mount`, `startMatch`, `tick`, `nextRound`,
// `closeMatch`) in typed, never-throwing functions. Same loading strategy as
// the other bridges (injectable fetch, `Function`-executed glue, unawaited
// `go.run`), with its own `wasm_exec.js` so it can never drift out of pairing
// with another module's. Every core call returns a JSON string, either
// `{"ok":true,...}` or `{"ok":false,"error":{"reason"}}`; this bridge parses
// it and validates the snapshot, so what reaches the UI is always well formed.
import type {
  CoreResult,
  FighterSnapshot,
  MatchSnapshot,
  RoundOutcome,
  SoundCue,
  StartMatchData,
  StartMatchRequest,
  TickInput,
} from "./core-types.ts";

const DEFAULT_WASM_EXEC_URL = "./wasm/core-wasm_exec.js";
const DEFAULT_WASM_BINARY_URL = "./wasm/core.wasm";

/** The `Go` runtime instance `wasm_exec.js` (via `new globalThis.Go()`) produces. */
interface GoRuntime {
  importObject: WebAssembly.Imports;
  run(instance: WebAssembly.Instance): Promise<void>;
}

interface OpenKakutouCoreGlobal {
  mount(name: string, bytes: Uint8Array): string;
  startMatch(requestJSON: string): string;
  tick(id: number, inputsJSON: string): string;
  nextRound(id: number): string;
  closeMatch(id: number): string;
}

export interface CoreWasmBridgeOptions {
  /** Fetches `core-wasm_exec.js`'s source text. Defaults to `fetch(DEFAULT_WASM_EXEC_URL)`. */
  fetchWasmExecSource?: () => Promise<string>;
  /** Fetches `core.wasm`'s raw bytes. Defaults to `fetch(DEFAULT_WASM_BINARY_URL)`. */
  fetchWasmBytes?: () => Promise<Uint8Array>;
}

/** Both players' inputs for one tick. */
export interface TickInputs {
  p1: TickInput;
  p2: TickInput;
}

/** One file to mount into the core: a slash-separated name from the game root, and its bytes. */
export interface CoreFile {
  name: string;
  bytes: Uint8Array;
}

async function defaultFetchWasmExecSource(): Promise<string> {
  const response = await fetch(DEFAULT_WASM_EXEC_URL);
  if (!response.ok) {
    throw new Error(
      `failed to fetch ${DEFAULT_WASM_EXEC_URL}: ${response.status} ${response.statusText}`,
    );
  }
  return response.text();
}

async function defaultFetchWasmBytes(): Promise<Uint8Array> {
  const response = await fetch(DEFAULT_WASM_BINARY_URL);
  if (!response.ok) {
    throw new Error(
      `failed to fetch ${DEFAULT_WASM_BINARY_URL}: ${response.status} ${response.statusText}`,
    );
  }
  return new Uint8Array(await response.arrayBuffer());
}

function getGoConstructor(): new () => GoRuntime {
  return (globalThis as unknown as { Go: new () => GoRuntime }).Go;
}

function getCore(): OpenKakutouCoreGlobal {
  return (globalThis as unknown as { OpenKakutouCore: OpenKakutouCoreGlobal })
    .OpenKakutouCore;
}

// Memoized so repeated calls neither re-fetch nor re-instantiate the module;
// reset by `resetCoreBridgeForTests`. Independent of the other bridges'
// own promises: each WASM binary needs its own Go runtime instance.
let readyPromise: Promise<void> | null = null;

// Names already mounted in this page's core: a file is mounted once per
// page, whatever the number of matches played.
const mountedNames = new Set<string>();

async function instantiateGoRuntime(
  options: CoreWasmBridgeOptions,
): Promise<void> {
  const fetchWasmExecSource =
    options.fetchWasmExecSource ?? defaultFetchWasmExecSource;
  const fetchWasmBytes = options.fetchWasmBytes ?? defaultFetchWasmBytes;

  const wasmExecSource = await fetchWasmExecSource();
  // wasm_exec.js assigns `globalThis.Go` itself, so running its source as a
  // function body works the same in a browser and under jsdom/Node.
  new Function(wasmExecSource)();

  const go = new (getGoConstructor())();
  const wasmBytes = await fetchWasmBytes();
  const { instance } = await WebAssembly.instantiate(
    wasmBytes as BufferSource,
    go.importObject,
  );

  // Not awaited: Go's main() registers OpenKakutouCore synchronously, then
  // blocks forever in select{}, so awaiting go.run would hang.
  go.run(instance);
}

function ensureGoRuntimeReady(options: CoreWasmBridgeOptions): Promise<void> {
  if (!readyPromise) {
    readyPromise = instantiateGoRuntime(options).catch((err: unknown) => {
      // Let a later call retry instead of keeping a rejected promise forever.
      readyPromise = null;
      throw err;
    });
  }
  return readyPromise;
}

/** Resets the memoized WASM instantiation and the mounted names. Test-only. */
export function resetCoreBridgeForTests(): void {
  readyPromise = null;
  mountedNames.clear();
}

/** Converts a failed load (missing asset, version-mismatched binary) into a typed error, so this module never rejects. */
async function ensureGoRuntimeReadyOrError(
  options: CoreWasmBridgeOptions,
): Promise<{ ok: true } | { ok: false; error: string }> {
  try {
    await ensureGoRuntimeReady(options);
    return { ok: true };
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : String(err);
    return {
      ok: false,
      error: `game core WASM module failed to load: ${message}`,
    };
  }
}

/** Runs one core call and parses its reply to an object, or an error. Never throws. */
function callCore(
  callName: string,
  call: (core: OpenKakutouCoreGlobal) => string,
): CoreResult<Record<string, unknown>> {
  let reply: string;
  try {
    reply = call(getCore());
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : String(err);
    return { ok: false, error: `game core ${callName} failed: ${message}` };
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(reply);
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : String(err);
    return {
      ok: false,
      error: `game core ${callName} returned malformed JSON: ${message}`,
    };
  }
  if (!isRecord(parsed)) {
    return { ok: false, error: `game core ${callName} returned no object` };
  }
  if (parsed.ok === true) return { ok: true, data: parsed };
  const failure = errorReason(parsed);
  if (failure === null) {
    return {
      ok: false,
      error: `game core ${callName} returned neither success nor an error reason`,
    };
  }
  return { ok: false, error: failure };
}

function errorReason(reply: Record<string, unknown>): string | null {
  if (reply.ok !== false) return null;
  if (!isRecord(reply.error)) return null;
  if (typeof reply.error.reason !== "string") return null;
  return reply.error.reason;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isFiniteNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value);
}

const ROUND_OUTCOMES: readonly string[] = [
  "none",
  "ko",
  "doubleKo",
  "timeout",
  "draw",
];

function isSide(value: unknown): value is 0 | 1 {
  return value === 0 || value === 1;
}

function parseFighter(value: unknown, index: number): FighterSnapshot | string {
  if (!isRecord(value)) return `fighter ${index} is not an object`;
  const numbers = [
    "x",
    "y",
    "anim",
    "animTime",
    "stateNo",
    "health",
    "power",
  ] as const;
  for (const key of numbers) {
    if (!isFiniteNumber(value[key])) {
      return `fighter ${index} has no numeric ${key}`;
    }
  }
  if (value.facing !== 1 && value.facing !== -1) {
    return `fighter ${index} has no facing of 1 or -1`;
  }
  if ((value.health as number) < 0 || (value.power as number) < 0) {
    return `fighter ${index} has a negative health or power`;
  }
  return value as unknown as FighterSnapshot;
}

function parseCue(value: unknown): SoundCue | string {
  if (!isRecord(value)) return "a sound cue is not an object";
  if (!isSide(value.side)) return "a sound cue has no side of 0 or 1";
  if (!isFiniteNumber(value.group) || !isFiniteNumber(value.sample)) {
    return "a sound cue has no numeric group and sample";
  }
  return value as unknown as SoundCue;
}

/** Validates a core reply as a `MatchSnapshot`; the whole snapshot is refused on any invalid field, never half-trusted. */
function parseSnapshot(
  reply: Record<string, unknown>,
): CoreResult<MatchSnapshot> {
  const counters = ["tick", "round", "timer", "camX"] as const;
  for (const key of counters) {
    if (!isFiniteNumber(reply[key])) {
      return { ok: false, error: `match snapshot has no numeric ${key}` };
    }
  }
  if (
    typeof reply.roundEnded !== "boolean" ||
    typeof reply.over !== "boolean"
  ) {
    return { ok: false, error: "match snapshot has no roundEnded/over flags" };
  }
  if (
    typeof reply.outcome !== "string" ||
    !ROUND_OUTCOMES.includes(reply.outcome)
  ) {
    return { ok: false, error: "match snapshot has an unknown outcome" };
  }
  for (const key of ["winner", "roundWinner"] as const) {
    if (reply[key] !== -1 && !isSide(reply[key])) {
      return { ok: false, error: `match snapshot has no ${key} of -1, 0 or 1` };
    }
  }
  const wins = reply.wins;
  if (
    !Array.isArray(wins) ||
    wins.length !== 2 ||
    !isFiniteNumber(wins[0]) ||
    !isFiniteNumber(wins[1])
  ) {
    return {
      ok: false,
      error: "match snapshot does not have exactly 2 numeric wins",
    };
  }
  if (!Array.isArray(reply.fighters) || reply.fighters.length !== 2) {
    return {
      ok: false,
      error: "match snapshot does not have exactly 2 fighters",
    };
  }
  const fighters: FighterSnapshot[] = [];
  for (const [index, raw] of reply.fighters.entries()) {
    const fighter = parseFighter(raw, index);
    if (typeof fighter === "string") return { ok: false, error: fighter };
    fighters.push(fighter);
  }
  if (!Array.isArray(reply.cues)) {
    return { ok: false, error: "match snapshot has no cues list" };
  }
  const cues: SoundCue[] = [];
  for (const raw of reply.cues) {
    const cue = parseCue(raw);
    if (typeof cue === "string") return { ok: false, error: cue };
    cues.push(cue);
  }
  return {
    ok: true,
    data: {
      tick: reply.tick as number,
      round: reply.round as number,
      timer: reply.timer as number,
      roundEnded: reply.roundEnded,
      outcome: reply.outcome as RoundOutcome,
      over: reply.over,
      winner: reply.winner as MatchSnapshot["winner"],
      roundWinner: reply.roundWinner as MatchSnapshot["roundWinner"],
      wins: [wins[0], wins[1]],
      camX: reply.camX as number,
      fighters: [fighters[0], fighters[1]],
      cues,
    },
  };
}

/** Runs a core call that replies with a snapshot, after making sure the module is loaded. */
async function callForSnapshot(
  options: CoreWasmBridgeOptions,
  callName: string,
  call: (core: OpenKakutouCoreGlobal) => string,
): Promise<
  CoreResult<{ reply: Record<string, unknown>; snapshot: MatchSnapshot }>
> {
  const ready = await ensureGoRuntimeReadyOrError(options);
  if (!ready.ok) return ready;
  const called = callCore(callName, call);
  if (!called.ok) return called;
  const snapshot = parseSnapshot(called.data);
  if (!snapshot.ok) return snapshot;
  return { ok: true, data: { reply: called.data, snapshot: snapshot.data } };
}

/**
 * Mounts files into the core's in-memory file system. A name already
 * mounted in this page is skipped (content is fetched once per page); the
 * first file the core refuses stops the call with its reason.
 */
export async function mountFiles(
  files: readonly CoreFile[],
  options: CoreWasmBridgeOptions = {},
): Promise<CoreResult<Record<string, never>>> {
  const ready = await ensureGoRuntimeReadyOrError(options);
  if (!ready.ok) return ready;
  for (const file of files) {
    if (mountedNames.has(file.name)) continue;
    const mounted = callCore("mount", (core) =>
      core.mount(file.name, file.bytes),
    );
    if (!mounted.ok) return mounted;
    mountedNames.add(file.name);
  }
  return { ok: true, data: {} };
}

/**
 * Starts round 1 of a match from `.def` files mounted earlier, and returns
 * its id, the values the HUD needs once and the state to draw first.
 */
export async function startMatch(
  request: StartMatchRequest,
  options: CoreWasmBridgeOptions = {},
): Promise<CoreResult<StartMatchData>> {
  const called = await callForSnapshot(options, "startMatch", (core) =>
    core.startMatch(JSON.stringify(request)),
  );
  if (!called.ok) return called;
  const { reply, snapshot } = called.data;
  for (const key of ["id", "roundTimerTicks", "roundsToWin"] as const) {
    if (!isFiniteNumber(reply[key])) {
      return { ok: false, error: `start reply has no numeric ${key}` };
    }
  }
  return {
    ok: true,
    data: {
      ...snapshot,
      id: reply.id as number,
      roundTimerTicks: reply.roundTimerTicks as number,
      roundsToWin: reply.roundsToWin as number,
    },
  };
}

/** Steps match `id` by one tick with both players' inputs; returns the state to draw. */
export async function tick(
  id: number,
  inputs: TickInputs,
  options: CoreWasmBridgeOptions = {},
): Promise<CoreResult<MatchSnapshot>> {
  const called = await callForSnapshot(options, "tick", (core) =>
    core.tick(id, JSON.stringify(inputs)),
  );
  if (!called.ok) return called;
  return { ok: true, data: called.data.snapshot };
}

/** Starts the round after one that ended; returns the state to draw before its first tick. */
export async function nextRound(
  id: number,
  options: CoreWasmBridgeOptions = {},
): Promise<CoreResult<MatchSnapshot>> {
  const called = await callForSnapshot(options, "nextRound", (core) =>
    core.nextRound(id),
  );
  if (!called.ok) return called;
  return { ok: true, data: called.data.snapshot };
}

/** Releases match `id`'s memory in the core. */
export async function closeMatch(
  id: number,
  options: CoreWasmBridgeOptions = {},
): Promise<CoreResult<Record<string, never>>> {
  const ready = await ensureGoRuntimeReadyOrError(options);
  if (!ready.ok) return ready;
  const closed = callCore("closeMatch", (core) => core.closeMatch(id));
  if (!closed.ok) return closed;
  return { ok: true, data: {} };
}

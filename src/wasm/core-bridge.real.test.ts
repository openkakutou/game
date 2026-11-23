// The real game core (`public/wasm/core.wasm`, built by `npm run wasm:core`)
// driven through the bridge with the native build's own test content
// (`native/testdata/content`): proves the JSON contract the bridge validates
// is the one the Go side really writes, not just the one the fake core in
// `core-bridge.test.ts` replays.
import { readFileSync, readdirSync } from "node:fs";
import path from "node:path";
import { beforeAll, describe, expect, it } from "vitest";
import {
  closeMatch,
  mountFiles,
  nextRound,
  resetCoreBridgeForTests,
  startMatch,
  tick,
} from "./core-bridge.ts";
import type { CoreFile } from "./core-bridge.ts";
import type { MatchSnapshot, StartMatchRequest } from "./core-types.ts";

const publicWasmDir = path.resolve(
  import.meta.dirname,
  "..",
  "..",
  "public",
  "wasm",
);
const contentRoot = path.resolve(
  import.meta.dirname,
  "..",
  "..",
  "native",
  "testdata",
  "content",
);
const realCore = {
  fetchWasmExecSource: async () =>
    readFileSync(path.join(publicWasmDir, "core-wasm_exec.js"), "utf-8"),
  fetchWasmBytes: async () =>
    new Uint8Array(readFileSync(path.join(publicWasmDir, "core.wasm"))),
};

function contentFiles(dir: string, prefix = ""): CoreFile[] {
  const files: CoreFile[] = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const name = `${prefix}${entry.name}`;
    if (entry.isDirectory()) {
      files.push(...contentFiles(path.join(dir, entry.name), `${name}/`));
    } else {
      files.push({
        name,
        bytes: new Uint8Array(readFileSync(path.join(dir, entry.name))),
      });
    }
  }
  return files;
}

// The fixture characters cannot walk, so the match is played on a stage
// narrow enough to start the fighters within reach of each other (the same
// trick as `native/cmd/core-wasm/smoke.mjs`).
function closeStage(): CoreFile {
  const arena = readFileSync(
    path.join(contentRoot, "stages", "arena.def"),
    "utf8",
  )
    .replace("leftbound = -160", "leftbound = -6")
    .replace("rightbound = 160", "rightbound = 6");
  // Copied into this realm's Uint8Array: the core only accepts that one.
  const bytes = new Uint8Array(new TextEncoder().encode(arena));
  return { name: "stages/close.def", bytes };
}

const request: StartMatchRequest = {
  p1: "chars/fighter/fighter.def",
  p2: "chars/rival/rival.def",
  stage: "stages/close.def",
  setup: { rounds: 3, timeLimitSeconds: 1 },
  p2cpu: false,
  viewWidth: 320,
};

const idle = { p1: {}, p2: {} };

beforeAll(async () => {
  resetCoreBridgeForTests();
  const mounted = await mountFiles(
    [...contentFiles(contentRoot), closeStage()],
    realCore,
  );
  if (!mounted.ok)
    throw new Error(`cannot mount the content: ${mounted.error}`);
});

describe("the real game core through the bridge", () => {
  it("starts round 1 with both fighters at their start positions, full health and the HUD values of the setup", async () => {
    const started = await startMatch(request, realCore);

    expect(started.ok).toBe(true);
    if (!started.ok) return;
    expect(started.data.roundsToWin).toBe(2);
    expect(started.data.roundTimerTicks).toBe(60);
    expect(started.data.round).toBe(1);
    expect(started.data.fighters[0]).toMatchObject({
      x: -6,
      facing: 1,
      health: 1000,
      power: 0,
    });
    expect(started.data.fighters[1]).toMatchObject({ x: 6, facing: -1 });
    await closeMatch(started.data.id, realCore);
  });

  it("steps ticks: the timer counts down, an attack hurts the other fighter and triggers its sound cue", async () => {
    const started = await startMatch(request, realCore);
    if (!started.ok) throw new Error(started.error);
    const id = started.data.id;

    // 300 ticks: the round's intros play, then the fight, where the time
    // counts and P1's attack lands.
    const cues: number[] = [];
    let last = started.data;
    for (let i = 0; i < 300; i++) {
      const input =
        i % 2 === 0 ? { p1: { buttons: { a: true } }, p2: {} } : idle;
      const stepped = await tick(id, input, realCore);
      if (!stepped.ok) throw new Error(stepped.error);
      last = { ...started.data, ...stepped.data };
      for (const cue of stepped.data.cues)
        cues.push(cue.group * 100 + cue.sample);
    }

    expect(last.tick).toBe(300);
    expect(last.timer).toBeLessThan(started.data.roundTimerTicks);
    expect(last.fighters[1].health).toBeLessThan(1000);
    expect(last.fighters[0].health).toBe(1000);
    // The fixture's attack state plays sound 5,0 on its first tick.
    expect(cues).toContain(500);
    await closeMatch(id, realCore);
  });

  it("ends the round at the time limit, then starts the next round on request", async () => {
    const started = await startMatch(request, realCore);
    if (!started.ok) throw new Error(started.error);
    const id = started.data.id;

    // The intros, the fight's 60 ticks, then the time over and the poses.
    let last: MatchSnapshot = started.data;
    for (let i = 0; i < 2000 && !last.roundEnded; i++) {
      const stepped = await tick(id, idle, realCore);
      if (!stepped.ok) throw new Error(stepped.error);
      last = stepped.data;
    }
    expect(last.roundEnded).toBe(true);
    expect(last.outcome).toBe("draw");
    expect(last.roundWinner).toBe(-1);
    expect(last.over).toBe(false);

    const refused = await tick(id, idle, realCore);
    expect(refused.ok).toBe(false);
    if (refused.ok) return;
    expect(refused.error).toContain("nextRound");

    const next = await nextRound(id, realCore);
    expect(next.ok).toBe(true);
    if (!next.ok) return;
    expect(next.data.round).toBe(2);
    expect(next.data.tick).toBe(0);
    expect(next.data.roundEnded).toBe(false);
    await closeMatch(id, realCore);
  });

  it("lets the core play player 2 when p2cpu is set: P2 acts on its own, which it never does without", async () => {
    // P2's state number over 600 idle ticks: it leaves standing (state 0)
    // only when something drives it.
    async function p2States(p2cpu: boolean): Promise<Set<number>> {
      const started = await startMatch({ ...request, p2cpu }, realCore);
      if (!started.ok) throw new Error(started.error);
      const states = new Set<number>();
      for (let i = 0; i < 600; i++) {
        const stepped = await tick(started.data.id, idle, realCore);
        if (!stepped.ok) break;
        states.add(stepped.data.fighters[1].stateNo);
        if (stepped.data.roundEnded) break;
      }
      await closeMatch(started.data.id, realCore);
      return states;
    }

    expect([...(await p2States(false))]).toEqual([0]);
    expect([...(await p2States(true))].some((s) => s !== 0)).toBe(true);
  });

  it("refuses a match of a character that was never mounted, with the core's reason", async () => {
    const started = await startMatch(
      { ...request, p1: "chars/nobody/nobody.def" },
      realCore,
    );

    expect(started.ok).toBe(false);
    if (started.ok) return;
    expect(started.error).toContain("chars/nobody/nobody.def");
  });

  it("refuses to tick a match that was closed", async () => {
    const started = await startMatch(request, realCore);
    if (!started.ok) throw new Error(started.error);
    await closeMatch(started.data.id, realCore);

    const stepped = await tick(started.data.id, idle, realCore);

    expect(stepped.ok).toBe(false);
  });
});

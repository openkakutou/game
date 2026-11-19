import { describe, expect, it } from "vitest";
import type { MatchSnapshot } from "../wasm/core-types.ts";
import { deriveMatchOutcome, deriveRoundOutcome } from "./outcome.ts";

type Result = Pick<
  MatchSnapshot,
  "round" | "roundEnded" | "roundWinner" | "over" | "winner"
>;

function result(overrides: Partial<Result> = {}): Result {
  return {
    round: 1,
    roundEnded: false,
    roundWinner: -1,
    over: false,
    winner: -1,
    ...overrides,
  };
}

describe("deriveRoundOutcome", () => {
  it("returns null while the round is still in progress", () => {
    expect(deriveRoundOutcome(result())).toBeNull();
  });

  it("resolves an ended round to the side the core names as its winner", () => {
    expect(
      deriveRoundOutcome(
        result({ round: 2, roundEnded: true, roundWinner: 1 }),
      ),
    ).toEqual({ round: 2, winner: "p2" });
    expect(
      deriveRoundOutcome(
        result({ round: 1, roundEnded: true, roundWinner: 0 }),
      ),
    ).toEqual({ round: 1, winner: "p1" });
  });

  it("resolves an ended round with no winner (double KO, equal-health timeout) to a draw", () => {
    expect(
      deriveRoundOutcome(
        result({ round: 3, roundEnded: true, roundWinner: -1 }),
      ),
    ).toEqual({ round: 3, winner: "draw" });
  });
});

describe("deriveMatchOutcome", () => {
  it("returns null while the match is still ongoing", () => {
    expect(deriveMatchOutcome(result({ roundEnded: true }))).toBeNull();
  });

  it("resolves to the match winner once the core reports the match over", () => {
    expect(
      deriveMatchOutcome(result({ roundEnded: true, over: true, winner: 1 })),
    ).toEqual({ winner: "p2" });
  });

  it("resolves a match over with no winner to a draw instead of leaving the result undefined", () => {
    expect(
      deriveMatchOutcome(result({ roundEnded: true, over: true, winner: -1 })),
    ).toEqual({ winner: "draw" });
  });
});

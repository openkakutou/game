// Pure, DOM-free derivation of a clear "who won" result from the core's
// match snapshot (`wasm/core-types.ts`): the core decides the outcome, the
// round winner and the match winner, and the bridge has already refused any
// value outside -1/0/1, so this only maps them to what the result screen
// shows.
import type { MatchSnapshot } from "../wasm/core-types.ts";

/** Who a round or the match resolved to: one side, or a draw (no winner) -- a double KO, a timeout at equal health. */
export type OutcomeWinner = "p1" | "p2" | "draw";

export interface RoundEndView {
  /** The round number that just ended. */
  round: number;
  winner: OutcomeWinner;
}

export interface MatchEndView {
  winner: OutcomeWinner;
}

function winnerFromSide(side: -1 | 0 | 1): OutcomeWinner {
  if (side === 0) return "p1";
  if (side === 1) return "p2";
  return "draw";
}

/** The round's result once it has ended, or `null` while it is still being fought. */
export function deriveRoundOutcome(
  snapshot: Pick<MatchSnapshot, "round" | "roundEnded" | "roundWinner">,
): RoundEndView | null {
  if (!snapshot.roundEnded) return null;
  return {
    round: snapshot.round,
    winner: winnerFromSide(snapshot.roundWinner),
  };
}

/** The match's result once the core reports it over, or `null` while it goes on. */
export function deriveMatchOutcome(
  snapshot: Pick<MatchSnapshot, "over" | "winner">,
): MatchEndView | null {
  if (!snapshot.over) return null;
  return { winner: winnerFromSide(snapshot.winner) };
}

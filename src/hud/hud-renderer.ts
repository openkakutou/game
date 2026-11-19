// Builds and updates the in-match HUD: a lifebar and power bar per player,
// plus the round display (round number, each player's round wins, best-of).
// Plain DOM (not drawn on the game canvas), built once by `createHud({ bestOf })` and
// mutated in place by `update()` -- no `createElement`/`appendChild` on the
// update path, matching this project's no-per-frame-allocation rendering
// discipline (see `.vibe/decisions/009`). It draws the core's match snapshot
// as it is: the values are the core's, already validated by
// `wasm/core-bridge.ts`, and only turned into bar fill percentages here.
import { onLocaleChange, t } from "../i18n/i18n.ts";
import type { MatchSnapshot } from "../wasm/core-types.ts";

/** Health a fighter starts a round with; a full lifebar. Mirrors `native/internal/game`'s `DefaultHealth`, which the core uses. */
export const MAX_HEALTH = 1000;

/** The power value of a full power bar. Mirrors `native/internal/game`'s `MaxPower`. */
export const MAX_POWER = 3000;

export interface HudOptions {
  /** The number of rounds the match is the best of, from the match setup. */
  bestOf: number;
  /** The health value that reads as a 100% full lifebar. Defaults to `MAX_HEALTH`. */
  maxHealth?: number;
  /** The power value that reads as a 100% full power bar. Defaults to `MAX_POWER`. */
  maxPower?: number;
}

interface HudRoundView {
  round: number;
  wins: [number, number];
  bestOf: number;
}

interface HudFighterView {
  health: number;
  power: number;
  /** Health as a percentage of the max, clamped to [0, 100]. */
  healthPercent: number;
  /** Power as a percentage of the max, clamped to [0, 100]. */
  powerPercent: number;
}

interface HudViewModel {
  fighters: [HudFighterView, HudFighterView];
  roundInfo: HudRoundView;
}

/** `value` as a percentage of `max`, clamped to [0, 100]; a non-positive `max` reads as 0% rather than dividing by zero. */
function clampPercent(value: number, max: number): number {
  if (max <= 0) return 0;
  const percent = (value / max) * 100;
  if (percent < 0) return 0;
  if (percent > 100) return 100;
  return percent;
}

export interface Hud {
  /** The HUD's root element -- the caller decides where to mount it (see `rendering/match-renderer.ts`). */
  element: HTMLElement;
  /** Applies one rendered frame's match snapshot. */
  update(snapshot: Pick<MatchSnapshot, "round" | "wins" | "fighters">): void;
  /** Releases this HUD's locale-change subscription. Call once when the match scene is torn down. */
  dispose(): void;
}

interface PlayerBarElements {
  wins: HTMLElement;
  health: HTMLElement;
  healthFill: HTMLElement;
  healthValueText: HTMLElement;
  power: HTMLElement;
  powerFill: HTMLElement;
  powerValueText: HTMLElement;
}

function buildPlayerColumn(index: 0 | 1): {
  element: HTMLElement;
  bars: PlayerBarElements;
} {
  const playerNumber = index + 1;
  const column = document.createElement("div");
  column.className = `hud__player hud__player--${playerNumber}`;

  const wins = document.createElement("p");
  wins.className = "hud__wins";
  column.appendChild(wins);

  function buildBar(kind: "health" | "power"): {
    bar: HTMLElement;
    fill: HTMLElement;
    valueText: HTMLElement;
  } {
    const bar = document.createElement("div");
    bar.className = `hud__${kind} hud__${kind}--${playerNumber}`;
    bar.setAttribute("role", "progressbar");
    bar.setAttribute("aria-valuemin", "0");

    const fill = document.createElement("div");
    fill.className = `hud__${kind}-fill hud__${kind}-fill--${playerNumber}`;
    bar.appendChild(fill);

    const valueText = document.createElement("span");
    valueText.className = "hud__value-text";
    bar.appendChild(valueText);

    column.appendChild(bar);
    return { bar, fill, valueText };
  }

  const health = buildBar("health");
  const power = buildBar("power");

  return {
    element: column,
    bars: {
      wins,
      health: health.bar,
      healthFill: health.fill,
      healthValueText: health.valueText,
      power: power.bar,
      powerFill: power.fill,
      powerValueText: power.valueText,
    },
  };
}

function roundInfoEqual(a: HudRoundView, b: HudRoundView): boolean {
  return (
    a.round === b.round && a.wins[0] === b.wins[0] && a.wins[1] === b.wins[1]
  );
}

/**
 * Builds a new HUD. Nothing is mounted anywhere yet -- the caller appends
 * `element` wherever it belongs (see `rendering/match-renderer.ts`'s
 * `renderMatch`).
 */
export function createHud(options: HudOptions): Hud {
  const maxHealth = options.maxHealth ?? MAX_HEALTH;
  const maxPower = options.maxPower ?? MAX_POWER;

  const element = document.createElement("div");
  element.className = "hud";

  const content = document.createElement("div");
  content.className = "hud__content";
  element.appendChild(content);

  const player1 = buildPlayerColumn(0);
  content.appendChild(player1.element);

  const round = document.createElement("div");
  round.className = "hud__round";
  const roundText = document.createElement("p");
  round.appendChild(roundText);
  content.appendChild(round);

  const player2 = buildPlayerColumn(1);
  content.appendChild(player2.element);

  const players: [PlayerBarElements, PlayerBarElements] = [
    player1.bars,
    player2.bars,
  ];

  const announcement = document.createElement("p");
  announcement.className = "hud__announcement";
  announcement.setAttribute("aria-live", "polite");
  element.appendChild(announcement);

  let last: HudViewModel | null = null;
  let lastAnnouncedRoundInfo: HudRoundView | null = null;

  function setBarLabels(): void {
    players[0].health.setAttribute(
      "aria-label",
      t("hud.playerHealth", "Player {{number}} health", { number: "1" }),
    );
    players[1].health.setAttribute(
      "aria-label",
      t("hud.playerHealth", "Player {{number}} health", { number: "2" }),
    );
    players[0].power.setAttribute(
      "aria-label",
      t("hud.playerPower", "Player {{number}} power", { number: "1" }),
    );
    players[1].power.setAttribute(
      "aria-label",
      t("hud.playerPower", "Player {{number}} power", { number: "2" }),
    );
  }
  setBarLabels();

  function renderBar(
    bar: PlayerBarElements["health"],
    fill: HTMLElement,
    valueText: HTMLElement,
    value: number,
    max: number,
    percent: number,
  ): void {
    bar.setAttribute("aria-valuemax", String(max));
    bar.setAttribute("aria-valuenow", String(value));
    const text = t("hud.valueOfMax", "{{value}} / {{max}}", {
      value: String(value),
      max: String(max),
    });
    bar.setAttribute("aria-valuetext", text);
    fill.style.width = `${percent}%`;
    valueText.textContent = text;
  }

  function renderRoundText(info: HudRoundView): void {
    roundText.textContent = t(
      "hud.roundInfo",
      "Round {{round}} · Best of {{bestOf}}",
      { round: String(info.round), bestOf: String(info.bestOf) },
    );
    players[0].wins.textContent = t("hud.wins", "Wins: {{count}}", {
      count: String(info.wins[0]),
    });
    players[1].wins.textContent = t("hud.wins", "Wins: {{count}}", {
      count: String(info.wins[1]),
    });
  }

  function renderBars(view: HudViewModel): void {
    for (const side of [0, 1] as const) {
      const fighter = view.fighters[side];
      renderBar(
        players[side].health,
        players[side].healthFill,
        players[side].healthValueText,
        fighter.health,
        maxHealth,
        fighter.healthPercent,
      );
      renderBar(
        players[side].power,
        players[side].powerFill,
        players[side].powerValueText,
        fighter.power,
        maxPower,
        fighter.powerPercent,
      );
    }
  }

  function render(view: HudViewModel): void {
    renderBars(view);
    renderRoundText(view.roundInfo);

    if (
      !lastAnnouncedRoundInfo ||
      !roundInfoEqual(lastAnnouncedRoundInfo, view.roundInfo)
    ) {
      announcement.textContent = t(
        "hud.roundAnnouncement",
        "Round {{round}} started. {{p1Wins}} - {{p2Wins}}.",
        {
          round: String(view.roundInfo.round),
          p1Wins: String(view.roundInfo.wins[0]),
          p2Wins: String(view.roundInfo.wins[1]),
        },
      );
      lastAnnouncedRoundInfo = view.roundInfo;
    }
    last = view;
  }

  function fighterView(fighter: {
    health: number;
    power: number;
  }): HudFighterView {
    return {
      health: fighter.health,
      power: fighter.power,
      healthPercent: clampPercent(fighter.health, maxHealth),
      powerPercent: clampPercent(fighter.power, maxPower),
    };
  }

  function update(
    snapshot: Pick<MatchSnapshot, "round" | "wins" | "fighters">,
  ): void {
    render({
      fighters: [
        fighterView(snapshot.fighters[0]),
        fighterView(snapshot.fighters[1]),
      ],
      roundInfo: {
        round: snapshot.round,
        wins: [snapshot.wins[0], snapshot.wins[1]],
        bestOf: options.bestOf,
      },
    });
  }

  const unsubscribe = onLocaleChange(() => {
    setBarLabels();
    if (last) {
      // The value-text strings depend on t(), so they need a fresh render on
      // a locale change -- re-running the render with the last known values,
      // since no new match data arrived.
      renderBars(last);
      renderRoundText(last.roundInfo);
    }
  });

  return {
    element,
    update,
    dispose: unsubscribe,
  };
}

import { describe, expect, it } from "vitest";

import type { AppDTO, TutorialPhase } from "../../api/types";
import { tutorialGuidance } from "./tutorial-copy";

function app(step: number, phase: TutorialPhase = ""): AppDTO {
  return {
    mode: "TUTORIAL",
    tutorial_step: step,
    tutorial_phase: phase,
    tutorial_portal_id: 42,
    tutorial_plane_id: 3,
    tutorial_observer_id: 7,
    expected_action: null,
  };
}

describe("tutorialGuidance", () => {
  it("tells the player to inspect the highlighted Portal card", () => {
    const guidance = tutorialGuidance(app(1));
    expect(guidance?.instruction).toBe(
      "Select the highlighted Portal on the Dashboard to inspect it.",
    );
    expect(JSON.stringify(guidance)).not.toMatch(
      /Open Details|Details button/i,
    );
  });

  it("keeps Step 0 general and free of later prices/rules", () => {
    const guidance = tutorialGuidance(app(0));
    expect(guidance).toMatchObject({
      title: "Welcome to Omenpath Research Lab",
      cta: "BEGIN_PRACTICE",
      waiting: false,
    });
    const copy = JSON.stringify(guidance);
    expect(copy).toMatch(
      /85\/85.*Dashboard.*Lab Summary.*Observers.*seven Slots.*Needs Attention.*Details.*Event Log/i,
    );
    for (const forbidden of [
      "SEND",
      "RECALL",
      "5 Energy",
      "20 Energy",
      "30 Energy",
      "decay",
      "CRITICAL",
      "LOST",
      "OUTBOUND",
    ])
      expect(copy).not.toContain(forbidden);
  });

  it.each([
    [
      1,
      "Read a Portal",
      /Portal Energy.*Lab Energy.*individual.*Time.*Stability.*Risk.*Recommendation.*History/i,
    ],
    [2, "Wait for the corridor", /Creatures.*SEND.*every 2 seconds.*wait/i],
    [
      3,
      "Send an Observer",
      /SEND costs 0.*AVAILABLE.*5.*15 seconds.*OUTBOUND/i,
    ],
    [
      4,
      "Stabilize an Omenpath",
      /costs 20.*0.*100.*\+1.*UNSTABLE.*85.*15 Portal Energy.*MEDIUM.*LOW/i,
    ],
    [
      5,
      "Respect CRITICAL risk",
      /CRITICAL.*SEND.*RECALL.*CLOSE costs 5.*CLOSED.*COLLAPSED.*drains Lab Energy.*20-second Override/i,
    ],
    [
      7,
      "Survive the return",
      /EXPLORED.*successful return.*terminal Portal.*LOST.*wait/i,
    ],
    [
      8,
      "Inspect the Event Log",
      /Global Log.*Portal History.*same source.*open Event Log/i,
    ],
    [
      9,
      "Training complete",
      /Natural Portals.*cost 30.*5-second sync.*first automatic return.*continuity/i,
    ],
  ])("describes authoritative Step %i", (step, title, meaning) => {
    const guidance = tutorialGuidance(app(step as number));
    expect(guidance?.title).toBe(title);
    expect(JSON.stringify(guidance)).toMatch(meaning as RegExp);
  });

  it.each([
    ["SEND_REPLACEMENT", /replacement.*Observer/i, false],
    ["WAIT_RESEARCH", /wait.*research/i, true],
    ["RECALL_READY", /RECALL costs 0.*longest-waiting.*INBOUND/i, false],
  ] as const)("uses Step 6 phase %s", (phase, meaning, waiting) => {
    const guidance = tutorialGuidance(app(6, phase));
    expect(guidance?.title).toBe("Bring the Observer home");
    expect(JSON.stringify(guidance)).toMatch(meaning);
    expect(guidance?.waiting).toBe(waiting);
  });

  it("returns null in LIVE mode", () => {
    expect(tutorialGuidance({ ...app(9), mode: "LIVE" })).toBeNull();
  });
});

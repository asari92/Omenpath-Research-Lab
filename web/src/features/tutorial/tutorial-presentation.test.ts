import { describe, expect, it } from "vitest";
import { snapshotAt } from "../../test/builders";
import {
  initialPresentation,
  tutorialPresentation,
} from "./tutorial-presentation";

const app = (step: number) => ({ ...snapshotAt().app, tutorial_step: step });
describe("tutorial presentation reducer", () => {
  it("one Forward jumps across all browsed history directly to current", () => {
    let state = initialPresentation(app(1));
    for (let step = 2; step <= 4; step++)
      state = tutorialPresentation(state, { type: "snapshot", app: app(step) });
    state = tutorialPresentation(state, { type: "back" });
    state = tutorialPresentation(state, { type: "back" });
    state = tutorialPresentation(state, { type: "back" });
    expect(state.shown?.tutorial_step).toBe(1);
    expect(
      tutorialPresentation(state, { type: "forward" }).shown?.tutorial_step,
    ).toBe(4);
  });
  it("keeps earlier shown lessons available when LOST repeats Step 6, without exposing future Step 7", () => {
    let state = initialPresentation(app(1));
    for (let step = 2; step <= 7; step++)
      state = tutorialPresentation(state, { type: "snapshot", app: app(step) });
    state = tutorialPresentation(state, {
      type: "snapshot",
      app: { ...app(6), tutorial_phase: "SEND_REPLACEMENT" },
    });
    expect(state.history.map((item) => item.tutorial_step)).toEqual([
      1, 2, 3, 4, 5, 6,
    ]);
    expect(state.shown?.tutorial_phase).toBe("SEND_REPLACEMENT");
    expect(tutorialPresentation(state, { type: "forward" })).toBe(state);
    expect(
      tutorialPresentation(state, { type: "back" }).shown?.tutorial_step,
    ).toBe(5);
  });
  it("keeps authority separate, queues multiple skipped cards, and ignores stale generations", () => {
    const initial = initialPresentation(app(1));
    const jumped = tutorialPresentation(initial, {
      type: "snapshot",
      app: app(4),
    });
    expect(jumped.current?.tutorial_step).toBe(4);
    expect(jumped.shown?.tutorial_step).toBe(2);
    expect(jumped.history.map((a) => a.tutorial_step)).toEqual([1, 2]);
    expect(jumped.queue.map((a) => a.tutorial_step)).toEqual([3, 4]);
    expect(
      tutorialPresentation(jumped, {
        type: "elapsed",
        generation: initial.generation,
      }),
    ).toBe(jumped);
    const advanced = tutorialPresentation(jumped, {
      type: "elapsed",
      generation: jumped.generation,
    });
    expect(advanced.shown?.tutorial_step).toBe(3);
    expect(advanced.replaying).toBe(true);
    const live = tutorialPresentation(advanced, {
      type: "snapshot",
      app: { ...app(9), mode: "LIVE" },
    });
    expect(live.history).toEqual([]);
    expect(live.queue).toEqual([]);
    expect(live.shown).toBeNull();
  });
  it("Back only visits shown cards and Forward cannot bypass a replay or future task", () => {
    let state = initialPresentation(app(1));
    state = tutorialPresentation(state, { type: "snapshot", app: app(3) });
    state = tutorialPresentation(state, { type: "back" });
    expect(state.shown?.tutorial_step).toBe(1);
    state = tutorialPresentation(state, { type: "forward" });
    expect(state.shown?.tutorial_step).toBe(2);
    expect(tutorialPresentation(state, { type: "forward" })).toBe(state);
  });
});

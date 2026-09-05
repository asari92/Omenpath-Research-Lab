import { describe, expect, it } from "vitest";
import { snapshotAt } from "../../test/builders";
import { initialPresentation, tutorialPresentation } from "./tutorial-presentation";

const app = (step: number) => ({ ...snapshotAt().app, tutorial_step: step });
describe("tutorial presentation reducer", () => {
  it("keeps authority separate, queues multiple skipped cards, and ignores stale generations", () => {
    const initial = initialPresentation(app(1));
    const jumped = tutorialPresentation(initial, { type: "snapshot", app: app(4) });
    expect(jumped.current?.tutorial_step).toBe(4);
    expect(jumped.shown?.tutorial_step).toBe(2);
    expect(jumped.history.map(a => a.tutorial_step)).toEqual([1, 2]);
    expect(jumped.queue.map(a => a.tutorial_step)).toEqual([3, 4]);
    expect(tutorialPresentation(jumped, { type: "elapsed", generation: initial.generation })).toBe(jumped);
    const advanced = tutorialPresentation(jumped, { type: "elapsed", generation: jumped.generation });
    expect(advanced.shown?.tutorial_step).toBe(3);
    expect(advanced.replaying).toBe(true);
    const live = tutorialPresentation(advanced, { type: "snapshot", app: {...app(9), mode:"LIVE"} });
    expect(live.history).toEqual([]);
    expect(live.queue).toEqual([]);
    expect(live.shown).toBeNull();
  });
  it("Back only visits shown cards and Forward cannot bypass a replay or future task", () => {
    let state = initialPresentation(app(1));
    state = tutorialPresentation(state, { type:"snapshot", app:app(3) });
    state = tutorialPresentation(state, { type:"back" });
    expect(state.shown?.tutorial_step).toBe(1);
    state = tutorialPresentation(state, { type:"forward" });
    expect(state.shown?.tutorial_step).toBe(2);
    expect(tutorialPresentation(state, {type:"forward"})).toBe(state);
  });
});

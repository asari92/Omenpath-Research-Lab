import { beforeEach, describe, expect, it } from "vitest";

import type { AppDTO } from "../../api/types";
import {
  consumeNavigationSignal,
  matchingNavigationSignal,
  recordNavigationIntent,
  resetNavigationIntentForTests,
} from "./navigation-signal";

function expectedDetails(id: number): AppDTO {
  return {
    mode: "TUTORIAL",
    tutorial_step: 1,
    tutorial_phase: "",
    tutorial_portal_id: id,
    tutorial_plane_id: 1,
    tutorial_observer_id: null,
    expected_action: "OPEN_PORTAL_DETAILS",
  };
}

describe("navigation signal", () => {
  beforeEach(resetNavigationIntentForTests);

  it("matches only the expected target Portal", () => {
    expect(
      matchingNavigationSignal(expectedDetails(42), { kind: "portal", id: 42 }),
    ).toEqual({
      signal: "PORTAL_DETAILS_OPENED",
      portal_id: 42,
    });
    expect(
      matchingNavigationSignal(expectedDetails(42), { kind: "portal", id: 43 }),
    ).toBeNull();
    expect(
      matchingNavigationSignal(
        { ...expectedDetails(42), expected_action: "OPEN_EVENT_LOG" },
        { kind: "portal", id: 42 },
      ),
    ).toBeNull();
  });

  it("requires and consumes a one-shot in-memory navigation intent", () => {
    recordNavigationIntent({ kind: "portal", id: 42 });
    expect(
      consumeNavigationSignal(expectedDetails(42), { kind: "portal", id: 42 }),
    ).toEqual({
      signal: "PORTAL_DETAILS_OPENED",
      portal_id: 42,
    });
    expect(
      consumeNavigationSignal(expectedDetails(42), { kind: "portal", id: 42 }),
    ).toBeNull();
  });

  it("matches Event Log only at the expected Tutorial step", () => {
    const app = {
      ...expectedDetails(42),
      tutorial_step: 8,
      expected_action: "OPEN_EVENT_LOG" as const,
    };
    expect(matchingNavigationSignal(app, { kind: "events" })).toEqual({
      signal: "EVENT_LOG_OPENED",
    });
    expect(
      matchingNavigationSignal(expectedDetails(42), { kind: "events" }),
    ).toBeNull();
  });
});

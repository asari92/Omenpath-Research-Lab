import { describe, expect, it } from "vitest";

import { snapshotAt } from "../../test/builders";
import {
  isExpectedCriticalSend,
  tutorialCommandTarget,
} from "./tutorial-actions";

describe("Tutorial action rules", () => {
  it("force-enables only the exact Step 5 critical SEND target", () => {
    const app = snapshotAt().app;
    app.tutorial_step = 5;
    app.expected_action = "ATTEMPT_CRITICAL_SEND";
    app.tutorial_portal_id = 42;
    expect(isExpectedCriticalSend(app, 42)).toBe(true);
    expect(isExpectedCriticalSend(app, 43)).toBe(false);
    expect(isExpectedCriticalSend({ ...app, mode: "LIVE" }, 42)).toBe(false);
    expect(
      isExpectedCriticalSend({ ...app, expected_action: "SEND_OBSERVER" }, 42),
    ).toBe(false);
  });

  it("maps only actionable tutorial states to their exact command", () => {
    const app = snapshotAt().app;
    app.tutorial_portal_id = 42;
    expect(
      tutorialCommandTarget({ ...app, expected_action: "SEND_OBSERVER" }),
    ).toEqual({ portalId: 42, command: "SEND" });
    expect(
      tutorialCommandTarget({ ...app, expected_action: "STABILIZE" }),
    ).toEqual({ portalId: 42, command: "STABILIZE" });
    expect(
      tutorialCommandTarget({ ...app, expected_action: "RECALL_OBSERVER" }),
    ).toEqual({ portalId: 42, command: "RECALL" });
    expect(
      tutorialCommandTarget({ ...app, expected_action: "WAIT_RESEARCH" }),
    ).toBeNull();
  });
});

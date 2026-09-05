import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Diagnostics } from "./Diagnostics";

describe("recommendation presentation", () => {
  it.each([
    [
      "SEND OBSERVER",
      "safe",
      "✓",
      "Begin an expedition with the advised travel margin.",
    ],
    [
      "RECALL OBSERVER",
      "safe",
      "✓",
      "Bring a waiting Observer home with the advised travel margin.",
    ],
    [
      "LEAVE OPEN",
      "suggested",
      "◇",
      "Keep this passage available; safety is not guaranteed.",
    ],
    [
      "WAIT FOR CORRIDOR",
      "suggested",
      "◇",
      "Let creatures clear the passage before Observer travel.",
    ],
    [
      "STABILIZE",
      "urgent",
      "!",
      "Remove instability and reinforce the passage for the mission.",
    ],
    [
      "CLOSE",
      "urgent",
      "!",
      "Close this passage when ready; any required confirmation still applies.",
    ],
    [
      null,
      "unavailable",
      "—",
      "No current recommendation for a terminal Portal.",
    ],
  ] as const)(
    "explains %s with explicit semantic state and icon",
    (recommendation, state, icon, explanation) => {
      render(
        <Diagnostics
          risk={recommendation ? "LOW" : null}
          recommendation={recommendation}
        />,
      );
      const advice = screen.getByLabelText("Recommendation guidance");
      expect(advice).toHaveAttribute("data-advice-state", state);
      expect(advice).toHaveTextContent(explanation);
      expect(advice).toHaveTextContent(icon);
      expect(advice).toHaveTextContent(new RegExp(state, "i"));
    },
  );
});

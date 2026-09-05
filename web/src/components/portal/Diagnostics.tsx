import type { Recommendation, RiskLevel } from "../../api/types";
import styles from "./Diagnostics.module.css";

const advicePresentation = {
  "SEND OBSERVER": {
    state: "safe",
    explanation: "Begin an expedition with the advised travel margin.",
  },
  "RECALL OBSERVER": {
    state: "safe",
    explanation:
      "Bring a waiting Observer home with the advised travel margin.",
  },
  "LEAVE OPEN": {
    state: "suggested",
    explanation: "Keep this passage available; safety is not guaranteed.",
  },
  "WAIT FOR CORRIDOR": {
    state: "suggested",
    explanation: "Let creatures clear the passage before Observer travel.",
  },
  STABILIZE: {
    state: "urgent",
    explanation:
      "Remove instability and reinforce the passage for the mission.",
  },
  CLOSE: {
    state: "urgent",
    explanation:
      "Close this passage when ready; any required confirmation still applies.",
  },
} as const satisfies Record<
  Recommendation,
  { state: string; explanation: string }
>;
const adviceStates = {
  safe: { icon: "✓", label: "Safe margin" },
  suggested: { icon: "◇", label: "Suggested" },
  urgent: { icon: "!", label: "Urgent" },
  unavailable: { icon: "—", label: "Unavailable" },
} as const;

export function Diagnostics({
  risk,
  recommendation,
}: {
  risk: RiskLevel | null;
  recommendation: Recommendation | null;
}) {
  const advice = recommendation
    ? advicePresentation[recommendation]
    : {
        state: "unavailable" as const,
        explanation: "No current recommendation for a terminal Portal.",
      };
  const presentation = adviceStates[advice.state];
  return (
    <section className={styles.diagnostics}>
      <h2>Diagnostics</h2>
      <dl>
        <div>
          <dt>Risk</dt>
          <dd data-risk={risk}>
            <span aria-hidden="true">
              {risk
                ? { LOW: "●", MEDIUM: "◐", HIGH: "▲", CRITICAL: "⚠" }[risk]
                : "—"}{" "}
            </span>
            {risk ?? "Not applicable"}
          </dd>
        </div>
        <div>
          <dt>Recommendation</dt>
          <dd data-recommendation={recommendation}>
            {recommendation ?? "Not applicable"}
          </dd>
        </div>
      </dl>
      <p
        className={styles.advice}
        aria-label="Recommendation guidance"
        data-advice-state={advice.state}
      >
        <strong>
          <span aria-hidden="true">{presentation.icon}</span>{" "}
          {presentation.label}
        </strong>{" "}
        {advice.explanation}
      </p>
      <details>
        <summary>How Risk Works</summary>
        <p data-testid="risk-explanation">
          LOW means comfortable operating conditions. MEDIUM calls for
          attention. HIGH means the corridor may soon become unsafe. CRITICAL
          means Observer transit is forbidden. Recommendation is guidance, not a
          restriction; command availability remains authoritative.
        </p>
      </details>
    </section>
  );
}

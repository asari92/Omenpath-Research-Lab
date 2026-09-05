import type { Recommendation, RiskLevel } from "../../api/types";
import styles from "./Diagnostics.module.css";

export function Diagnostics({
  risk,
  recommendation,
}: {
  risk: RiskLevel | null;
  recommendation: Recommendation | null;
}) {
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
            <span aria-hidden="true">
              {recommendation === "CLOSE"
                ? "⊗"
                : recommendation === "STABILIZE"
                  ? "✦"
                  : "→"}{" "}
            </span>
            {recommendation ?? "Not applicable"}
          </dd>
        </div>
      </dl>
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

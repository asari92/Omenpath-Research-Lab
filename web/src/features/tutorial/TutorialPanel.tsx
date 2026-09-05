import { useSnapshotState } from "../../state/SnapshotProvider";
import { tutorialGuidance } from "./tutorial-copy";
import styles from "./TutorialPanel.module.css";

export function TutorialPanel() {
  const { snapshot } = useSnapshotState();
  if (!snapshot) return null;
  const guidance = tutorialGuidance(snapshot.app);
  if (!guidance) return null;
  const ctaLabel =
    guidance.cta === "BEGIN_PRACTICE"
      ? "Begin Practice"
      : guidance.cta === "TRY_SEND"
        ? "Try Send"
        : guidance.cta === "START_LIVE"
          ? "Start Live"
          : null;
  return (
    <aside aria-label="Tutorial" className={styles.panel}>
      <header>
        <span>Step {snapshot.app.tutorial_step}</span>
        <strong>{guidance.title}</strong>
        {guidance.waiting && <span>Waiting</span>}
      </header>
      <p className={styles.instruction}>{guidance.instruction}</p>
      <p>{guidance.explanation[0]}</p>
      <details>
        <summary>More context</summary>
        {guidance.explanation.slice(1).map((paragraph) => (
          <p key={paragraph}>{paragraph}</p>
        ))}
        <p>
          Targets: Portal {snapshot.app.tutorial_portal_id ?? "—"}, Plane{" "}
          {snapshot.app.tutorial_plane_id ?? "—"}, Observer{" "}
          {snapshot.app.tutorial_observer_id ?? "—"}
        </p>
      </details>
      {ctaLabel && <button type="button">{ctaLabel}</button>}
    </aside>
  );
}

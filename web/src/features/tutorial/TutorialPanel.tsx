import { ApiError } from "../../api/errors";
import { useFeedback } from "../../components/feedback/FeedbackProvider";
import { useSnapshotContext } from "../../state/SnapshotProvider";
import { useSnapshotState } from "../../state/SnapshotProvider";
import { isExpectedCriticalSend } from "./tutorial-actions";
import { tutorialGuidance } from "./tutorial-copy";
import styles from "./TutorialPanel.module.css";
import { commandsEnabled } from "../../state/command-health";

export function TutorialPanel() {
  const { api, store } = useSnapshotContext();
  const state = useSnapshotState();
  const { commandKeys, snapshot } = state;
  const feedback = useFeedback();
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
  const ctaKey = guidance.cta ? `TUTORIAL:${guidance.cta}` : null;
  const ctaBusy = ctaKey !== null && commandKeys.has(ctaKey);

  const runCTA = async () => {
    if (!commandsEnabled(store.getState())) return;
    if (!guidance.cta || !ctaKey || store.getState().commandKeys.has(ctaKey)) {
      return;
    }
    const current = store.getState().snapshot;
    if (!current || current.app.mode !== "TUTORIAL") return;
    store.beginCommand(ctaKey);
    try {
      if (
        guidance.cta === "BEGIN_PRACTICE" &&
        current.app.expected_action === "COMPLETE_INTRO"
      ) {
        store.acceptSnapshot(
          await api.tutorialSignal({ signal: "TUTORIAL_INTRO_COMPLETED" }),
        );
      } else if (
        guidance.cta === "TRY_SEND" &&
        current.app.tutorial_portal_id !== null &&
        isExpectedCriticalSend(current.app, current.app.tutorial_portal_id)
      ) {
        try {
          store.acceptSnapshot(
            await api.sendObserver(current.app.tutorial_portal_id, false),
          );
        } catch (error: unknown) {
          if (
            !(error instanceof ApiError) ||
            error.code !== "PORTAL_CRITICAL_RISK"
          ) {
            throw error;
          }
        }
      } else if (
        guidance.cta === "START_LIVE" &&
        current.app.expected_action === "START_LIVE"
      ) {
        store.acceptSnapshot(await api.startLive());
      }
    } catch (error: unknown) {
      feedback.notify(
        error instanceof Error ? error.message : "Tutorial action failed",
      );
    } finally {
      store.endCommand(ctaKey);
    }
  };

  const reset = async () => {
    if (!commandsEnabled(store.getState())) return;
    const key = "TUTORIAL:RESET";
    if (store.getState().commandKeys.has(key)) return;
    store.beginCommand(key);
    try {
      const confirmed = await feedback.confirmAction(
        "Reset Tutorial",
        "Tutorial progress",
        "Energy, Observers, exploration and Tutorial Event history will be reset by the Laboratory.",
      );
      if (!confirmed || !commandsEnabled(store.getState())) return;
      store.acceptSnapshot(await api.resetTutorial());
    } catch (error: unknown) {
      feedback.notify(
        error instanceof Error ? error.message : "Tutorial reset failed",
      );
    } finally {
      store.endCommand(key);
    }
  };
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
      <div className={styles.actions}>
        {ctaLabel && (
          <button
            disabled={ctaBusy || !commandsEnabled(state)}
            onClick={() => void runCTA()}
            type="button"
          >
            {ctaBusy ? "Working…" : ctaLabel}
          </button>
        )}
        <button
          disabled={
            commandKeys.has("TUTORIAL:RESET") || !commandsEnabled(state)
          }
          onClick={() => void reset()}
          type="button"
        >
          Reset Tutorial
        </button>
      </div>
    </aside>
  );
}

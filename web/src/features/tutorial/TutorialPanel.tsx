import { ApiError } from "../../api/errors";
import { useFeedback } from "../../components/feedback/FeedbackProvider";
import { useSnapshotContext } from "../../state/SnapshotProvider";
import { useSnapshotState } from "../../state/SnapshotProvider";
import { isExpectedCriticalSend } from "./tutorial-actions";
import { tutorialGuidance } from "./tutorial-copy";
import styles from "./TutorialPanel.module.css";
import { commandsEnabled } from "../../state/command-health";
import { useTutorialPresentation } from "./useTutorialPresentation";
import { tutorialSystem } from "./tutorial-copy";
import { useNavigate } from "react-router-dom";

export function TutorialPanel() {
  const navigate = useNavigate();
  const { api, store } = useSnapshotContext();
  const state = useSnapshotState();
  const { commandKeys, snapshot } = state;
  const feedback = useFeedback();
  const { presentation, back, forward } = useTutorialPresentation(
    snapshot?.app ?? null,
  );
  const shown = presentation.shown;
  if (!snapshot || !shown) return null;
  const guidance = tutorialGuidance(shown);
  if (!guidance) return null;
  const historical =
    presentation.replaying ||
    presentation.cursor < presentation.history.length - 1;
  const system = tutorialSystem(shown);
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
    if (historical) return;
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
        const next = await api.startLive();
        store.acceptSnapshot(next);
        if (next.app.mode === "LIVE") navigate("/");
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
    <aside
      aria-label="Tutorial"
      className={styles.panel}
      data-completed={historical || undefined}
    >
      <header>
        <span>Step {shown.tutorial_step}</span>
        <strong>{guidance.title}</strong>
        {historical ? (
          <span>Completed</span>
        ) : (
          guidance.waiting && <span>Waiting</span>
        )}
      </header>
      <p className={styles.instruction}>{guidance.instruction}</p>
      {guidance.explanation.map((paragraph) => (
        <p key={paragraph}>{paragraph}</p>
      ))}
      <p>System: {system.action}</p>
      <p>Complete when: {system.completion}</p>
      <div className={styles.actions}>
        <button
          type="button"
          onClick={back}
          disabled={presentation.cursor === 0}
        >
          Back
        </button>
        <button
          type="button"
          onClick={forward}
          disabled={presentation.cursor === presentation.history.length - 1}
        >
          Forward
        </button>
        <span>
          {historical
            ? `Current objective: Step ${snapshot.app.tutorial_step}`
            : "Current objective"}
        </span>
        {ctaLabel && !historical && (
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

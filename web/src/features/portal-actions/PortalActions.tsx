import type { QuickActionsDTO } from "../../api/types";
import { useFeedback } from "../../components/feedback/FeedbackProvider";
import { unavailableActionCopy } from "./action-copy";
import styles from "./PortalActions.module.css";

export type PortalCommand = "STABILIZE" | "CLOSE" | "SEND" | "RECALL";

interface CommandView {
  command: PortalCommand;
  label: string;
  shortLabel: string;
  available: boolean;
  reason: string | null;
}

export interface PortalActionsProps {
  portalId: number | null;
  quickActions: QuickActionsDTO | null;
  busyKey?: string | null;
  onCommand?(command: PortalCommand): void | Promise<void>;
  expectedCriticalSend?: boolean;
  highlightedCommand?: PortalCommand | null;
  offline?: boolean;
  outcome?: "success" | "error" | null;
}

export function PortalActions({
  portalId,
  quickActions,
  busyKey = null,
  onCommand,
  expectedCriticalSend = false,
  highlightedCommand = null,
  offline = false,
  outcome = null,
}: PortalActionsProps) {
  const feedback = useFeedback();
  const emptyReason = "No Portal occupies this Slot";
  const commands: CommandView[] = [
    {
      command: "STABILIZE",
      label: "Stabilize",
      shortLabel: "Stabilize",
      available: quickActions?.can_stabilize ?? false,
      reason: quickActions?.stabilize_unavailable_reason ?? emptyReason,
    },
    {
      command: "SEND",
      label: "Send Observer",
      shortLabel: "Send",
      available: quickActions?.can_send_observer ?? false,
      reason: quickActions?.send_observer_unavailable_reason ?? emptyReason,
    },
    {
      command: "RECALL",
      label: "Recall Observer",
      shortLabel: "Recall",
      available: quickActions?.can_recall_observer ?? false,
      reason: quickActions?.recall_observer_unavailable_reason ?? emptyReason,
    },
    {
      command: "CLOSE",
      label: "Close",
      shortLabel: "Close",
      available: quickActions?.can_close ?? false,
      reason: quickActions?.close_unavailable_reason ?? emptyReason,
    },
  ];
  if (expectedCriticalSend && commands[1].reason === "PORTAL_CRITICAL_RISK") {
    commands[1] = { ...commands[1], available: true };
  }

  return (
    <div className={styles.wrapper} data-outcome={outcome ?? undefined}>
      <div aria-label="Portal commands" className={styles.actions} role="group">
        {commands.map((item) => {
          const key = `${portalId ?? "empty"}:${item.command}`;
          const busy = busyKey === key;
          return (
            <button
              aria-label={item.label}
              aria-disabled={!item.available || offline}
              aria-busy={busy}
              className={item.available ? styles.available : styles.unavailable}
              data-command={item.command}
              data-tutorial-command={
                highlightedCommand === item.command || undefined
              }
              disabled={busy || portalId === null}
              key={item.command}
              onClick={() => {
                if (offline) {
                  feedback.notify(
                    "Planar paths unstable. Wait for the link to recover.",
                  );
                  return;
                }
                if (!item.available || portalId === null) {
                  feedback.notify(
                    item.reason === emptyReason
                      ? emptyReason
                      : unavailableActionCopy(item.reason ?? "UNKNOWN"),
                  );
                  return;
                }
                void Promise.resolve(onCommand?.(item.command)).catch(
                  (error: unknown) => {
                    feedback.notify(
                      error instanceof Error ? error.message : "Command failed",
                    );
                  },
                );
              }}
              type="button"
            >
              {busy ? "Working…" : item.shortLabel}
            </button>
          );
        })}
      </div>
    </div>
  );
}

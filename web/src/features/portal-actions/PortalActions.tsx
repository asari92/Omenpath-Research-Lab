import { useState } from "react";

import type { QuickActionsDTO } from "../../api/types";
import { unavailableActionCopy } from "./action-copy";
import styles from "./PortalActions.module.css";

export type PortalCommand = "STABILIZE" | "CLOSE" | "SEND" | "RECALL";

interface CommandView {
  command: PortalCommand;
  label: string;
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
}

export function PortalActions({
  portalId,
  quickActions,
  busyKey = null,
  onCommand,
  expectedCriticalSend = false,
  highlightedCommand = null,
}: PortalActionsProps) {
  const [notice, setNotice] = useState<string | null>(null);
  const emptyReason = "No Portal occupies this Slot";
  const commands: CommandView[] = [
    {
      command: "STABILIZE",
      label: "Stabilize",
      available: quickActions?.can_stabilize ?? false,
      reason: quickActions?.stabilize_unavailable_reason ?? emptyReason,
    },
    {
      command: "CLOSE",
      label: "Close",
      available: quickActions?.can_close ?? false,
      reason: quickActions?.close_unavailable_reason ?? emptyReason,
    },
    {
      command: "SEND",
      label: "Send Observer",
      available: quickActions?.can_send_observer ?? false,
      reason: quickActions?.send_observer_unavailable_reason ?? emptyReason,
    },
    {
      command: "RECALL",
      label: "Recall Observer",
      available: quickActions?.can_recall_observer ?? false,
      reason: quickActions?.recall_observer_unavailable_reason ?? emptyReason,
    },
  ];
  if (expectedCriticalSend && commands[2].reason === "PORTAL_CRITICAL_RISK") {
    commands[2] = { ...commands[2], available: true };
  }

  return (
    <div className={styles.wrapper}>
      <div aria-label="Portal commands" className={styles.actions} role="group">
        {commands.map((item) => {
          const key = `${portalId ?? "empty"}:${item.command}`;
          const busy = busyKey === key;
          return (
            <button
              aria-label={item.label}
              aria-disabled={!item.available}
              className={item.available ? styles.available : styles.unavailable}
              data-tutorial-command={
                highlightedCommand === item.command || undefined
              }
              disabled={busy}
              key={item.command}
              onClick={() => {
                if (!item.available || portalId === null) {
                  setNotice(
                    item.reason === emptyReason
                      ? emptyReason
                      : unavailableActionCopy(item.reason ?? "UNKNOWN"),
                  );
                  return;
                }
                setNotice(null);
                void Promise.resolve(onCommand?.(item.command)).catch(
                  (error: unknown) => {
                    setNotice(
                      error instanceof Error ? error.message : "Command failed",
                    );
                  },
                );
              }}
              type="button"
            >
              {busy ? "Working…" : item.label}
            </button>
          );
        })}
      </div>
      {notice && <p role="status">{notice}</p>}
    </div>
  );
}

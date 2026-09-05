import { useCallback } from "react";

import { ApiError } from "../../api/errors";
import { useFeedback } from "../../components/feedback/FeedbackProvider";
import {
  useSnapshotContext,
  useSnapshotState,
} from "../../state/SnapshotProvider";
import type { PortalCommand } from "./PortalActions";

export function usePortalCommand(portalId: number) {
  const { api, store } = useSnapshotContext();
  const { commandKeys, snapshot: currentSnapshot } = useSnapshotState();
  const feedback = useFeedback();
  const busyKey =
    [...commandKeys].find((key) => key.startsWith(`${portalId}:`)) ?? null;

  const run = useCallback(
    async (command: PortalCommand) => {
      const key = `${portalId}:${command}`;
      if (store.getState().commandKeys.has(key)) return;
      store.beginCommand(key);
      const invoke = (confirm: boolean) =>
        command === "STABILIZE"
          ? api.stabilize(portalId)
          : command === "CLOSE"
            ? api.close(portalId, confirm)
            : command === "SEND"
              ? api.sendObserver(portalId, confirm)
              : api.recallObserver(portalId, confirm);
      try {
        let next;
        try {
          next = await invoke(false);
        } catch (error: unknown) {
          if (!(error instanceof ApiError) || !error.confirmable) throw error;
          store.endCommand(key);
          const labels: Record<PortalCommand, string> = {
            STABILIZE: "Stabilize",
            CLOSE: "Close",
            SEND: "Send Observer",
            RECALL: "Recall Observer",
          };
          const portalName =
            currentSnapshot?.slots.find((slot) => slot.portal?.id === portalId)
              ?.portal?.name ?? `Portal ${portalId}`;
          const confirmed = await feedback.confirmAction(
            labels[command],
            portalName,
            error.message,
          );
          if (!confirmed) return;
          store.beginCommand(key);
          next = await invoke(true);
        }
        store.acceptSnapshot(next);
      } catch (error: unknown) {
        feedback.notify(
          error instanceof Error ? error.message : "Portal command failed",
        );
      } finally {
        store.endCommand(key);
      }
    },
    [api, currentSnapshot, feedback, portalId, store],
  );

  return { busyKey, run };
}

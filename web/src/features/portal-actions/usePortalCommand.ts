import { useCallback, useState } from "react";

import { ApiError } from "../../api/errors";
import { useFeedback } from "../../components/feedback/FeedbackProvider";
import {
  useSnapshotContext,
  useSnapshotState,
} from "../../state/SnapshotProvider";
import type { PortalCommand } from "./PortalActions";
import { isExpectedCriticalSend } from "../tutorial/tutorial-actions";

export function usePortalCommand(portalId: number) {
  const [outcome, setOutcome] = useState<"success" | "error" | null>(null);
  const { api, store } = useSnapshotContext();
  const { commandKeys, snapshot: currentSnapshot } = useSnapshotState();
  const feedback = useFeedback();
  const busyKey =
    [...commandKeys].find((key) => key.startsWith(`${portalId}:`)) ?? null;

  const run = useCallback(
    async (command: PortalCommand) => {
      if (store.getState().connection !== "connected") {
        feedback.notify("Planar paths unstable. Wait for the link to recover.");
        setOutcome("error");
        return;
      }
      setOutcome(null);
      const key = `${portalId}:${command}`;
      if (store.getState().commandKeys.has(key)) return;
      store.beginCommand(key);
      const appAtStart = store.getState().snapshot?.app;
      const expectedCriticalRejection =
        command === "SEND" &&
        appAtStart !== undefined &&
        isExpectedCriticalSend(appAtStart, portalId);
      const invoke = (confirm: boolean) => {
        if (store.getState().connection !== "connected") {
          throw new Error(
            "Planar paths unstable. Wait for the link to recover.",
          );
        }
        return command === "STABILIZE"
          ? api.stabilize(portalId)
          : command === "CLOSE"
            ? api.close(portalId, confirm)
            : command === "SEND"
              ? api.sendObserver(portalId, confirm)
              : api.recallObserver(portalId, confirm);
      };
      try {
        let next;
        try {
          next = await invoke(false);
        } catch (error: unknown) {
          if (
            expectedCriticalRejection &&
            error instanceof ApiError &&
            error.code === "PORTAL_CRITICAL_RISK"
          ) {
            return;
          }
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
        setOutcome("success");
        feedback.notify(
          `${command.replaceAll("_", " ")} completed for Portal ${portalId}.`,
        );
      } catch (error: unknown) {
        setOutcome("error");
        feedback.notify(
          error instanceof Error ? error.message : "Portal command failed",
        );
      } finally {
        store.endCommand(key);
      }
    },
    [api, currentSnapshot, feedback, portalId, store],
  );

  return { busyKey, run, outcome };
}

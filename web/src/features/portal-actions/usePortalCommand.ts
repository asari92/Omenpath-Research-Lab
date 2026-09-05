import { useCallback } from "react";

import {
  useSnapshotContext,
  useSnapshotState,
} from "../../state/SnapshotProvider";
import type { PortalCommand } from "./PortalActions";

export function usePortalCommand(portalId: number) {
  const { api, store } = useSnapshotContext();
  const { commandKeys } = useSnapshotState();
  const busyKey =
    [...commandKeys].find((key) => key.startsWith(`${portalId}:`)) ?? null;

  const run = useCallback(
    async (command: PortalCommand) => {
      const key = `${portalId}:${command}`;
      if (store.getState().commandKeys.has(key)) return;
      store.beginCommand(key);
      try {
        const snapshot = await (command === "STABILIZE"
          ? api.stabilize(portalId)
          : command === "CLOSE"
            ? api.close(portalId)
            : command === "SEND"
              ? api.sendObserver(portalId)
              : api.recallObserver(portalId));
        store.acceptSnapshot(snapshot);
      } finally {
        store.endCommand(key);
      }
    },
    [api, portalId, store],
  );

  return { busyKey, run };
}

import { useState } from "react";
import { connectionHealth } from "../../state/command-health";

import {
  useSnapshotContext,
  useSnapshotState,
} from "../../state/SnapshotProvider";

export function ConnectionState() {
  const { api, store } = useSnapshotContext();
  const state = useSnapshotState();
  const { bootstrap, protocolError } = state;
  const health = connectionHealth(state);
  const [retrying, setRetrying] = useState(false);
  const failed = bootstrap === "failed" || protocolError !== null;
  return (
    <aside aria-label="Connection state" data-connection={health}>
      <p aria-live="polite" role="status">
        <span aria-hidden="true">● </span>
        {health === "disconnected"
          ? "Disconnected from the planes"
          : health === "connected"
            ? "Planar link stable"
            : "Planar paths unstable"}
      </p>
      {failed && (
        <div role="alert">
          <span>{protocolError ?? "Unable to refresh Laboratory state"}</span>
          <button
            disabled={retrying}
            onClick={async () => {
              setRetrying(true);
              try {
                store.acceptSnapshot(await api.state());
              } catch (error: unknown) {
                store.setProtocolError(
                  error instanceof Error
                    ? error.message
                    : "Connection retry failed",
                );
              } finally {
                setRetrying(false);
              }
            }}
            type="button"
          >
            Retry connection
          </button>
        </div>
      )}
    </aside>
  );
}

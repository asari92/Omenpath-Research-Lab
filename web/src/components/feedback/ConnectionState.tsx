import { useState } from "react";

import {
  useSnapshotContext,
  useSnapshotState,
} from "../../state/SnapshotProvider";

export function ConnectionState() {
  const { api, store } = useSnapshotContext();
  const { bootstrap, connection, protocolError } = useSnapshotState();
  const [retrying, setRetrying] = useState(false);
  const failed = bootstrap === "failed" || protocolError !== null;
  return (
    <aside aria-label="Connection state">
      <p aria-live="polite" role="status">
        Connection: {connection}
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

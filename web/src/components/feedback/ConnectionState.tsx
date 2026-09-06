import { connectionHealth } from "../../state/command-health";

import {
  useSnapshotContext,
  useSnapshotState,
} from "../../state/SnapshotProvider";

export function ConnectionState() {
  const { retryConnection } = useSnapshotContext();
  const state = useSnapshotState();
  const { bootstrap, protocolError } = state;
  const health = connectionHealth(state);
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
            disabled={bootstrap === "loading"}
            onClick={retryConnection}
            type="button"
          >
            Retry connection
          </button>
        </div>
      )}
    </aside>
  );
}

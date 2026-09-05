import type { StateSnapshot } from "../api/types";

export type BootstrapState = "idle" | "loading" | "ready" | "failed";
export type ConnectionState =
  "connecting" | "connected" | "reconnecting" | "offline";

export interface SnapshotStoreState {
  snapshot: StateSnapshot | null;
  bootstrap: BootstrapState;
  connection: ConnectionState;
  commandKeys: ReadonlySet<string>;
  protocolError: string | null;
}

export interface SnapshotStore {
  getState(): SnapshotStoreState;
  subscribe(listener: () => void): () => void;
  acceptSnapshot(snapshot: StateSnapshot): boolean;
  setBootstrap(value: BootstrapState): void;
  setConnection(value: ConnectionState): void;
  setProtocolError(message: string | null): void;
  beginCommand(key: string): void;
  endCommand(key: string): void;
}

export function shouldAcceptSnapshot(
  currentGeneratedAt: string | null,
  candidateGeneratedAt: string,
): boolean {
  const candidateTime = Date.parse(candidateGeneratedAt);
  if (Number.isNaN(candidateTime)) return false;
  if (currentGeneratedAt === null) return true;
  const currentTime = Date.parse(currentGeneratedAt);
  return !Number.isNaN(currentTime) && candidateTime >= currentTime;
}

export function createSnapshotStore(): SnapshotStore {
  let state: SnapshotStoreState = {
    snapshot: null,
    bootstrap: "idle",
    connection: "offline",
    commandKeys: new Set<string>(),
    protocolError: null,
  };
  const listeners = new Set<() => void>();
  const update = (next: SnapshotStoreState) => {
    state = next;
    listeners.forEach((listener) => listener());
  };

  return {
    getState: () => state,
    subscribe(listener) {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    acceptSnapshot(snapshot) {
      if (
        !shouldAcceptSnapshot(
          state.snapshot?.generated_at ?? null,
          snapshot.generated_at,
        )
      ) {
        if (Number.isNaN(Date.parse(snapshot.generated_at))) {
          update({
            ...state,
            protocolError: "Invalid generated_at in state snapshot",
          });
        }
        return false;
      }
      update({ ...state, snapshot, bootstrap: "ready", protocolError: null });
      return true;
    },
    setBootstrap(bootstrap) {
      update({ ...state, bootstrap });
    },
    setConnection(connection) {
      update({ ...state, connection });
    },
    setProtocolError(protocolError) {
      update({ ...state, protocolError });
    },
    beginCommand(key) {
      const commandKeys = new Set(state.commandKeys);
      commandKeys.add(key);
      update({ ...state, commandKeys });
    },
    endCommand(key) {
      const commandKeys = new Set(state.commandKeys);
      commandKeys.delete(key);
      update({ ...state, commandKeys });
    },
  };
}

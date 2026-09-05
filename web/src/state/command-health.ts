import type { SnapshotStoreState } from "./snapshot-store";

export function connectionHealth(
  state: SnapshotStoreState,
): "connected" | "reconnecting" | "disconnected" {
  if (
    state.bootstrap === "failed" ||
    state.protocolError !== null ||
    state.connection === "offline"
  )
    return "disconnected";
  if (
    state.bootstrap !== "ready" ||
    state.snapshot === null ||
    state.connection !== "connected"
  )
    return "reconnecting";
  return "connected";
}

export function commandsEnabled(state: SnapshotStoreState): boolean {
  return connectionHealth(state) === "connected";
}

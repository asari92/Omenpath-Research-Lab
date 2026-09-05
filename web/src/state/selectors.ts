import type { StateSnapshot } from "../api/types";

export function requireSnapshot(snapshot: StateSnapshot | null): StateSnapshot {
  if (!snapshot) throw new Error("State snapshot is not ready");
  return snapshot;
}

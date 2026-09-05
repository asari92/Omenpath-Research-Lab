import type { SlotDTO, StateSnapshot } from "../api/types";

export function requireSnapshot(snapshot: StateSnapshot | null): StateSnapshot {
  if (!snapshot) throw new Error("State snapshot is not ready");
  return snapshot;
}

export function sevenSlots(snapshot: StateSnapshot): readonly SlotDTO[] {
  const byIndex = new Map(
    snapshot.slots.map((slot) => [slot.slot_index, slot]),
  );
  const result = Array.from({ length: 7 }, (_, index) =>
    byIndex.get(index + 1),
  );
  if (result.some((slot) => slot === undefined) || byIndex.size !== 7) {
    throw new Error("State snapshot must contain exactly Slot 1..7");
  }
  return result as readonly SlotDTO[];
}

export function formatEnergy(value: number): string {
  return `${value.toFixed(1)}%`;
}

export function formatRemaining(totalSeconds: number): string {
  const safe = Math.max(0, Math.floor(totalSeconds));
  const minutes = Math.floor(safe / 60);
  const seconds = safe % 60;
  return `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
}

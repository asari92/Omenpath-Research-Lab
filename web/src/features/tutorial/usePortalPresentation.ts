import { useEffect, useState } from "react";
import type { SlotDTO, StateSnapshot } from "../../api/types";

interface Ghost {
  slot: SlotDTO;
  replacement: number;
  key: string;
  generation: number;
}
interface Presentation {
  snapshot: StateSnapshot | null;
  ghost: Ghost | null;
  generation: number;
}

export function usePortalPresentation(snapshot: StateSnapshot | null) {
  const [state, setState] = useState<Presentation>({
    snapshot,
    ghost: null,
    generation: 0,
  });
  if (state.snapshot !== snapshot) {
    const previous = state.snapshot;
    const oldTarget = previous?.app.tutorial_portal_id;
    const newTarget = snapshot?.app.tutorial_portal_id;
    const oldSlot = previous?.slots.find(
      (slot) => slot.portal?.id === oldTarget,
    );
    const retry =
      snapshot?.app.mode === "TUTORIAL" &&
      snapshot.app.tutorial_step > 0 &&
      previous?.app.mode === "TUTORIAL" &&
      previous.app.tutorial_step === snapshot.app.tutorial_step &&
      oldTarget !== newTarget &&
      newTarget != null &&
      oldSlot?.portal &&
      !snapshot.slots.some((slot) => slot.portal?.id === oldTarget);
    const reset =
      snapshot?.app.mode !== "TUTORIAL" || snapshot.app.tutorial_step === 0;
    const generation = state.generation + 1;
    const ghost = retry
      ? {
          slot: {
            ...oldSlot,
            portal: { ...oldSlot.portal!, status: "CLOSED" as const },
          },
          replacement: newTarget,
          key: `${oldSlot.slot_index}:${oldTarget}:${newTarget}`,
          generation,
        }
      : reset
        ? null
        : state.ghost;
    setState({ snapshot, ghost, generation });
  }
  const ghost = state.ghost;
  useEffect(() => {
    if (!ghost) return;
    const timer = window.setTimeout(
      () =>
        setState((current) =>
          current.ghost?.key === ghost.key &&
          current.ghost.generation === ghost.generation
            ? { ...current, ghost: null }
            : current,
        ),
      2000,
    );
    return () => window.clearTimeout(timer);
  }, [ghost]);
  const slots = snapshot?.slots.map((slot) =>
    ghost
      ? slot.slot_index === ghost.slot.slot_index
        ? ghost.slot
        : slot.portal?.id === ghost.replacement
          ? { ...slot, portal: null }
          : slot
      : slot,
  );
  return { slots, ghost };
}

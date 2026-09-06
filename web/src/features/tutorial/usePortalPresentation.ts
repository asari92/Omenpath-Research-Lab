import { useEffect, useRef, useState } from "react";
import type { SlotDTO, StateSnapshot } from "../../api/types";

interface Ghost {
  slot: SlotDTO;
  replacement: number | null;
  key: string;
  timerKey: string;
}
interface Presentation {
  snapshot: StateSnapshot | null;
  ghosts: Ghost[];
  generation: number;
}

export function usePortalPresentation(snapshot: StateSnapshot | null) {
  const [state, setState] = useState<Presentation>({
    snapshot,
    ghosts: [],
    generation: 0,
  });
  const timers = useRef(new Map<string, number>());
  if (state.snapshot !== snapshot) {
    const previous = state.snapshot;
    const generation = state.generation + 1;
    const reset =
      !snapshot ||
      (snapshot.app.mode === "TUTORIAL" && snapshot.app.tutorial_step === 0);
    const ghosts = reset
      ? []
      : state.ghosts.map((ghost) => {
          const replacement =
            snapshot.slots.find((s) => s.slot_index === ghost.slot.slot_index)
              ?.portal?.id ?? ghost.replacement;
          return {
            ...ghost,
            replacement,
            key: `${ghost.slot.slot_index}:${ghost.slot.portal!.id}:${replacement ?? "empty"}`,
          };
        });
    if (!reset)
      for (const old of previous?.slots ?? []) {
        if (
          !old.portal ||
          ghosts.some((g) => g.slot.slot_index === old.slot_index) ||
          snapshot.slots.some((s) => s.portal?.id === old.portal!.id)
        )
          continue;
        const sameSlot =
          snapshot.slots.find((s) => s.slot_index === old.slot_index)?.portal
            ?.id ?? null;
        const tutorialRetry =
          previous?.app.mode === "TUTORIAL" &&
          snapshot.app.mode === "TUTORIAL" &&
          previous.app.tutorial_step === snapshot.app.tutorial_step &&
          previous.app.tutorial_portal_id === old.portal.id;
        const replacement =
          sameSlot ?? (tutorialRetry ? snapshot.app.tutorial_portal_id : null);
        ghosts.push({
          // Slot snapshots contain only open portals. This is a generic terminal
          // visual, not a claim about whether the domain closed or collapsed it.
          slot: { ...old, portal: { ...old.portal, status: "CLOSED" } },
          replacement,
          key: `${old.slot_index}:${old.portal.id}:${replacement ?? "empty"}`,
          timerKey: `${old.slot_index}:${old.portal.id}:${generation}`,
        });
      }
    setState({ snapshot, ghosts, generation });
  }
  const ghosts = state.ghosts;
  useEffect(() => {
    const pending = timers.current;
    for (const [key, timer] of pending)
      if (!ghosts.some((g) => g.timerKey === key)) {
        window.clearTimeout(timer);
        pending.delete(key);
      }
    for (const ghost of ghosts)
      if (!pending.has(ghost.timerKey)) {
        pending.set(
          ghost.timerKey,
          window.setTimeout(() => {
            pending.delete(ghost.timerKey);
            setState((current) => ({
              ...current,
              ghosts: current.ghosts.filter(
                (g) => g.timerKey !== ghost.timerKey,
              ),
            }));
          }, 2000),
        );
      }
  }, [ghosts]);
  useEffect(() => {
    const pending = timers.current;
    return () => {
      pending.forEach((timer) => window.clearTimeout(timer));
      pending.clear();
    };
  }, []);
  const slots = snapshot?.slots.map((slot) => {
    const ghost = ghosts.find((g) => g.slot.slot_index === slot.slot_index);
    if (ghost) return ghost.slot;
    return ghosts.some((g) => g.replacement === slot.portal?.id)
      ? { ...slot, portal: null }
      : slot;
  });
  return { slots, ghosts };
}

import type { AppDTO } from "../../api/types";

export interface TutorialPresentation {
  current: AppDTO | null;
  shown: AppDTO | null;
  history: AppDTO[];
  queue: AppDTO[];
  cursor: number;
  replaying: boolean;
  generation: number;
}
type Action =
  | { type: "snapshot"; app: AppDTO | null }
  | { type: "back" }
  | { type: "forward" }
  | { type: "elapsed"; generation: number };

export function initialPresentation(
  app: AppDTO | null,
  generation = 0,
): TutorialPresentation {
  const current = app?.mode === "TUTORIAL" ? app : null;
  return {
    current,
    shown: current,
    history: current ? [current] : [],
    queue: [],
    cursor: 0,
    replaying: false,
    generation,
  };
}

function show(
  state: TutorialPresentation,
  app: AppDTO,
  queue: AppDTO[],
): TutorialPresentation {
  const history = [...state.history, app];
  return {
    ...state,
    shown: app,
    history,
    queue,
    cursor: history.length - 1,
    replaying: queue.length > 0,
    generation: state.generation + 1,
  };
}

// This reducer only selects explanatory cards. It never writes gameplay state or sends signals.
export function tutorialPresentation(
  state: TutorialPresentation,
  action: Action,
): TutorialPresentation {
  if (action.type === "back" || action.type === "forward") {
    const cursor = Math.max(
      0,
      Math.min(
        state.history.length - 1,
        state.cursor + (action.type === "back" ? -1 : 1),
      ),
    );
    return cursor === state.cursor
      ? state
      : { ...state, cursor, shown: state.history[cursor] ?? null };
  }
  if (action.type === "elapsed") {
    if (action.generation !== state.generation || !state.replaying)
      return state;
    return show(state, state.queue[0], state.queue.slice(1));
  }
  const app = action.app;
  if (
    app?.mode !== "TUTORIAL" ||
    !state.current ||
    (app.tutorial_step === 0 && state.current.tutorial_step !== 0)
  ) {
    return initialPresentation(app, state.generation + 1);
  }
  if (app.tutorial_step < state.current.tutorial_step) {
    const history = state.history.filter(
      (item) => item.tutorial_step < app.tutorial_step,
    );
    return show({ ...state, current: app, history }, app, []);
  }
  if (app.tutorial_step === state.current.tutorial_step) {
    const history = state.history.map((item) =>
      item.tutorial_step === app.tutorial_step ? app : item,
    );
    return {
      ...state,
      current: app,
      history,
      shown: history[state.cursor] ?? null,
      queue: state.queue.map((item) =>
        item.tutorial_step === app.tutorial_step ? app : item,
      ),
    };
  }
  const missed: AppDTO[] = [];
  for (
    let step = state.current.tutorial_step + 1;
    step < app.tutorial_step;
    step++
  ) {
    // Historical intermediate identities are unknown: do not invent command targets.
    missed.push({
      ...app,
      tutorial_step: step,
      tutorial_phase: "",
      tutorial_portal_id: null,
      tutorial_plane_id: null,
      tutorial_observer_id: null,
      expected_action: null,
    });
  }
  const queue = [...state.queue, ...missed, app];
  const updated = { ...state, current: app };
  return state.replaying
    ? { ...updated, queue }
    : show(updated, queue[0], queue.slice(1));
}

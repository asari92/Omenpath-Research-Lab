import { useEffect, useReducer, useRef, useState } from "react";
import type { AppDTO } from "../../api/types";
import {
  initialPresentation,
  tutorialPresentation,
} from "./tutorial-presentation";

export function useTutorialPresentation(app: AppDTO | null) {
  const [accepted, setAccepted] = useState(app);
  const [presentation, dispatch] = useReducer(
    tutorialPresentation,
    app,
    initialPresentation,
  );
  if (accepted !== app) {
    setAccepted(app);
    dispatch({ type: "snapshot", app });
  }
  const { generation, replaying } = presentation;
  const budget = useRef({ generation: -1, remaining: 7000, token: 0 });
  const atObjective = presentation.cursor === presentation.history.length - 1;
  useEffect(() => {
    if (!replaying || !atObjective) return;
    if (budget.current.generation !== generation)
      budget.current = {
        generation,
        remaining: 7000,
        token: budget.current.token,
      };
    const started = Date.now();
    const remaining = budget.current.remaining;
    const token = ++budget.current.token;
    const timer = window.setTimeout(() => {
      if (budget.current.token === token)
        dispatch({ type: "elapsed", generation });
    }, remaining);
    return () => {
      window.clearTimeout(timer);
      budget.current = {
        generation,
        remaining: Math.max(0, remaining - (Date.now() - started)),
        token: budget.current.token + 1,
      };
    };
  }, [generation, replaying, atObjective]);
  return {
    presentation,
    back: () => dispatch({ type: "back" }),
    forward: () => dispatch({ type: "forward" }),
  };
}

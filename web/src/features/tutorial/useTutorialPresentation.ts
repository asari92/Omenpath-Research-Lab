import { useEffect, useReducer, useState } from "react";
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
  useEffect(() => {
    if (!replaying) return;
    const timer = window.setTimeout(
      () => dispatch({ type: "elapsed", generation }),
      7000,
    );
    return () => window.clearTimeout(timer);
  }, [generation, replaying]);
  return {
    presentation,
    back: () => dispatch({ type: "back" }),
    forward: () => dispatch({ type: "forward" }),
  };
}

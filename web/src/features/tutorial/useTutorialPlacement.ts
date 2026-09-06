import { useRef } from "react";
import type { AppDTO } from "../../api/types";

// The approved composition keeps the parchment at the top centre. It remains
// pointer-transparent except for its own controls, so card actions stay usable.
export function useTutorialPlacement(pathname: string, app: AppDTO | null) {
  void pathname;
  void app;
  return useRef<HTMLDivElement>(null);
}

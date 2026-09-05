import type { AppDTO } from "../../api/types";
import type { PortalCommand } from "../portal-actions/PortalActions";

export function isExpectedCriticalSend(app: AppDTO, portalId: number): boolean {
  return (
    app.mode === "TUTORIAL" &&
    app.expected_action === "ATTEMPT_CRITICAL_SEND" &&
    app.tutorial_portal_id === portalId
  );
}

export function tutorialCommandTarget(
  app: AppDTO,
): { portalId: number; command: PortalCommand } | null {
  if (app.mode !== "TUTORIAL" || app.tutorial_portal_id === null) return null;
  const command =
    app.expected_action === "SEND_OBSERVER"
      ? "SEND"
      : app.expected_action === "STABILIZE"
        ? "STABILIZE"
        : app.expected_action === "ATTEMPT_CRITICAL_SEND"
          ? "SEND"
          : app.expected_action === "RECALL_OBSERVER"
            ? "RECALL"
            : null;
  return command === null
    ? null
    : { portalId: app.tutorial_portal_id, command };
}

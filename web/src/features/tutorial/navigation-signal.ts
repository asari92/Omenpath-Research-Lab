import type { AppDTO, TutorialSignalRequest } from "../../api/types";

export type NavigationDestination =
  { kind: "portal"; id: number } | { kind: "events" };

let navigationIntent: NavigationDestination | null = null;

function sameDestination(
  a: NavigationDestination,
  b: NavigationDestination,
): boolean {
  return a.kind === "events"
    ? b.kind === "events"
    : b.kind === "portal" && a.id === b.id;
}

export function matchingNavigationSignal(
  app: AppDTO | null,
  destination: NavigationDestination,
): TutorialSignalRequest | null {
  if (!app || app.mode !== "TUTORIAL") return null;
  if (
    destination.kind === "portal" &&
    app.expected_action === "OPEN_PORTAL_DETAILS" &&
    app.tutorial_portal_id === destination.id
  ) {
    return { signal: "PORTAL_DETAILS_OPENED", portal_id: destination.id };
  }
  if (
    destination.kind === "events" &&
    app.expected_action === "OPEN_EVENT_LOG"
  ) {
    return { signal: "EVENT_LOG_OPENED" };
  }
  return null;
}

export function recordNavigationIntent(
  destination: NavigationDestination,
): void {
  navigationIntent = destination;
}

export function consumeNavigationSignal(
  app: AppDTO | null,
  destination: NavigationDestination,
): TutorialSignalRequest | null {
  const intent = navigationIntent;
  navigationIntent = null;
  if (!intent || !sameDestination(intent, destination)) return null;
  return matchingNavigationSignal(app, destination);
}

export function resetNavigationIntentForTests(): void {
  navigationIntent = null;
}

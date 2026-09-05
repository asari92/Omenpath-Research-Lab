import { expect, it } from "vitest";
import { observerTransit, snapshotAt } from "../test/builders";
import { observerTransitForPortal } from "./selectors";

it("selects authoritative transit by exact portal ID and returns null when idle", () => {
  const snapshot = snapshotAt();
  const outbound = observerTransit(42, 9);
  const returning = observerTransit(43, 3, "RETURNING");
  snapshot.observer_transits = [returning, outbound];
  expect(observerTransitForPortal(snapshot, 42)).toBe(outbound);
  expect(observerTransitForPortal(snapshot, 43)).toBe(returning);
  expect(observerTransitForPortal(snapshot, 44)).toBeNull();
});

import { EmptyPortalSlot } from "../components/dashboard/EmptyPortalSlot";
import { NeedsAttention } from "../components/dashboard/NeedsAttention";
import { PortalSlot } from "../components/dashboard/PortalSlot";
import { sevenSlots } from "../state/selectors";
import { useSnapshotState } from "../state/SnapshotProvider";
import styles from "./DashboardPage.module.css";
import { usePortalPresentation } from "../features/tutorial/usePortalPresentation";

export function DashboardPage() {
  const { snapshot, bootstrap, protocolError } = useSnapshotState();
  const presentation = usePortalPresentation(snapshot);
  if (!snapshot) {
    return (
      <section>
        <h1>Laboratory Overview</h1>
        <p>
          {protocolError ??
            (bootstrap === "failed"
              ? "Unable to load Laboratory state."
              : "Connecting…")}
        </p>
      </section>
    );
  }

  let slots;
  try {
    slots = sevenSlots({
      ...snapshot,
      slots: presentation.slots ?? snapshot.slots,
    });
  } catch (error) {
    return (
      <section>
        <h1>Laboratory Overview</h1>
        <p role="alert">
          {error instanceof Error ? error.message : "Invalid Slot data"}
        </p>
      </section>
    );
  }

  return (
    <section className={styles.dashboard}>
      <header className={styles.titleRow}>
        <h1>Laboratory Overview</h1>
        <NeedsAttention snapshot={snapshot} />
      </header>
      {snapshot.portals.active === 0 && (
        <p className={styles.waiting}>Waiting for an Omenpath…</p>
      )}
      <div className={styles.board} data-testid="portal-board">
        {slots.map((slot) =>
          slot.portal ? (
            <PortalSlot
              key={slot.slot_index}
              ghost={presentation.ghosts.some(
                (ghost) => ghost.slot.slot_index === slot.slot_index,
              )}
              needsAttention={
                snapshot.needs_attention_portal_id === slot.portal.id
              }
              portal={slot.portal}
              slotIndex={slot.slot_index}
              tutorialTarget={
                snapshot.app.mode === "TUTORIAL" &&
                snapshot.app.tutorial_portal_id === slot.portal.id
              }
            />
          ) : (
            <EmptyPortalSlot
              key={slot.slot_index}
              slotIndex={slot.slot_index}
            />
          ),
        )}
      </div>
    </section>
  );
}

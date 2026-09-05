import { EmptyPortalSlot } from "../components/dashboard/EmptyPortalSlot";
import { LabSummary } from "../components/dashboard/LabSummary";
import { NeedsAttention } from "../components/dashboard/NeedsAttention";
import { PortalSlot } from "../components/dashboard/PortalSlot";
import { sevenSlots } from "../state/selectors";
import { useSnapshotState } from "../state/SnapshotProvider";
import styles from "./DashboardPage.module.css";

export function DashboardPage() {
  const { snapshot, bootstrap, protocolError } = useSnapshotState();
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
    slots = sevenSlots(snapshot);
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
      <h1>Laboratory Overview</h1>
      <LabSummary snapshot={snapshot} />
      {snapshot.lab.leyline_override_active && (
        <p className={styles.override} role="status">
          Leyline Override active until{" "}
          {snapshot.lab.leyline_override_until?.slice(11, 19)} UTC
        </p>
      )}
      <NeedsAttention snapshot={snapshot} />
      {snapshot.portals.active === 0 && (
        <p className={styles.waiting}>Waiting for an Omenpath…</p>
      )}
      <div className={styles.board} data-testid="portal-board">
        {slots.map((slot) =>
          slot.portal ? (
            <PortalSlot
              key={slot.slot_index}
              needsAttention={
                snapshot.needs_attention_portal_id === slot.portal.id
              }
              portal={slot.portal}
              slotIndex={slot.slot_index}
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

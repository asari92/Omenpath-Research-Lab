import { PortalActions } from "../../features/portal-actions/PortalActions";
import styles from "./PortalSlot.module.css";

export function EmptyPortalSlot({ slotIndex }: { slotIndex: number }) {
  return (
    <article
      className={`${styles.slot} ${styles.empty}`}
      data-testid="portal-slot"
    >
      <header>
        <span>Slot {slotIndex}</span>
        <span>EMPTY</span>
      </header>
      <div className={styles.visual}>
        <div className={styles.emptyMark} aria-hidden="true" />
      </div>
      <h2>Unbound Omenpath</h2>
      <p>Awaiting Portal</p>
      <dl className={styles.metrics} aria-hidden="true">
        <div>—</div>
      </dl>
      <div className={styles.transit} />
      <PortalActions portalId={null} quickActions={null} />
      <span className={styles.detailsPlaceholder}>Details</span>
    </article>
  );
}

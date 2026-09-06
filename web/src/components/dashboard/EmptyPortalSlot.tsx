import { PortalActions } from "../../features/portal-actions/PortalActions";
import styles from "./PortalSlot.module.css";

export function EmptyPortalSlot({ slotIndex }: { slotIndex: number }) {
  return (
    <article
      className={`${styles.slot} ${styles.empty}`}
      data-testid="portal-slot"
    >
      <header>
        <span className={styles.slotNumber}>Slot {slotIndex}</span>
        <strong>Unbound</strong>
        <span>EMPTY</span>
      </header>
      <div className={styles.visual}>
        <div className={styles.emptyMark} aria-hidden="true" />
      </div>
      <span className={styles.identity}>Unbound Omenpath</span>
      <p className={styles.portalState}>Awaiting Portal</p>
      <dl className={styles.metrics} aria-hidden="true">
        <div>—</div>
      </dl>
      <div className={styles.transit} />
      <div className={styles.commandLayer}>
        <PortalActions portalId={null} quickActions={null} />
      </div>
    </article>
  );
}

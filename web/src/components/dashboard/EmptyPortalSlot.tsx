import { PortalActions } from "../../features/portal-actions/PortalActions";
import styles from "./PortalSlot.module.css";

export function EmptyPortalSlot({ slotIndex }: { slotIndex: number }) {
  return (
    <article
      className={`${styles.slot} ${styles.empty}`}
      data-testid="portal-slot"
    >
      <h2>Slot {slotIndex}</h2>
      <div className={styles.emptyMark} aria-hidden="true" />
      <p>Awaiting Portal</p>
      <PortalActions portalId={null} quickActions={null} />
    </article>
  );
}

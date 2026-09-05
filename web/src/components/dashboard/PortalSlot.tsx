import { Link } from "react-router-dom";

import type { SlotPortalDTO } from "../../api/types";
import { PortalActions } from "../../features/portal-actions/PortalActions";
import { usePortalCommand } from "../../features/portal-actions/usePortalCommand";
import { recordNavigationIntent } from "../../features/tutorial/navigation-signal";
import {
  isExpectedCriticalSend,
  tutorialCommandTarget,
} from "../../features/tutorial/tutorial-actions";
import { PortalEffect } from "../../portal-fx/PortalEffect";
import {
  formatEnergy,
  formatRemaining,
  observerTransitForPortal,
} from "../../state/selectors";
import { ObserverTransit } from "../portal/ObserverTransit";
import styles from "./PortalSlot.module.css";
import { useSnapshotState } from "../../state/SnapshotProvider";

export function PortalSlot({
  slotIndex,
  portal,
  needsAttention,
  tutorialTarget = false,
}: {
  slotIndex: number;
  portal: SlotPortalDTO;
  needsAttention: boolean;
  tutorialTarget?: boolean;
}) {
  const { snapshot, connection } = useSnapshotState();
  const command = usePortalCommand(portal.id);
  const target = snapshot ? tutorialCommandTarget(snapshot.app) : null;
  return (
    <article
      className={styles.slot}
      data-testid="portal-slot"
      data-stability={portal.stability}
      data-tutorial-target={tutorialTarget || undefined}
    >
      <header>
        <span>Slot {slotIndex}</span>
        <span>{portal.stability}</span>
      </header>
      <div className={styles.visual}>
        <PortalEffect
          density={needsAttention ? "high" : "low"}
          planeId={portal.destination_plane_id}
          planeName={portal.destination_plane_name}
          portalId={portal.id}
        />
      </div>
      <h2>{portal.name}</h2>
      <p>
        {portal.destination_plane_name} ·{" "}
        {portal.destination_explored ? "EXPLORED" : "UNEXPLORED"} ·{" "}
        {portal.status}
      </p>
      <dl className={styles.metrics}>
        <div>
          <dt>Energy</dt>
          <dd>{formatEnergy(portal.energy)}</dd>
        </div>
        <div>
          <dt>Time</dt>
          <dd>{formatRemaining(portal.time_remaining_seconds)}</dd>
        </div>
        <div>
          <dt>Creatures</dt>
          <dd>{portal.creatures_inside}</dd>
        </div>
      </dl>
      <div className={styles.transit}>
        {snapshot && (
          <ObserverTransit
            transit={observerTransitForPortal(snapshot, portal.id)}
          />
        )}
      </div>
      <PortalActions
        offline={connection !== "connected"}
        outcome={command.outcome}
        busyKey={command.busyKey}
        expectedCriticalSend={
          snapshot ? isExpectedCriticalSend(snapshot.app, portal.id) : false
        }
        highlightedCommand={
          target?.portalId === portal.id ? target.command : null
        }
        onCommand={command.run}
        portalId={portal.id}
        quickActions={portal.quick_actions}
      />
      <Link
        onClick={() =>
          recordNavigationIntent({ kind: "portal", id: portal.id })
        }
        to={`/portals/${portal.id}`}
      >
        Details
      </Link>
    </article>
  );
}

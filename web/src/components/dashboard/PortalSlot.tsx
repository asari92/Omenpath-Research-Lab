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
import { commandsEnabled } from "../../state/command-health";

export function PortalSlot({
  slotIndex,
  portal,
  needsAttention,
  tutorialTarget = false,
  ghost = false,
}: {
  slotIndex: number;
  portal: SlotPortalDTO;
  needsAttention: boolean;
  tutorialTarget?: boolean;
  ghost?: boolean;
}) {
  const state = useSnapshotState();
  const { snapshot } = state;
  const command = usePortalCommand(portal.id);
  const target = snapshot ? tutorialCommandTarget(snapshot.app) : null;
  return (
    <article
      className={styles.slot}
      data-testid="portal-slot"
      data-ghost={ghost || undefined}
      data-stability={portal.stability}
      data-tutorial-target={(!ghost && tutorialTarget) || undefined}
    >
      <header>
        <span>Slot {slotIndex}</span>
        <span>{portal.stability}</span>
      </header>
      <div
        className={styles.visual}
        data-testid={ghost ? "portal-ghost" : undefined}
      >
        {ghost && (
          <span className={styles.ghostName}>{portal.name} · Ended</span>
        )}
        <PortalEffect
          key={portal.id}
          status={portal.status}
          exiting={ghost}
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
        {ghost ? "ENDED" : portal.status}
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
        offline={!commandsEnabled(state)}
        outcome={command.outcome}
        busyKey={command.busyKey}
        expectedCriticalSend={
          snapshot ? isExpectedCriticalSend(snapshot.app, portal.id) : false
        }
        highlightedCommand={
          target?.portalId === portal.id ? target.command : null
        }
        onCommand={command.run}
        portalId={ghost ? null : portal.id}
        quickActions={ghost ? null : portal.quick_actions}
      />
      {ghost ? (
        <span>Portal ended</span>
      ) : (
        <Link
          onClick={() =>
            recordNavigationIntent({ kind: "portal", id: portal.id })
          }
          to={`/portals/${portal.id}`}
        >
          Details
        </Link>
      )}
    </article>
  );
}

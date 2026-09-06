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
        <span className={styles.slotNumber}>Slot {slotIndex}</span>
        <strong>{portal.destination_plane_name}</strong>
        <span
          aria-label={`Stability ${portal.stability}`}
          className={styles.stability}
          data-stability={portal.stability}
        >
          {portal.stability}
        </span>
      </header>
      <span className={styles.identity}>{portal.name}</span>
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
      <p className={styles.portalState}>
        {portal.destination_explored ? "EXPLORED" : "UNEXPLORED"} ·{" "}
        {ghost ? "ENDED" : portal.status}
      </p>
      <dl className={styles.metrics}>
        <div>
          <dt>Energy</dt>
          <dd data-testid="portal-energy" data-value-kind="energy">
            <span aria-hidden="true">⚡</span> {formatEnergy(portal.energy)}
          </dd>
        </div>
        <div>
          <dt>Time</dt>
          <dd data-testid="portal-time" data-value-kind="time">
            {formatRemaining(portal.time_remaining_seconds)}
          </dd>
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
      <div className={styles.commandLayer}>
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
      </div>
      {ghost ? (
        <span>Portal ended</span>
      ) : (
        <Link
          aria-label={`Inspect ${portal.name} on ${portal.destination_plane_name}`}
          className={styles.cardLink}
          onClick={() =>
            recordNavigationIntent({ kind: "portal", id: portal.id })
          }
          onKeyDown={(event) => {
            if (event.key !== "Enter" && event.key !== " ") return;
            event.preventDefault();
            event.currentTarget.click();
          }}
          to={`/portals/${portal.id}`}
        />
      )}
    </article>
  );
}

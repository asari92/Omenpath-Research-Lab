import {
  useEffect,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";
import { useParams } from "react-router-dom";

import { ApiError } from "../api/errors";
import { createLiveResource } from "../api/live-resource";
import { EventList } from "../components/events/EventList";
import { DestinationFacts } from "../components/portal/DestinationFacts";
import { Diagnostics } from "../components/portal/Diagnostics";
import { ObserverTransit } from "../components/portal/ObserverTransit";
import { PortalFacts } from "../components/portal/PortalFacts";
import { PortalActions } from "../features/portal-actions/PortalActions";
import { usePortalCommand } from "../features/portal-actions/usePortalCommand";
import { consumeNavigationSignal } from "../features/tutorial/navigation-signal";
import {
  isExpectedCriticalSend,
  tutorialCommandTarget,
} from "../features/tutorial/tutorial-actions";
import { PortalEffect } from "../portal-fx/PortalEffect";
import {
  useSnapshotContext,
  useSnapshotState,
} from "../state/SnapshotProvider";
import styles from "./PortalDetailsPage.module.css";
import { commandsEnabled } from "../state/command-health";

function parsePortalID(value: string | undefined): number | null {
  if (!value || !/^\d+$/.test(value)) return null;
  const id = Number(value);
  return Number.isSafeInteger(id) && id > 0 ? id : null;
}

function PortalDetailsResource({ id }: { id: number }) {
  const { api, store } = useSnapshotContext();
  const snapshotState = useSnapshotState();
  const { snapshot } = snapshotState;
  const enabled = commandsEnabled(snapshotState);
  const [signalError, setSignalError] = useState<string | null>(null);
  const lifecycle = useRef(0);
  const command = usePortalCommand(id);
  const resource = useMemo(
    () => createLiveResource((signal) => api.portal(id, signal)),
    [api, id],
  );
  const details = useSyncExternalStore(
    resource.subscribe,
    resource.getSnapshot,
    resource.getSnapshot,
  );

  useEffect(() => resource.refresh(), [resource, snapshot?.generated_at]);
  useEffect(() => {
    lifecycle.current += 1;
    const generation = lifecycle.current;
    return () => {
      queueMicrotask(() => {
        if (lifecycle.current === generation) resource.dispose();
      });
    };
  }, [resource]);
  useEffect(() => {
    if (!commandsEnabled(store.getState())) return;
    const request = consumeNavigationSignal(snapshot?.app ?? null, {
      kind: "portal",
      id,
    });
    if (!request) return;
    void api
      .tutorialSignal(request)
      .then((next) => store.acceptSnapshot(next))
      .catch((error: unknown) => {
        setSignalError(
          error instanceof Error ? error.message : "Tutorial signal failed",
        );
      });
  }, [api, id, snapshot?.app, store, enabled]);

  if (details.error instanceof ApiError && details.error.status === 404) {
    return <p role="alert">Portal Not Found</p>;
  }
  if (details.error && !details.data) {
    return (
      <div role="alert">
        <p>Unable to load Portal Details.</p>
        <button onClick={() => resource.refresh()} type="button">
          Retry
        </button>
      </div>
    );
  }
  if (!details.data)
    return <p>{details.loading ? "Loading Portal…" : "Portal unavailable"}</p>;

  const value = details.data;
  const tutorialTarget = snapshot ? tutorialCommandTarget(snapshot.app) : null;
  return (
    <>
      {details.error && (
        <p role="status">
          Updates unavailable.{" "}
          <button type="button" onClick={() => resource.refresh()}>
            Retry
          </button>
        </p>
      )}
      {signalError && <p role="status">{signalError}</p>}
      <div className={styles.body}>
        <div className={styles.facts}>
          <PortalFacts portal={value.portal} />
          <ObserverTransit transit={value.observer_transit} />
        </div>
        <div className={styles.hero}>
          <PortalEffect
            density="high"
            planeId={value.destination.plane_id}
            planeName={value.destination.name}
            portalId={value.portal.id}
            status={value.portal.status}
          />
          <section className={styles.actions} aria-label="Portal actions">
            <h2>Actions</h2>
            <PortalActions
              offline={!enabled}
              outcome={command.outcome}
              busyKey={command.busyKey}
              expectedCriticalSend={
                snapshot ? isExpectedCriticalSend(snapshot.app, id) : false
              }
              highlightedCommand={
                tutorialTarget?.portalId === id ? tutorialTarget.command : null
              }
              onCommand={command.run}
              portalId={id}
              quickActions={value.portal.quick_actions}
            />
          </section>
        </div>
        <div className={styles.facts}>
          <DestinationFacts destination={value.destination} />
          <Diagnostics
            risk={value.risk_level}
            recommendation={value.recommendation}
          />
        </div>
      </div>
      <details className={styles.history}>
        <summary>History ({value.history.length})</summary>
        <div
          className={styles.historyList}
          role="region"
          aria-label="Portal history"
          tabIndex={0}
        >
          <EventList events={value.history} />
        </div>
      </details>
    </>
  );
}

export function PortalDetailsPage() {
  const { id: routeID } = useParams();
  const id = parsePortalID(routeID);
  return (
    <section className={styles.page}>
      <h1>Portal Details</h1>
      {id === null ? (
        <p role="alert">Invalid Portal ID</p>
      ) : (
        <PortalDetailsResource id={id} />
      )}
    </section>
  );
}

import {
  useEffect,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";

import { createLiveResource } from "../api/live-resource";
import { EventList } from "../components/events/EventList";
import { useFeedback } from "../components/feedback/FeedbackProvider";
import { EventFilters } from "../features/event-log/EventFilters";
import {
  filterEvents,
  type EventFilter,
} from "../features/event-log/event-filter";
import { consumeNavigationSignal } from "../features/tutorial/navigation-signal";
import {
  useSnapshotContext,
  useSnapshotState,
} from "../state/SnapshotProvider";
import styles from "./EventLogPage.module.css";
import { commandsEnabled } from "../state/command-health";

const emptyFilter: EventFilter = {
  eventTypes: new Set(),
  portalId: null,
  observerId: null,
  planeId: null,
};

export function EventLogPage() {
  const { api, store } = useSnapshotContext();
  const snapshotState = useSnapshotState();
  const { snapshot } = snapshotState;
  const enabled = commandsEnabled(snapshotState);
  const feedback = useFeedback();
  const [filter, setFilter] = useState<EventFilter>(emptyFilter);
  const lifecycle = useRef(0);
  const resource = useMemo(
    () => createLiveResource((signal) => api.events(signal)),
    [api],
  );
  const state = useSyncExternalStore(
    resource.subscribe,
    resource.getSnapshot,
    resource.getSnapshot,
  );
  useEffect(() => resource.refresh(), [resource, snapshot?.generated_at]);
  useEffect(() => {
    lifecycle.current += 1;
    const generation = lifecycle.current;
    return () =>
      queueMicrotask(() => {
        if (lifecycle.current === generation) resource.dispose();
      });
  }, [resource]);
  useEffect(() => {
    if (!commandsEnabled(store.getState())) return;
    const request = consumeNavigationSignal(snapshot?.app ?? null, {
      kind: "events",
    });
    if (!request) return;
    void api
      .tutorialSignal(request)
      .then((next) => store.acceptSnapshot(next))
      .catch((error: unknown) =>
        feedback.notify(
          error instanceof Error ? error.message : "Tutorial signal failed",
        ),
      );
  }, [api, feedback, snapshot?.app, store, enabled]);

  const visible = filterEvents(state.data ?? [], filter);
  return (
    <section className={styles.page}>
      <h1>Event Log</h1>
      <EventFilters onChange={setFilter} />
      {state.error && !state.data ? (
        <div role="alert">
          <p>Unable to load Event Log.</p>
          <button onClick={() => resource.refresh()} type="button">
            Retry
          </button>
        </div>
      ) : (
        <EventList
          emptyMessage={
            state.data?.length === 0
              ? "No events recorded yet."
              : "No events match the active filters."
          }
          events={visible}
        />
      )}
    </section>
  );
}

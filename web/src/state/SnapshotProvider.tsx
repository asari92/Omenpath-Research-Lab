import {
  createContext,
  type ReactNode,
  useContext,
  useEffect,
  useMemo,
  useSyncExternalStore,
} from "react";

import { createApiClient, type OmenpathApi } from "../api/client";
import { createRealtimeClient } from "../api/realtime";
import {
  createSnapshotStore,
  type SnapshotStore,
  type SnapshotStoreState,
} from "./snapshot-store";

interface SnapshotContextValue {
  store: SnapshotStore;
  api: OmenpathApi;
}

const SnapshotContext = createContext<SnapshotContextValue | null>(null);

export interface SnapshotProviderProps {
  children: ReactNode;
  store?: SnapshotStore;
  api?: OmenpathApi;
}

export function SnapshotProvider({
  children,
  store: suppliedStore,
  api: suppliedApi,
}: SnapshotProviderProps) {
  const value = useMemo(
    () => ({
      store: suppliedStore ?? createSnapshotStore(),
      api: suppliedApi ?? createApiClient(),
    }),
    [suppliedApi, suppliedStore],
  );

  useEffect(() => {
    const controller = new AbortController();
    let realtime: ReturnType<typeof createRealtimeClient> | null = null;
    value.store.setBootstrap("loading");
    void value.api
      .state(controller.signal)
      .then((snapshot) => {
        if (controller.signal.aborted) return;
        if (!value.store.acceptSnapshot(snapshot)) {
          value.store.setBootstrap("failed");
          return;
        }
        if (typeof WebSocket !== "undefined") {
          realtime = createRealtimeClient(value.store);
          realtime.start();
        }
      })
      .catch((error: unknown) => {
        if (!controller.signal.aborted) {
          value.store.setBootstrap("failed");
          value.store.setProtocolError(
            error instanceof Error ? error.message : "State bootstrap failed",
          );
        }
      });
    return () => {
      controller.abort();
      realtime?.stop();
    };
  }, [value]);

  return (
    <SnapshotContext.Provider value={value}>
      {children}
    </SnapshotContext.Provider>
  );
}

export function useSnapshotContext(): SnapshotContextValue {
  const value = useContext(SnapshotContext);
  if (!value)
    throw new Error("useSnapshotContext must be used inside SnapshotProvider");
  return value;
}

export function useSnapshotState(): SnapshotStoreState {
  const { store } = useSnapshotContext();
  return useSyncExternalStore(store.subscribe, store.getState, store.getState);
}

import type { SnapshotStore } from "../state/snapshot-store";
import type { StateSnapshot } from "./types";

interface WebSocketLike {
  readyState: number;
  onopen: (() => void) | null;
  onmessage: ((event: MessageEvent<string>) => void) | null;
  onclose: (() => void) | null;
  onerror: (() => void) | null;
  close(): void;
}

interface WebSocketConstructor {
  new (url: string): WebSocketLike;
}

export interface RealtimeClient {
  start(): void;
  stop(): void;
  nextDelay(): number;
}

const delays = [1000, 2000, 5000, 10000] as const;

function realtimeURL(location: Location): string {
  const protocol = location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${location.host}/ws/lab`;
}

export function createRealtimeClient(
  store: SnapshotStore,
  WebSocketImpl: WebSocketConstructor = WebSocket as unknown as WebSocketConstructor,
): RealtimeClient {
  let socket: WebSocketLike | null = null;
  let timer: ReturnType<typeof setTimeout> | null = null;
  let stopped = true;
  let attempt = 0;
  let scheduledDelay: number = delays[0];

  const connect = () => {
    if (stopped) return;
    store.setConnection(attempt === 0 ? "connecting" : "reconnecting");
    socket = new WebSocketImpl(realtimeURL(window.location));
    socket.onopen = () => store.setConnection("connected");
    socket.onmessage = (event) => {
      try {
        const snapshot = JSON.parse(event.data) as StateSnapshot;
        if (store.acceptSnapshot(snapshot)) attempt = 0;
      } catch {
        store.setProtocolError("Invalid WebSocket snapshot");
      }
    };
    socket.onerror = () => store.setConnection("reconnecting");
    socket.onclose = () => {
      if (stopped) return;
      store.setConnection("reconnecting");
      scheduledDelay = delays[Math.min(attempt, delays.length - 1)];
      attempt += 1;
      timer = setTimeout(connect, scheduledDelay);
    };
  };

  return {
    start() {
      if (!stopped) return;
      stopped = false;
      connect();
    },
    stop() {
      stopped = true;
      if (timer !== null) clearTimeout(timer);
      timer = null;
      if (socket) {
        socket.onclose = null;
        socket.close();
      }
      socket = null;
      store.setConnection("offline");
    },
    nextDelay: () => scheduledDelay,
  };
}

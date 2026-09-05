import { ApiError, isApiErrorBody } from "./errors";
import type {
  EventDTO,
  PortalDetails,
  StateSnapshot,
  TutorialSignalRequest,
} from "./types";

export { ApiError } from "./errors";

export interface OmenpathApi {
  state(signal?: AbortSignal): Promise<StateSnapshot>;
  portal(id: number, signal?: AbortSignal): Promise<PortalDetails>;
  events(signal?: AbortSignal): Promise<EventDTO[]>;
  stabilize(id: number): Promise<StateSnapshot>;
  close(id: number, confirm?: boolean): Promise<StateSnapshot>;
  sendObserver(id: number, confirm?: boolean): Promise<StateSnapshot>;
  recallObserver(id: number, confirm?: boolean): Promise<StateSnapshot>;
  openExtraction(planeId: number): Promise<StateSnapshot>;
  startTutorial(): Promise<StateSnapshot>;
  resetTutorial(): Promise<StateSnapshot>;
  tutorialSignal(request: TutorialSignalRequest): Promise<StateSnapshot>;
  startLive(): Promise<StateSnapshot>;
}

async function requestJSON<T>(
  baseURL: string,
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const response = await fetch(`${baseURL}${path}`, {
    ...init,
    headers: {
      Accept: "application/json",
      ...(init.body === undefined
        ? {}
        : { "Content-Type": "application/json" }),
      ...init.headers,
    },
  });
  const body: unknown = await response.json();
  if (!response.ok) {
    if (isApiErrorBody(body)) {
      throw new ApiError(
        response.status,
        body.error.code,
        body.error.confirmable,
        body.error.message,
      );
    }
    throw new ApiError(
      response.status,
      "INVALID_RESPONSE",
      false,
      "Invalid server error response",
    );
  }
  return body as T;
}

function post<T>(baseURL: string, path: string, body: object): Promise<T> {
  return requestJSON<T>(baseURL, path, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function createApiClient(baseURL = ""): OmenpathApi {
  return {
    state: (signal) => requestJSON(baseURL, "/api/state", { signal }),
    portal: (id, signal) =>
      requestJSON(baseURL, `/api/portals/${id}`, { signal }),
    events: (signal) => requestJSON(baseURL, "/api/events", { signal }),
    stabilize: (id) => post(baseURL, `/api/portals/${id}/stabilize`, {}),
    close: (id, confirm = false) =>
      post(baseURL, `/api/portals/${id}/close`, { confirm }),
    sendObserver: (id, confirm = false) =>
      post(baseURL, `/api/portals/${id}/send-observer`, { confirm }),
    recallObserver: (id, confirm = false) =>
      post(baseURL, `/api/portals/${id}/recall-observer`, { confirm }),
    openExtraction: (planeId) =>
      post(baseURL, "/api/extraction/open", { plane_id: planeId }),
    startTutorial: () => post(baseURL, "/api/tutorial/start", {}),
    resetTutorial: () => post(baseURL, "/api/tutorial/reset", {}),
    tutorialSignal: (request) => post(baseURL, "/api/tutorial/signal", request),
    startLive: () => post(baseURL, "/api/live/start", {}),
  };
}

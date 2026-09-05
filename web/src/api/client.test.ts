import { afterEach, describe, expect, it, vi } from 'vitest';

import { portalDetails, snapshotAt } from '../test/builders';
import { ApiError, createApiClient } from './client';

function jsonResponse(value: unknown, status = 200): Response {
  return new Response(JSON.stringify(value), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

describe('API client', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('GET state/details/events parses success and structured errors', async () => {
    const snapshot = snapshotAt();
    const details = portalDetails();
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse(snapshot))
      .mockResolvedValueOnce(jsonResponse(details))
      .mockResolvedValueOnce(jsonResponse([]))
      .mockResolvedValueOnce(
        jsonResponse(
          { error: { code: 'PORTAL_NOT_FOUND', message: 'portal not found', confirmable: false } },
          404,
        ),
      );
    vi.stubGlobal('fetch', fetchMock);
    const api = createApiClient();

    await expect(api.state()).resolves.toEqual(snapshot);
    await expect(api.portal(42)).resolves.toEqual(details);
    await expect(api.events()).resolves.toEqual([]);
    await expect(api.portal(99)).rejects.toEqual(
      expect.objectContaining<ApiError>({
        status: 404,
        code: 'PORTAL_NOT_FOUND',
        confirmable: false,
      }),
    );
  });

  it('each portal command uses exact endpoint and confirm body', async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(jsonResponse(snapshotAt()));
    vi.stubGlobal('fetch', fetchMock);
    const api = createApiClient();

    await api.stabilize(7);
    await api.close(7, true);
    await api.sendObserver(7, false);
    await api.recallObserver(7, true);

    expect(fetchMock.mock.calls.map(([url, init]) => [url, init?.body])).toEqual([
      ['/api/portals/7/stabilize', '{}'],
      ['/api/portals/7/close', '{"confirm":true}'],
      ['/api/portals/7/send-observer', '{"confirm":false}'],
      ['/api/portals/7/recall-observer', '{"confirm":true}'],
    ]);
  });

  it('extraction/tutorial/signal/live use exact request shapes', async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(jsonResponse(snapshotAt()));
    vi.stubGlobal('fetch', fetchMock);
    const api = createApiClient();

    await api.openExtraction(3);
    await api.startTutorial();
    await api.resetTutorial();
    await api.tutorialSignal({ signal: 'PORTAL_DETAILS_OPENED', portal_id: 42 });
    await api.startLive();

    expect(fetchMock.mock.calls.map(([url, init]) => [url, init?.body])).toEqual([
      ['/api/extraction/open', '{"plane_id":3}'],
      ['/api/tutorial/start', '{}'],
      ['/api/tutorial/reset', '{}'],
      ['/api/tutorial/signal', '{"signal":"PORTAL_DETAILS_OPENED","portal_id":42}'],
      ['/api/live/start', '{}'],
    ]);
  });
});

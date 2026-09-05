import { describe, expect, it, vi } from 'vitest';

import { snapshotAt } from '../test/builders';
import { createSnapshotStore } from './snapshot-store';

describe('snapshot store', () => {
  it('bootstrap snapshot is accepted atomically', () => {
    const store = createSnapshotStore();
    const listener = vi.fn();
    store.subscribe(listener);

    expect(store.acceptSnapshot(snapshotAt())).toBe(true);
    expect(store.getState().snapshot?.lab.current_energy).toBe(100);
    expect(listener).toHaveBeenCalledOnce();
  });

  it('older REST response cannot replace newer WebSocket snapshot', () => {
    const store = createSnapshotStore();
    store.acceptSnapshot(snapshotAt('2026-09-05T10:00:02Z', 72));

    expect(store.acceptSnapshot(snapshotAt('2026-09-05T10:00:01Z', 10))).toBe(false);
    expect(store.getState().snapshot?.lab.current_energy).toBe(72);
  });

  it('equal generated_at is idempotently accepted', () => {
    const store = createSnapshotStore();
    store.acceptSnapshot(snapshotAt('2026-09-05T10:00:02Z', 72));

    expect(store.acceptSnapshot(snapshotAt('2026-09-05T10:00:02Z', 73))).toBe(true);
    expect(store.getState().snapshot?.lab.current_energy).toBe(73);
  });

  it('invalid generated_at becomes protocol error', () => {
    const store = createSnapshotStore();

    expect(store.acceptSnapshot(snapshotAt('invalid'))).toBe(false);
    expect(store.getState().protocolError).toMatch(/generated_at/i);
  });

  it('one command key does not block another command', () => {
    const store = createSnapshotStore();
    store.beginCommand('42:CLOSE');

    expect(store.getState().commandKeys.has('42:CLOSE')).toBe(true);
    expect(store.getState().commandKeys.has('42:SEND')).toBe(false);
    store.endCommand('42:CLOSE');
    expect(store.getState().commandKeys).toHaveLength(0);
  });
});

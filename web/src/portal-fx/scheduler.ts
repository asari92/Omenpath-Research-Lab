export type PortalFrame = (timestamp: number) => void;

export interface PortalScheduler {
  register(frame: PortalFrame): () => void;
}

export function createPortalScheduler(
  requestFrame: (
    callback: FrameRequestCallback,
  ) => number = requestAnimationFrame,
  cancelFrame: (handle: number) => void = cancelAnimationFrame,
): PortalScheduler {
  const callbacks = new Set<PortalFrame>();
  let handle: number | null = null;

  const schedule = () => {
    if (handle === null && callbacks.size > 0) handle = requestFrame(tick);
  };
  const tick = (timestamp: number) => {
    handle = null;
    callbacks.forEach((callback) => callback(timestamp));
    schedule();
  };

  return {
    register(frame) {
      callbacks.add(frame);
      schedule();
      return () => {
        callbacks.delete(frame);
        if (callbacks.size === 0 && handle !== null) {
          cancelFrame(handle);
          handle = null;
        }
      };
    },
  };
}

export const portalScheduler = createPortalScheduler();

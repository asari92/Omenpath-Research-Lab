import { describe, expect, it, vi } from "vitest";

import { createPortalScheduler } from "./scheduler";

describe("shared portal scheduler", () => {
  it("uses one RAF for every registered portal and stops after unregistration", () => {
    const frames: FrameRequestCallback[] = [];
    const raf = vi.fn((callback: FrameRequestCallback) => {
      frames.push(callback);
      return frames.length;
    });
    const cancel = vi.fn();
    const scheduler = createPortalScheduler(raf, cancel);
    const first = vi.fn();
    const second = vi.fn();

    const removeFirst = scheduler.register(first);
    const removeSecond = scheduler.register(second);
    expect(raf).toHaveBeenCalledOnce();
    frames.shift()?.(16);
    expect(first).toHaveBeenCalledWith(16);
    expect(second).toHaveBeenCalledWith(16);

    removeFirst();
    removeSecond();
    expect(cancel).toHaveBeenCalledOnce();
  });
});

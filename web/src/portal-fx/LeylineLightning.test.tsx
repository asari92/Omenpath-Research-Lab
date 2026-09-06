import { act, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { LeylineLightning, lightningCanvasScale } from "./LeylineLightning";

function canvasContext() {
  return {
    beginPath: vi.fn(),
    clearRect: vi.fn(),
    lineTo: vi.fn(),
    moveTo: vi.fn(),
    restore: vi.fn(),
    save: vi.fn(),
    setLineDash: vi.fn(),
    setTransform: vi.fn(),
    stroke: vi.fn(),
  } as unknown as CanvasRenderingContext2D;
}

describe("LeylineLightning", () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("sleeps between strikes instead of running a permanent animation frame loop", () => {
    vi.useFakeTimers();
    vi.stubGlobal("matchMedia", () => ({ matches: false }));
    vi.stubGlobal(
      "ResizeObserver",
      class {
        constructor(private readonly callback: ResizeObserverCallback) {}
        observe() {
          this.callback([], this as unknown as ResizeObserver);
        }
        disconnect() {}
      },
    );
    let nextFrame: FrameRequestCallback | null = null;
    const requestFrame = vi.fn((callback: FrameRequestCallback) => {
      nextFrame = callback;
      return requestFrame.mock.calls.length;
    });
    const runFrame = (timestamp: number) => {
      const callback = nextFrame as FrameRequestCallback | null;
      expect(callback).not.toBeNull();
      callback?.(timestamp);
    };
    const cancelFrame = vi.fn();
    vi.stubGlobal("requestAnimationFrame", requestFrame);
    vi.stubGlobal("cancelAnimationFrame", cancelFrame);
    const context = canvasContext();
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(
      context,
    );

    const view = render(<LeylineLightning active />);
    expect(requestFrame).not.toHaveBeenCalled();

    act(() => vi.advanceTimersByTime(120));
    expect(requestFrame).toHaveBeenCalledOnce();
    const bornAt = performance.now();
    act(() => runFrame(bornAt));
    const clearsAfterFirstDraw = vi.mocked(context.clearRect).mock.calls.length;
    act(() => runFrame(bornAt + 16));
    expect(context.clearRect).toHaveBeenCalledTimes(clearsAfterFirstDraw);
    act(() => runFrame(bornAt + 34));
    expect(context.clearRect).toHaveBeenCalledTimes(clearsAfterFirstDraw + 1);
    const callsBeforeExpiry = requestFrame.mock.calls.length;
    act(() => runFrame(bornAt + 466));

    expect(requestFrame).toHaveBeenCalledTimes(callsBeforeExpiry);
    expect(vi.getTimerCount()).toBe(1);
    view.unmount();
    expect(vi.getTimerCount()).toBe(0);
  });

  it("limits backing resolution for a full-viewport canvas", () => {
    expect(lightningCanvasScale(1400, 900, 2)).toBe(1);
    expect(lightningCanvasScale(390, 760, 3)).toBe(1.25);
    expect(lightningCanvasScale(1400, 900, 1)).toBe(1);
  });
});

import { act, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { PortalEffect } from "./PortalEffect";
import { portalScheduler } from "./scheduler";

describe("PortalEffect", () => {
  it.each(["CLOSED", "COLLAPSED"] as const)(
    "keeps historical %s artwork static unless an exit is explicitly requested",
    (status) => {
      render(
        <PortalEffect
          portalId={8}
          planeId={1}
          planeName="Agyrem"
          density="static"
          status={status}
        />,
      );
      expect(screen.getByRole("img").parentElement).toHaveAttribute(
        "data-motion",
        "static",
      );
      expect(screen.getByRole("img").parentElement).toHaveAttribute(
        "data-status",
        status,
      );
    },
  );
  it("unregisters heavy animation immediately when the motion preference changes", () => {
    const callbacks = new Set<() => void>();
    const media = {
      matches: false,
      addEventListener: (_event: string, cb: () => void) => callbacks.add(cb),
      removeEventListener: (_event: string, cb: () => void) =>
        callbacks.delete(cb),
    };
    vi.stubGlobal("matchMedia", () => media);
    vi.stubGlobal(
      "IntersectionObserver",
      class {
        observe() {}
        disconnect() {}
      },
    );
    const context = vi
      .spyOn(HTMLCanvasElement.prototype, "getContext")
      .mockReturnValue({} as CanvasRenderingContext2D);
    const unregister = vi.fn();
    const register = vi
      .spyOn(portalScheduler, "register")
      .mockReturnValue(unregister);
    try {
      render(
        <PortalEffect
          portalId={8}
          planeId={1}
          planeName="Agyrem"
          density="high"
        />,
      );
      expect(register).toHaveBeenCalledOnce();
      act(() => {
        media.matches = true;
        callbacks.forEach((cb) => cb());
      });
      expect(unregister).toHaveBeenCalledOnce();
    } finally {
      context.mockRestore();
      register.mockRestore();
      vi.unstubAllGlobals();
    }
  });
  it("does not start a canvas renderer when reduced motion is requested", () => {
    vi.stubGlobal("matchMedia", () => ({
      matches: true,
      addEventListener: () => {},
      removeEventListener: () => {},
    }));
    vi.stubGlobal(
      "IntersectionObserver",
      class {
        observe() {}
        disconnect() {}
      },
    );
    const context = vi.spyOn(HTMLCanvasElement.prototype, "getContext");
    try {
      render(
        <PortalEffect
          portalId={8}
          planeId={1}
          planeName="Agyrem"
          density="high"
        />,
      );
      expect(context).not.toHaveBeenCalled();
    } finally {
      context.mockRestore();
      vi.unstubAllGlobals();
    }
  });
  it("marks entrance and terminal motion while terminal artwork stays semantic", () => {
    const view = render(
      <PortalEffect
        portalId={8}
        planeId={1}
        planeName="Agyrem"
        density="static"
      />,
    );
    expect(screen.getByRole("img").parentElement).toHaveAttribute(
      "data-motion",
      "entering",
    );
    view.rerender(
      <PortalEffect
        portalId={8}
        planeId={1}
        planeName="Agyrem"
        density="static"
        status="CLOSED"
        exiting
      />,
    );
    expect(screen.getByRole("img").parentElement).toHaveAttribute(
      "data-motion",
      "terminal",
    );
  });
  it("keeps local Plane art semantic and Canvas decorative", () => {
    render(
      <PortalEffect
        portalId={7}
        planeId={1}
        planeName="The Abyss"
        density="low"
      />,
    );

    expect(screen.getByRole("img", { name: "The Abyss" })).toHaveAttribute(
      "src",
      expect.stringMatching(/^\/planes\//),
    );
    expect(screen.getByTestId("portal-canvas")).toHaveAttribute(
      "aria-hidden",
      "true",
    );
  });
});

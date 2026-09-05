import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PortalEffect } from "./PortalEffect";

describe("PortalEffect", () => {
  it("marks entrance and terminal motion while terminal artwork stays semantic", () => {
    const view = render(<PortalEffect portalId={8} planeId={1} planeName="Agyrem" density="static" />);
    expect(screen.getByRole("img").parentElement).toHaveAttribute("data-motion", "entering");
    view.rerender(<PortalEffect portalId={8} planeId={1} planeName="Agyrem" density="static" status="CLOSED" />);
    expect(screen.getByRole("img").parentElement).toHaveAttribute("data-motion", "terminal");
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

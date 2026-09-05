import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PortalEffect } from "./PortalEffect";

describe("PortalEffect", () => {
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

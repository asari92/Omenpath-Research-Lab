import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ArtCreditsDialog } from "./ArtCreditsDialog";
import { planeArtEntries } from "../../assets/plane-art";

describe("ArtCreditsDialog", () => {
  it("renders safe source, credit and policy links", () => {
    render(<ArtCreditsDialog open onClose={vi.fn()} />);

    const source = screen.getAllByRole("link", { name: /source/i })[0];
    expect(source).toHaveAttribute("target", "_blank");
    expect(source).toHaveAttribute("rel", "noreferrer");
    expect(screen.getByText(/unofficial fan content/i)).toBeInTheDocument();
    const links = screen.getAllByRole("link", { name: /^source$/i });
    expect(links).toHaveLength(85);
    planeArtEntries().forEach((entry, index) => {
      expect(links[index]).toHaveAttribute("href", entry.source_url);
      expect(entry.source_url).toMatch(
        /^(https:\/\/|\/plane-art-provenance\.json$)/,
      );
    });
    expect(
      screen.getByText(
        /OpenAI image generation — original Bloomburrow interpretation/,
      ),
    ).toBeInTheDocument();
  });
});

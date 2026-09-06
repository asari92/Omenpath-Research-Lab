import { readFile } from "node:fs/promises";
import { createHash } from "node:crypto";
import path from "node:path";
import audit from "../../../data/plane_art_audit.json";

import sharp from "sharp";
import { describe, expect, it } from "vitest";

import seed from "../../../data/mtg_planes_expanded_seed.json";
import manifest from "../../../data/plane_images_manifest.json";
import { planeArt, planeArtEntries } from "./plane-art";

describe("local plane artwork", () => {
  it("passes the complete reviewed text-free artwork audit", () => {
    expect(audit.map((entry) => entry.plane_id)).toEqual(
      seed.planes.map((plane) => plane.id),
    );
    expect(
      audit
        .filter(
          (entry) => !entry.reviewed || !entry.text_free || !entry.frame_free,
        )
        .map((entry) => entry.plane_id),
    ).toEqual([]);
    expect(
      manifest.entries
        .filter((entry) => entry.fallback)
        .map((entry) => entry.plane_id),
    ).toEqual([]);
  });
  it("maps every seed ID exactly once to a unique local file", () => {
    const seedIDs = seed.planes.map((plane) => plane.id).sort((a, b) => a - b);
    const entries = planeArtEntries();
    expect(
      entries.map((entry) => entry.plane_id).sort((a, b) => a - b),
    ).toEqual(seedIDs);
    expect(new Set(entries.map((entry) => entry.local_path)).size).toBe(85);
    expect(entries.every((entry) => !/^https?:/i.test(entry.local_path))).toBe(
      true,
    );
  });

  it("stores bounded WebP assets with provenance", async () => {
    for (const entry of manifest.entries) {
      expect(entry).toEqual(
        expect.objectContaining({
          source_url: expect.any(String),
          credit: expect.any(String),
          policy_url: expect.any(String),
          sha256: expect.stringMatching(/^[a-f0-9]{64}$/),
          fallback: expect.any(Boolean),
        }),
      );
      const absolute = path.resolve(
        process.cwd(),
        "public",
        entry.local_path.replace(/^\//, ""),
      );
      const file = await readFile(absolute);
      const metadata = await sharp(file).metadata();
      expect(metadata.format).toBe("webp");
      expect(metadata.width).toBe(512);
      expect(metadata.height).toBe(512);
      expect(createHash("sha256").update(file).digest("hex")).toBe(
        entry.sha256,
      );
      expect(entry.fallback).toBe(false);
      await expect(sharp(file).raw().toBuffer()).resolves.toBeInstanceOf(
        Buffer,
      );
      expect(file.byteLength).toBeLessThanOrEqual(180 * 1024);
    }
  });

  it("unknown Plane ID returns the committed generic fallback", () => {
    expect(planeArt(9999).local_path).toBe("/planes/fallback.webp");
  });
});

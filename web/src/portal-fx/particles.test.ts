import { describe, expect, it } from "vitest";

import {
  densityProfile,
  distance,
  dot,
  radialUnit,
  spawnParticle,
} from "./particles";

describe("portal particles", () => {
  it("spawns on the circumference with outward and tangential motion", () => {
    const values = [0.25, 0.5, 0.5, 0.5];
    const particle = spawnParticle({
      center: { x: 88, y: 88 },
      radius: 65,
      random: () => values.shift() ?? 0.5,
    });

    expect(distance(particle.position, { x: 88, y: 88 })).toBeCloseTo(65, 5);
    expect(
      dot(radialUnit(particle, { x: 88, y: 88 }), particle.velocity),
    ).toBeGreaterThan(0);
    const radial = radialUnit(particle, { x: 88, y: 88 });
    expect(
      Math.abs(radial.x * particle.velocity.y - radial.y * particle.velocity.x),
    ).toBeGreaterThan(0);
  });

  it("caps high density at 170 and low at 42", () => {
    expect(densityProfile("high").maxParticles).toBe(170);
    expect(densityProfile("low").maxParticles).toBe(42);
    expect(densityProfile("static").spawnPerSecond).toBe(0);
  });
});

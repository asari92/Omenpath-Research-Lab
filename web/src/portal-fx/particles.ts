export interface Point {
  x: number;
  y: number;
}

export interface PortalParticle {
  position: Point;
  velocity: Point;
  age: number;
  lifetime: number;
  size: number;
}

export type PortalEffectDensity = "high" | "low" | "static";

export interface ParticleProfile {
  maxParticles: number;
  spawnPerSecond: number;
  tangentialSpeed: readonly [number, number];
  outwardSpeed: readonly [number, number];
}

const profiles: Record<PortalEffectDensity, ParticleProfile> = {
  high: {
    maxParticles: 170,
    spawnPerSecond: 86,
    tangentialSpeed: [42, 92],
    outwardSpeed: [18, 58],
  },
  low: {
    maxParticles: 42,
    spawnPerSecond: 18,
    tangentialSpeed: [32, 68],
    outwardSpeed: [12, 38],
  },
  static: {
    maxParticles: 0,
    spawnPerSecond: 0,
    tangentialSpeed: [0, 0],
    outwardSpeed: [0, 0],
  },
};

export function densityProfile(density: PortalEffectDensity): ParticleProfile {
  return profiles[density];
}

export function spawnParticle({
  center,
  radius,
  random = Math.random,
  profile = profiles.high,
}: {
  center: Point;
  radius: number;
  random?: () => number;
  profile?: ParticleProfile;
}): PortalParticle {
  const angle = random() * Math.PI * 2;
  const radial = { x: Math.cos(angle), y: Math.sin(angle) };
  const tangent = { x: -radial.y, y: radial.x };
  const direction = random() < 0.5 ? -1 : 1;
  const tangential = between(profile.tangentialSpeed, random());
  const outward = between(profile.outwardSpeed, random());
  return {
    position: {
      x: center.x + radial.x * radius,
      y: center.y + radial.y * radius,
    },
    velocity: {
      x: radial.x * outward + tangent.x * tangential * direction,
      y: radial.y * outward + tangent.y * tangential * direction,
    },
    age: 0,
    lifetime: 0.45 + random() * 0.7,
    size: 0.8 + random() * 1.8,
  };
}

export function stepParticle(
  particle: PortalParticle,
  seconds: number,
): PortalParticle {
  return {
    ...particle,
    position: {
      x: particle.position.x + particle.velocity.x * seconds,
      y: particle.position.y + particle.velocity.y * seconds,
    },
    age: particle.age + seconds,
  };
}

export function radialUnit(particle: PortalParticle, center: Point): Point {
  const x = particle.position.x - center.x;
  const y = particle.position.y - center.y;
  const length = Math.hypot(x, y) || 1;
  return { x: x / length, y: y / length };
}

export function distance(a: Point, b: Point): number {
  return Math.hypot(a.x - b.x, a.y - b.y);
}

export function dot(a: Point, b: Point): number {
  return a.x * b.x + a.y * b.y;
}

function between(range: readonly [number, number], sample: number): number {
  return range[0] + (range[1] - range[0]) * sample;
}

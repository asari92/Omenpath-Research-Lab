import { useEffect, useRef } from "react";

import { planeArt } from "../assets/plane-art";
import styles from "./PortalEffect.module.css";
import {
  densityProfile,
  type PortalEffectDensity,
  type PortalParticle,
  spawnParticle,
  stepParticle,
} from "./particles";
import { portalScheduler } from "./scheduler";

export interface PortalEffectProps {
  portalId: number;
  planeId: number;
  planeName: string;
  density: PortalEffectDensity;
}

export function PortalEffect({
  portalId,
  planeId,
  planeName,
  density,
}: PortalEffectProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const art = planeArt(planeId);
  const hue = (portalId * 67 + 118) % 360;

  useEffect(() => {
    const canvas = canvasRef.current;
    if (
      !canvas ||
      density === "static" ||
      typeof IntersectionObserver === "undefined"
    )
      return;
    const reduced =
      window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false;
    if (reduced) return;
    const context = canvas.getContext("2d");
    if (!context) return;
    const profile = densityProfile(density);
    let particles: PortalParticle[] = [];
    let visible = true;
    let previous = performance.now();
    let spawnCarry = 0;
    const observer = new IntersectionObserver(([entry]) => {
      visible = entry?.isIntersecting ?? false;
    });
    observer.observe(canvas);
    const unregister = portalScheduler.register((timestamp) => {
      if (!visible || document.hidden) {
        previous = timestamp;
        return;
      }
      const seconds = Math.min(
        0.05,
        Math.max(0, (timestamp - previous) / 1000),
      );
      previous = timestamp;
      const size = canvas.clientWidth || 176;
      if (canvas.width !== size || canvas.height !== size) {
        canvas.width = size;
        canvas.height = size;
      }
      spawnCarry += profile.spawnPerSecond * seconds;
      while (spawnCarry >= 1 && particles.length < profile.maxParticles) {
        particles.push(
          spawnParticle({
            center: { x: size / 2, y: size / 2 },
            radius: size * 0.39,
            profile,
          }),
        );
        spawnCarry -= 1;
      }
      particles = particles
        .map((particle) => stepParticle(particle, seconds))
        .filter((p) => p.age < p.lifetime);
      context.clearRect(0, 0, size, size);
      context.fillStyle = `hsl(${hue} 95% 70%)`;
      context.shadowColor = `hsl(${hue} 100% 58%)`;
      context.shadowBlur = 8;
      for (const particle of particles) {
        context.globalAlpha = Math.max(0, 1 - particle.age / particle.lifetime);
        context.beginPath();
        context.arc(
          particle.position.x,
          particle.position.y,
          particle.size,
          0,
          Math.PI * 2,
        );
        context.fill();
      }
      context.globalAlpha = 1;
    });
    return () => {
      observer.disconnect();
      unregister();
    };
  }, [density, hue]);

  return (
    <div
      className={styles.portal}
      style={{ "--portal-hue": hue } as React.CSSProperties}
    >
      <img alt={planeName} className={styles.art} src={art.local_path} />
      <canvas
        aria-hidden="true"
        className={styles.canvas}
        data-testid="portal-canvas"
        ref={canvasRef}
      />
    </div>
  );
}

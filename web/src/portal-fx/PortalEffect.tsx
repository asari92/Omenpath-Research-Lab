import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useSyncExternalStore,
} from "react";

import { planeArt } from "../assets/plane-art";
import type { PortalStatus } from "../api/types";
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
  status?: PortalStatus;
}

export function PortalEffect({
  portalId,
  planeId,
  planeName,
  density,
  status = "OPEN",
}: PortalEffectProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const art = planeArt(planeId);
  const hue = (portalId * 67 + 118) % 360;
  const media = useMemo(
    () => window.matchMedia?.("(prefers-reduced-motion: reduce)"),
    [],
  );
  const subscribe = useCallback(
    (notify: () => void) => {
      media?.addEventListener?.("change", notify);
      return () => media?.removeEventListener?.("change", notify);
    },
    [media],
  );
  const reduced = useSyncExternalStore(
    subscribe,
    () => media?.matches ?? false,
    () => false,
  );

  useEffect(() => {
    const canvas = canvasRef.current;
    if (
      !canvas ||
      density === "static" ||
      status !== "OPEN" ||
      typeof IntersectionObserver === "undefined"
    )
      return;
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
  }, [density, hue, status, reduced]);

  return (
    <div
      className={styles.portal}
      data-status={status}
      data-motion={status === "OPEN" ? "entering" : "terminal"}
      style={{ "--portal-hue": hue } as React.CSSProperties}
    >
      <img alt={planeName} className={styles.art} src={art.local_path} />
      <canvas
        aria-hidden="true"
        className={styles.canvas}
        data-density={density}
        data-testid="portal-canvas"
        ref={canvasRef}
      />
    </div>
  );
}

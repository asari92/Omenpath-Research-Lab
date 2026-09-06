import { useEffect, useRef } from "react";

// Segmented Canvas lightning approach adapted from Javascript-Lightning-Effect
// by diwsi (MIT): https://github.com/diwsi/Javascript-Lightning-Effect

interface Point {
  x: number;
  y: number;
}

interface BoltPath {
  points: Point[];
  strength: number;
}

interface Strike {
  bornAt: number;
  paths: BoltPath[];
}

const initialStrikeDelay = 120;
const strikeLifetime = 465;
const drawInterval = 1000 / 30;
const largeCanvasArea = 800_000;

const randomBetween = (minimum: number, maximum: number) =>
  minimum + Math.random() * (maximum - minimum);

export function lightningCanvasScale(
  width: number,
  height: number,
  devicePixelRatio: number,
): number {
  const maximum = width * height >= largeCanvasArea ? 1 : 1.25;
  return Math.min(Math.max(1, devicePixelRatio || 1), maximum);
}

function boltPath(
  start: Point,
  end: Point,
  segments: number,
  jitter: number,
): Point[] {
  const dx = end.x - start.x;
  const dy = end.y - start.y;
  const length = Math.max(1, Math.hypot(dx, dy));
  const normalX = -dy / length;
  const normalY = dx / length;
  let previousOffset = 0;

  return Array.from({ length: segments + 1 }, (_, index) => {
    const progress = index / segments;
    if (index === 0) return start;
    if (index === segments) return end;
    const envelope = Math.sin(Math.PI * progress);
    previousOffset =
      previousOffset * 0.28 + randomBetween(-jitter, jitter) * envelope;
    return {
      x: start.x + dx * progress + normalX * previousOffset,
      y: start.y + dy * progress + normalY * previousOffset,
    };
  });
}

function clampPoint(point: Point, width: number, height: number): Point {
  return {
    x: Math.max(0, Math.min(width, point.x)),
    y: Math.max(0, Math.min(height, point.y)),
  };
}

function createStrike(width: number, height: number, bornAt: number): Strike {
  const horizontal = Math.random() > 0.5;
  const reverse = Math.random() > 0.5;
  const start = horizontal
    ? {
        x: reverse ? width : 0,
        y: randomBetween(height * 0.08, height * 0.92),
      }
    : {
        x: randomBetween(width * 0.08, width * 0.92),
        y: reverse ? height : 0,
      };
  const end = horizontal
    ? {
        x: reverse ? 0 : width,
        y: randomBetween(height * 0.08, height * 0.92),
      }
    : {
        x: randomBetween(width * 0.08, width * 0.92),
        y: reverse ? 0 : height,
      };
  const span = Math.hypot(end.x - start.x, end.y - start.y);
  const main = boltPath(start, end, 34, span * 0.035);
  const paths: BoltPath[] = [{ points: main, strength: 1 }];
  const branchCount = 3 + Math.floor(Math.random() * 4);

  for (let branchIndex = 0; branchIndex < branchCount; branchIndex += 1) {
    const anchorIndex =
      5 + Math.floor(Math.random() * Math.max(1, main.length - 11));
    const anchor = main[anchorIndex];
    const previous = main[Math.max(0, anchorIndex - 1)];
    const direction = Math.atan2(anchor.y - previous.y, anchor.x - previous.x);
    const side = Math.random() > 0.5 ? 1 : -1;
    const branchLength = randomBetween(
      Math.min(width, height) * 0.12,
      Math.min(width, height) * 0.32,
    );
    const angle = direction + side * randomBetween(0.45, 1.05);
    const branchEnd = clampPoint(
      {
        x: anchor.x + Math.cos(angle) * branchLength,
        y: anchor.y + Math.sin(angle) * branchLength,
      },
      width,
      height,
    );
    const branch = boltPath(anchor, branchEnd, 14, branchLength * 0.07);
    paths.push({ points: branch, strength: randomBetween(0.45, 0.72) });

    if (Math.random() > 0.55) {
      const twigAnchor = branch[Math.floor(branch.length * 0.58)];
      const twigLength = branchLength * randomBetween(0.28, 0.46);
      const twigAngle = angle - side * randomBetween(0.65, 1.15);
      const twigEnd = clampPoint(
        {
          x: twigAnchor.x + Math.cos(twigAngle) * twigLength,
          y: twigAnchor.y + Math.sin(twigAngle) * twigLength,
        },
        width,
        height,
      );
      paths.push({
        points: boltPath(twigAnchor, twigEnd, 8, twigLength * 0.08),
        strength: randomBetween(0.25, 0.4),
      });
    }
  }

  return { bornAt, paths };
}

function tracePath(context: CanvasRenderingContext2D, points: Point[]) {
  context.beginPath();
  context.moveTo(points[0].x, points[0].y);
  for (let index = 1; index < points.length; index += 1) {
    context.lineTo(points[index].x, points[index].y);
  }
  context.stroke();
}

function drawStrike(
  context: CanvasRenderingContext2D,
  strike: Strike,
  intensity: number,
  age: number,
) {
  context.lineCap = "round";
  context.lineJoin = "round";
  context.globalCompositeOperation = "lighter";

  for (const path of strike.paths) {
    const alpha = intensity * path.strength;
    context.setLineDash([]);
    context.strokeStyle = `rgba(113, 53, 220, ${alpha * 0.34})`;
    context.shadowColor = "#995cff";
    context.shadowBlur = 20 * path.strength;
    context.lineWidth = 7 * path.strength;
    tracePath(context, path.points);

    context.strokeStyle = `rgba(229, 219, 255, ${alpha * 0.94})`;
    context.shadowColor = "#c6a8ff";
    context.shadowBlur = 7 * path.strength;
    context.lineWidth = Math.max(0.65, 1.65 * path.strength);
    tracePath(context, path.points);

    context.setLineDash([12, 38]);
    context.lineDashOffset = -age * 0.42;
    context.strokeStyle = `rgba(255, 255, 255, ${alpha * 0.9})`;
    context.shadowBlur = 4;
    context.lineWidth = Math.max(0.5, path.strength);
    tracePath(context, path.points);
  }

  context.setLineDash([]);
  context.globalAlpha = 1;
  context.globalCompositeOperation = "source-over";
}

function strikeIntensity(age: number): number {
  if (age < 42) return age / 42;
  if (age < 90) return 1;
  if (age < 145) return 0.18;
  if (age < 205) return 0.72;
  return Math.max(0, 0.55 * (1 - (age - 205) / 260));
}

export function LeylineLightning({
  active,
  className,
}: {
  active: boolean;
  className?: string;
}) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!active || !canvas || typeof ResizeObserver === "undefined") return;
    const context = canvas.getContext("2d");
    if (!context) return;

    let width = 1;
    let height = 1;
    let frame = 0;
    let strikeTimer = 0;
    let strike: Strike | null = null;
    let lastDrawAt = Number.NEGATIVE_INFINITY;
    let stopped = false;
    const reducedMotion =
      window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false;

    const resize = () => {
      const bounds = canvas.getBoundingClientRect();
      width = Math.max(1, bounds.width);
      height = Math.max(1, bounds.height);
      const scale = lightningCanvasScale(
        width,
        height,
        window.devicePixelRatio,
      );
      const backingWidth = Math.round(width * scale);
      const backingHeight = Math.round(height * scale);
      if (canvas.width !== backingWidth) canvas.width = backingWidth;
      if (canvas.height !== backingHeight) canvas.height = backingHeight;
      context.setTransform(scale, 0, 0, scale, 0, 0);
    };
    const clear = () => {
      context.save();
      context.setTransform(1, 0, 0, 1, 0, 0);
      context.clearRect(0, 0, canvas.width, canvas.height);
      context.restore();
    };
    const observer = new ResizeObserver(resize);
    observer.observe(canvas);
    resize();

    if (reducedMotion) {
      strike = createStrike(width, height, 0);
      clear();
      drawStrike(context, strike, 0.2, 0);
      return () => observer.disconnect();
    }

    const cancelWork = () => {
      if (frame !== 0) cancelAnimationFrame(frame);
      if (strikeTimer !== 0) window.clearTimeout(strikeTimer);
      frame = 0;
      strikeTimer = 0;
    };
    const scheduleStrike = (delay: number) => {
      if (stopped || document.hidden) return;
      strikeTimer = window.setTimeout(() => {
        strikeTimer = 0;
        if (stopped || document.hidden) return;
        strike = createStrike(width, height, performance.now());
        lastDrawAt = Number.NEGATIVE_INFINITY;
        frame = requestAnimationFrame(render);
      }, delay);
    };
    const render = (timestamp: number) => {
      frame = 0;
      if (stopped || document.hidden || !strike) return;
      const age = timestamp - strike.bornAt;
      if (age > strikeLifetime) {
        clear();
        strike = null;
        scheduleStrike(randomBetween(720, 1450) - strikeLifetime);
        return;
      }
      if (timestamp - lastDrawAt >= drawInterval - 1) {
        clear();
        drawStrike(context, strike, strikeIntensity(age), age);
        lastDrawAt = timestamp;
      }
      frame = requestAnimationFrame(render);
    };
    const handleVisibility = () => {
      cancelWork();
      strike = null;
      clear();
      if (!document.hidden) scheduleStrike(initialStrikeDelay);
    };
    document.addEventListener("visibilitychange", handleVisibility);
    scheduleStrike(initialStrikeDelay);

    return () => {
      stopped = true;
      cancelWork();
      observer.disconnect();
      document.removeEventListener("visibilitychange", handleVisibility);
      clear();
    };
  }, [active]);

  return (
    <canvas
      aria-hidden="true"
      className={className}
      data-active={active || undefined}
      data-testid="leyline-energy-layer"
      ref={canvasRef}
    />
  );
}

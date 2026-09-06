import { useLayoutEffect, useRef } from "react";
import type { AppDTO } from "../../api/types";

// Place the floating book on the opposite side of the current exercise.
// Measuring presentation never changes simulation state or command targets.
export function useTutorialPlacement(pathname: string, app: AppDTO | null) {
  const ref = useRef<HTMLDivElement>(null);
  useLayoutEffect(() => {
    const host = ref.current;
    const main = host?.parentElement;
    if (!host || !main) return;
    const position = () => {
      const target =
        main.querySelector('[data-tutorial-target="true"]') ??
        main.querySelector('[data-tutorial-command="true"]') ??
        (pathname.startsWith("/portals/")
          ? main.querySelector('[aria-label="Portal commands"]')
          : null);
      const area = main.getBoundingClientRect();
      const book = host.getBoundingClientRect();
      const targetBox = target?.getBoundingClientRect();
      const top = 8;
      const bottom = Math.max(top, area.height - book.height - 8);
      const overlapsTop =
        targetBox &&
        area.top + top + book.height > targetBox.top &&
        area.top + top < targetBox.bottom;
      host.style.top = `${overlapsTop ? bottom : top}px`;
    };
    position();
    const resize =
      typeof ResizeObserver === "undefined"
        ? null
        : new ResizeObserver(position);
    resize?.observe(host);
    resize?.observe(main);
    const mutation = new MutationObserver(position);
    mutation.observe(main, { childList: true, subtree: true });
    window.addEventListener("resize", position);
    return () => {
      resize?.disconnect();
      mutation.disconnect();
      window.removeEventListener("resize", position);
    };
  }, [pathname, app]);
  return ref;
}

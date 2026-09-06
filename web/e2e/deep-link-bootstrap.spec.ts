import { expect, test } from "@playwright/test";
import type { StateSnapshot } from "../src/api/types";

for (const routePath of ["/events", "/portals/1"]) {
  test(`fresh direct ${routePath} establishes one cookie before route REST and WebSocket`, async ({ page, context }) => {
    let release!: () => void;
    const held = new Promise<void>((resolve) => { release = resolve; });
    let stateStarted!: () => void;
    const started = new Promise<void>((resolve) => { stateStarted = resolve; });
    const resourceCookies: Promise<string | undefined>[] = [];
    const cookieResponses: Promise<string | undefined>[] = [];
    const frames: StateSnapshot[] = [];
    page.on("request", (request) => {
      if (new URL(request.url()).pathname === `/api${routePath}`)
        resourceCookies.push(request.allHeaders().then((headers) => headers.cookie));
    });
    page.on("response", (response) => {
      if (new URL(response.url()).pathname.startsWith("/api/"))
        cookieResponses.push(response.allHeaders().then((headers) => headers["set-cookie"]));
    });
    page.on("websocket", (socket) => {
      if (new URL(socket.url()).pathname !== "/ws/lab") return;
      socket.on("framereceived", ({ payload }) => frames.push(JSON.parse(payload.toString()) as StateSnapshot));
    });
    await page.route("**/api/state", async (route) => {
      stateStarted();
      await held;
      await route.continue();
    });
    try {
      await page.goto(routePath);
      await started;
      // Intentionally hold bootstrap so an eager route fetch wins the race.
      await page.waitForTimeout(250);
      expect(resourceCookies).toHaveLength(0);
      expect(frames).toHaveLength(0);
      expect(await context.cookies()).toHaveLength(0);
      release();
      await expect.poll(() => frames.length).toBeGreaterThan(0);
      await expect.poll(() => resourceCookies.length).toBeGreaterThan(0);
      if (routePath === "/events")
        await expect(page.getByText("No events recorded yet.")).toBeVisible();
      else
        await expect(page.getByText("Portal Not Found")).toBeVisible();
      const cookies = (await context.cookies()).filter((cookie) => cookie.name === "omenpath_session");
      expect(cookies).toHaveLength(1);
      expect((await Promise.all(cookieResponses)).filter((header) => header?.includes("omenpath_session="))).toHaveLength(1);
      const response = await page.request.post("/api/tutorial/signal", { data: { signal: "TUTORIAL_INTRO_COMPLETED" } });
      expect(response.ok()).toBe(true);
      await expect.poll(() => frames.at(-1)?.app.tutorial_portal_id).toBe(1);
      if (routePath === "/events")
        await expect(page.getByTestId("event-row")).toContainText("Portal opened");
      else
        await expect(page.getByRole("heading", { name: "Diagnostics" })).toBeVisible();
      expect((await Promise.all(resourceCookies)).every((header) => header?.includes(`omenpath_session=${cookies[0].value}`))).toBe(true);
      expect((await context.cookies()).find((cookie) => cookie.name === cookies[0].name)?.value).toBe(cookies[0].value);
    } finally {
      release();
    }
  });
}

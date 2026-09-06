import { expect, test, type Page } from "@playwright/test";
import type { StateSnapshot } from "../src/api/types";
import { resetLaboratory } from "./helpers/laboratory";

const state = async (page: Page): Promise<StateSnapshot> => {
  const response = await page.request.get("/api/state");
  expect(response.ok()).toBe(true);
  return response.json();
};
const begin = async (page: Page) => {
  await page.getByRole("button", { name: "Begin Practice" }).click();
  await expect(page.getByLabel("Tutorial")).toContainText("Step 1");
};
const captureSnapshots = (page: Page) => {
  const frames: StateSnapshot[] = [];
  page.on("websocket", (socket) => {
    if (!new URL(socket.url()).pathname.endsWith("/ws/lab")) return;
    socket.on("framereceived", ({ payload }) =>
      frames.push(JSON.parse(payload.toString()) as StateSnapshot),
    );
  });
  return frames;
};

test("reset helper resets the already-playing browser laboratory", async ({
  page,
}) => {
  await page.goto("/");
  await begin(page);
  expect((await state(page)).app.tutorial_step).toBe(1);
  await resetLaboratory(page);
  expect((await state(page)).app.tutorial_step).toBe(0);
  await expect(page.getByLabel("Tutorial")).toContainText("Step 0");
});

test("browser laboratories isolate matching IDs, actions, events and WebSocket; reload resumes and cleared cookie starts anew", async ({
  page,
  context,
  browser,
  baseURL,
}) => {
  const other = await browser.newContext({
    baseURL,
    viewport: page.viewportSize(),
  });
  try {
    const second = await other.newPage();
    const framesA = captureSnapshots(page);
    const framesB = captureSnapshots(second);
    await Promise.all([page.goto("/"), second.goto("/")]);
    await expect.poll(() => framesA.length).toBeGreaterThan(0);
    await expect.poll(() => framesB.length).toBeGreaterThan(0);
    const cookieA = (await context.cookies()).find(
      (cookie) => cookie.name === "omenpath_session",
    )!;
    const cookieB = (await other.cookies()).find(
      (cookie) => cookie.name === "omenpath_session",
    )!;
    expect(cookieA.value).toMatch(/^[A-Za-z0-9_-]{43}$/);
    expect(cookieB.value).not.toBe(cookieA.value);
    expect(cookieA.httpOnly).toBe(true);
    expect(cookieA.sameSite).toBe("Lax");
    expect(await page.evaluate(() => document.cookie)).not.toContain(
      "omenpath_session",
    );
    await Promise.all([begin(page), begin(second)]);
    expect((await state(page)).app.tutorial_portal_id).toBe(1);
    expect((await state(second)).app.tutorial_portal_id).toBe(1);
    const beforeB = framesB.length;
    const closed = await page.request.post("/api/portals/1/close", {
      data: { confirm: true },
    });
    expect(closed.ok()).toBe(true);
    const commandState: StateSnapshot = await closed.json();
    const commandCutoff = Date.parse(commandState.generated_at);
    expect(Number.isFinite(commandCutoff)).toBe(true);
    await expect.poll(() => framesA.at(-1)?.app.tutorial_portal_id).toBe(2);
    // A queued pre-command frame cannot satisfy the post-action tick evidence.
    await expect
      .poll(() =>
        framesB
          .slice(beforeB)
          .some(
            (snapshot) => Date.parse(snapshot.generated_at) > commandCutoff,
          ),
      )
      .toBe(true);
    const detailA = await (await page.request.get("/api/portals/1")).json();
    const detailB = await (await second.request.get("/api/portals/1")).json();
    expect(detailA.portal.status).toBe("CLOSED");
    expect(detailB.portal.status).toBe("OPEN");
    expect((await second.request.get("/api/portals/2")).status()).toBe(404);
    const eventsA = await (await page.request.get("/api/events")).text();
    const eventsB = await (await second.request.get("/api/events")).text();
    expect(eventsA).toContain("PORTAL_CLOSED");
    expect(eventsB).not.toContain("PORTAL_CLOSED");
    const visiblePayload = JSON.stringify([
      await state(page),
      detailA,
      eventsA,
      framesA,
    ]);
    expect(visiblePayload).not.toMatch(/"(?:lab_id|token|token_hash)"/);
    expect(visiblePayload).not.toContain(cookieA.value);
    expect(await page.locator("body").innerText()).not.toContain(cookieA.value);
    await page.reload();
    expect(
      (await context.cookies()).find((cookie) => cookie.name === cookieA.name)
        ?.value,
    ).toBe(cookieA.value);
    expect((await state(page)).app.tutorial_portal_id).toBe(2);
    expect(await (await page.request.get("/api/events")).text()).toBe(eventsA);
    // Navigate away first so the old live socket is actually closed.
    await page.goto("about:blank");
    await context.clearCookies();
    await page.goto("/");
    await expect(
      page.getByRole("button", { name: "Begin Practice" }),
    ).toBeVisible();
    const freshCookie = (await context.cookies()).find(
      (cookie) => cookie.name === cookieA.name,
    )!;
    expect(freshCookie.value).not.toBe(cookieA.value);
    expect((await state(page)).app.tutorial_step).toBe(0);
    expect(await (await page.request.get("/api/events")).json()).toEqual([]);
    expect((await page.request.get("/api/portals/1")).status()).toBe(404);
    expect((await state(second)).app.tutorial_portal_id).toBe(1);
    expect(
      framesB
        .slice(beforeB)
        .every(
          (snapshot) =>
            snapshot.app.tutorial_portal_id === 1 &&
            snapshot.slots[0].portal?.id === 1 &&
            snapshot.slots[0].portal.status === "OPEN" &&
            snapshot.portals.closed === 0 &&
            snapshot.slots.every((slot) => slot.portal?.id !== 2),
        ),
    ).toBe(true);
  } finally {
    await other.close();
  }
});

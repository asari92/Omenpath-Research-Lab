import { expect, test } from "@playwright/test";
import { snapshotAt } from "../src/test/builders";

test.beforeEach(async ({ page }) => {
  const request = page.request;
  await request.post("http://127.0.0.1:18080/api/tutorial/reset", { data: {} });
  await request.post("http://127.0.0.1:18080/api/tutorial/signal", {
    data: { signal: "EVENT_LOG_OPENED" },
  });
});

test("global Event Log renders safe shared rows and wrong-step navigation does not advance", async ({
  page,
}) => {
  const request = page.request;
  await page.goto("/");
  await page.getByRole("link", { name: "Event Log" }).click();
  await expect(page.getByRole("heading", { name: "Event Log" })).toBeVisible();
  await expect(page.getByTestId("event-row")).toHaveCount(1);
  await expect(
    page.getByTestId("event-row").getByText("Action rejected", { exact: true }),
  ).toBeVisible();
  const state = await (
    await request.get("http://127.0.0.1:18080/api/state")
  ).json();
  expect(state.app.tutorial_step).toBe(0);
});

test("Event timestamps stack above titles and only results scroll", async ({ page }) => {
  const snapshot = snapshotAt(); snapshot.app.mode = "LIVE";
  await page.route("**/api/state", (route) => route.fulfill({ json: snapshot }));
  await page.routeWebSocket("**/ws/lab", () => {});
  await page.route("**/api/events", (route) => route.fulfill({ json: Array.from({ length: 50 }, (_, id) => ({ id, event_type: "PORTAL_OPENED", created_at: "2026-09-06T15:04:05Z", message: "A way opens", payload_json: null, portal_id: 1, observer_id: null, plane_id: 1 })) }));
  await page.goto("/events");
  await expect(page.getByText("15:04:05-06-09-2026")).toHaveCount(50);
  const first = page.getByTestId("event-row").first();
  const time = await first.locator("time").boundingBox();
  const title = await first.locator("strong").boundingBox();
  expect(title!.y).toBeGreaterThanOrEqual(time!.y + time!.height);
  expect(await page.getByRole("region", { name: "Event results" }).evaluate((n) => n.scrollHeight > n.clientHeight)).toBe(true);
  expect(await page.getByRole("main").evaluate((n) => n.scrollHeight <= n.clientHeight)).toBe(true);
  expect(await page.locator("html").evaluate((n) => n.scrollHeight <= n.clientHeight)).toBe(true);
});

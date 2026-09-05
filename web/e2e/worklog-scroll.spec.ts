import { expect, test } from "@playwright/test";
import { snapshotAt } from "../src/test/builders";

test("long-form Worklog scrolls inside the viewport and Dashboard stays fixed", async ({
  page,
}) => {
  const snapshot = snapshotAt();
  snapshot.app.mode = "LIVE";
  await page.route("**/api/state", (route) =>
    route.fulfill({ json: snapshot }),
  );
  await page.routeWebSocket("**/ws/lab", () => {});
  await page.goto("/ai-worklog");
  await expect(
    page.getByRole("heading", { name: "AI Worklog — Current", exact: true }),
  ).toBeVisible();
  const main = page.getByRole("main");
  await page.locator("article").hover({ position: { x: 40, y: 40 } });
  await page.mouse.wheel(0, 2000);
  await expect
    .poll(() => main.evaluate((node) => node.scrollTop))
    .toBeGreaterThan(100);
  const scroll = await main.evaluate((node) => {
    node.scrollTop = node.scrollHeight;
    return { top: node.scrollTop, overflow: getComputedStyle(node).overflowY };
  });
  expect(scroll.top).toBeGreaterThan(1000);
  expect(scroll.overflow).toBe("auto");
  await expect(page.locator("article > :last-child")).toBeInViewport();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollHeight <= innerHeight,
    ),
  ).toBe(true);
  await page.getByRole("link", { name: "Dashboard", exact: true }).click();
  await expect(page.getByTestId("portal-slot")).toHaveCount(7);
  expect(
    await main.evaluate((node) => node.scrollHeight <= node.clientHeight),
  ).toBe(true);
  expect(
    await page
      .getByTestId("portal-board")
      .evaluate((node) => node.scrollHeight <= node.clientHeight),
  ).toBe(true);
  expect(
    await page.evaluate(
      () => document.documentElement.scrollHeight <= innerHeight,
    ),
  ).toBe(true);
});

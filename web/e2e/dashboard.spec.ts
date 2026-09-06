import { expect, test } from "@playwright/test";
import { portalDetails, snapshotAt } from "../src/test/builders";

test.beforeEach(async ({ page }) => {
  const snapshot = snapshotAt();
  snapshot.app.mode = "LIVE";
  snapshot.portals.active = 1;
  snapshot.slots[0].portal = {
    ...portalDetails(1).portal,
    destination_plane_id: 1,
    destination_plane_name: "Agyrem",
    destination_explored: false,
  };
  await page.route("**/api/state", (route) =>
    route.fulfill({ json: snapshot }),
  );
  await page.routeWebSocket("**/ws/lab", () => {});
});

test("all slots have equal visible geometry and commands", async ({
  page,
}, info) => {
  await page.goto("/");
  await expect(page.getByTestId("portal-slot")).toHaveCount(7);
  const boxes = await page.getByTestId("portal-slot").evaluateAll((nodes) =>
    nodes.map((n) => {
      const r = n.getBoundingClientRect();
      return { x: r.x, y: r.y, width: r.width, height: r.height };
    }),
  );
  for (const key of ["width", "height"] as const)
    expect(
      Math.max(...boxes.map((b) => b[key])) -
        Math.min(...boxes.map((b) => b[key])),
    ).toBeLessThanOrEqual(1);
  const rows = [...new Set(boxes.map((b) => Math.round(b.y)))];
  expect(
    rows.map((y) => boxes.filter((b) => Math.round(b.y) === y).length),
  ).toEqual(info.project.name === "desktop" ? [4, 3] : [2, 2, 2, 1]);
  const board = await page.getByTestId("portal-board").boundingBox();
  const lastRow = boxes.filter((b) => Math.round(b.y) === rows.at(-1));
  expect(
    Math.abs(
      (lastRow[0].x + lastRow.at(-1)!.x + lastRow.at(-1)!.width) / 2 -
        (board!.x + board!.width / 2),
    ),
  ).toBeLessThanOrEqual(1);
  const buttons = page.getByTestId("portal-board").getByRole("button");
  await expect(buttons).toHaveCount(28);
  for (const button of await buttons.all())
    await expect(button).toBeInViewport({ ratio: 1 });
  expect(
    await page.evaluate(
      () =>
        document.documentElement.scrollHeight <= innerHeight &&
        document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  expect(
    await page
      .getByTestId("portal-board")
      .evaluate(
        (n) =>
          n.scrollHeight <= n.clientHeight && n.scrollWidth <= n.clientWidth,
      ),
  ).toBe(true);
  await page.screenshot({ path: info.outputPath("dashboard.png") });
});

test("desktop keeps seven stable slots in a 4 + 3 board", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name !== "desktop");
  await page.goto("/");
  await expect(page.getByTestId("portal-slot")).toHaveCount(7);
  const firstRow = await page
    .getByTestId("portal-slot")
    .evaluateAll((slots) =>
      slots
        .slice(0, 4)
        .map((slot) => Math.round(slot.getBoundingClientRect().top)),
    );
  expect(new Set(firstRow).size).toBe(1);
});

test("phone keeps all seven slots and four commands per slot without board scroll", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name !== "phone");
  await page.goto("/");
  await expect(page.getByTestId("portal-slot")).toHaveCount(7);
  await expect(page.getByRole("button", { name: "Close" })).toHaveCount(7);
  const boardFits = await page
    .getByTestId("portal-board")
    .evaluate((node) => node.scrollHeight <= node.clientHeight);
  expect(boardFits).toBe(true);
  const pageFits = await page.evaluate(
    () => document.documentElement.scrollHeight <= innerHeight,
  );
  expect(pageFits).toBe(true);
});

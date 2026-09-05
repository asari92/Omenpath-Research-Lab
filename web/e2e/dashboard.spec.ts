import { expect, test } from "@playwright/test";

test.beforeEach(async ({ request }) => {
  await request.post("http://127.0.0.1:18080/api/tutorial/reset", { data: {} });
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

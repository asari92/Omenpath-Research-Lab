import { expect, test } from "@playwright/test";

test.beforeEach(async ({ request }) => {
  await request.post("http://127.0.0.1:18080/api/tutorial/reset", { data: {} });
});

test("desktop shell keeps a stable sidebar and supports browser history", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name !== "desktop");
  const consoleErrors: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") consoleErrors.push(message.text());
  });
  await page.goto("/");
  const navigation = page.getByRole("navigation", {
    name: "Primary navigation",
  });
  const main = page.getByRole("main");
  const [navBox, mainBox] = await Promise.all([
    navigation.boundingBox(),
    main.boundingBox(),
  ]);
  expect(navBox).not.toBeNull();
  expect(mainBox).not.toBeNull();
  expect(navBox!.x + navBox!.width).toBeLessThanOrEqual(mainBox!.x);

  await page.getByRole("link", { name: "Event Log" }).click();
  await page.getByRole("link", { name: "AI Worklog" }).click();
  await expect(
    page.getByRole("heading", { name: "AI Worklog — Current" }),
  ).toBeVisible();
  await page.goBack();
  await expect(page.getByRole("heading", { name: "Event Log" })).toBeVisible();
  await page.goForward();
  await expect(
    page.getByRole("heading", { name: "AI Worklog — Current" }),
  ).toBeVisible();
  expect(consoleErrors).toEqual([]);
});

test("phone keeps primary navigation below every Slot control", async ({
  page,
}, testInfo) => {
  test.skip(testInfo.project.name !== "phone");
  await page.goto("/");
  await expect(page.getByTestId("portal-slot")).toHaveCount(7);
  const navigation = page.getByRole("navigation", {
    name: "Primary navigation",
  });
  const navBox = await navigation.boundingBox();
  const lastControlBox = await page
    .getByTestId("portal-slot")
    .last()
    .getByRole("button", { name: "Recall Observer" })
    .boundingBox();
  expect(navBox).not.toBeNull();
  expect(lastControlBox).not.toBeNull();
  expect(navBox!.y).toBeGreaterThanOrEqual(
    lastControlBox!.y + lastControlBox!.height,
  );
  expect(navBox!.y + navBox!.height).toBeLessThanOrEqual(760);
});

test("artwork routes stay local and credits expose artist, source and policy", async ({
  page,
}) => {
  const remoteImages: string[] = [];
  page.on("request", (request) => {
    const url = new URL(request.url());
    if (request.resourceType() === "image" && url.hostname !== "127.0.0.1") {
      remoteImages.push(request.url());
    }
  });
  await page.goto("/");
  await page.getByRole("button", { name: "Artwork Credits" }).click();
  const dialog = page.getByRole("dialog", { name: "Plane Artwork Credits" });
  await expect(
    dialog.getByRole("link", { name: "Source" }).first(),
  ).toBeVisible();
  await expect(
    dialog.getByRole("link", { name: "Policy" }).first(),
  ).toBeVisible();
  await expect(dialog.locator("li").first()).toContainText(/—/);
  expect(remoteImages).toEqual([]);
});

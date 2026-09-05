import { expect, test } from "@playwright/test";

const apiState = async (
  request: import("@playwright/test").APIRequestContext,
) => (await request.get("http://127.0.0.1:18080/api/state")).json();

test("the visible Tutorial journey reaches Live without client-side progress", async ({
  page,
  request,
}, testInfo) => {
  test.skip(testInfo.project.name !== "desktop");
  test.setTimeout(240_000);
  await request.post("http://127.0.0.1:18080/api/tutorial/reset", { data: {} });
  await page.goto("/");

  await expect(page.getByLabel("Tutorial")).toContainText("Step 0");
  await page.getByRole("button", { name: "Begin Practice" }).click();
  await expect(page.getByLabel("Tutorial")).toContainText("Step 1");
  await page
    .locator('[data-tutorial-target="true"]')
    .getByRole("link", { name: "Details" })
    .click();
  await expect(page.getByLabel("Tutorial")).toContainText("Step 2");
  await page.getByRole("link", { name: "Dashboard" }).click();

  await expect(page.getByLabel("Tutorial")).toContainText("Step 3", {
    timeout: 45_000,
  });
  await page.locator('[data-tutorial-command="true"]').click();
  await expect(page.getByLabel("Tutorial")).toContainText("Step 4");
  await page.locator('[data-tutorial-command="true"]').click();
  await expect(page.getByLabel("Tutorial")).toContainText("Step 5");
  await page.getByRole("button", { name: "Try Send" }).click();

  await expect
    .poll(async () => (await apiState(request)).app.tutorial_step)
    .toBe(6);
  const firstPhase = (await apiState(request)).app.expected_action;
  if (firstPhase === "SEND_OBSERVER") {
    await page.locator('[data-tutorial-command="true"]').click();
  }
  await expect
    .poll(async () => (await apiState(request)).app.expected_action, {
      timeout: 90_000,
    })
    .toBe("RECALL_OBSERVER");
  await page.locator('[data-tutorial-command="true"]').click();
  await expect
    .poll(async () => (await apiState(request)).app.expected_action, {
      timeout: 45_000,
    })
    .toBe("OPEN_EVENT_LOG");

  await page.getByRole("link", { name: "Event Log" }).click();
  await expect(page.getByLabel("Tutorial")).toContainText("Step 9");
  const beforeLive = await apiState(request);
  const eventsBefore = await (
    await request.get("http://127.0.0.1:18080/api/events")
  ).json();
  await page.getByRole("button", { name: "Start Live" }).click();
  await expect(page.getByLabel("Tutorial")).toHaveCount(0);

  const live = await apiState(request);
  const eventsAfter = await (
    await request.get("http://127.0.0.1:18080/api/events")
  ).json();
  expect(live.app.mode).toBe("LIVE");
  expect(live.portals.active).toBe(0);
  expect(live.lab.current_energy).toBeGreaterThanOrEqual(
    beforeLive.lab.current_energy,
  );
  expect(live.exploration).toEqual(beforeLive.exploration);
  expect(live.observers).toEqual(beforeLive.observers);
  expect(live.planes).toEqual(beforeLive.planes);
  expect(eventsAfter.length).toBeGreaterThanOrEqual(eventsBefore.length);
});

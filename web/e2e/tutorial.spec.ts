import { expect, test } from "@playwright/test";
import { portalDetails, snapshotAt } from "../src/test/builders";

test("presentation replays missed cards for 7s and terminal target for 2s", async ({
  page,
}) => {
  const clockStart = new Date("2026-09-06T10:00:00Z");
  await page.clock.install({ time: clockStart });
  await page.clock.pauseAt(clockStart);
  const initial = snapshotAt();
  initial.app.tutorial_step = 1;
  initial.app.tutorial_portal_id = 1;
  initial.portals.active = 1;
  const portal = (id: number) => ({
    ...portalDetails(id).portal,
    destination_plane_id: 1,
    destination_plane_name: "Agyrem",
    destination_explored: false,
  });
  initial.slots[0].portal = portal(1);
  await page.route("**/api/state", (route) => route.fulfill({ json: initial }));
  let socket: import("@playwright/test").WebSocketRoute;
  await page.routeWebSocket("**/ws/lab", (ws) => {
    socket = ws;
  });
  await page.goto("/");
  await expect(page.getByLabel("Tutorial")).toContainText("Step 1");
  await expect(page.getByText("More context")).toHaveCount(0);
  const next = {
    ...initial,
    generated_at: "2026-09-05T10:00:01Z",
    app: { ...initial.app, tutorial_step: 3 },
  };
  socket!.send(JSON.stringify(next));
  await expect(page.getByLabel("Tutorial")).toContainText("Step 2");
  await expect(page.getByLabel("Tutorial")).toContainText("Completed");
  await page.clock.runFor(6999);
  await expect(page.getByLabel("Tutorial")).toContainText("Step 2");
  await page.clock.runFor(1);
  await expect(page.getByLabel("Tutorial")).toContainText("Step 3");
  await page.getByRole("button", { name: "Back", exact: true }).click();
  await expect(page.getByLabel("Tutorial")).toContainText("Step 2");
  await page.getByRole("button", { name: "Forward", exact: true }).click();
  await expect(page.getByLabel("Tutorial")).toContainText("Step 3");
  await expect(
    page.getByRole("button", { name: "Forward", exact: true }),
  ).toBeDisabled();
  const replaced = {
    ...next,
    generated_at: "2026-09-05T10:00:02Z",
    app: { ...next.app, tutorial_portal_id: 2 },
    slots: next.slots.map((s) =>
      s.slot_index === 1 ? { ...s, portal: portal(2) } : s,
    ),
  };
  socket!.send(JSON.stringify(replaced));
  await expect(page.getByTestId("portal-ghost")).toHaveCount(1);
  await page.clock.runFor(1999);
  await expect(page.getByTestId("portal-ghost")).toHaveCount(1);
  await page.clock.runFor(1);
  await expect(page.getByTestId("portal-ghost")).toHaveCount(0);
  const target = page.locator('[data-tutorial-target="true"]');
  await expect(
    target.getByRole("button", { name: "Send Observer" }),
  ).toBeVisible();
  await target
    .getByRole("button", { name: "Send Observer" })
    .click({ trial: true });
  const panelBox = await page.getByLabel("Tutorial").boundingBox();
  const commandsBox = await target
    .getByRole("group", { name: "Portal commands" })
    .boundingBox();
  expect(
    panelBox!.y + panelBox!.height <= commandsBox!.y ||
      panelBox!.y >= commandsBox!.y + commandsBox!.height,
  ).toBe(true);
  expect(
    await page.evaluate(
      () => document.documentElement.scrollHeight <= innerHeight,
    ),
  ).toBe(true);
  await page.emulateMedia({ reducedMotion: "reduce" });
  await expect(target.locator('[data-motion="entering"]')).toHaveCSS(
    "animation-duration",
    "0.12s",
  );
});

const apiState = async (
  request: import("@playwright/test").APIRequestContext,
) => (await request.get("http://127.0.0.1:18080/api/state")).json();

test("the visible Tutorial journey reaches Live without client-side progress", async ({
  page,
}, testInfo) => {
  const request = page.request;
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

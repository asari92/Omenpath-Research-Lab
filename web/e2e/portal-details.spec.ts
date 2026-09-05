import { expect, test } from "@playwright/test";
import { portalDetails, snapshotAt } from "../src/test/builders";

async function prepareStepOne(
  request: import("@playwright/test").APIRequestContext,
) {
  await request.post("http://127.0.0.1:18080/api/tutorial/reset", { data: {} });
  const response = await request.post(
    "http://127.0.0.1:18080/api/tutorial/signal",
    {
      data: { signal: "TUTORIAL_INTRO_COMPLETED" },
    },
  );
  return response.json();
}

test("matching Details click emits one explicit Tutorial signal", async ({
  page,
}) => {
  const request = page.request;
  const snapshot = await prepareStepOne(request);
  const portalID = snapshot.app.tutorial_portal_id;
  await page.goto("/");
  await page
    .getByTestId("portal-slot")
    .filter({ hasText: `#${String(portalID).padStart(4, "0")}` })
    .getByRole("link", { name: "Details" })
    .click();
  await expect(page).toHaveURL(`/portals/${portalID}`);
  await expect(
    page.getByRole("heading", { name: "Diagnostics" }),
  ).toBeVisible();
  await expect
    .poll(
      async () =>
        (await (await request.get("http://127.0.0.1:18080/api/state")).json())
          .app.tutorial_step,
    )
    .toBe(2);
});

test("direct Details load performs GET without advancing Tutorial", async ({
  page,
}) => {
  const request = page.request;
  const snapshot = await prepareStepOne(request);
  const portalID = snapshot.app.tutorial_portal_id;
  await page.goto(`/portals/${portalID}`);
  await expect(
    page.getByRole("heading", { name: "Diagnostics" }),
  ).toBeVisible();
  const state = await (
    await request.get("http://127.0.0.1:18080/api/state")
  ).json();
  expect(state.app.expected_action).toBe("OPEN_PORTAL_DETAILS");
});

test("Details fits the viewport and only expanded History scrolls", async ({
  page,
}) => {
  const snapshot = snapshotAt();
  snapshot.app.mode = "LIVE";
  const details = portalDetails();
  details.history = Array.from({ length: 40 }, (_, id) => ({
    id,
    event_type: "PORTAL_OPENED" as const,
    portal_id: 42,
    observer_id: null,
    plane_id: 1,
    message: `Connection record ${id}`,
    created_at: "2026-09-06T15:04:05Z",
    payload_json: null,
  }));
  await page.route("**/api/state", (route) =>
    route.fulfill({ json: snapshot }),
  );
  await page.route("**/api/portals/42", (route) =>
    route.fulfill({ json: details }),
  );
  await page.routeWebSocket("**/ws/lab", () => {});
  await page.goto("/portals/42");
  await page.getByText("History (40)").click();
  const history = page.getByRole("region", { name: "Portal history" });
  expect(await history.evaluate((n) => n.scrollHeight > n.clientHeight)).toBe(
    true,
  );
  for (const region of [page.locator("html"), page.getByRole("main")]) {
    expect(
      await region.evaluate(
        (n) =>
          n.scrollHeight <= n.clientHeight && n.scrollWidth <= n.clientWidth,
      ),
    ).toBe(true);
  }
  for (const action of await page
    .getByRole("button", {
      name: /stabilize|close|send observer|recall observer/i,
    })
    .all())
    await expect(action).toBeInViewport({ ratio: 1 });
  for (const fact of await page.locator("main dt, main dd").all())
    await expect(fact).toBeInViewport({ ratio: 1 });
  const portal = await page.getByRole("img", { name: "Agyrem" }).boundingBox();
  const actions = await page
    .getByRole("region", { name: "Portal actions" })
    .boundingBox();
  expect(actions!.y).toBeGreaterThan(portal!.y + portal!.height);
  await page.getByText("How Risk Works").click();
  await expect(page.getByTestId("risk-explanation")).toBeInViewport({
    ratio: 1,
  });
});

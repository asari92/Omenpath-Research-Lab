import { expect, test } from "@playwright/test";

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
  request,
}) => {
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
  request,
}) => {
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

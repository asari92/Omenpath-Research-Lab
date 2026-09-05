import { expect, test } from "@playwright/test";

test.beforeEach(async ({ request }) => {
  await request.post("http://127.0.0.1:18080/api/tutorial/reset", { data: {} });
  await request.post("http://127.0.0.1:18080/api/tutorial/signal", {
    data: { signal: "EVENT_LOG_OPENED" },
  });
});

test("global Event Log renders safe shared rows and wrong-step navigation does not advance", async ({
  page,
  request,
}) => {
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

import { expect, test } from "@playwright/test";

test.beforeEach(async ({ request }) => {
  await request.post("http://127.0.0.1:18080/api/tutorial/reset", { data: {} });
});

test("Extraction chooser exposes all local Plane choices and preserves rejected selection", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Open Extraction" }).click();
  await expect(
    page.getByRole("dialog", { name: "Open Extraction Portal" }),
  ).toBeVisible();
  await expect(page.getByTestId("plane-card")).toHaveCount(85);
  await page.getByRole("button", { name: "Agyrem" }).click();
  await page.getByRole("button", { name: "Open Extraction" }).last().click();
  await expect(page.getByRole("alert")).toBeVisible();
  await expect(page.getByRole("button", { name: "Agyrem" })).toHaveAttribute(
    "aria-pressed",
    "true",
  );
});

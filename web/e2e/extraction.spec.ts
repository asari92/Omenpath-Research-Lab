import { expect, test } from "@playwright/test";
import { resetLaboratory } from "./helpers/laboratory";

test.beforeEach(async ({ page }) => {
  await resetLaboratory(page);
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
  const selected = page.getByTestId("plane-card").first();
  await selected.click();
  await page.getByRole("button", { name: "Open Extraction" }).last().click();
  await expect(page.getByRole("alert")).toBeVisible();
  await expect(selected).toHaveAttribute("aria-pressed", "true");
});

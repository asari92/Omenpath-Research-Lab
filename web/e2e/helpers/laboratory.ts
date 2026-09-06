import { expect, type APIRequestContext, type Page } from "@playwright/test";

// Preserve the existing reset path until the browser-cookie regression is green.
export async function resetLaboratory(page: Page, request: APIRequestContext) {
  await page.goto("/");
  await expect(page.getByLabel("Tutorial")).toBeVisible();
  const response = await request.post("/api/tutorial/reset", { data: {} });
  expect(response.ok()).toBe(true);
  await page.reload();
}

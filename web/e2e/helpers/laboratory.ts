import { expect, type Page } from "@playwright/test";

// Bootstrap sets the HttpOnly session before using this browser's cookie jar.
export async function resetLaboratory(page: Page) {
  const [bootstrap] = await Promise.all([
    page.waitForResponse(
      (response) => new URL(response.url()).pathname === "/api/state",
    ),
    page.goto("/"),
  ]);
  expect(bootstrap.ok()).toBe(true);
  await bootstrap.finished();
  const response = await page.request.post("/api/tutorial/reset", { data: {} });
  expect(response.ok()).toBe(true);
  await page.reload();
}

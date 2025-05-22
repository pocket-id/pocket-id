// frontend/tests/api-key.spec.ts
import { expect, test } from "@playwright/test";
import { apiKeys } from "../data";
import { cleanupBackend } from "../utils/cleanup.util";

test.describe("API Key Management", () => {
  test.beforeEach(async ({ page }) => {
    await cleanupBackend();
    await page.goto("/settings/admin/api-keys");
  });

  test("Create new API key", async ({ page }) => {
    await page.getByRole("button", { name: "Add API Key" }).click();

    // Fill out the API key form
    const name = "New Test API Key";
    await page.getByLabel("Name").fill(name);
    await page.getByLabel("Description").fill("Created by automated test");

    // Choose the date
    const currentDate = new Date();
    await page.getByLabel("Expires At").click();
    await page.getByLabel("Select year").click();
    // Select the next year
    await page.getByText((currentDate.getFullYear() + 1).toString()).click();
    // Select the first day of the month
    await page
      .getByRole("button", { name: /([A-Z][a-z]+), ([A-Z][a-z]+) 1, (\d{4})/ })
      .first()
      .click();

    // Submit the form
    await page.getByRole("button", { name: "Save" }).click();

    // Verify the success dialog appears
    await expect(
      page.getByRole("heading", { name: "API Key Created" })
    ).toBeVisible();

    // Verify the key details are shown
    await expect(page.getByRole("cell", { name })).toBeVisible();

    // Verify the token is displayed (should be 32 characters)
    const token = await page.locator(".font-mono").textContent();
    expect(token?.length).toBe(32);

    // Close the dialog
    await page.getByRole("button", { name: "Close" }).click();

    await page.reload();

    // Verify the key appears in the list
    await expect(page.getByRole("cell", { name }).first()).toContainText(name);
  });

  test("Revoke API key", async ({ page }) => {
    const apiKey = apiKeys[0];

    await page
      .getByRole("row", { name: apiKey.name })
      .getByRole("button", { name: "Revoke" })
      .click();

    await page.getByText("Revoke", { exact: true }).click();

    // Verify success message
    await expect(page.locator('[data-type="success"]')).toHaveText(
      "API key revoked successfully"
    );

    // Verify key is no longer in the list
    await expect(
      page.getByRole("cell", { name: apiKey.name })
    ).not.toBeVisible();
  });
});

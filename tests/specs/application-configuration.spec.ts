import test, { expect } from "@playwright/test";
import { cleanupBackend } from "../utils/cleanup.util";

test.beforeEach(cleanupBackend);

test("Update general configuration", async ({ page }) => {
  await page.goto("/settings/admin/application-configuration");

  await page
    .getByLabel("Application Name", { exact: true })
    .fill("Updated Name");
  await page.getByLabel("Session Duration").fill("30");
  await page.getByRole("button", { name: "Save" }).first().click();

  await expect(page.locator('[data-type="success"]')).toHaveText(
    "Application configuration updated successfully"
  );
  await expect(page.getByTestId("application-name")).toHaveText("Updated Name");

  await page.reload();

  await expect(
    page.getByLabel("Application Name", { exact: true })
  ).toHaveValue("Updated Name");
  await expect(page.getByLabel("Session Duration")).toHaveValue("30");
});

test("Update email configuration", async ({ page }) => {
  await page.goto("/settings/admin/application-configuration");

  await page.getByRole("button", { name: "Expand card" }).nth(1).click();

  await page.getByLabel("SMTP Host").fill("smtp.gmail.com");
  await page.getByLabel("SMTP Port").fill("587");
  await page.getByLabel("SMTP User").fill("test@gmail.com");
  await page.getByLabel("SMTP Password").fill("password");
  await page.getByLabel("SMTP From").fill("test@gmail.com");
  await page.getByLabel("Email Login Notification").click();
  await page.getByLabel("Email Login Code Requested by User").click();
  await page.getByLabel("Email Login Code from Admin").click();
  await page.getByLabel("API Key Expiration").click();

  await page.getByRole("button", { name: "Save" }).nth(1).click();

  await expect(page.locator('[data-type="success"]')).toHaveText(
    "Email configuration updated successfully"
  );

  await page.reload();

  await expect(page.getByLabel("SMTP Host")).toHaveValue("smtp.gmail.com");
  await expect(page.getByLabel("SMTP Port")).toHaveValue("587");
  await expect(page.getByLabel("SMTP User")).toHaveValue("test@gmail.com");
  await expect(page.getByLabel("SMTP Password")).toHaveValue("password");
  await expect(page.getByLabel("SMTP From")).toHaveValue("test@gmail.com");
  await expect(page.getByLabel("Email Login Notification")).toBeChecked();
  await expect(
    page.getByLabel("Email Login Code Requested by User")
  ).toBeChecked();
  await expect(page.getByLabel("Email Login Code from Admin")).toBeChecked();
  await expect(page.getByLabel("API Key Expiration")).toBeChecked();
});

test("Update application images", async ({ page }) => {
  await page.goto("/settings/admin/application-configuration");

  await page.getByRole("button", { name: "Expand card" }).nth(3).click();

  await page
    .getByLabel("Favicon")
    .setInputFiles("assets/w3-schools-favicon.ico");
  await page
    .getByLabel("Light Mode Logo")
    .setInputFiles("assets/pingvin-share-logo.png");
  await page
    .getByLabel("Dark Mode Logo")
    .setInputFiles("assets/nextcloud-logo.png");
  await page
    .getByLabel("Background Image")
    .setInputFiles("assets/clouds.jpg");
  await page.getByRole("button", { name: "Save" }).nth(1).click();

  await expect(page.locator('[data-type="success"]')).toHaveText(
    "Images updated successfully"
  );

  await page.request
    .get("/api/application-configuration/favicon")
    .then((res) => expect.soft(res.status()).toBe(200));
  await page.request
    .get("/api/application-configuration/logo?light=true")
    .then((res) => expect.soft(res.status()).toBe(200));
  await page.request
    .get("/api/application-configuration/logo?light=false")
    .then((res) => expect.soft(res.status()).toBe(200));
  await page.request
    .get("/api/application-configuration/background-image")
    .then((res) => expect.soft(res.status()).toBe(200));
});

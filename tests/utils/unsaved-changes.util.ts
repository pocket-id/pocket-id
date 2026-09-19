import { expect, type Page } from '@playwright/test';

export async function saveUnsavedChanges(page: Page) {
	await page.getByRole('button', { name: 'Save', exact: true }).click();
	await expect(page.getByText('Changes saved successfully', { exact: true })).toBeVisible();
}

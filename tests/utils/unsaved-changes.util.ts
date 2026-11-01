import { expect, type Page } from '@playwright/test';

export function unsavedChangesBar(page: Page) {
	return page.getByRole('region', { name: 'Unsaved changes' });
}

export async function saveUnsavedChanges(page: Page) {
	await page.getByRole('button', { name: 'Save', exact: true }).click();
	await expect(unsavedChangesBar(page).getByText('Changes saved', { exact: true })).toBeVisible();
}

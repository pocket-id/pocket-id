import { expect, type Page } from '@playwright/test';

export function unsavedChangesBar(page: Page) {
	return page.getByRole('region', { name: 'Unsaved changes' });
}

export async function saveUnsavedChanges(page: Page) {
	await page.getByRole('button', { name: 'Save', exact: true }).click();
	// The bar keeps the message of an earlier save in an inert layer while it shows pending changes, so only a settled layer counts
	await expect(
		unsavedChangesBar(page)
			.locator(':scope > span:not([inert])')
			.getByText('Changes saved', { exact: true })
	).toBeVisible();
}

import test, { expect } from '@playwright/test';
import { userGroups } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';

test.beforeEach(async () => await cleanupBackend());

test('Filter users by group from a later page and clear the filter', async ({ page }) => {
	for (let index = 0; index < 21; index++) {
		const created = await page.request.post('/api/users', {
			data: {
				username: `member-${index}`,
				firstName: `Member ${index}`,
				lastName: 'Test',
				displayName: `Member ${index} Test`,
				email: `member-${index}@example.invalid`,
				isAdmin: false,
				disabled: false,
				emailVerified: false,
				userGroupIds: [userGroups.developers.id]
			}
		});
		expect(created.ok()).toBeTruthy();
	}
	await page.goto('/settings/admin/users?user-list-page=2');
	await expect(page.getByRole('table').getByRole('row')).toHaveCount(5);
	const filter = page.getByLabel('Filter by user groups');
	await filter.click();
	await page.getByRole('option', { name: userGroups.designers.name, exact: true }).click();
	await filter.click();
	await expect(page).toHaveURL(/user-list-page=1/);
	await expect(page.getByRole('table').getByRole('row')).toHaveCount(2);
	await expect(page.getByRole('row', { name: 'Tim Cook' })).toBeVisible();
	await expect(page.getByRole('row', { name: 'Craig Federighi' })).not.toBeVisible();
	await filter.click();
	await page.getByRole('option', { name: userGroups.designers.name, exact: true }).click();
	await filter.click();
	await expect(page.getByRole('table').getByRole('row')).toHaveCount(21);
});

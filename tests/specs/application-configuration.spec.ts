import { expect, test, type Page } from '@playwright/test';
import { cleanupBackend } from '../utils/cleanup.util';

test.beforeEach(async ({ page }) => {
	await cleanupBackend();
	await page.goto('/settings/admin/application-configuration');
});

test('Update general configuration', async ({ page }) => {
	await page.getByLabel('Application Name', { exact: true }).fill('Updated Name');
	await page.getByLabel('Session Duration').fill('30');

	await page.getByRole('button', { name: 'Home Page' }).click();
	await page.getByRole('option', { name: 'My Apps' }).click();

	await page.getByRole('button', { name: 'Save' }).first().click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'Application configuration updated successfully'
	);

	await page.reload();

	await expect(page.getByLabel('Application Name', { exact: true })).toHaveValue('Updated Name');
	await expect(page.getByLabel('Session Duration')).toHaveValue('30');

	await page.getByRole('link', { name: 'Logo' }).click();
	await page.waitForURL('/settings/apps');
});

test.describe('Update user creation configuration', () => {
	test.beforeEach(async ({ page }) => {
		await page.getByRole('tab', { name: 'Users and groups' }).click();
		await page.getByRole('button', { name: 'Expand card' }).first().click();
	});

	test('should save sign up mode', async ({ page }) => {
		await page.getByRole('button', { name: 'Enable User Signups' }).click();
		await page.getByRole('option', { name: 'Open Signup' }).click();

		await page.getByRole('button', { name: 'Save' }).last().click();

		await expect(page.locator('[data-type="success"]').last()).toHaveText(
			'User creation settings updated successfully.'
		);

		await page.reload();

		await expect(page.getByRole('button', { name: 'Enable User Signups' })).toBeVisible();
	});

	test('should save default user groups for new signups', async ({ page }) => {
		await page.getByRole('combobox', { name: 'User Groups' }).click();
		await page.getByRole('option', { name: 'Developers' }).click();
		await page.getByRole('option', { name: 'Designers' }).click();
		await page.getByRole('combobox', { name: 'User Groups' }).click();

		await page.getByRole('button', { name: 'Save' }).last().click();

		await expect(page.locator('[data-type="success"]').last()).toHaveText(
			'User creation settings updated successfully.'
		);

		await page.reload();

		await page.getByRole('combobox', { name: 'User Groups' }).click();

		await expect(page.getByRole('option', { name: 'Developers' })).toBeChecked();
		await expect(page.getByRole('option', { name: 'Designers' })).toBeChecked();
	});

	test('should create, edit, validate, and delete custom fields', async ({ page }) => {
		await openCustomFieldsCard(page);

		await addCustomField(page, {
			displayName: 'Employee code',
			key: 'employeeCode',
			defaultValue: 'WRONG',
			validationRegex: '^EMP-[0-9]+$',
			validationErrorMessage: 'Use EMP-###'
		});
		await expect(page.getByText('Use EMP-###')).toBeVisible();
		await page.getByLabel('Default value').fill('EMP-001');
		await page.getByRole('button', { name: 'Save' }).last().click();

		await expect(page.locator('[data-type="success"]').last()).toHaveText(
			'Custom fields updated successfully'
		);

		await addCustomField(page, {
			displayName: 'Weekly hours',
			key: 'weeklyHours',
			type: 'Number',
			target: 'Users and groups',
			required: true,
			defaultValue: '40'
		});

		await expect(page.locator('[data-type="success"]').last()).toHaveText(
			'Custom fields updated successfully'
		);
		await expect(page.getByRole('row', { name: /Employee code/ })).toBeVisible();
		await expect(page.getByRole('row', { name: /Weekly hours/ })).toBeVisible();

		await page.getByRole('row', { name: /Employee code/ }).click();
		await expect(page.getByLabel('Type')).toBeDisabled();
		await page.getByLabel('Display Name').fill('Employee code updated');
		await page.getByRole('button', { name: 'Save' }).last().click();

		await expect(page.locator('[data-type="success"]').last()).toHaveText(
			'Custom fields updated successfully'
		);

		const weeklyHoursRow = page.getByRole('row', { name: /Weekly hours/ });
		await weeklyHoursRow.getByRole('button').click();
		await page.getByRole('menuitem', { name: 'Delete' }).click();
		await page.getByRole('alertdialog').getByRole('button', { name: 'Delete' }).click();
		await expect(page.locator('[data-type="success"]').last()).toHaveText(
			'Custom fields updated successfully'
		);

		await page.reload();

		await page.getByRole('tab', { name: 'Users and groups' }).click();
		await openCustomFieldsCard(page, false);
		await expect(page.getByText('Employee code updated')).toBeVisible();
		await expect(page.getByText('Weekly hours')).not.toBeVisible();
	});
});

async function openCustomFieldsCard(page: Page, navigate = true) {
	if (navigate) {
		await page.goto('/settings/admin/application-configuration');
		await page.getByRole('tab', { name: 'Users and groups' }).click();
	}
	const addButton = page.getByRole('button', { name: 'Add custom field' });
	if (!(await addButton.isVisible().catch(() => false))) {
		await page.getByRole('button', { name: 'Expand card' }).nth(1).click();
	}
	await expect(addButton).toBeVisible();
}

async function addCustomField(
	page: Page,
	field: {
		displayName: string;
		key: string;
		type?: 'Text' | 'Number' | 'Boolean';
		target?: 'Users' | 'Groups' | 'Users and groups';
		required?: boolean;
		userEditable?: boolean;
		defaultValue?: string;
		validationRegex?: string;
		validationErrorMessage?: string;
	}
) {
	await page.getByRole('button', { name: 'Add custom field' }).click();
	await page.getByLabel('Display Name').fill(field.displayName);
	await page.getByLabel('Key').fill(field.key);

	if (field.type && field.type !== 'Text') {
		await page.getByLabel('Type').click();
		await page.getByRole('option', { name: field.type, exact: true }).click();
	}
	if (field.target && field.target !== 'Users') {
		await page.getByLabel('Available for').click();
		await page.getByRole('option', { name: field.target, exact: true }).click();
	}
	if (field.required) {
		await page.getByLabel('Required').click();
	}
	if (field.userEditable) {
		await page.getByLabel('User editable').click();
	}
	if (field.defaultValue !== undefined) {
		if (field.type === 'Boolean') {
			await page.getByLabel('Default value').click();
			await page
				.getByRole('option', { name: field.defaultValue === 'true' ? 'Yes' : 'No', exact: true })
				.click();
		} else {
			await page.getByLabel('Default value').fill(field.defaultValue);
		}
	}
	if (field.validationRegex) {
		await page.getByLabel('Validation regex').fill(field.validationRegex);
	}
	if (field.validationErrorMessage) {
		await page.getByLabel('Validation error message').fill(field.validationErrorMessage);
	}
	await page.getByRole('button', { name: 'Save' }).last().click();
}

test('Update email configuration', async ({ page }) => {
	await page.getByRole('tab', { name: 'Email' }).click();
	await page.getByRole('button', { name: 'Expand card' }).first().click();

	await page.getByLabel('SMTP Host').fill('smtp.gmail.com');
	await page.getByLabel('SMTP Port').fill('587');
	await page.getByLabel('SMTP User').fill('test@gmail.com');
	await page.getByLabel('SMTP Password').fill('password');
	await page.getByLabel('SMTP From').fill('test@gmail.com');
	await page.getByLabel('Email Login Notification').click();
	await page.getByLabel('Email Login Code Requested by User').click();
	await page.getByLabel('Email Login Code from Admin').click();
	await page.getByLabel('API Key Expiration').click();

	await page.getByRole('button', { name: 'Save' }).last().click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'Email configuration updated successfully'
	);

	await page.reload();

	await expect(page.getByLabel('SMTP Host')).toHaveValue('smtp.gmail.com');
	await expect(page.getByLabel('SMTP Port')).toHaveValue('587');
	await expect(page.getByLabel('SMTP User')).toHaveValue('test@gmail.com');
	await expect(page.getByLabel('SMTP Password')).toHaveValue('password');
	await expect(page.getByLabel('SMTP From')).toHaveValue('test@gmail.com');
	await expect(page.getByLabel('Email Login Notification')).toBeChecked();
	await expect(page.getByLabel('Email Login Code Requested by User')).toBeChecked();
	await expect(page.getByLabel('Email Login Code from Admin')).toBeChecked();
	await expect(page.getByLabel('API Key Expiration')).toBeChecked();
});

test.describe('Update application images', () => {
	test.beforeEach(async ({ page }) => {
		await page.getByRole('button', { name: 'Expand card' }).last().click();
	});

	test('should upload images', async ({ page }) => {
		await page.getByLabel('Favicon').setInputFiles('resources/images/w3-schools-favicon.ico');
		await page
			.getByLabel('Light Mode Logo')
			.setInputFiles('resources/images/pingvin-share-logo.png');
		await page.getByLabel('Dark Mode Logo').setInputFiles('resources/images/cloud-logo.png');
		await page.getByLabel('Email Logo').setInputFiles('resources/images/pingvin-share-logo.png');
		await page
			.getByLabel('Default Profile Picture')
			.setInputFiles('resources/images/pingvin-share-logo.png');
		await page.getByLabel('Background Image').setInputFiles('resources/images/clouds.jpg');
		await page.getByRole('button', { name: 'Save' }).last().click();

		await expect(page.locator('[data-type="success"]')).toHaveText(
			'Images updated successfully. It may take a few minutes to update.'
		);

		await page.request
			.get('/api/application-images/favicon')
			.then((res) => expect.soft(res.status()).toBe(200));
		await page.request
			.get('/api/application-images/logo?light=true')
			.then((res) => expect.soft(res.status()).toBe(200));
		await page.request
			.get('/api/application-images/logo?light=false')
			.then((res) => expect.soft(res.status()).toBe(200));
		await page.request
			.get('/api/application-images/email')
			.then((res) => expect.soft(res.status()).toBe(200));
		await page.request
			.get('/api/application-images/background')
			.then((res) => expect.soft(res.status()).toBe(200));
	});

	test('should only allow png/jpeg for email logo', async ({ page }) => {
		const emailLogoInput = page.getByLabel('Email Logo');

		await emailLogoInput.setInputFiles('resources/images/cloud-logo.svg');
		await page.getByRole('button', { name: 'Save' }).last().click();

		await expect(page.locator('[data-type="error"]')).toHaveText(
			'File must be of type .png or .jpg/jpeg'
		);
	});
});

import test, { expect } from '@playwright/test';
import { cleanupBackend } from './utils/cleanup.util';

test.beforeEach(cleanupBackend);

test('Update general configuration', async ({ page }) => {
	await page.goto('/settings/admin/application-configuration');

	await page.getByLabel('Application Name', { exact: true }).fill('Updated Name');
	await page.getByLabel('Session Duration').fill('30');
	await page.getByRole('button', { name: 'Save' }).first().click();

	await expect(page.getByRole('status')).toHaveText(
		'Application configuration updated successfully'
	);
	await expect(page.getByTestId('application-name')).toHaveText('Updated Name');

	await page.reload();

	await expect(page.getByLabel('Application Name', { exact: true })).toHaveValue('Updated Name');
	await expect(page.getByLabel('Session Duration')).toHaveValue('30');
});

test('Update email configuration', async ({ page }) => {
	await page.goto('/settings/admin/application-configuration');

	await page.getByRole('button', { name: 'Expand card' }).nth(1).click();

	await page.getByLabel('SMTP Host').fill('smtp.gmail.com');
	await page.getByLabel('SMTP Port').fill('587');
	await page.getByLabel('SMTP User').fill('test@gmail.com');
	await page.getByLabel('SMTP Password').fill('password');
	await page.getByLabel('SMTP From').fill('test@gmail.com');
	await page.getByLabel('Email Login Notification').click();
	await page.getByLabel('Email Login Code Requested by User').click();
	await page.getByLabel('Email Login Code from Admin').click();
	await page.getByLabel('API Key Expiration').click();

	await page.getByRole('button', { name: 'Save' }).nth(1).click();

	await expect(page.getByRole('status')).toHaveText('Email configuration updated successfully');

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

test('Update LDAP configuration', async ({ page }) => {
	await page.goto('/settings/admin/application-configuration');

	await page.getByRole('button', { name: 'Expand card' }).nth(2).click();

	await page.getByLabel('LDAP URL').fill('ldap://localhost:389');
	await page.getByLabel('LDAP Bind DN').fill('cn=admin,dc=example,dc=com');
	await page.getByLabel('LDAP Bind Password').fill('password');
	await page.getByLabel('LDAP Base DN').fill('dc=example,dc=com');
	await page.getByLabel('User Search Filter').fill('(objectClass=person)');
	await page.getByLabel('Groups Search Filter').fill('(objectClass=groupOfUniqueNames)');
	await page.getByLabel('User Unique Identifier Attribute').fill('uuid');
	await page.getByLabel('Username Attribute').fill('uid');
	await page.getByLabel('User Mail Attribute').fill('mail');
	await page.getByLabel('User First Name Attribute').fill('givenName');
	await page.getByLabel('User Last Name Attribute').fill('sn');
	await page.getByLabel('Group Unique Identifier Attribute').fill('uuid');
	await page.getByLabel('Group Name Attribute').fill('cn');
	await page.getByLabel('Admin Group Name').fill('admin');

	await page.getByRole('button', { name: 'Enable' }).click();

	await expect(page.getByRole('status')).toHaveText('LDAP configuration updated successfully');

	await page.reload();

	await expect(page.getByRole('button', { name: 'Disable' })).toBeVisible();
	await expect(page.getByLabel('LDAP URL')).toHaveValue('ldap://localhost:389');
	await expect(page.getByLabel('LDAP Bind DN')).toHaveValue('cn=admin,dc=example,dc=com');
	await expect(page.getByLabel('LDAP Bind Password')).toHaveValue('password');
	await expect(page.getByLabel('LDAP Base DN')).toHaveValue('dc=example,dc=com');
	await page.getByLabel('User Search Filter').fill('(objectClass=person)');
	await page.getByLabel('Groups Search Filter').fill('(objectClass=groupOfUniqueNames)');
	await expect(page.getByLabel('User Unique Identifier Attribute')).toHaveValue('uuid');
	await expect(page.getByLabel('Username Attribute')).toHaveValue('uid');
	await expect(page.getByLabel('User Mail Attribute')).toHaveValue('mail');
	await expect(page.getByLabel('User First Name Attribute')).toHaveValue('givenName');
	await expect(page.getByLabel('User Last Name Attribute')).toHaveValue('sn');
	await expect(page.getByLabel('Admin Group Name')).toHaveValue('admin');
});

test('Update application images', async ({ page }) => {
	await page.goto('/settings/admin/application-configuration');

	await page.getByRole('button', { name: 'Expand card' }).nth(3).click();

	await page.getByLabel('Favicon').setInputFiles('tests/assets/w3-schools-favicon.ico');
	await page.getByLabel('Light Mode Logo').setInputFiles('tests/assets/pingvin-share-logo.png');
	await page.getByLabel('Dark Mode Logo').setInputFiles('tests/assets/nextcloud-logo.png');
	await page.getByLabel('Background Image').setInputFiles('tests/assets/clouds.jpg');
	await page.getByRole('button', { name: 'Save' }).nth(1).click();

	await expect(page.getByRole('status')).toHaveText('Images updated successfully');

	await page.request
		.get('/api/application-configuration/favicon')
		.then((res) => expect.soft(res.status()).toBe(200));
	await page.request
		.get('/api/application-configuration/logo?light=true')
		.then((res) => expect.soft(res.status()).toBe(200));
	await page.request
		.get('/api/application-configuration/logo?light=false')
		.then((res) => expect.soft(res.status()).toBe(200));
	await page.request
		.get('/api/application-configuration/background-image')
		.then((res) => expect.soft(res.status()).toBe(200));
});

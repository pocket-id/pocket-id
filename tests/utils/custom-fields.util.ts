import { expect, type Page } from '@playwright/test';

export type CustomField = {
	id: string;
	key: string;
	displayName: string;
	type: 'string' | 'number' | 'boolean';
	target: 'user' | 'group' | 'both';
	required: boolean;
	userEditable: boolean;
	defaultValue?: string;
	validationRegex?: string;
	validationErrorMessage?: string;
};

export type CustomFieldValue = {
	customFieldId: string;
	value: string;
};

export async function updateUserCustomFieldsViaApi(
	page: Page,
	userId: string,
	customFieldValues: CustomFieldValue[]
) {
	const userResponse = await page.request.get(`/api/users/${userId}`);
	expect(userResponse.ok()).toBeTruthy();
	const user = await userResponse.json();

	const updateResponse = await page.request.put(`/api/users/${userId}`, {
		data: {
			firstName: user.firstName,
			lastName: user.lastName,
			displayName: user.displayName,
			email: user.email,
			emailVerified: user.emailVerified,
			username: user.username,
			isAdmin: user.isAdmin,
			disabled: user.disabled,
			customFieldValues
		}
	});
	expect(updateResponse.ok()).toBeTruthy();
}

export async function updateUserGroupCustomFieldsViaApi(
	page: Page,
	userGroupId: string,
	customFieldValues: CustomFieldValue[]
) {
	const userGroupResponse = await page.request.get(`/api/user-groups/${userGroupId}`);
	expect(userGroupResponse.ok()).toBeTruthy();
	const userGroup = await userGroupResponse.json();

	const updateResponse = await page.request.put(`/api/user-groups/${userGroupId}`, {
		data: {
			friendlyName: userGroup.friendlyName,
			name: userGroup.name,
			ldapId: userGroup.ldapId,
			customFieldValues
		}
	});
	expect(updateResponse.ok()).toBeTruthy();
}

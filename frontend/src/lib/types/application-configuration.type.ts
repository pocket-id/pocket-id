import type { CustomFieldValue, CustomField } from './custom-field.type';

export type AppConfig = {
	appName: string;
	homePageUrl: string;
	allowOwnAccountEdit: boolean;
	allowUserSignups: 'disabled' | 'withToken' | 'open';
	emailOneTimeAccessAsUnauthenticatedEnabled: boolean;
	emailOneTimeAccessAsAdminEnabled: boolean;
	emailVerificationEnabled: boolean;
	ldapEnabled: boolean;
	disableAnimations: boolean;
	uiConfigDisabled: boolean;
	accentColor: string;
	requireUserEmail: boolean;
	customFields: CustomField[];
};

export type AllAppConfig = AppConfig & {
	// General
	sessionDuration: number;
	emailsVerified: boolean;
	signupDefaultUserGroupIDs: string[];
	// Email
	smtpHost: string;
	smtpPort: string;
	smtpFrom: string;
	smtpUser: string;
	smtpPassword: string;
	smtpTls: 'none' | 'starttls' | 'tls';
	smtpSkipCertVerify: boolean;
	emailLoginNotificationEnabled: boolean;
	emailApiKeyExpirationEnabled: boolean;
	// LDAP
	ldapUrl: string;
	ldapBindDn: string;
	ldapBindPassword: string;
	ldapBase: string;
	ldapUserSearchFilter: string;
	ldapUserGroupSearchFilter: string;
	ldapSkipCertVerify: boolean;
	ldapAttributeUserUniqueIdentifier: string;
	ldapAttributeUserUsername: string;
	ldapAttributeUserEmail: string;
	ldapAttributeUserFirstName: string;
	ldapAttributeUserLastName: string;
	ldapAttributeUserDisplayName: string;
	ldapAttributeUserProfilePicture: string;
	ldapAttributeGroupMember: string;
	ldapAttributeGroupUniqueIdentifier: string;
	ldapAttributeGroupName: string;
	ldapAdminGroupName: string;
	ldapSoftDeleteUsers: boolean;
};

export type AppConfigRawResponse = {
	key: string;
	type: string;
	value: string;
}[];

export type AppVersionInformation = {
	isUpToDate: boolean | null;
	newestVersion: string | null;
	currentVersion: string;
};

export type AppConfig = {
	appName: string;
	allowOwnAccountEdit: boolean;
	allowUserSignups: 'disabled' | 'withToken' | 'open';
	emailOneTimeAccessAsUnauthenticatedEnabled: boolean;
	emailOneTimeAccessAsAdminEnabled: boolean;
	ldapEnabled: boolean;
	disableAnimations: boolean;
	uiConfigDisabled: boolean;
	accentColor: string;
};

export type AllAppConfig = AppConfig & {
	// General
	sessionDuration: number;
	emailsVerified: boolean;
	// Email
	smtpHost: string;
	smtpPort: number;
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
	ldapAttributeUserProfilePicture: string;
	ldapAttributeGroupMember: string;
	ldapAttributeGroupUniqueIdentifier: string;
	ldapAttributeGroupName: string;
	ldapAttributeAdminGroup: string;
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

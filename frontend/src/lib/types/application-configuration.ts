export type AppConfig = {
	appName: string;
	allowOwnAccountEdit: boolean;
	emailOneTimeAccessAsUnauthenticatedEnabled: boolean;
	emailOneTimeAccessAsAdminEnabled: boolean;
	ldapEnabled: boolean;
	disableAnimations: boolean;
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
	isUpToDate?: boolean;
	newestVersion?: string;
	currentVersion: string;
};

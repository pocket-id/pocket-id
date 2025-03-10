export const users = {
	tim: {
		id: 'f4b89dc2-62fb-46bf-9f5f-c34f4eafe93e',
		firstname: 'Tim',
		lastname: 'Cook',
		email: 'tim.cook@test.com',
		username: 'tim'
	},
	craig: {
		id: '1cd19686-f9a6-43f4-a41f-14a0bf5b4036',
		firstname: 'Craig',
		lastname: 'Federighi',
		email: 'craig.federighi@test.com',
		username: 'craig'
	},
	steve: {
		firstname: 'Steve',
		lastname: 'Jobs',
		email: 'steve.jobs@test.com',
		username: 'steve'
	}
};

export const oidcClients = {
	nextcloud: {
		id: '3654a746-35d4-4321-ac61-0bdcff2b4055',
		name: 'Nextcloud',
		callbackUrl: 'http://nextcloud/auth/callback',
		logoutCallbackUrl: 'http://nextcloud/auth/logout/callback'
	},
	immich: {
		id: '606c7782-f2b1-49e5-8ea9-26eb1b06d018',
		name: 'Immich',
		callbackUrl: 'http://immich/auth/callback'
	},
	pingvinShare: {
		name: 'Pingvin Share',
		callbackUrl: 'http://pingvin.share/auth/callback',
		secondCallbackUrl: 'http://pingvin.share/auth/callback2'
	}
};

export const userGroups = {
	developers: {
		id: '4110f814-56f1-4b28-8998-752b69bc97c0e',
		friendlyName: 'Developers',
		name: 'developers'
	},
	designers: {
		id: 'adab18bf-f89d-4087-9ee1-70ff15b48211',
		friendlyName: 'Designers',
		name: 'designers'
	},
	humanResources: {
		friendlyName: 'Human Resources',
		name: 'human_resources'
	}
};

export const oneTimeAccessTokens = [
	{ token: 'HPe6k6uiDRRVuAQV', expired: false },
	{ token: 'YCGDtftvsvYWiXd0', expired: true }
];

export const apiKeys = [
	{
		id: '5f1fa856-c164-4295-961e-175a0d22d725',
		key: '6c34966f57ef2bb7857649aff0e7ab3ad67af93c846342ced3f5a07be8706c20',
		name: 'Test API Key'
	}
];

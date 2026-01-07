export const users = {
	tim: {
		id: 'f4b89dc2-62fb-46bf-9f5f-c34f4eafe93e',
		firstname: 'Tim',
		lastname: 'Cook',
		displayName: 'Tim Cook',
		email: 'tim.cook@test.com',
		username: 'tim'
	},
	craig: {
		id: '1cd19686-f9a6-43f4-a41f-14a0bf5b4036',
		firstname: 'Craig',
		lastname: 'Federighi',
		displayName: 'Craig Federighi',
		email: 'craig.federighi@test.com',
		username: 'craig'
	},
	steve: {
		firstname: 'Steve',
		lastname: 'Jobs',
		displayName: 'Steve Jobs',
		email: 'steve.jobs@test.com',
		username: 'steve'
	}
};

export const oidcClients = {
	nextcloud: {
		id: '3654a746-35d4-4321-ac61-0bdcff2b4055',
		name: 'Nextcloud',
		callbackUrl: 'http://nextcloud/auth/callback',
		logoutCallbackUrl: 'http://nextcloud/auth/logout/callback',
		secret: 'w2mUeZISmEvIDMEDvpY0PnxQIpj1m3zY',
		launchURL: 'https://nextcloud.local'
	},
	immich: {
		id: '606c7782-f2b1-49e5-8ea9-26eb1b06d018',
		name: 'Immich',
		callbackUrl: 'http://immich/auth/callback',
		secret: 'PYjrE9u4v9GVqXKi52eur0eb2Ci4kc0x'
	},
	tailscale: {
		id: '7c21a609-96b5-4011-9900-272b8d31a9d1',
		name: 'Tailscale',
		callbackUrl: 'http://tailscale/auth/callback',
		secret: 'n4VfQeXlTzA6yKpWbR9uJcMdSx2qH0Lo'
	},
	federated: {
		id: 'c48232ff-ff65-45ed-ae96-7afa8a9b443b',
		name: 'Federated',
		callbackUrl: 'http://federated/auth/callback',
		federatedJWT: {
			issuer: 'https://external-idp.local',
			audience: 'api://PocketID',
			subject: 'c48232ff-ff65-45ed-ae96-7afa8a9b443b'
		},
		accessCodes: ['federated']
	},
	scim: {
		id: 'c46d2090-37a0-4f2b-8748-6aa53b0c1afa',
		name: 'SCIM Client',
		callbackUrl: 'http://scim.client/auth/callback',
		secret: 'nQbiuMRG7FpdK2EnDd5MBivWQeKFXohn'
	},
	pingvinShare: {
		name: 'Pingvin Share',
		callbackUrl: 'http://pingvin.share/auth/callback',
		secondCallbackUrl: 'http://pingvin.share/auth/callback2',
		launchURL: 'https://pingvin-share.local'
	}
};

export const userGroups = {
	developers: {
		id: 'c7ae7c01-28a3-4f3c-9572-1ee734ea8368',
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
		name: 'Test API Key',
		expired: false
	},
	{
		id: '98900330-7a7b-48fe-881b-2cc6ad049976',
		key: '141ff8ac9db640ba93630099de83d0ead8e7ac673e3a7d31b4fd7ff2252e6389',
		name: 'Expired API Key',
		expired: true
	}
];

export const refreshTokens = [
	{
		token: 'ou87UDg249r1StBLYkMEqy9TXDbV5HmGuDpMcZDo',
		clientId: oidcClients.nextcloud.id,
		userId: 'f4b89dc2-62fb-46bf-9f5f-c34f4eafe93e',
		expired: false
	},
	{
		token: 'X4vqwtRyCUaq51UafHea4Fsg8Km6CAns6vp3tuX4',
		clientId: oidcClients.nextcloud.id,
		userId: 'f4b89dc2-62fb-46bf-9f5f-c34f4eafe93e',
		expired: true
	}
];

export const signupTokens = {
	valid: {
		id: 'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
		token: 'VALID1234567890A',
		expiresAt: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
		usageLimit: 1,
		usageCount: 0,
		createdAt: new Date().toISOString()
	},
	partiallyUsed: {
		id: 'dc3c9c96-714e-48eb-926e-2d7c7858e6cf',
		token: 'PARTIAL567890ABC',
		expiresAt: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
		usageLimit: 5,
		usageCount: 2,
		createdAt: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString()
	},
	expired: {
		id: '44de1863-ffa5-4db1-9507-4887cd7a1e3f',
		token: 'EXPIRED34567890B',
		expiresAt: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
		usageLimit: 3,
		usageCount: 1,
		createdAt: new Date(Date.now() - 3 * 24 * 60 * 60 * 1000).toISOString()
	},
	fullyUsed: {
		id: 'f1b1678b-7720-4d8b-8f91-1dbff1e2d02b',
		token: 'FULLYUSED567890C',
		expiresAt: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
		usageLimit: 1,
		usageCount: 1,
		createdAt: new Date(Date.now() - 1 * 60 * 60 * 1000).toISOString()
	}
};

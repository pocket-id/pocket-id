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
		logoutCallbackUrl: 'http://nextcloud/auth/logout/callback',
		secret: 'w2mUeZISmEvIDMEDvpY0PnxQIpj1m3zY'
	},
	immich: {
		id: '606c7782-f2b1-49e5-8ea9-26eb1b06d018',
		name: 'Immich',
		callbackUrl: 'http://immich/auth/callback',
		secret: 'PYjrE9u4v9GVqXKi52eur0eb2Ci4kc0x'
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

export const refreshTokens = [
	{
		token: 'ou87UDg249r1StBLYkMEqy9TXDbV5HmGuDpMcZDo',
		clientId: oidcClients.nextcloud.id,
		expired: false
	},
	{
		token: 'X4vqwtRyCUaq51UafHea4Fsg8Km6CAns6vp3tuX4',
		clientId: oidcClients.nextcloud.id,
		expired: true
	}
];

export const idTokens = [
	{
		token:
			'eyJhbGciOiJSUzI1NiIsImtpZCI6Ijh1SER3M002cmY4IiwidHlwIjoiSldUIn0.eyJhdWQiOiIzNjU0YTc0Ni0zNWQ0LTQzMjEtYWM2MS0wYmRjZmYyYjQwNTUiLCJlbWFpbCI6InRpbS5jb29rQHRlc3QuY29tIiwiZW1haWxfdmVyaWZpZWQiOnRydWUsImV4cCI6MTY5MDAwMDAwMSwiZmFtaWx5X25hbWUiOiJUaW0iLCJnaXZlbl9uYW1lIjoiQ29vayIsImlhdCI6MTY5MDAwMDAwMCwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdCIsIm5hbWUiOiJUaW0gQ29vayIsIm5vbmNlIjoib1cxQTFPNzhHUTE1RDczT3NIRXg3V1FLajdacXZITFp1XzM3bWRYSXFBUSIsInN1YiI6IjRiODlkYzItNjJmYi00NmJmLTlmNWYtYzM0ZjRlYWZlOTNlIiwidHlwZSI6ImlkLXRva2VuIn0.noxQ-sCNHh7f8EaySJT7oF0DlmjYcM-FdMPH45Yuuvt5-bTpLLkggN9aq8RILmkGL9xUVsfZbYkWV5EkGobxfIoXITE98xH54BQwtpOjLL_HZLF4kFXarUyGLGO3zeVJAQzyofVz_1rKfDlZdi5Zmm-91cO5OiOtshfluDqt1h1D-E5h4ShT0eN7apvSvQnD7806-3tfxP0GHE-HuerR1Qbv9p0uWmuhT0CkVIM-K2dKBHdhLtquRqxNp2EuD_T-HA3WJgvkTTWp-JZ6NqvWDMy3M-jB-_Bs9eABERlTSTp7H2XCMGbwRSBZDmSn-97LPwc-NO5JYEkgZOeVr_r6qg',
		clientId: oidcClients.nextcloud.id,
		expired: true
	},
	{
		token:
			'eyJhbGciOiJSUzI1NiIsImtpZCI6Ijh1SER3M002cmY4IiwidHlwIjoiSldUIn0.eyJhdWQiOiIzNjU0YTc0Ni0zNWQ0LTQzMjEtYWM2MS0wYmRjZmYyYjQwNTUiLCJlbWFpbCI6InRpbS5jb29rQHRlc3QuY29tIiwiZW1haWxfdmVyaWZpZWQiOnRydWUsImV4cCI6MjY5MDAwMDAwMSwiZmFtaWx5X25hbWUiOiJUaW0iLCJnaXZlbl9uYW1lIjoiQ29vayIsImlhdCI6MTY5MDAwMDAwMCwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdCIsIm5hbWUiOiJUaW0gQ29vayIsIm5vbmNlIjoib1cxQTFPNzhHUTE1RDczT3NIRXg3V1FLajdacXZITFp1XzM3bWRYSXFBUSIsInN1YiI6IjRiODlkYzItNjJmYi00NmJmLTlmNWYtYzM0ZjRlYWZlOTNlIiwidHlwZSI6ImlkLXRva2VuIn0.ry7s3xP4-vvvZPzcRCvR1yBl32Pi09ryC6Z-67E1P4xChe8MaMoiQgImS5ZNbZiYzBN4cdkQsExXZK1FP-kMD019k3uNKPq0fIREBwrT9wXPqQJlLSBmN-tVkjLm90-b310SG5p65aajWvMkcPmJleG6y24_zoPFr3ISGI87vV6zdyoqG55pc-GkT7FwiEFIZJGQAzl7u1uOi7sQrda8Y6rF_SCC-f9I4PnHblnaTne8pfXe9jXKJeY1ZKj2Qh9dRPhWCLPHHV1YErUyoMP9oeMVzYpno-pBYVOiT9Ktl6CpG-jqB8smKqDEhZrSejgZ256h34f8jNL1SEhpM-4_cQ',
		clientId: oidcClients.nextcloud.id,
		expired: false
	}
];

export const accessTokens = [
	{
		token:
			'eyJhbGciOiJSUzI1NiIsImtpZCI6Ijh1SER3M002cmY4IiwidHlwIjoiSldUIn0.eyJhdWQiOlsiMzY1NGE3NDYtMzVkNC00MzIxLWFjNjEtMGJkY2ZmMmI0MDU1Il0sImV4cCI6MTc0Mzk3MjI4MywiaWF0IjoxNzQzOTY4NjgzLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0Iiwic3ViIjoiZjRiODlkYzItNjJmYi00NmJmLTlmNWYtYzM0ZjRlYWZlOTNlIiwidHlwZSI6Im9hdXRoLWFjY2Vzcy10b2tlbiJ9.REFSDFsGso9u7WxpyMmMVvjQMgulbidQNUft-kBRg7nw5LN9pOWhO0Zlr1tZnnrA1LenZRv0BvLIf0qekwGEC4FOPmJ6-As2ggIcoBIXpUR2A4Hhuy0FtqbCUgIkda1Dcx9w1Rmfzi0eHY_-1H_98rDgS5RxqweNA_YP3RsnJqBsc9GYhDarrf1nyCOplshGOEiyisUGoU2TaURI6DTcCiDzVOm_esZqokoZTpKlQw6ZugDDObro0eWYgROo97_3cqPRgRjSYBYRAGCHhZom3bFkjmz3wqpeoGmUNgL022x3-gl7QjurpJMQrKJ7wkFs0bh2uFnnngnh2w6m4j8-5w',
		clientId: oidcClients.nextcloud.id,
		expired: true
	},
	{
		token:
			'eyJhbGciOiJSUzI1NiIsImtpZCI6Ijh1SER3M002cmY4IiwidHlwIjoiSldUIn0.eyJhdWQiOlsiMzY1NGE3NDYtMzVkNC00MzIxLWFjNjEtMGJkY2ZmMmI0MDU1Il0sImV4cCI6Mjc0Mzk3MjI4MywiaWF0IjoxNzQzOTY4NjgzLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0Iiwic3ViIjoiZjRiODlkYzItNjJmYi00NmJmLTlmNWYtYzM0ZjRlYWZlOTNlIiwidHlwZSI6Im9hdXRoLWFjY2Vzcy10b2tlbiJ9.FaFsHJS_8wbvQvctftNTPyzAe9IhbpJiHIkhg28RrFRFfnBMq0QycmTUh00MJPXkUfd_j5tcCnXybF1efHsq6WbP4AWFG_TJMUyz7a9SYt1lGR8dxo3eys0YAX5eJQ5YoVTKNrivSKrC37Rg3VlcZVWXp6KBAxRWVl3OUlquSC6q7HNKAKg8sbBJiGpUJ37wwanOTE2XhYGvFB2_gxS36tvOuSTV3CVg_7Fctej7gNhKMXBFMJiIFurxZaeNud8620xtv-vJX6ALa1Qu1SkWhhZN2Yx3WuODZNlni3rUps-THoEdqh62jNwItE9wB7C0fGEKuUqVIllaF9I_7i2s3w',
		clientId: oidcClients.nextcloud.id,
		expired: false
	}
];

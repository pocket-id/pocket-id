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
			'eyJhbGciOiJSUzI1NiIsImtpZCI6Ijh1SER3M002cmY4IiwidHlwIjoiaWQrand0In0.eyJhdWQiOiIzNjU0YTc0Ni0zNWQ0LTQzMjEtYWM2MS0wYmRjZmYyYjQwNTUiLCJlbWFpbCI6InRpbS5jb29rQHRlc3QuY29tIiwiZW1haWxfdmVyaWZpZWQiOnRydWUsImV4cCI6MTY5MDAwMDAwMSwiZmFtaWx5X25hbWUiOiJUaW0iLCJnaXZlbl9uYW1lIjoiQ29vayIsImlhdCI6MTY5MDAwMDAwMCwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdCIsIm5hbWUiOiJUaW0gQ29vayIsIm5vbmNlIjoib1cxQTFPNzhHUTE1RDczT3NIRXg3V1FLajdacXZITFp1XzM3bWRYSXFBUSIsInN1YiI6IjRiODlkYzItNjJmYi00NmJmLTlmNWYtYzM0ZjRlYWZlOTNlIn0.LHwNnp9WxFc_NbIVsBC41trA-1LUBxTfKwIqfgGP4WC5j39M2Rmc0G4rw7J96tfwyEobwgPFAP0YJ3BqMaZgHT4Zu0rYSenU-yv_CICWiLL4csyeojlqbqDKDiOD3Gsl4_ZUuo8UuN190RGz6HlxmTwxpmceerSFpx6dBtA6chYZfgnUf289DRWIgTsNrXnkohZRa8zWc8bjbw_hj1u7H6Ev9Yu3U2k4K0cHWZLFjQiPWt3JBaWNAldSEn2q7a3Rkyv17_Gx8Nwl5L4ugWKV8M1YkcHbEkYCJKaJCbZi13R89yH1E0EOfHYXK5Z0KqBq47eTYRGRUtFiP-uTlUDQUQ',
		clientId: oidcClients.nextcloud.id,
		expired: true
	},
	{
		token:
			'eyJhbGciOiJSUzI1NiIsImtpZCI6Ijh1SER3M002cmY4IiwidHlwIjoiaWQrand0In0.eyJhdWQiOiIzNjU0YTc0Ni0zNWQ0LTQzMjEtYWM2MS0wYmRjZmYyYjQwNTUiLCJlbWFpbCI6InRpbS5jb29rQHRlc3QuY29tIiwiZW1haWxfdmVyaWZpZWQiOnRydWUsImV4cCI6MjY5MDAwMDAwMSwiZmFtaWx5X25hbWUiOiJUaW0iLCJnaXZlbl9uYW1lIjoiQ29vayIsImlhdCI6MTY5MDAwMDAwMCwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdCIsIm5hbWUiOiJUaW0gQ29vayIsIm5vbmNlIjoib1cxQTFPNzhHUTE1RDczT3NIRXg3V1FLajdacXZITFp1XzM3bWRYSXFBUSIsInN1YiI6IjRiODlkYzItNjJmYi00NmJmLTlmNWYtYzM0ZjRlYWZlOTNlIn0.yG21sES1HMyQg6GeJtd-6sUJ5a_QBS-hHq3mDTjRoMkL604RxprPvIJ-ypYhzcV5LwlTiD-7jJQ2Z95uUb82aNek55V5Pzq_rcLM5EtHh2bHSegt_1QXcpBzl8mWB1AIZBSRzFDaB1msnkyxGnndJk4VHpUVStvubcldxksH3e9v286x9ak4oTNoaLy4kMi4KAE8WCwrqsYc1iieLOSFTRHjpM9YxWa8X9hGNsikC85NJ0tj1pG9I4QTG62h4ZqJ4-jFWe5dogg_vd9Sk7tA3f9S779XSCG6hpj1V-sxQqLCy9uAmB2URP4N60jamKTn2TCxc1R7xgQ7M9Rc9ty68g',
		clientId: oidcClients.nextcloud.id,
		expired: false
	}
];

export const accessTokens = [
	{
		token:
			'eyJhbGciOiJSUzI1NiIsImtpZCI6Ijh1SER3M002cmY4IiwidHlwIjoiYXQrand0In0.eyJhdWQiOlsiMzY1NGE3NDYtMzVkNC00MzIxLWFjNjEtMGJkY2ZmMmI0MDU1Il0sImV4cCI6MTc0Mzk3MjI4MywiaWF0IjoxNzQzOTY4NjgzLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0Iiwic3ViIjoiZjRiODlkYzItNjJmYi00NmJmLTlmNWYtYzM0ZjRlYWZlOTNlIn0.iwkQR96BKTJugh87_YOrDb7hXSWsv0RJXrqrqxHn3rwhcKNxwGnYAhTiQ12wyi-77-AFkzUlgs9E9pwgVi3_sE37QCVZ3YZzHjbg5crmT1EJ4f8gN8hw5cDqC3ny0R8rhgNzzirpZNe-i7SXzWCIySyEVh7MGFTPqNA-1ZlGh06FuOFRb22GVaHfrDkpE2RhkeZ-ZLlua9pbTcT1T9CihlCrW8JKTUwT2QspCwtnaJGs34iH77sHry31cTYVyOqd5q218tg_N4ky9iV6k7mK6b7uaPsjYHrtpfK1tp-9_MSp6Fqzw6wu_vrvg5WrIWwiREaz_wJj-SjIuBR5TlntdA',
		clientId: oidcClients.nextcloud.id,
		expired: true
	},
	{
		token:
			'eyJhbGciOiJSUzI1NiIsImtpZCI6Ijh1SER3M002cmY4IiwidHlwIjoiYXQrand0In0.eyJhdWQiOlsiMzY1NGE3NDYtMzVkNC00MzIxLWFjNjEtMGJkY2ZmMmI0MDU1Il0sImV4cCI6Mjc0Mzk3MjI4MywiaWF0IjoxNzQzOTY4NjgzLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0Iiwic3ViIjoiZjRiODlkYzItNjJmYi00NmJmLTlmNWYtYzM0ZjRlYWZlOTNlIn0.lZMEohQeOi6oKDsKLKDDRYJIJNedUilvCLCi6XLADcHPtKlFJbPqH8IuQxuzryeIYAnTILsjvTkxkHAeRoQZCXQR7oS5BguGx6MtQYjgj--GpLBQ39r_nz-SEfhKtuMzEzPsN1raxOH8jWbnPM7zHxf5NIz7AHDKtCSWRA3JlE9kgAU7S-RRc6xP_BYVPDB97J6k-xuO5zxcdNTb92j8pZWbPPokv6CGG9CTPNzcrNHf-M98M6GE8SVM-8R2MAbpUCqTkTc_O46GHEexZzif2Wg8K5O-htiSQnwumoXXN08zKHCzCAvSdSa9JRMB-cgP7jsM7I6itUBXWxgvWDK3rA',
		clientId: oidcClients.nextcloud.id,
		expired: false
	}
];

export type Session = {
	id: string;
	browser: string;
	os: string;
	deviceType: 'desktop' | 'mobile' | 'tablet';
	userAgent: string;
	ipAddress: string;
	country: string;
	city: string;
	authenticationMethod: string;
	current: boolean;
	createdAt: string;
	lastActiveAt: string;
	expiresAt: string;
};

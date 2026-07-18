import type { User } from './user.type';

export type DeviceLoginRequest = {
	id: string;
	userCode: string;
	verificationUri: string;
	verificationUriComplete: string;
	expiresAt: string;
	interval: number;
};

export type DeviceLoginVerificationInfo = {
	userCode: string;
	device: string;
	ipAddress?: string;
	expiresAt: string;
};

export type DeviceLoginDecision = 'approve' | 'deny';

export type DeviceLoginExchangeResult = User | null;

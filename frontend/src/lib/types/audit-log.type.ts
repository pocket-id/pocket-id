export type AuditLog = {
	id: string;
	event: string;
	ipAddress: string;
	country?: string;
	city?: string;
	device: string;
	username?: string;
	createdAt: string;
	data: any;
};

export type AuditLog = {
	id: string;
	event: string;
	ipAddress: string;
	country?: string;
	city?: string;
	device: string;
	userID: string;
	username?: string;
	actorUsername?: string;
	createdAt: string;
	data: any;
};

export type AuditLogFilter = {
	userID: string;
	event: string;
	location: string;
	clientName: string;
};

export type AuditLog = {
	id: string;
	event: string;
	ipAddress: string;
	country?: string;
	city?: string;
	device: string;
	userId: string;
	username?: string;
	createdAt: string;
	data: any;
};

export type AuditLogFilter = {
	userId: string,
	event: string,
	clientName: string,
}
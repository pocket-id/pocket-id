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

// RFC 3339 bounds of the date filter: from is inclusive, to exclusive, and a missing bound is open
export type AuditLogDateRange = {
	from?: string;
	to?: string;
};

export type AuditLogFilter = {
	userID: string;
	event: string;
	location: string;
	clientName: string;
};

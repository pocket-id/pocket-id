export type SigningStatus = {
	enabled: boolean;
	certificate: {
		subject: string;
		issuer: string;
		emails: string[];
		notBefore: string;
		notAfter: string;
		fingerprint: string;
		status: 'valid' | 'expiringSoon' | 'expired' | 'notYetValid';
		senderMatches: boolean;
	} | null;
};

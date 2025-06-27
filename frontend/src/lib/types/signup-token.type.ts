export interface SignupTokenDto {
	id: string;
	token: string;
	expiresAt: string;
	usageLimit: number;
	usageCount: number;
	createdAt: string;
}

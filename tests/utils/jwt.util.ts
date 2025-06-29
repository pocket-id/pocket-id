import * as jose from 'jose';
import playwrightConfig from '../playwright.config';

const PRIVATE_KEY_STRING = `{"alg":"RS256","d":"mvMDWSdPPvcum0c0iEHE2gbqtV2NKMmLwrl9E6K7g8lTV95SePLnW_bwyMPV7EGp7PQk3l17I5XRhFjze7GqTnFIOgKzMianPs7jv2ELtBMGK0xOPATgu1iGb70xZ6vcvuEfRyY3dJ0zr4jpUdVuXwKmx9rK4IdZn2dFCKfvSuspqIpz11RhF1ALrqDLkxGVv7ZwNh0_VhJZU9hcjG5l6xc7rQEKpPRkZp0IdjkGS8Z0FskoVaiRIWAbZuiVFB9WCW8k1czC4HQTPLpII01bUQx2ludbm0UlXRgVU9ptUUbU7GAImQqTOW8LfPGklEvcgzlIlR_oqw4P9yBxLi-yMQ","dp":"pvNCSnnhbo8Igw9psPR-DicxFnkXlu_ix4gpy6efTrxA-z1VDFDioJ814vKQNioYDzpyAP1gfMPhRkvG_q0hRZsJah3Sb9dfA-WkhSWY7lURQP4yIBTMU0PF_rEATuS7lRciYk1SOx5fqXZd3m_LP0vpBC4Ujlq6NAq6CIjCnms","dq":"TtUVGCCkPNgfOLmkYXu7dxxUCV5kB01-xAEK2OY0n0pG8vfDophH4_D_ZC7nvJ8J9uDhs_3JStexq1lIvaWtG99RNTChIEDzpdn6GH9yaVcb_eB4uJjrNm64FhF8PGCCwxA-xMCZMaARKwhMB2_IOMkxUbWboL3gnhJ2rDO_QO0","e":"AQAB","kid":"8uHDw3M6rf8","kty":"RSA","n":"yaeEL0VKoPBXIAaWXsUgmu05lAvEIIdJn0FX9lHh4JE5UY9B83C5sCNdhs9iSWzpeP11EVjWp8i3Yv2CF7c7u50BXnVBGtxpZpFC-585UXacoJ0chUmarL9GRFJcM1nPHBTFu68aRrn1rIKNHUkNaaxFo0NFGl_4EDDTO8HwawTjwkPoQlRzeByhlvGPVvwgB3Fn93B8QJ_cZhXKxJvjjrC_8Pk76heC_ntEMru71Ix77BoC3j2TuyiN7m9RNBW8BU5q6lKoIdvIeZfTFLzi37iufyfvMrJTixp9zhNB1NxlLCeOZl2MXegtiGqd2H3cbAyqoOiv9ihUWTfXj7SxJw","p":"_Yylc9e07CKdqNRD2EosMC2mrhrEa9j5oY_l00Qyy4-jmCA59Q9viyqvveRo0U7cRvFA5BWgWN6GGLh1DG3X-QBqVr0dnk3uzbobb55RYUXyPLuBZI2q6w2oasbiDwPdY7KpkVv_H-bpITQlyDvO8hhucA6rUV7F6KTQVz8M3Ms","q":"y5p3hch-7jJ21TkAhp_Vk1fLCAuD4tbErwQs2of9ja8sB4iJOs5Wn6HD3P7Mc8Plye7qaLHvzc8I5g0tPKWvC0DPd_FLPXiWwMVAzee3NUX_oGeJNOQp11y1w_KqdO9qZqHSEPZ3NcFL_SZMFgggxhM1uzRiPzsVN0lnD_6prZU","qi":"2Grt6uXHm61ji3xSdkBWNtUnj19vS1-7rFJp5SoYztVQVThf_W52BAiXKBdYZDRVoItC_VS2NvAOjeJjhYO_xQ_q3hK7MdtuXfEPpLnyXKkmWo3lrJ26wbeF6l05LexCkI7ShsOuSt-dsyaTJTszuKDIA6YOfWvfo3aVZmlWRaI","use":"sig"}`;

type User = {
	id: string;
	email: string;
	firstname: string;
	lastname: string;
};

const privateKey = JSON.parse(PRIVATE_KEY_STRING);
const privateKeyImported = await jose.importJWK(privateKey, 'RS256');

export async function generateIdToken(user: User, clientId: string, expired = false) {
	const now = Math.floor(Date.now() / 1000);
	const expiration = expired ? now + 1 : now + 1000000000; // Either expired or valid for a long time

	const payload = {
		aud: clientId,
		email: user.email,
		email_verified: true,
		exp: expiration,
		family_name: user.lastname,
		given_name: user.firstname,
		iat: now,
		iss: playwrightConfig.use!.baseURL,
		name: `${user.firstname} ${user.lastname}`,
		nonce: 'oW1A1O78GQ15D73OsHEx7WQKj7ZqvHLZu_37mdXIqAQ',
		sub: user.id,
		type: 'id-token'
	};

	return await new jose.SignJWT(payload)
		.setProtectedHeader({ alg: 'RS256', kid: privateKey.kid, typ: 'JWT' })
		.sign(privateKeyImported);
}

export async function generateOauthAccessToken(user: User, clientId: string, expired = false) {
	const now = Math.floor(Date.now() / 1000);
	const expiration = expired ? now - 1000 : now + 1000000000; // Either expired or valid for a long time

	const payload = {
		aud: [clientId],
		exp: expiration,
		iat: now,
		iss: playwrightConfig.use!.baseURL,
		sub: user.id,
		type: 'oauth-access-token'
	};

	return await new jose.SignJWT(payload)
		.setProtectedHeader({ alg: 'RS256', kid: privateKey.kid, typ: 'JWT' })
		.sign(privateKeyImported);
}

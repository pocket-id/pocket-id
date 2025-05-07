import type { RequestHandler } from '@sveltejs/kit';
import axios from 'axios';

export const GET: RequestHandler = async () => {
	const backendOK = await axios
		.get(process!.env!.INTERNAL_BACKEND_URL + '/healthz')
		.then(() => true, () => false);

	return new Response(
		backendOK ? `{"status":"HEALTHY"}` : `{"status":"UNHEALTHY"}`,
		{
			status: backendOK ? 200 : 500,
			headers: {
				'content-type': 'application/json'
			}
		}
	);
};

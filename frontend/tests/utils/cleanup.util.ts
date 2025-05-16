import axios from 'axios';
import playwrightConfig from '../../playwright.config';

export async function cleanupBackend() {
	await axios.post(playwrightConfig.use!.baseURL + '/api/test/reset');
}

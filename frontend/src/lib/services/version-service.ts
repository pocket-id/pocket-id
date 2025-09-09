import { version as currentVersion } from '$app/environment';
import axios from 'axios';

async function getNewestVersion() {
	const response = await axios
		.get('/api/version/latest', {
			timeout: 2000
		})
		.then((res) => res.data);

	return response.latestVersion;
}

function getCurrentVersion() {
	return currentVersion;
}

export default {
	getNewestVersion,
	getCurrentVersion,
};

import fs from 'fs';

async function globalSetup() {
	await fs.promises.mkdir('.tmp', { recursive: true });
}

export default globalSetup;

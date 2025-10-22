import fs from 'fs';

async function globalTeardown() {
	await fs.promises.rm('.tmp', { recursive: true, force: true });
}

export default globalTeardown;

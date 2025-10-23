import fs from 'fs';
import { pathFromRoot } from 'utils/fs.util';

async function globalTeardown() {
	const tmpPath = pathFromRoot('.tmp');
	await fs.promises.rm(tmpPath, { recursive: true, force: true });
}

export default globalTeardown;

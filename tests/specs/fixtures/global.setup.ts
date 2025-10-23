import fs from 'fs';
import { pathFromRoot } from 'utils/fs.util';

async function globalSetup() {
	const tmpPath = pathFromRoot('.tmp');
	await fs.promises.mkdir(tmpPath, { recursive: true });
}

export default globalSetup;

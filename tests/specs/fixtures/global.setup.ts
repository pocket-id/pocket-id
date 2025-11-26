import fs from 'fs';
import { tmpDir } from 'utils/fs.util';

async function globalSetup() {
	await fs.promises.mkdir(tmpDir, { recursive: true });
}

export default globalSetup;

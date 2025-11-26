import fs from 'fs';
import { tmpDir } from 'utils/fs.util';

async function globalTeardown() {
	await fs.promises.rm(tmpDir, { recursive: true, force: true });
}

export default globalTeardown;

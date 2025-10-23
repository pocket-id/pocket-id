import path from 'path';

export const tmpDir = path.join(process.env.RUNNER_TEMP, 'playwright') || pathFromRoot('.tmp');

export function pathFromRoot(p: string): string {
	return path.resolve(path.dirname(new URL(import.meta.url).pathname), '..', p);
}

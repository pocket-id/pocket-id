import path from 'path';

// CI runs several shards side by side on one runner, so each needs its own scratch files
export const tmpDir = pathFromRoot(
	process.env.E2E_SHARD ? `.tmp/shard-${process.env.E2E_SHARD}` : '.tmp'
);
export const authFile = path.join(tmpDir, 'auth', 'user.json');

export function pathFromRoot(p: string): string {
	return path.resolve(path.dirname(new URL(import.meta.url).pathname), '..', p);
}

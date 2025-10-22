import path from 'path';

export function pathFromRoot(p: string): string {
	return path.resolve(path.dirname(new URL(import.meta.url).pathname), '..', p);
}

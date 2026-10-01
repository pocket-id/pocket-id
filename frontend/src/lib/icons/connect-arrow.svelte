<script module lang="ts">
	type Point = [number, number];

	// Two pen passes over the same loop, slightly offset so the arrow reads as hand-drawn
	// Each pass is a chain of cubic curves: a start point followed by three points per curve
	const STROKES: Point[][] = [
		[
			[-1.1, 0.2],
			[27.4, 0.1],
			[60.2, 2.2],
			[77.8, -4.4],
			[93.1, -9.9],
			[94.2, -28.5],
			[81.1, -31.8],
			[65.7, -35.1],
			[57, -16.4],
			[67.9, -3.3],
			[76.7, 7.7],
			[98.6, 5.5],
			[120.5, -1.1],
			[135.8, -5.5],
			[149, -3.3],
			[159, -1.9]
		],
		[
			[0.5, -0.6],
			[28.8, -0.9],
			[61.6, 1.1],
			[79.1, -5.6],
			[94.6, -11.4],
			[95.2, -29.3],
			[81.9, -32.6],
			[66.4, -35.6],
			[56.1, -17.6],
			[66.6, -4.2],
			[75.6, 8.6],
			[99.4, 6.6],
			[121.2, -0.3],
			[136.6, -4.4],
			[150.1, -2.6],
			[159.9, -2.4]
		]
	];

	// Both arms of the head are drawn twice as well and meet at the end of the second pass
	const HEAD =
		'M137.15 3.58 C144.96 0.13, 151.62 -1.33, 159.9 -2.4 M137.15 3.58 C143.85 1.87, 149.96 -0.23, 159.9 -2.4 M138.63 -12.45 C146.01 -10.77, 152.19 -7.08, 159.9 -2.4 M138.63 -12.45 C144.94 -9.33, 150.61 -6.62, 159.9 -2.4';
	const TIP = STROKES[1][STROKES[1].length - 1];

	const lerp = (a: Point, b: Point, t: number): Point => [
		a[0] + (b[0] - a[0]) * t,
		a[1] + (b[1] - a[1]) * t
	];

	// Pull a pass towards the straight line between its ends, with both ends squeezed towards the middle
	function straighten(points: Point[], taut: number, squeeze: number) {
		const first = points[0];
		const last = points[points.length - 1];
		const middle = lerp(first, last, 0.5);
		const start = lerp(first, middle, squeeze);
		const end = lerp(last, middle, squeeze);
		return points.map((point, i) => lerp(point, lerp(start, end, i / (points.length - 1)), taut));
	}

	function toPath([start, ...curves]: Point[]) {
		let d = `M${start}`;
		for (let i = 0; i < curves.length; i += 3) {
			d += ` C${curves[i]} ${curves[i + 1]} ${curves[i + 2]}`;
		}
		return d;
	}
</script>

<script lang="ts">
	import { mode } from 'mode-watcher';

	let {
		class: className,
		taut = 0,
		squeeze = 0
	}: {
		class?: string;
		// How far the arrow has been pulled straight, from 0 for the drawn loop to 1 for a straight line
		taut?: number;
		// How far both ends have moved towards the middle, from 0 at rest to 1 where they meet
		squeeze?: number;
	} = $props();

	const strokes = $derived(STROKES.map((points) => straighten(points, taut, squeeze)));
	const tip = $derived(strokes[1][strokes[1].length - 1]);
</script>

<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 181.05 72.62" class={className}>
	<g
		transform="translate(11.145 36.748)"
		stroke={mode.current == 'dark' ? 'white' : 'black'}
		stroke-width="2"
		stroke-linecap="round"
		fill="none"
	>
		{#each strokes as points, i (i)}
			<path d={toPath(points)}></path>
		{/each}
		<path d={HEAD} transform="translate({tip[0] - TIP[0]} {tip[1] - TIP[1]})"></path>
	</g>
</svg>

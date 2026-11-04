# @pocket-id/ui

The Svelte components, design tokens and fonts of [Pocket ID](https://pocket-id.org), built on [bits-ui](https://bits-ui.com) and Tailwind CSS.

## Installation

```sh
pnpm add @pocket-id/ui bits-ui
```

It needs Svelte 5 and Tailwind CSS 4. Install `svelte-sonner` too if you use the toaster.

The components import [Solar](https://icon-sets.iconify.design/solar/) icons through `unplugin-icons`, so add it with the icon set and register its Vite plugin:

```sh
pnpm add -D unplugin-icons @iconify-json/solar
```

```ts
// vite.config.ts
import Icons from 'unplugin-icons/vite';

export default defineConfig({
	plugins: [sveltekit(), Icons({ compiler: 'svelte' })]
});
```

Import the styles in your Tailwind entry stylesheet:

```css
@import 'tailwindcss';
@import '@pocket-id/ui/styles.css';
```

`styles.css` includes the tokens, fonts and base styles, and registers the components as a Tailwind source.
For the tokens and fonts without the components, import `@pocket-id/ui/theme.css` instead.

## Usage

```svelte
<script lang="ts">
	import { Button } from '@pocket-id/ui/button';
	import * as Dialog from '@pocket-id/ui/dialog';
	import { cn } from '@pocket-id/ui/utils';
</script>

<Button variant="outline">Sign in</Button>
```

Dark mode follows a `dark` class on an ancestor, usually `<html>`.

### Labels

A few components render text of their own, such as the screen reader label of a close button.
It is English by default, and you can translate it once in your root layout:

```ts
import { setUiLabels } from '@pocket-id/ui/labels';

setUiLabels({ close: () => 'Schliessen', optional: () => '(optional)' });
```

## Fonts

The bundled fonts are DM Sans, Young Serif and JetBrains Mono, licensed under the SIL Open Font License (see `dist/fonts`).

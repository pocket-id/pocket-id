<script lang="ts" module>
  import { cn, type WithElementRef } from "$lib/utils/style.js";
  import type { HTMLButtonAttributes } from "svelte/elements";
  import {
    buttonVariants,
    type ButtonVariant,
    type ButtonSize,
  } from "$lib/components/ui/button/button.svelte";

  export type DropdownButtonTriggerProps =
    WithElementRef<HTMLButtonAttributes> & {
      variant?: ButtonVariant;
      size?: ButtonSize;
      builders?: any[];
    };
</script>

<script lang="ts">
  import ChevronDown from "@lucide/svelte/icons/chevron-down";

  let {
    class: className,
    variant = "default",
    size = "default",
    ref = $bindable(null),
    type = "button",
    disabled,
    builders = [],
    children,
    ...restProps
  }: DropdownButtonTriggerProps = $props();
</script>

<button
  bind:this={ref}
  use:builders[0]
  data-slot="dropdown-button-trigger"
  class={cn(
    buttonVariants({ variant, size }),
    "rounded-l-none px-2 border-l border-l-background/20",
    className
  )}
  {type}
  {disabled}
  {...restProps}
>
  {#if children}
    {@render children()}
  {:else}
    <ChevronDown class="size-4" />
  {/if}
</button>

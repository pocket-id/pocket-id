<script lang="ts" generics="T extends string">
	import * as Select from '#lib/components/ui/select/index.ts';

	let {
		value,
		options,
		onValueChange,
		...triggerProps
	}: {
		value: T;
		options: { value: T; label: string; description?: string }[];
		onValueChange: (value: T) => void;
		id?: string;
		'aria-describedby'?: string;
		'aria-invalid'?: boolean;
	} = $props();
</script>

<Select.Root
	type="single"
	{value}
	allowDeselect={false}
	onValueChange={(v) => onValueChange(v as T)}
>
	<Select.Trigger {...triggerProps} size="sm" class="min-w-40">
		{options.find((option) => option.value === value)?.label ?? value}
	</Select.Trigger>
	<Select.Content align="end" class="max-w-80">
		{#each options as option (option.value)}
			<Select.Item value={option.value} label={option.label}>
				<span class="flex flex-col items-start gap-0.5 whitespace-normal">
					<span>{option.label}</span>
					{#if option.description}
						<span class="text-caption text-fg-muted">{option.description}</span>
					{/if}
				</span>
			</Select.Item>
		{/each}
	</Select.Content>
</Select.Root>

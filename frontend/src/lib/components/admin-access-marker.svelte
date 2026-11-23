<script lang="ts">
	import { m } from '#lib/paraglide/messages.js';
	import TriangleAlertIcon from '~icons/solar/danger-triangle-bold-duotone';
</script>

<span
	class="text-warning-fg inline-flex shrink-0 align-[-2px]"
	title={m.can_lead_to_full_admin_access()}
>
	<TriangleAlertIcon class="size-3.5" aria-hidden="true" />
	<span class="sr-only">{m.can_lead_to_full_admin_access()}</span>
</span>

<script lang="ts">
	import { base } from '$app/paths';
	import { formatNumber } from '$lib/utils/format';

	type Variant = 'default' | 'success' | 'warning' | 'error' | 'primary';

	interface Props {
		label: string;
		value: number;
		variant?: Variant;
		icon?: string;
		href?: string;
	}

	let { label, value, variant = 'default', icon, href }: Props = $props();

	const fullHref = href ? base + href : undefined;

	const variantClasses: Record<Variant, string> = {
		default: 'text-surface-900 dark:text-surface-100',
		success: 'text-success-500',
		warning: 'text-warning-500',
		error: 'text-error-500',
		primary: 'text-primary-500'
	};
</script>

{#if fullHref}
<a href={fullHref} class="card preset-outlined-surface-200-800 bg-surface-50-950 p-4 block hover:ring-2 hover:ring-primary-500/50 transition-all cursor-pointer">
	<div class="flex items-start justify-between">
		<div>
			<p class="text-sm text-surface-500">{label}</p>
			<p class="text-3xl font-bold {variantClasses[variant]}">{formatNumber(value)}</p>
		</div>
		{#if icon}
			<div class="p-2 rounded-lg bg-surface-100 dark:bg-surface-700">
				{@html icon}
			</div>
		{/if}
	</div>
</a>
{:else}
<div class="card preset-outlined-surface-200-800 bg-surface-50-950 p-4">
	<div class="flex items-start justify-between">
		<div>
			<p class="text-sm text-surface-500">{label}</p>
			<p class="text-3xl font-bold {variantClasses[variant]}">{formatNumber(value)}</p>
		</div>
		{#if icon}
			<div class="p-2 rounded-lg bg-surface-100 dark:bg-surface-700">
				{@html icon}
			</div>
		{/if}
	</div>
</div>
{/if}

<script lang="ts">
	import type { VaultSummary } from '$lib/api/types';

	interface Props {
		vault: VaultSummary;
		onClick?: () => void;
		onDelete?: () => void;
	}

	let { vault, onClick, onDelete }: Props = $props();
</script>

<div
	class="card preset-outlined-surface-200-800 bg-surface-50-950 p-4 cursor-pointer hover:shadow-lg transition-shadow"
	onclick={onClick}
	onkeydown={(e) => e.key === 'Enter' && onClick?.()}
	role="button"
	tabindex="0"
>
	<div class="flex items-start justify-between mb-3">
		<div class="flex items-center gap-2">
			<svg class="w-5 h-5 text-primary-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"/>
			</svg>
			<h3 class="font-semibold text-lg">{vault.name}</h3>
		</div>
		<button
			type="button"
			class="btn btn-sm preset-outlined-error-500 opacity-0 group-hover:opacity-100 transition-opacity"
			onclick={(e) => {
				e.stopPropagation();
				onDelete?.();
			}}
			title="Delete vault"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
			</svg>
		</button>
	</div>

	{#if vault.description}
		<p class="text-sm text-surface-500 mb-3 line-clamp-2">{vault.description}</p>
	{/if}

	<div class="flex items-center justify-between">
		<div class="flex items-center gap-2">
			<span class="badge preset-filled-primary-500">
				{vault.entry_count} {vault.entry_count === 1 ? 'entry' : 'entries'}
			</span>
		</div>
		<svg class="w-4 h-4 text-surface-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
		</svg>
	</div>
</div>

<style>
	.card:hover button {
		opacity: 1;
	}
</style>

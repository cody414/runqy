<script lang="ts">
	import { Switch } from '@skeletonlabs/skeleton-svelte';
	import type { VaultEntryView } from '$lib/api/types';

	interface Props {
		open?: boolean;
		loading?: boolean;
		mode?: 'create' | 'edit';
		entry?: VaultEntryView | null;
		onsave?: (key: string, value: string, isSecret: boolean) => void;
		oncancel?: () => void;
	}

	let {
		open = $bindable(false),
		loading = false,
		mode = 'create',
		entry = null,
		onsave,
		oncancel
	}: Props = $props();

	let key = $state('');
	let value = $state('');
	let isSecret = $state(true);
	let error = $state('');

	function handleSubmit() {
		error = '';
		if (!key.trim()) {
			error = 'Key is required';
			return;
		}
		if (!value.trim()) {
			error = 'Value is required';
			return;
		}
		if (!/^[a-zA-Z0-9_-]+$/.test(key.trim())) {
			error = 'Key can only contain letters, numbers, hyphens, and underscores';
			return;
		}
		onsave?.(key.trim(), value, isSecret);
	}

	function handleCancel() {
		key = '';
		value = '';
		isSecret = true;
		error = '';
		oncancel?.();
		open = false;
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			handleCancel();
		}
	}

	$effect(() => {
		if (open && entry && mode === 'edit') {
			key = entry.key;
			value = '';
			isSecret = entry.is_secret;
		} else if (!open) {
			key = '';
			value = '';
			isSecret = true;
			error = '';
		}
	});
</script>

{#if open}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div
		class="fixed inset-0 z-50 bg-surface-950/50 flex justify-center items-center p-4"
		onclick={handleBackdropClick}
	>
		<div class="card preset-outlined-surface-200-800 bg-surface-100-900 w-full max-w-md p-6 shadow-xl">
			<h2 class="h4 mb-4">
				{mode === 'create' ? 'Add Entry' : 'Edit Entry'}
			</h2>

			<form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }}>
				<div class="space-y-4">
					<label class="label">
						<span class="label-text">Key</span>
						<input
							type="text"
							bind:value={key}
							placeholder="API_KEY"
							class="input"
							disabled={loading || mode === 'edit'}
						/>
					</label>

					<label class="label">
						<span class="label-text">Value</span>
						<textarea
							bind:value={value}
							placeholder={mode === 'edit' ? 'Enter new value...' : 'Enter value...'}
							class="textarea font-mono text-sm"
							rows={4}
							disabled={loading}
						></textarea>
						{#if mode === 'edit'}
							<span class="label-text text-surface-500">Enter a new value to update this entry</span>
						{/if}
					</label>

					<div class="flex items-center gap-3">
						<Switch bind:checked={isSecret} disabled={loading} />
						<span class="text-sm">Secret value</span>
						<span class="text-xs text-surface-500">
							{isSecret ? 'Value will be masked in UI' : 'Value will be visible in UI'}
						</span>
					</div>

					{#if error}
						<aside class="alert preset-filled-error-500">
							<p>{error}</p>
						</aside>
					{/if}
				</div>

				<footer class="flex justify-end gap-2 mt-6">
					<button type="button" class="btn preset-tonal" onclick={handleCancel} disabled={loading}>
						Cancel
					</button>
					<button type="submit" class="btn preset-filled" disabled={loading}>
						{#if loading}
							Saving...
						{:else}
							{mode === 'create' ? 'Add Entry' : 'Update Entry'}
						{/if}
					</button>
				</footer>
			</form>
		</div>
	</div>
{/if}

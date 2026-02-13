<script lang="ts">
	interface Props {
		open?: boolean;
		loading?: boolean;
		oncreate?: (name: string, description: string) => void;
		oncancel?: () => void;
	}

	let {
		open = $bindable(false),
		loading = false,
		oncreate,
		oncancel
	}: Props = $props();

	let name = $state('');
	let description = $state('');
	let error = $state('');

	function handleSubmit() {
		error = '';
		if (!name.trim()) {
			error = 'Vault name is required';
			return;
		}
		if (!/^[a-zA-Z0-9_-]+$/.test(name.trim())) {
			error = 'Vault name can only contain letters, numbers, hyphens, and underscores';
			return;
		}
		oncreate?.(name.trim(), description.trim());
	}

	function handleCancel() {
		name = '';
		description = '';
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
		if (!open) {
			name = '';
			description = '';
			error = '';
		}
	});
</script>

{#if open}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div
		class="fixed inset-0 z-50 bg-surface-950/70 backdrop-blur-sm flex justify-center items-center p-4"
		onclick={handleBackdropClick}
	>
		<div class="card preset-outlined-surface-200-800 bg-surface-100-900 ring-1 ring-surface-300 dark:ring-surface-600 w-full max-w-md p-6 shadow-xl">
			<h2 class="h4 mb-4">Create Vault</h2>

			<form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }}>
				<div class="space-y-4">
					<label class="label">
						<span class="label-text">Name</span>
						<input
							type="text"
							bind:value={name}
							placeholder="my-vault"
							class="input"
							disabled={loading}
						/>
					</label>

					<label class="label">
						<span class="label-text">Description <span class="text-surface-500">(optional)</span></span>
						<textarea
							bind:value={description}
							placeholder="Store secrets for production environment..."
							class="textarea"
							rows={3}
							disabled={loading}
						></textarea>
					</label>

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
							Creating...
						{:else}
							Create Vault
						{/if}
					</button>
				</footer>
			</form>
		</div>
	</div>
{/if}

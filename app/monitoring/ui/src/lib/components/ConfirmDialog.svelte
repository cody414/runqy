<script lang="ts">
	interface Props {
		open?: boolean;
		title?: string;
		message?: string;
		confirmText?: string;
		cancelText?: string;
		variant?: 'default' | 'danger';
		onconfirm?: () => void;
		oncancel?: () => void;
	}

	let {
		open = $bindable(false),
		title = 'Confirm',
		message = 'Are you sure?',
		confirmText = 'Confirm',
		cancelText = 'Cancel',
		variant = 'default',
		onconfirm,
		oncancel
	}: Props = $props();

	function handleConfirm() {
		onconfirm?.();
		open = false;
	}

	function handleCancel() {
		oncancel?.();
		open = false;
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			handleCancel();
		}
	}
</script>

{#if open}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div
		class="fixed inset-0 z-50 bg-surface-950/70 backdrop-blur-sm flex justify-center items-center p-4"
		onclick={handleBackdropClick}
	>
		<div class="card preset-outlined-surface-200-800 bg-surface-100-900 ring-1 ring-surface-300 dark:ring-surface-600 w-full max-w-md p-6 space-y-4 shadow-xl">
			<h2 class="h4">{title}</h2>
			<p class="text-surface-600-400">{message}</p>
			<footer class="flex justify-end gap-2 pt-2">
				<button type="button" class="btn preset-tonal" onclick={handleCancel}>
					{cancelText}
				</button>
				<button
					type="button"
					class="btn {variant === 'danger' ? 'preset-filled-error-500' : 'preset-filled'}"
					onclick={handleConfirm}
				>
					{confirmText}
				</button>
			</footer>
		</div>
	</div>
{/if}

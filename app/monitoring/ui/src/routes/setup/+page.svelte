<script lang="ts">
	import { goto } from '$app/navigation';
	import { base } from '$app/paths';
	import { authStore } from '$lib/stores/auth';

	let email = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let localError = $state<string | null>(null);

	function validatePassword(): boolean {
		if (password.length < 8) {
			localError = 'Password must be at least 8 characters';
			return false;
		}
		if (password !== confirmPassword) {
			localError = 'Passwords do not match';
			return false;
		}
		localError = null;
		return true;
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();

		if (!validatePassword()) {
			return;
		}

		const success = await authStore.setup(email, password, confirmPassword);
		if (success) {
			goto(base || '/');
		}
	}
</script>

<svelte:head>
	<title>Setup Admin - runqy Monitor</title>
</svelte:head>

<div class="min-h-screen flex items-center justify-center bg-surface-100 dark:bg-surface-900 p-4">
	<div class="card bg-surface-50-950 p-8 w-full max-w-md shadow-xl">
		<div class="text-center mb-8">
			<div class="flex justify-center mb-4">
				<svg viewBox="0 0 100 100" class="w-16 h-16">
					<line x1="32" y1="50" x2="72" y2="22" stroke="#64748B" stroke-width="5" stroke-linecap="round" />
					<line x1="32" y1="50" x2="78" y2="50" stroke="#64748B" stroke-width="5" stroke-linecap="round" />
					<line x1="32" y1="50" x2="72" y2="78" stroke="#64748B" stroke-width="5" stroke-linecap="round" />
					<circle cx="32" cy="50" r="16" fill="#3B82F6" />
					<circle cx="72" cy="22" r="11" fill="#E2E8F0" />
					<circle cx="78" cy="50" r="11" fill="#E2E8F0" />
					<circle cx="72" cy="78" r="11" fill="#E2E8F0" />
				</svg>
			</div>
			<h1 class="text-2xl font-bold">Welcome to runqy</h1>
			<p class="text-surface-500 mt-2">Create your admin account to get started</p>
		</div>

		{#if localError || $authStore.error}
			<div class="mb-4 p-3 bg-error-100 dark:bg-error-900 text-error-700 dark:text-error-300 rounded-lg text-sm">
				{localError || $authStore.error}
			</div>
		{/if}

		<form onsubmit={handleSubmit} class="space-y-4">
			<div>
				<label class="label" for="email">
					<span class="label-text">Email</span>
				</label>
				<input
					id="email"
					type="email"
					class="input"
					placeholder="admin@example.com"
					bind:value={email}
					required
					disabled={$authStore.loading}
				/>
			</div>

			<div>
				<label class="label" for="password">
					<span class="label-text">Password</span>
				</label>
				<input
					id="password"
					type="password"
					class="input"
					placeholder="Minimum 8 characters"
					bind:value={password}
					required
					minlength="8"
					disabled={$authStore.loading}
				/>
			</div>

			<div>
				<label class="label" for="confirm-password">
					<span class="label-text">Confirm Password</span>
				</label>
				<input
					id="confirm-password"
					type="password"
					class="input"
					placeholder="Confirm your password"
					bind:value={confirmPassword}
					required
					minlength="8"
					disabled={$authStore.loading}
				/>
			</div>

			<button
				type="submit"
				class="btn preset-filled w-full"
				disabled={$authStore.loading}
			>
				{#if $authStore.loading}
					<span class="animate-spin mr-2">
						<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					</span>
					Creating account...
				{:else}
					Create Admin Account
				{/if}
			</button>
		</form>

		<p class="mt-6 text-center text-xs text-surface-500">
			This will be the only admin account for this runqy instance.
		</p>
	</div>
</div>

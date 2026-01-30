<script lang="ts">
	import { settings, type Theme, type ViewDensity } from '$lib/stores/settings';

	const themes: { value: Theme; label: string; description: string }[] = [
		{ value: 'system', label: 'System', description: 'Follow system preference' },
		{ value: 'light', label: 'Light', description: 'Light mode' },
		{ value: 'dark', label: 'Dark', description: 'Dark mode' }
	];

	const densities: { value: ViewDensity; label: string; description: string }[] = [
		{ value: 'comfortable', label: 'Comfortable', description: 'More spacing between elements' },
		{ value: 'compact', label: 'Compact', description: 'Reduced spacing for more data' }
	];

	const pollIntervals = [
		{ value: 2, label: '2 seconds' },
		{ value: 5, label: '5 seconds' },
		{ value: 10, label: '10 seconds' },
		{ value: 30, label: '30 seconds' },
		{ value: 60, label: '1 minute' }
	];

	function handlePollIntervalChange(e: Event) {
		settings.setPollInterval(parseInt((e.target as HTMLSelectElement).value));
	}
</script>

<svelte:head>
	<title>Settings - runqy Monitor</title>
</svelte:head>

<div class="p-6 max-w-2xl">
	<h1 class="text-2xl font-bold mb-6">Settings</h1>

	<div class="space-y-8">
		<!-- Theme -->
		<div class="card p-6">
			<h2 class="text-lg font-semibold mb-4">Appearance</h2>

			<div class="space-y-4">
				<div>
					<label class="text-sm font-medium text-surface-600 dark:text-surface-400">Theme</label>
					<div class="mt-2 grid grid-cols-3 gap-3">
						{#each themes as theme (theme.value)}
							<button
								type="button"
								class="card p-4 text-center transition-all {$settings.theme === theme.value
									? 'ring-2 ring-primary-500'
									: 'hover:bg-surface-100 dark:hover:bg-surface-700'}"
								onclick={() => settings.setTheme(theme.value)}
							>
								<div class="mb-2">
									{#if theme.value === 'system'}
										<svg class="w-8 h-8 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
											/>
										</svg>
									{:else if theme.value === 'light'}
										<svg class="w-8 h-8 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"
											/>
										</svg>
									{:else}
										<svg class="w-8 h-8 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"
											/>
										</svg>
									{/if}
								</div>
								<div class="font-medium">{theme.label}</div>
								<div class="text-xs text-surface-500">{theme.description}</div>
							</button>
						{/each}
					</div>
				</div>

				<div>
					<label class="text-sm font-medium text-surface-600 dark:text-surface-400">View Density</label>
					<div class="mt-2 grid grid-cols-2 gap-3">
						{#each densities as density (density.value)}
							<button
								type="button"
								class="card p-4 text-left transition-all {$settings.viewDensity === density.value
									? 'ring-2 ring-primary-500'
									: 'hover:bg-surface-100 dark:hover:bg-surface-700'}"
								onclick={() => settings.setViewDensity(density.value)}
							>
								<div class="font-medium">{density.label}</div>
								<div class="text-xs text-surface-500">{density.description}</div>
							</button>
						{/each}
					</div>
				</div>
			</div>
		</div>

		<!-- Polling -->
		<div class="card p-6">
			<h2 class="text-lg font-semibold mb-4">Data Refresh</h2>

			<div>
				<label class="text-sm font-medium text-surface-600 dark:text-surface-400" for="poll-interval">
					Auto-refresh interval
				</label>
				<p class="text-xs text-surface-500 mb-2">How often to refresh data from the server</p>
				<select
					id="poll-interval"
					class="select max-w-xs"
					value={$settings.pollInterval}
					onchange={handlePollIntervalChange}
				>
					{#each pollIntervals as interval (interval.value)}
						<option value={interval.value}>{interval.label}</option>
					{/each}
				</select>
			</div>
		</div>

		<!-- Sidebar -->
		<div class="card p-6">
			<h2 class="text-lg font-semibold mb-4">Navigation</h2>

			<div class="flex items-center justify-between">
				<div>
					<div class="font-medium">Collapsed Sidebar</div>
					<div class="text-xs text-surface-500">Show only icons in the sidebar</div>
				</div>
				<label class="relative inline-flex items-center cursor-pointer">
					<input
						type="checkbox"
						class="sr-only peer"
						checked={$settings.sidebarCollapsed}
						onchange={() => settings.toggleSidebar()}
					/>
					<div
						class="w-11 h-6 bg-surface-300 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 dark:peer-focus:ring-primary-800 rounded-full peer dark:bg-surface-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-surface-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-surface-600 peer-checked:bg-primary-500"
					></div>
				</label>
			</div>
		</div>

		<!-- Reset -->
		<div class="card p-6">
			<h2 class="text-lg font-semibold mb-4">Reset</h2>
			<p class="text-sm text-surface-500 mb-4">Reset all settings to their default values</p>
			<button type="button" class="btn preset-outlined-error-500" onclick={() => settings.reset()}>
				Reset to Defaults
			</button>
		</div>
	</div>
</div>

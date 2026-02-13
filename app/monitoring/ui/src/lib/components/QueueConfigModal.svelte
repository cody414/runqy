<script lang="ts">
	import { SegmentedControl } from '@skeletonlabs/skeleton-svelte';
	import type { QueueConfigDetail, DeploymentConfig, VaultSummary } from '$lib/api/types';
	import { getVaults, getVault } from '$lib/api/client';

	interface SubQueue {
		name: string;
		priority: number;
	}

	interface QueueToCreate {
		name: string;
		priority: number;
		deployment: DeploymentConfig | null;
	}

	interface Props {
		open?: boolean;
		loading?: boolean;
		mode?: 'create' | 'edit';
		config?: QueueConfigDetail | null;
		existingQueues?: string[];
		onsave?: (queues: QueueToCreate[]) => void;
		onsavesubqueue?: (parentQueue: string, subqueueName: string, priority: number) => void;
		oncancel?: () => void;
	}

	let {
		open = $bindable(false),
		loading = false,
		mode = 'create',
		config = null,
		existingQueues = [],
		onsave,
		onsavesubqueue,
		oncancel
	}: Props = $props();

	let createType = $state<'queue' | 'subqueue'>('queue');
	let selectedParentQueue = $state('');
	let subqueueName = $state('');
	let subqueuePriority = $state(1);
	let queueName = $state('');
	let subQueues = $state<SubQueue[]>([]);
	let gitUrl = $state('');
	let branch = $state('main');
	let codePath = $state('');
	let startupCmd = $state('');
	let deploymentMode = $state('long_running');
	let startupTimeout = $state(60);
	let redisStorage = $state(false);
	let selectedVaults = $state<Set<string>>(new Set());
	let gitTokenVault = $state('');
	let gitTokenKey = $state('');
	let availableVaults = $state<VaultSummary[]>([]);
	let selectedVaultEntries = $state<string[]>([]);
	let loadingVaults = $state(false);
	let loadingEntries = $state(false);
	let error = $state('');

	let parentQueues = $derived(() => {
		const parents = new Set<string>();
		for (const q of existingQueues) {
			const dotIndex = q.lastIndexOf('.');
			if (dotIndex > 0) {
				parents.add(q.substring(0, dotIndex));
			} else {
				parents.add(q);
			}
		}
		return Array.from(parents).sort();
	});

	async function loadVaults() {
		loadingVaults = true;
		try {
			const response = await getVaults();
			availableVaults = response.vaults || [];
		} catch (e) {
			console.error('Failed to load vaults:', e);
			availableVaults = [];
		} finally {
			loadingVaults = false;
		}
	}

	async function loadVaultEntries(vaultName: string) {
		if (!vaultName) {
			selectedVaultEntries = [];
			return;
		}
		loadingEntries = true;
		try {
			const vault = await getVault(vaultName);
			selectedVaultEntries = vault.entries?.map(e => e.key) || [];
		} catch (e) {
			console.error('Failed to load vault entries:', e);
			selectedVaultEntries = [];
		} finally {
			loadingEntries = false;
		}
	}

	function toggleVault(vaultName: string) {
		const newSet = new Set(selectedVaults);
		if (newSet.has(vaultName)) {
			newSet.delete(vaultName);
		} else {
			newSet.add(vaultName);
		}
		selectedVaults = newSet;
	}

	function addSubQueue() {
		subQueues = [...subQueues, { name: '', priority: 1 }];
	}

	function removeSubQueue(index: number) {
		subQueues = subQueues.filter((_, i) => i !== index);
	}

	function handleSubmit() {
		error = '';

		if (mode === 'create' && createType === 'subqueue') {
			if (!selectedParentQueue) {
				error = 'Please select a parent queue';
				return;
			}
			if (!subqueueName.trim()) {
				error = 'Subqueue name is required';
				return;
			}
			if (!/^[a-zA-Z0-9_-]+$/.test(subqueueName.trim())) {
				error = 'Subqueue name can only contain letters, numbers, hyphens, and underscores';
				return;
			}
			if (subqueuePriority < 1) {
				error = 'Priority must be at least 1';
				return;
			}
			onsavesubqueue?.(selectedParentQueue, subqueueName.trim(), subqueuePriority);
			return;
		}

		if (!queueName.trim()) {
			error = 'Queue name is required';
			return;
		}
		if (!/^[a-zA-Z0-9_-]+$/.test(queueName.trim())) {
			error = 'Queue name can only contain letters, numbers, hyphens, and underscores';
			return;
		}

		for (const sq of subQueues) {
			if (!sq.name.trim()) {
				error = 'All subqueue names are required';
				return;
			}
			if (!/^[a-zA-Z0-9_-]+$/.test(sq.name.trim())) {
				error = 'Subqueue names can only contain letters, numbers, hyphens, and underscores';
				return;
			}
			if (sq.priority < 1) {
				error = 'All subqueue priorities must be at least 1';
				return;
			}
		}

		let deployment: DeploymentConfig | null = null;
		const hasDeploymentConfig = gitUrl.trim() || startupCmd.trim();

		if (mode === 'create' || hasDeploymentConfig) {
			if (mode === 'create' && !gitUrl.trim()) {
				error = 'Git URL is required';
				return;
			}
			if (mode === 'create' && !startupCmd.trim()) {
				error = 'Startup command is required';
				return;
			}

			if (gitUrl.trim() && startupCmd.trim()) {
				const vaultsArray = Array.from(selectedVaults);
				let gitTokenRef = '';
				if (gitTokenVault && gitTokenKey) {
					gitTokenRef = `vault://${gitTokenVault}/${gitTokenKey}`;
				}

				deployment = {
					git_url: gitUrl.trim(),
					branch: branch.trim() || 'main',
					code_path: codePath.trim() || undefined,
					startup_cmd: startupCmd.trim(),
					mode: deploymentMode,
					startup_timeout_secs: startupTimeout,
					redis_storage: redisStorage,
					vaults: vaultsArray.length > 0 ? vaultsArray : undefined,
					git_token: gitTokenRef || undefined
				};
			}
		}

		const queuesToCreate: QueueToCreate[] = [];

		if (subQueues.length === 0) {
			queuesToCreate.push({
				name: `${queueName.trim()}.default`,
				priority: 1,
				deployment
			});
		} else {
			for (const sq of subQueues) {
				queuesToCreate.push({
					name: `${queueName.trim()}.${sq.name.trim()}`,
					priority: sq.priority,
					deployment
				});
			}
		}

		onsave?.(queuesToCreate);
	}

	function handleCancel() {
		resetForm();
		oncancel?.();
		open = false;
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			handleCancel();
		}
	}

	function resetForm() {
		createType = 'queue';
		selectedParentQueue = '';
		subqueueName = '';
		subqueuePriority = 1;
		queueName = '';
		subQueues = [];
		gitUrl = '';
		branch = 'main';
		codePath = '';
		startupCmd = '';
		deploymentMode = 'long_running';
		startupTimeout = 60;
		redisStorage = false;
		selectedVaults = new Set();
		gitTokenVault = '';
		gitTokenKey = '';
		error = '';
	}

	$effect(() => {
		if (open) {
			loadVaults();
			if (config && mode === 'edit') {
				const dotIndex = config.name.lastIndexOf('.');
				if (dotIndex > 0) {
					queueName = config.name.substring(0, dotIndex);
					const subName = config.name.substring(dotIndex + 1);
					if (subName !== 'default') {
						subQueues = [{ name: subName, priority: config.priority }];
					}
				} else {
					queueName = config.name;
				}
				if (config.deployment) {
					gitUrl = config.deployment.git_url || '';
					branch = config.deployment.branch || 'main';
					codePath = config.deployment.code_path || '';
					startupCmd = config.deployment.startup_cmd || '';
					deploymentMode = config.deployment.mode || 'long_running';
					startupTimeout = config.deployment.startup_timeout_secs || 60;
					redisStorage = config.deployment.redis_storage || false;
					selectedVaults = new Set(config.deployment.vaults || []);
					if (config.deployment.git_token?.startsWith('vault://')) {
						const ref = config.deployment.git_token.substring(8);
						const parts = ref.split('/');
						if (parts.length >= 2) {
							gitTokenVault = parts[0];
							gitTokenKey = parts.slice(1).join('/');
							loadVaultEntries(gitTokenVault);
						}
					}
				}
			}
		} else {
			resetForm();
		}
	});

	$effect(() => {
		if (gitTokenVault) {
			loadVaultEntries(gitTokenVault);
		} else {
			selectedVaultEntries = [];
			gitTokenKey = '';
		}
	});
</script>

{#if open}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div
		class="fixed inset-0 z-50 bg-surface-950/70 backdrop-blur-sm flex justify-center items-center p-4 overflow-y-auto"
		onclick={handleBackdropClick}
	>
		<div class="card preset-outlined-surface-200-800 bg-surface-100-900 ring-1 ring-surface-300 dark:ring-surface-600 w-full max-w-2xl p-6 shadow-xl max-h-[90vh] overflow-y-auto">
			<h2 class="h4 mb-4">
				{mode === 'create' ? 'Create Queue' : 'Edit Queue Configuration'}
			</h2>

			<form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }}>
				<div class="space-y-4">
					{#if mode === 'create'}
						<SegmentedControl
							name="createType"
							value={createType}
							onValueChange={(e) => createType = e.value as 'queue' | 'subqueue'}
						>
							<SegmentedControl.Item value="queue">New Queue</SegmentedControl.Item>
							<SegmentedControl.Item value="subqueue" disabled={parentQueues().length === 0}>Add Subqueue</SegmentedControl.Item>
						</SegmentedControl>
					{/if}

					{#if mode === 'create' && createType === 'subqueue'}
						<div class="space-y-4">
							<label class="label">
								<span class="label-text">Parent Queue</span>
								<select bind:value={selectedParentQueue} class="select" disabled={loading}>
									<option value="">Select a queue...</option>
									{#each parentQueues() as parent}
										<option value={parent}>{parent}</option>
									{/each}
								</select>
							</label>

							<div class="grid grid-cols-2 gap-4">
								<label class="label">
									<span class="label-text">Subqueue Name</span>
									<div class="input-group grid-cols-[auto_1fr]">
										<span class="ig-cell">{selectedParentQueue || '...'}.</span>
										<input type="text" bind:value={subqueueName} placeholder="high" class="ig-input" disabled={loading} />
									</div>
								</label>
								<label class="label">
									<span class="label-text">Priority</span>
									<input type="number" bind:value={subqueuePriority} min="1" class="input" disabled={loading} />
								</label>
							</div>
							<p class="text-sm text-surface-500">
								Higher priority subqueues are processed first.
							</p>
						</div>
					{:else}
						<div class="space-y-4">
							<div class="flex items-end gap-4">
								<label class="label flex-1">
									<span class="label-text">Queue Name</span>
									<input type="text" bind:value={queueName} placeholder="inference" class="input" disabled={loading || mode === 'edit'} />
								</label>
								<button type="button" class="btn preset-tonal" onclick={addSubQueue} disabled={loading}>
									+ Add Subqueue
								</button>
							</div>

							{#if subQueues.length === 0}
								<p class="text-sm text-surface-500 py-2">
									No subqueues defined. A <code>.default</code> subqueue will be created automatically.
								</p>
							{:else}
								<div class="card preset-outlined p-3 space-y-2">
									{#each subQueues as sq, index}
										<div class="flex items-center gap-2">
											<span class="text-surface-500 text-sm">{queueName || '...'}.</span>
											<input type="text" bind:value={sq.name} placeholder="high" class="input flex-1" disabled={loading} />
											<span class="text-xs text-surface-500">Priority:</span>
											<input type="number" bind:value={sq.priority} min="1" class="input w-16" disabled={loading} />
											<button type="button" class="btn-icon preset-tonal-error" onclick={() => removeSubQueue(index)} disabled={loading}>
												<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
												</svg>
											</button>
										</div>
									{/each}
								</div>
							{/if}

							<hr class="hr" />

							<div>
								<h5 class="h5 mb-1">Deployment Configuration</h5>
								<p class="text-sm text-surface-500 mb-4">Configure Git-based code deployment for workers</p>
							</div>

							<div class="space-y-4">
								<label class="label">
									<span class="label-text">Git URL</span>
									<input type="text" bind:value={gitUrl} placeholder="https://github.com/org/repo.git" class="input font-mono text-sm" disabled={loading} />
								</label>

								<div class="grid grid-cols-2 gap-4">
									<label class="label">
										<span class="label-text">Branch</span>
										<input type="text" bind:value={branch} placeholder="main" class="input" disabled={loading} />
									</label>
									<label class="label">
										<span class="label-text">Code Path <span class="text-surface-500">(optional)</span></span>
										<input type="text" bind:value={codePath} placeholder="./src" class="input" disabled={loading} />
									</label>
								</div>

								<label class="label">
									<span class="label-text">Startup Command</span>
									<input type="text" bind:value={startupCmd} placeholder="python main.py" class="input font-mono text-sm" disabled={loading} />
								</label>

								<div class="grid grid-cols-2 gap-4">
									<label class="label">
										<span class="label-text">Mode</span>
										<select bind:value={deploymentMode} class="select" disabled={loading}>
											<option value="long_running">Long Running</option>
											<option value="one_shot">One Shot</option>
										</select>
									</label>
									<label class="label">
										<span class="label-text">Startup Timeout (sec)</span>
										<input type="number" bind:value={startupTimeout} min="1" class="input" disabled={loading} />
									</label>
								</div>

								<label class="flex items-center gap-3 cursor-pointer">
									<input type="checkbox" bind:checked={redisStorage} class="checkbox" disabled={loading} />
									<span class="text-sm">Redis Storage</span>
								</label>

								<div>
									<span class="label-text">Vaults</span>
									{#if loadingVaults}
										<p class="text-sm text-surface-500">Loading vaults...</p>
									{:else if availableVaults.length === 0}
										<p class="text-sm text-surface-500">No vaults available</p>
									{:else}
										<div class="card preset-outlined p-2 max-h-32 overflow-y-auto grid grid-cols-2 gap-2 mt-1">
											{#each availableVaults as vault}
												<label class="flex items-center gap-2 cursor-pointer p-1 rounded hover:preset-tonal">
													<input type="checkbox" checked={selectedVaults.has(vault.name)} onchange={() => toggleVault(vault.name)} class="checkbox" disabled={loading} />
													<span class="text-sm">{vault.name}</span>
													<span class="text-xs text-surface-400">({vault.entry_count})</span>
												</label>
											{/each}
										</div>
									{/if}
								</div>

								<div>
									<span class="label-text">Git Token (from vault)</span>
									<div class="grid grid-cols-2 gap-2 mt-1">
										<select bind:value={gitTokenVault} class="select" disabled={loading || availableVaults.length === 0}>
											<option value="">Select vault...</option>
											{#each availableVaults as vault}
												<option value={vault.name}>{vault.name}</option>
											{/each}
										</select>
										<select bind:value={gitTokenKey} class="select" disabled={loading || !gitTokenVault || selectedVaultEntries.length === 0}>
											<option value="">Select key...</option>
											{#each selectedVaultEntries as key}
												<option value={key}>{key}</option>
											{/each}
										</select>
									</div>
									{#if gitTokenVault && gitTokenKey}
										<p class="text-xs text-surface-500 mt-1">
											Reference: <code>vault://{gitTokenVault}/{gitTokenKey}</code>
										</p>
									{/if}
								</div>
							</div>
						</div>
					{/if}

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
						{:else if mode === 'create' && createType === 'subqueue'}
							Add Subqueue
						{:else if mode === 'create'}
							Create Queue
						{:else}
							Update Queue
						{/if}
					</button>
				</footer>
			</form>
		</div>
	</div>
{/if}

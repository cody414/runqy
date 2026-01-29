<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import type { Task, TaskState } from '$lib/api/types';
	import {
		getQueueInfo,
		getTasks,
		pauseQueue,
		resumeQueue,
		deleteTask,
		cancelTask,
		runTask,
		archiveTask,
		deleteAllTasks,
		archiveAllTasks,
		runAllRetryTasks,
		runAllArchivedTasks,
		batchDeleteTasks,
		batchArchiveTasks,
		batchRunTasks
	} from '$lib/api/client';
	import { settings } from '$lib/stores/settings';
	import { formatNumber, formatBytes, formatDuration } from '$lib/utils/format';
	import TaskTable from '$lib/components/TaskTable.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';

	$: qname = decodeURIComponent($page.params.qname);

	type TabState = 'active' | 'pending' | 'retry' | 'archived' | 'completed';
	const tabs: { label: string; state: TabState }[] = [
		{ label: 'Active', state: 'active' },
		{ label: 'Pending', state: 'pending' },
		{ label: 'Retry', state: 'retry' },
		{ label: 'Archived', state: 'archived' },
		{ label: 'Completed', state: 'completed' }
	];

	let activeTab: TabState = 'active';
	let queueInfo: Awaited<ReturnType<typeof getQueueInfo>> | null = null;
	let tasks: Task[] = [];
	let taskCounts: Record<TabState, number> = {
		active: 0,
		pending: 0,
		retry: 0,
		archived: 0,
		completed: 0
	};
	let loading = false;
	let error: string | null = null;
	let selectedIds = new Set<string>();
	let pollInterval: ReturnType<typeof setInterval> | null = null;

	let confirmDialog = {
		open: false,
		title: '',
		message: '',
		action: () => {}
	};

	async function loadQueueInfo() {
		try {
			queueInfo = await getQueueInfo(qname);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load queue info';
		}
	}

	async function loadTasks() {
		loading = true;
		error = null;
		try {
			const response = await getTasks(qname, activeTab, 1, 100);
			tasks = response.tasks || [];
			taskCounts[activeTab] = response.stats?.total || tasks.length;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load tasks';
			tasks = [];
		} finally {
			loading = false;
		}
	}

	async function loadAllCounts() {
		// Load counts for all tabs in parallel
		const results = await Promise.allSettled(
			tabs.map(async (tab) => {
				const response = await getTasks(qname, tab.state, 1, 1);
				return { state: tab.state, count: response.stats?.total || 0 };
			})
		);

		results.forEach((result) => {
			if (result.status === 'fulfilled') {
				taskCounts[result.value.state] = result.value.count;
			}
		});
		taskCounts = taskCounts; // Trigger reactivity
	}

	async function loadData() {
		await Promise.all([loadQueueInfo(), loadTasks()]);
	}

	onMount(() => {
		loadData();
		loadAllCounts();
		pollInterval = setInterval(loadData, $settings.pollInterval * 1000);
	});

	onDestroy(() => {
		if (pollInterval) clearInterval(pollInterval);
	});

	// Reload tasks when tab changes
	$: if (activeTab) {
		loadTasks();
		selectedIds = new Set();
	}

	async function handlePause() {
		try {
			await pauseQueue(qname);
			await loadQueueInfo();
		} catch (e) {
			console.error('Failed to pause queue:', e);
		}
	}

	async function handleResume() {
		try {
			await resumeQueue(qname);
			await loadQueueInfo();
		} catch (e) {
			console.error('Failed to resume queue:', e);
		}
	}

	function handleSelect(e: CustomEvent<{ id: string; selected: boolean }>) {
		if (e.detail.selected) {
			selectedIds.add(e.detail.id);
		} else {
			selectedIds.delete(e.detail.id);
		}
		selectedIds = selectedIds; // Trigger reactivity
	}

	function handleSelectAll(e: CustomEvent<{ selected: boolean }>) {
		if (e.detail.selected) {
			selectedIds = new Set(tasks.map((t) => t.id));
		} else {
			selectedIds = new Set();
		}
	}

	async function handleTaskAction(e: CustomEvent<{ action: string; taskId: string }>) {
		const { action, taskId } = e.detail;
		try {
			switch (action) {
				case 'cancel':
					await cancelTask(qname, taskId);
					break;
				case 'run':
					await runTask(qname, taskId);
					break;
				case 'archive':
					await archiveTask(qname, taskId, activeTab);
					break;
				case 'delete':
					await deleteTask(qname, taskId, activeTab);
					break;
			}
			await loadTasks();
			await loadAllCounts();
		} catch (e) {
			console.error(`Failed to ${action} task:`, e);
		}
	}

	// Bulk actions
	async function handleBulkDelete() {
		if (selectedIds.size === 0) return;
		confirmDialog = {
			open: true,
			title: 'Delete Tasks',
			message: `Are you sure you want to delete ${selectedIds.size} task(s)?`,
			action: async () => {
				try {
					await batchDeleteTasks(qname, Array.from(selectedIds), activeTab);
					selectedIds = new Set();
					await loadTasks();
					await loadAllCounts();
				} catch (e) {
					console.error('Failed to delete tasks:', e);
				}
			}
		};
	}

	async function handleBulkArchive() {
		if (selectedIds.size === 0) return;
		try {
			await batchArchiveTasks(qname, Array.from(selectedIds), activeTab);
			selectedIds = new Set();
			await loadTasks();
			await loadAllCounts();
		} catch (e) {
			console.error('Failed to archive tasks:', e);
		}
	}

	async function handleBulkRun() {
		if (selectedIds.size === 0) return;
		try {
			await batchRunTasks(qname, Array.from(selectedIds));
			selectedIds = new Set();
			await loadTasks();
			await loadAllCounts();
		} catch (e) {
			console.error('Failed to run tasks:', e);
		}
	}

	// Bulk all actions
	function handleDeleteAll() {
		confirmDialog = {
			open: true,
			title: 'Delete All Tasks',
			message: `Are you sure you want to delete all ${taskCounts[activeTab]} ${activeTab} task(s)?`,
			action: async () => {
				try {
					await deleteAllTasks(qname, activeTab);
					await loadTasks();
					await loadAllCounts();
				} catch (e) {
					console.error('Failed to delete all tasks:', e);
				}
			}
		};
	}

	async function handleArchiveAll() {
		try {
			await archiveAllTasks(qname, activeTab);
			await loadTasks();
			await loadAllCounts();
		} catch (e) {
			console.error('Failed to archive all tasks:', e);
		}
	}

	async function handleRunAll() {
		try {
			if (activeTab === 'retry') {
				await runAllRetryTasks(qname);
			} else if (activeTab === 'archived') {
				await runAllArchivedTasks(qname);
			}
			await loadTasks();
			await loadAllCounts();
		} catch (e) {
			console.error('Failed to run all tasks:', e);
		}
	}

	$: canBulkRun = activeTab === 'retry' || activeTab === 'archived';
	$: canBulkArchive = activeTab === 'pending' || activeTab === 'retry';
	$: canBulkDelete = activeTab !== 'active';
</script>

<svelte:head>
	<title>{qname} - Queues - runqy</title>
</svelte:head>

<div class="p-6 space-y-6">
	<!-- Breadcrumb -->
	<nav class="flex items-center gap-2 text-sm">
		<a href="/queues" class="text-surface-500 hover:text-primary-500">Queues</a>
		<span class="text-surface-400">/</span>
		<span class="font-medium">{qname}</span>
	</nav>

	<!-- Header -->
	<div class="flex items-start justify-between">
		<div>
			<div class="flex items-center gap-3">
				<h1 class="text-2xl font-bold">{qname}</h1>
				{#if queueInfo}
					{#if queueInfo.paused}
						<span class="badge variant-filled-warning">Paused</span>
					{:else}
						<span class="badge variant-filled-success">Running</span>
					{/if}
				{/if}
			</div>
			{#if queueInfo}
				<div class="mt-2 flex items-center gap-4 text-sm text-surface-500">
					<span>Latency: {queueInfo.display_latency}</span>
					<span>Memory: {formatBytes(queueInfo.memory_usage_bytes)}</span>
					{#if queueInfo.groups > 0}
						<span>Groups: {queueInfo.groups}</span>
					{/if}
				</div>
			{/if}
		</div>

		<div class="flex items-center gap-2">
			{#if queueInfo?.paused}
				<button type="button" class="btn variant-filled-success" on:click={handleResume}>
					<svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"
						/>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
					Resume
				</button>
			{:else}
				<button type="button" class="btn variant-filled-warning" on:click={handlePause}>
					<svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z"
						/>
					</svg>
					Pause
				</button>
			{/if}
			<button type="button" class="btn variant-ghost-surface" on:click={loadData}>
				<svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
					/>
				</svg>
				Refresh
			</button>
		</div>
	</div>

	<!-- Tabs -->
	<div class="flex items-center gap-1 border-b border-surface-300 dark:border-surface-600">
		{#each tabs as tab}
			<button
				type="button"
				class="px-4 py-2 font-medium text-sm border-b-2 transition-colors {activeTab === tab.state
					? 'border-primary-500 text-primary-500'
					: 'border-transparent text-surface-500 hover:text-surface-900 dark:hover:text-surface-100'}"
				on:click={() => (activeTab = tab.state)}
			>
				{tab.label}
				<span
					class="ml-2 px-2 py-0.5 rounded-full text-xs {activeTab === tab.state
						? 'bg-primary-500 text-white'
						: 'bg-surface-200 dark:bg-surface-700'}"
				>
					{formatNumber(taskCounts[tab.state])}
				</span>
			</button>
		{/each}
	</div>

	<!-- Bulk Actions -->
	{#if selectedIds.size > 0 || taskCounts[activeTab] > 0}
		<div class="flex items-center gap-3 p-3 rounded-lg bg-surface-100 dark:bg-surface-800">
			{#if selectedIds.size > 0}
				<span class="text-sm text-surface-500">{selectedIds.size} selected</span>
				<div class="h-4 w-px bg-surface-300 dark:bg-surface-600"></div>
			{/if}

			{#if canBulkRun}
				<button
					type="button"
					class="btn btn-sm variant-ghost-success"
					on:click={selectedIds.size > 0 ? handleBulkRun : handleRunAll}
				>
					{selectedIds.size > 0 ? 'Run Selected' : 'Run All'}
				</button>
			{/if}

			{#if canBulkArchive}
				<button
					type="button"
					class="btn btn-sm variant-ghost-surface"
					on:click={selectedIds.size > 0 ? handleBulkArchive : handleArchiveAll}
				>
					{selectedIds.size > 0 ? 'Archive Selected' : 'Archive All'}
				</button>
			{/if}

			{#if canBulkDelete}
				<button
					type="button"
					class="btn btn-sm variant-ghost-error"
					on:click={selectedIds.size > 0 ? handleBulkDelete : handleDeleteAll}
				>
					{selectedIds.size > 0 ? 'Delete Selected' : 'Delete All'}
				</button>
			{/if}
		</div>
	{/if}

	<!-- Error State -->
	{#if error}
		<div class="card p-4 variant-ghost-error">
			<p class="text-error-500">{error}</p>
		</div>
	{/if}

	<!-- Task Table -->
	<TaskTable
		{tasks}
		state={activeTab}
		{loading}
		{selectedIds}
		on:select={handleSelect}
		on:selectAll={handleSelectAll}
		on:action={handleTaskAction}
	/>
</div>

<ConfirmDialog
	bind:open={confirmDialog.open}
	title={confirmDialog.title}
	message={confirmDialog.message}
	variant="danger"
	confirmText="Delete"
	on:confirm={confirmDialog.action}
/>

<script lang="ts">
	import type { Worker } from '$lib/api/types';
	import { formatRelativeTime, truncateId } from '$lib/utils/format';
	import { workersStore } from '$lib/stores/workers';

	let { worker }: { worker: Worker } = $props();

	// Parse queues string like "map[queue1:5 queue2:5]" to array of {name, priority}
	function parseQueues(queuesStr: string): { name: string; priority: number }[] {
		const match = queuesStr.match(/map\[(.*)\]/);
		if (!match) return [];
		return match[1].split(' ').map(q => {
			const [name, priorityStr] = q.split(':');
			return { name, priority: parseInt(priorityStr, 10) || 0 };
		}).filter(q => q.name);
	}

	// Check if worker is processing (has active tasks on a server with matching id)
	let isProcessing = $derived(
		$workersStore.servers.some(s =>
			s.id === worker.worker_id && s.active_workers && s.active_workers.length > 0
		)
	);

	// Get worker status: 'processing' | 'idle' | 'stale' | 'bootstrapping' | 'stopped'
	let status = $derived.by(() => {
		if (worker.is_stale) return 'stale';
		if (worker.status === 'bootstrapping') return 'bootstrapping';
		if (worker.status === 'stopped') return 'stopped';
		if (worker.status === 'running' && isProcessing) return 'processing';
		if (worker.status === 'running') return 'idle';
		return 'stopped'; // fallback for unknown statuses
	});

	let queues = $derived(parseQueues(worker.queues));
	let startedDate = $derived(new Date(worker.started_at * 1000));
	let lastBeatDate = $derived(new Date(worker.last_beat * 1000));
</script>

<div class="card preset-outlined-surface-200-800 bg-surface-50-950 p-4">
	<div class="flex items-start justify-between mb-4">
		<div>
			<h3 class="font-semibold font-mono text-sm" title={worker.worker_id}>
				{truncateId(worker.worker_id, 20)}
			</h3>
			<p class="text-xs text-surface-500 mt-1">
				Concurrency: {worker.concurrency}
			</p>
		</div>
		<div>
			{#if status === 'processing'}
				<span class="badge preset-filled-primary-500 text-xs">Processing</span>
			{:else if status === 'idle'}
				<span class="badge preset-filled-success-500 text-xs">Idle</span>
			{:else if status === 'stale'}
				<span class="badge preset-filled-warning-500 text-xs">Stale</span>
			{:else if status === 'bootstrapping'}
				<span class="badge preset-filled-secondary-500 text-xs">Bootstrapping</span>
			{:else}
				<span class="badge preset-filled-surface-500 text-xs">Stopped</span>
			{/if}
		</div>
	</div>

	<div class="mb-4">
		<div class="text-xs text-surface-500 mb-1">Queues</div>
		<div class="flex flex-wrap gap-1">
			{#each queues as queue (queue.name)}
				<span class="badge preset-outlined-primary-500 text-xs">
					{queue.name}
					{#if queue.priority === 0 && status === 'bootstrapping'}
						<svg class="w-3 h-3 ml-1 animate-spin inline-block" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{:else}
						<span class="text-surface-400 ml-1">p:{queue.priority}</span>
					{/if}
				</span>
			{/each}
		</div>
	</div>

	<div class="border-t border-surface-300 dark:border-surface-600 pt-4 space-y-2 text-xs text-surface-500">
		<div class="flex justify-between">
			<span>Started:</span>
			<span>{formatRelativeTime(startedDate)}</span>
		</div>
		<div class="flex justify-between">
			<span>Last heartbeat:</span>
			<span>{formatRelativeTime(lastBeatDate)}</span>
		</div>
	</div>
</div>

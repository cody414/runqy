import { writable, derived } from 'svelte/store';
import type { Queue } from '$lib/api/types';
import { getQueues } from '$lib/api/client';

interface QueuesState {
	queues: Queue[];
	loading: boolean;
	error: string | null;
	lastUpdated: Date | null;
}

function createQueuesStore() {
	const { subscribe, set, update } = writable<QueuesState>({
		queues: [],
		loading: false,
		error: null,
		lastUpdated: null
	});

	return {
		subscribe,
		fetch: async () => {
			update((s) => ({ ...s, loading: true, error: null }));
			try {
				const response = await getQueues();
				update((s) => ({
					...s,
					queues: response.queues || [],
					loading: false,
					lastUpdated: new Date()
				}));
			} catch (e) {
				update((s) => ({
					...s,
					loading: false,
					error: e instanceof Error ? e.message : 'Failed to fetch queues'
				}));
			}
		},
		setQueues: (queues: Queue[]) => {
			update((s) => ({ ...s, queues, lastUpdated: new Date() }));
		}
	};
}

export const queuesStore = createQueuesStore();

// Derived stores for common calculations
export const totalStats = derived(queuesStore, ($state) => {
	const queues = $state.queues;
	return {
		totalQueues: queues.length,
		totalPending: queues.reduce((sum, q) => sum + (q.pending || 0), 0),
		totalActive: queues.reduce((sum, q) => sum + (q.active || 0), 0),
		totalProcessed: queues.reduce((sum, q) => sum + (q.processed || 0), 0),
		totalFailed: queues.reduce((sum, q) => sum + (q.failed || 0), 0),
		totalRetry: queues.reduce((sum, q) => sum + (q.retry || 0), 0),
		totalArchived: queues.reduce((sum, q) => sum + (q.archived || 0), 0),
		pausedQueues: queues.filter((q) => q.paused).length
	};
});

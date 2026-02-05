import { writable, derived } from 'svelte/store';
import type { Worker, ServerInfo } from '$lib/api/types';
import { getWorkers, getServers } from '$lib/api/client';

interface WorkersState {
	workers: Worker[];
	servers: ServerInfo[];
	loading: boolean;
	error: string | null;
	lastUpdated: Date | null;
}

function createWorkersStore() {
	const { subscribe, set, update } = writable<WorkersState>({
		workers: [],
		servers: [],
		loading: false,
		error: null,
		lastUpdated: null
	});

	return {
		subscribe,
		fetch: async () => {
			update((s) => ({ ...s, loading: true, error: null }));
			try {
				const [workersResponse, serversResponse] = await Promise.all([
					getWorkers(),
					getServers()
				]);
				update((s) => ({
					...s,
					workers: workersResponse.workers || [],
					servers: serversResponse.servers || [],
					loading: false,
					lastUpdated: new Date()
				}));
			} catch (e) {
				update((s) => ({
					...s,
					loading: false,
					error: e instanceof Error ? e.message : 'Failed to fetch workers'
				}));
			}
		}
	};
}

export const workersStore = createWorkersStore();

// Derived stores
export const workerStats = derived(workersStore, ($state) => {
	const workers = $state.workers;
	const servers = $state.servers;

	// Count active workers (currently processing tasks) from servers
	const activeWorkerCount = servers.reduce(
		(sum, server) => sum + (server.active_workers?.length || 0),
		0
	);

	// Ready workers = running and not stale
	const readyWorkers = workers.filter((w) => w.status === 'running' && !w.is_stale).length;

	// Processing = min of active tasks and ready workers (can't have more processing than ready)
	const processing = Math.min(activeWorkerCount, readyWorkers);

	// Idle = ready workers that are not processing
	const idle = Math.max(0, readyWorkers - processing);

	const stale = workers.filter((w) => w.is_stale).length;
	const bootstrapping = workers.filter((w) => w.status === 'bootstrapping' && !w.is_stale).length;
	const stopped = workers.filter((w) => w.status === 'stopped' && !w.is_stale).length;

	return {
		total: workers.length,
		processing,
		idle,
		stale,
		bootstrapping,
		stopped,
		totalCapacity: workers.reduce((sum, w) => sum + (w.concurrency || 0), 0)
	};
});

import { writable, derived } from 'svelte/store';
import type { DailyStats } from '$lib/api/types';
import { getQueueStats } from '$lib/api/client';
import { queuesStore, totalStats } from './queues';

// Time range options for the chart
export const TIME_RANGES = {
	'today': 0,
	'7d': 7,
	'30d': 30
} as const;

export type TimeRangeKey = keyof typeof TIME_RANGES;

interface QueueStatsState {
	// Raw stats from API: { queueName: DailyStats[] }
	rawStats: { [queueName: string]: DailyStats[] };
	loading: boolean;
	error: string | null;
	lastUpdated: Date | null;
	selectedTimeRange: TimeRangeKey;
	selectedQueue: string | null; // null = all queues
}

function createQueueStatsStore() {
	const { subscribe, set, update } = writable<QueueStatsState>({
		rawStats: {},
		loading: false,
		error: null,
		lastUpdated: null,
		selectedTimeRange: '7d',
		selectedQueue: null
	});

	return {
		subscribe,
		fetch: async () => {
			update((s) => ({ ...s, loading: true, error: null }));
			try {
				const response = await getQueueStats();
				update((s) => ({
					...s,
					rawStats: response.stats || {},
					loading: false,
					lastUpdated: new Date()
				}));
			} catch (e) {
				update((s) => ({
					...s,
					loading: false,
					error: e instanceof Error ? e.message : 'Failed to fetch queue stats'
				}));
			}
		},
		setTimeRange: (range: TimeRangeKey) => {
			update((s) => ({ ...s, selectedTimeRange: range }));
		},
		setQueue: (queue: string | null) => {
			update((s) => ({ ...s, selectedQueue: queue }));
		},
		reset: () => {
			set({
				rawStats: {},
				loading: false,
				error: null,
				lastUpdated: null,
				selectedTimeRange: '7d',
				selectedQueue: null
			});
		}
	};
}

export const queueStatsStore = createQueueStatsStore();

// Data point for throughput chart
export interface ThroughputDataPoint {
	date: string; // "YYYY-MM-DD"
	processed: number;
	failed: number;
}

// Derived store for daily throughput data (aggregated across all queues or filtered)
export const dailyThroughputData = derived(
	queueStatsStore,
	($stats): ThroughputDataPoint[] => {
		const { rawStats, selectedTimeRange, selectedQueue } = $stats;

		if (Object.keys(rawStats).length === 0) return [];

		// Get the number of days to include
		const days = TIME_RANGES[selectedTimeRange];
		if (days === 0) return []; // "today" is handled separately

		// Aggregate stats across queues
		const dateMap = new Map<string, { processed: number; failed: number }>();

		const queuesToInclude = selectedQueue
			? [selectedQueue]
			: Object.keys(rawStats);

		for (const queueName of queuesToInclude) {
			const queueStats = rawStats[queueName];
			if (!queueStats) continue;

			for (const stat of queueStats) {
				const existing = dateMap.get(stat.date) || { processed: 0, failed: 0 };
				dateMap.set(stat.date, {
					processed: existing.processed + stat.processed,
					failed: existing.failed + stat.failed
				});
			}
		}

		// Convert to array and sort by date
		const dataPoints: ThroughputDataPoint[] = [];
		for (const [date, stats] of dateMap) {
			dataPoints.push({
				date,
				processed: stats.processed,
				failed: stats.failed
			});
		}

		dataPoints.sort((a, b) => a.date.localeCompare(b.date));

		// Filter to requested number of days
		if (days > 0 && dataPoints.length > days) {
			return dataPoints.slice(-days);
		}

		return dataPoints;
	}
);

// Today's stats from the queues store (live data)
export interface TodayStats {
	processed: number;
	failed: number;
	succeeded: number;
}

export const todayStats = derived(totalStats, ($total): TodayStats => {
	return {
		processed: $total.totalProcessed,
		failed: $total.totalFailed,
		succeeded: $total.totalProcessed - $total.totalFailed
	};
});

// List of available queues for filtering
export const availableQueues = derived(queueStatsStore, ($stats): string[] => {
	return Object.keys($stats.rawStats).sort();
});

// Legacy exports for backwards compatibility (metrics store that was Prometheus-based)
// These are kept for any existing code that might reference them
export const metricsStore = queueStatsStore;
export const throughputData = dailyThroughputData;

<script lang="ts">
	import { Chart, Area, Axis, Svg, Tooltip, Highlight } from 'layerchart';
	import { scaleTime, scaleLinear } from 'd3-scale';
	import { stack, stackOrderNone, stackOffsetNone, curveMonotoneX } from 'd3-shape';
	import {
		queueStatsStore,
		dailyThroughputData,
		todayStats,
		availableQueues,
		type TimeRangeKey
	} from '$lib/stores/metrics';
	import { queuesStore } from '$lib/stores/queues';
	import { onMount, onDestroy } from 'svelte';
	import { settings } from '$lib/stores/settings';

	interface Props {
		height?: number;
	}

	let { height = 300 }: Props = $props();

	let pollInterval: ReturnType<typeof setInterval> | null = null;

	// Time range buttons
	const timeRanges: { key: TimeRangeKey; label: string }[] = [
		{ key: 'today', label: 'Today' },
		{ key: '7d', label: '7D' },
		{ key: '30d', label: '30D' }
	];

	function selectTimeRange(range: TimeRangeKey) {
		queueStatsStore.setTimeRange(range);
	}

	function selectQueue(queue: string | null) {
		queueStatsStore.setQueue(queue);
	}

	// Prepare data for the line/area chart (7D/30D views)
	let areaChartData = $derived.by(() => {
		const data = $dailyThroughputData;
		if (data.length === 0)
			return { series: [], xDomain: [new Date(), new Date()], yDomain: [0, 1], data: [] };

		// Convert date strings to Date objects
		const withDates = data.map((d) => ({
			...d,
			timestamp: new Date(d.date + 'T12:00:00') // Noon to avoid timezone issues
		}));

		// Stack generator
		const stackGen = stack<(typeof withDates)[0]>()
			.keys(['processed', 'failed'])
			.order(stackOrderNone)
			.offset(stackOffsetNone);

		const series = stackGen(withDates);

		// Calculate domains
		const xDomain = [withDates[0].timestamp, withDates[withDates.length - 1].timestamp];
		const maxY = Math.max(...series.flatMap((s) => s.flatMap((d) => d[1])));
		const yDomain = [0, Math.max(maxY, 10)]; // Ensure at least some range

		return { series, xDomain, yDomain, data: withDates };
	});

	// Prepare data for the bar chart (Today view)
	let barChartData = $derived.by(() => {
		const stats = $todayStats;
		return [
			{ label: 'Processed', value: stats.processed, color: 'rgb(34, 197, 94)' },
			{ label: 'Failed', value: stats.failed, color: 'rgb(239, 68, 68)' }
		];
	});

	let barMaxValue = $derived(Math.max(1, ...barChartData.map((d) => d.value)));

	// Format functions
	function formatDate(date: Date): string {
		return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
	}

	function formatNumber(val: number): string {
		if (val >= 1000000) return (val / 1000000).toFixed(1) + 'M';
		if (val >= 1000) return (val / 1000).toFixed(1) + 'K';
		return val.toFixed(0);
	}

	function formatLargeNumber(val: number): string {
		return val.toLocaleString();
	}

	onMount(() => {
		queueStatsStore.fetch();
		queuesStore.fetch();
		pollInterval = setInterval(() => {
			queueStatsStore.fetch();
			queuesStore.fetch();
		}, $settings.pollInterval * 1000);
	});

	onDestroy(() => {
		if (pollInterval) clearInterval(pollInterval);
	});
</script>

<div class="rq-card p-5">
	<div class="flex items-center justify-between mb-4 flex-wrap gap-2">
		<h3 class="text-lg font-semibold">Task History</h3>
		<div class="flex items-center gap-3">
			<!-- Queue filter dropdown -->
			{#if $availableQueues.length > 1}
				<select
					class="select select-sm w-32"
					value={$queueStatsStore.selectedQueue || ''}
					onchange={(e) => selectQueue(e.currentTarget.value || null)}
				>
					<option value="">All Queues</option>
					{#each $availableQueues as queue}
						<option value={queue}>{queue}</option>
					{/each}
				</select>
			{/if}
			<!-- Time range buttons -->
			<div class="flex gap-1">
				{#each timeRanges as { key, label }}
					<button
						type="button"
						class="{$queueStatsStore.selectedTimeRange === key
							? 'rq-pill-active'
							: 'rq-pill'}"
						onclick={() => selectTimeRange(key)}
					>
						{label}
					</button>
				{/each}
			</div>
		</div>
	</div>

	{#if $queueStatsStore.loading && Object.keys($queueStatsStore.rawStats).length === 0}
		<div class="flex items-center justify-center" style="height: {height}px">
			<div class="animate-pulse text-surface-500">Loading stats...</div>
		</div>
	{:else if $queueStatsStore.error}
		<div class="flex items-center justify-center" style="height: {height}px">
			<p class="text-error-500">{$queueStatsStore.error}</p>
		</div>
	{:else if $queueStatsStore.selectedTimeRange === 'today'}
		<!-- Today View - Bar Chart -->
		<div style="height: {height}px" class="flex items-center justify-center">
			<div class="flex gap-16 items-end">
				{#each barChartData as item}
					<div class="flex flex-col items-center">
						<div
							class="w-24 rounded-t transition-all duration-300"
							style="background-color: {item.color}; height: {Math.max(
								20,
								(item.value / Math.max(barMaxValue, 1)) * (height - 100)
							)}px;"
						></div>
						<div class="mt-3 text-center">
							<div class="text-2xl font-bold">{formatLargeNumber(item.value)}</div>
							<div class="text-sm text-surface-500">{item.label}</div>
						</div>
					</div>
				{/each}
			</div>
		</div>
	{:else if $dailyThroughputData.length === 0}
		<div class="flex items-center justify-center" style="height: {height}px">
			<p class="text-surface-500">No historical data available</p>
		</div>
	{:else}
		<!-- 7D/30D View - Area Chart -->
		<div style="height: {height}px">
			<Chart
				data={areaChartData.data}
				x={(d: { timestamp: Date }) => d.timestamp}
				xScale={scaleTime()}
				xDomain={areaChartData.xDomain}
				yScale={scaleLinear()}
				yDomain={areaChartData.yDomain}
				yNice
				padding={{ top: 20, right: 20, bottom: 40, left: 50 }}
				tooltip={{ mode: 'bisect-x' }}
			>
				<Svg>
					<!-- Grid lines -->
					<g class="grid-lines">
						{#each [0.25, 0.5, 0.75, 1] as ratio}
							<line
								x1="0"
								x2="100%"
								y1="{(1 - ratio) * 100}%"
								y2="{(1 - ratio) * 100}%"
								stroke="currentColor"
								stroke-opacity="0.1"
							/>
						{/each}
					</g>

					<!-- Stacked areas -->
					{#each areaChartData.series as s, i}
						{@const colors = ['rgb(34, 197, 94)', 'rgb(239, 68, 68)']}
						{@const fillOpacity = [0.6, 0.6]}
						<Area
							data={s}
							x={(d) => d.data.timestamp}
							y0={(d) => d[0]}
							y1={(d) => d[1]}
							fill={colors[i]}
							fillOpacity={fillOpacity[i]}
							curve={curveMonotoneX}
							tweened
						/>
					{/each}

					<!-- Axes -->
					<Axis placement="bottom" format={formatDate} ticks={6} classes={{ tickLabel: '!fill-[#9ca3af] !stroke-[#111118]' }} />
					<Axis placement="left" format={formatNumber} ticks={5} classes={{ tickLabel: '!fill-[#9ca3af] !stroke-[#111118]' }} />

					<!-- Highlight on hover -->
					<Highlight lines />
				</Svg>

				<Tooltip.Root let:data>
					{#if data}
						<div class="bg-surface-800 text-white p-2 rounded shadow-lg text-sm">
							<div class="font-medium mb-1">{data.date}</div>
							<div class="flex justify-between gap-4">
								<span class="text-success-400">Processed:</span>
								<span>{formatLargeNumber(data.processed)}</span>
							</div>
							<div class="flex justify-between gap-4">
								<span class="text-error-400">Failed:</span>
								<span>{formatLargeNumber(data.failed)}</span>
							</div>
						</div>
					{/if}
				</Tooltip.Root>
			</Chart>
		</div>
	{/if}

	<!-- Legend -->
	<div class="flex items-center justify-center gap-6 mt-3">
		<div class="flex items-center gap-2">
			<div class="w-3 h-3 rounded" style="background-color: rgb(34, 197, 94);"></div>
			<span class="text-sm text-surface-600 dark:text-surface-400">Processed</span>
		</div>
		<div class="flex items-center gap-2">
			<div class="w-3 h-3 rounded" style="background-color: rgb(239, 68, 68);"></div>
			<span class="text-sm text-surface-600 dark:text-surface-400">Failed</span>
		</div>
	</div>
</div>

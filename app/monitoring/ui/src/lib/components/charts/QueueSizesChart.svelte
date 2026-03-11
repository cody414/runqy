<script lang="ts">
	import { Chart, Bars, Axis, Svg, Tooltip } from 'layerchart';
	import { scaleBand, scaleLinear } from 'd3-scale';
	import { queuesStore } from '$lib/stores/queues';
	import { goto } from '$app/navigation';
	import { base } from '$app/paths';

	interface Props {
		height?: number;
		maxQueues?: number;
	}

	let { height = 250, maxQueues = 8 }: Props = $props();

	// Colors for each state
	const stateColors = {
		pending: 'rgb(234, 179, 8)',   // yellow-500
		active: 'rgb(59, 130, 246)',   // blue-500
		retry: 'rgb(249, 115, 22)'    // orange-500
	};

	const stateLabels = {
		pending: 'Pending',
		active: 'Active',
		retry: 'Retry'
	};

	interface ChartDataPoint {
		queue: string;
		state: string;
		value: number;
	}

	interface ChartData {
		flattened: ChartDataPoint[];
		queues: string[];
	}

	// Transform queue data into chart format
	let chartData = $derived.by((): ChartData => {
		const queues = $queuesStore.queues;
		if (queues.length === 0) return { flattened: [], queues: [] };

		// Sort by total tasks and take top N
		const sorted = [...queues]
			.map(q => ({
				queue: q.queue,
				pending: q.pending || 0,
				active: q.active || 0,
				retry: q.retry || 0,
				total: (q.pending || 0) + (q.active || 0) + (q.retry || 0)
			}))
			.sort((a, b) => b.total - a.total)
			.slice(0, maxQueues);

		// Flatten for grouped bar chart
		const flattened: ChartDataPoint[] = [];
		for (const q of sorted) {
			flattened.push({ queue: q.queue, state: 'pending', value: q.pending });
			flattened.push({ queue: q.queue, state: 'active', value: q.active });
			flattened.push({ queue: q.queue, state: 'retry', value: q.retry });
		}

		return { flattened, queues: sorted.map(q => q.queue) };
	});

	// Calculate max value for y domain
	let maxValue = $derived(
		Math.max(1, ...chartData.flattened.map(d => d.value))
	);

	function formatNumber(val: number): string {
		if (val >= 1000000) return (val / 1000000).toFixed(1) + 'M';
		if (val >= 1000) return (val / 1000).toFixed(1) + 'K';
		return val.toString();
	}

	function truncateQueueName(name: string): { line1: string; line2: string } {
		const dotIndex = name.indexOf('.');
		const parent = dotIndex >= 0 ? name.slice(0, dotIndex) : name;
		const sub = dotIndex >= 0 ? name.slice(dotIndex) : '';

		const parts = parent.split('-');
		const shortParent = parts.length >= 2
			? parts.map(p => p.slice(0, 2)).join('-')
			: parent.slice(0, 4);

		const shortSub = sub ? '.' + sub.slice(1, 5) : '';

		return { line1: shortParent, line2: shortSub };
	}

	function handleQueueClick(qname: string) {
		goto(`${base}/queues/${encodeURIComponent(qname)}`);
	}
</script>

<div class="rq-card p-5">
	<div class="flex items-center justify-between mb-4">
		<h3 class="text-lg font-semibold">Live Queue Sizes</h3>
		{#if $queuesStore.queues.length > maxQueues}
			<span class="text-sm text-surface-500">
				Top {maxQueues} of {$queuesStore.queues.length} queues
			</span>
		{/if}
	</div>

	{#if $queuesStore.loading && $queuesStore.queues.length === 0}
		<div class="flex items-center justify-center" style="height: {height}px">
			<div class="animate-pulse text-surface-500">Loading queues...</div>
		</div>
	{:else if $queuesStore.queues.length === 0}
		<div class="flex items-center justify-center" style="height: {height}px">
			<p class="text-surface-500">No queues available</p>
		</div>
	{:else if chartData.queues.length > 0}
		<div style="height: {height}px">
			<Chart
				data={chartData.flattened}
				x={(d: ChartDataPoint) => d.queue}
				xScale={scaleBand().padding(0.3)}
				xDomain={chartData.queues}
				y={(d: ChartDataPoint) => d.value}
				yScale={scaleLinear()}
				yDomain={[0, maxValue]}
				yNice
				padding={{ top: 20, right: 20, bottom: 54, left: 50 }}
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

					<!-- Grouped bars for each queue -->
					{#each chartData.queues as queue}
						{@const queueData = chartData.flattened.filter(d => d.queue === queue)}
						{@const states = ['pending', 'active', 'retry']}
						{#each states as state, i}
							{@const d = queueData.find(x => x.state === state)}
							{#if d && d.value > 0}
								<Bars
									data={[d]}
									groupBy="state"
									inset={2}
									radius={2}
									fill={stateColors[state as keyof typeof stateColors]}
									onclick={() => handleQueueClick(queue)}
									class="cursor-pointer hover:opacity-80 transition-opacity"
								/>
							{/if}
						{/each}
					{/each}

					<!-- Axes -->
					<Axis placement="bottom" classes={{ tickLabel: '!fill-[#9ca3af] !stroke-[#111118]' }}>
						<svelte:fragment slot="tickLabel" let:labelProps>
							{@const label = truncateQueueName(labelProps.value)}
							<text
								x={labelProps.x}
								y={labelProps.y}
								dy={4}
								text-anchor="middle"
								dominant-baseline="hanging"
							>
								<tspan x={labelProps.x} fill="#9ca3af" stroke="#111118" stroke-width="0.3" font-size="0.75rem">{label.line1}</tspan>
								{#if label.line2}
									<tspan x={labelProps.x} dy="1.1em" fill="#9ca3af" stroke="#111118" stroke-width="0.3" font-size="0.7rem">{label.line2}</tspan>
								{/if}
							</text>
						</svelte:fragment>
					</Axis>
					<Axis placement="left" format={formatNumber} ticks={5} classes={{ tickLabel: '!fill-[#9ca3af] !stroke-[#111118]' }} />
				</Svg>

				<Tooltip.Root let:data>
					{#if data}
						<div class="bg-surface-800 text-white p-2 rounded shadow-lg text-sm">
							<div class="font-medium mb-1">{data.queue}</div>
							<div class="flex justify-between gap-4">
								<span>{stateLabels[data.state as keyof typeof stateLabels]}:</span>
								<span>{formatNumber(data.value)}</span>
							</div>
						</div>
					{/if}
				</Tooltip.Root>
			</Chart>
		</div>

		<!-- Legend -->
		<div class="flex items-center justify-center gap-4 mt-3 flex-wrap">
			{#each Object.entries(stateColors) as [state, color]}
				<div class="flex items-center gap-2">
					<div class="w-3 h-3 rounded" style="background-color: {color};"></div>
					<span class="text-sm text-surface-600 dark:text-surface-400">
						{stateLabels[state as keyof typeof stateLabels]}
					</span>
				</div>
			{/each}
		</div>
	{/if}
</div>

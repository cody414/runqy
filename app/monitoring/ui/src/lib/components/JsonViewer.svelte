<script lang="ts">
	import { tryParseJson } from '$lib/utils/format';

	export let data: string | unknown;
	export let collapsed: boolean = true;
	export let maxHeight: string = '400px';

	let isCollapsed = collapsed;
	let copySuccess = false;

	$: parsed = typeof data === 'string' ? tryParseJson(data) : data;
	$: displayData = parsed !== null ? parsed : data;
	$: isValidJson = parsed !== null;
	$: formattedJson = isValidJson ? JSON.stringify(displayData, null, 2) : String(data);

	function toggleCollapse() {
		isCollapsed = !isCollapsed;
	}

	async function copyToClipboard() {
		try {
			await navigator.clipboard.writeText(formattedJson);
			copySuccess = true;
			setTimeout(() => (copySuccess = false), 2000);
		} catch (err) {
			console.error('Failed to copy:', err);
		}
	}

	function syntaxHighlight(json: string): string {
		return json.replace(
			/("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g,
			(match) => {
				let cls = 'json-number';
				if (/^"/.test(match)) {
					if (/:$/.test(match)) {
						cls = 'json-key';
					} else {
						cls = 'json-string';
					}
				} else if (/true|false/.test(match)) {
					cls = 'json-boolean';
				} else if (/null/.test(match)) {
					cls = 'json-null';
				}
				return `<span class="${cls}">${match}</span>`;
			}
		);
	}
</script>

<div class="json-viewer rounded-lg bg-surface-100 dark:bg-surface-800 border border-surface-300 dark:border-surface-600">
	<div class="flex items-center justify-between px-3 py-2 border-b border-surface-300 dark:border-surface-600">
		<button
			type="button"
			class="flex items-center gap-2 text-sm text-surface-600 dark:text-surface-400 hover:text-surface-900 dark:hover:text-surface-100"
			on:click={toggleCollapse}
		>
			<svg
				class="w-4 h-4 transition-transform {isCollapsed ? '' : 'rotate-90'}"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
			</svg>
			<span>{isValidJson ? 'JSON' : 'Raw'}</span>
			{#if isValidJson && typeof displayData === 'object' && displayData !== null}
				<span class="text-xs opacity-60">
					{Array.isArray(displayData)
						? `[${displayData.length} items]`
						: `{${Object.keys(displayData).length} keys}`}
				</span>
			{/if}
		</button>

		<button
			type="button"
			class="p-1.5 rounded hover:bg-surface-200 dark:hover:bg-surface-700 text-surface-600 dark:text-surface-400"
			on:click={copyToClipboard}
			title="Copy to clipboard"
		>
			{#if copySuccess}
				<svg class="w-4 h-4 text-success-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
				</svg>
			{:else}
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
					/>
				</svg>
			{/if}
		</button>
	</div>

	{#if !isCollapsed}
		<div class="overflow-auto scrollbar-thin" style="max-height: {maxHeight}">
			<pre
				class="p-3 text-sm font-mono whitespace-pre-wrap break-all">{#if isValidJson}{@html syntaxHighlight(formattedJson)}{:else}{formattedJson}{/if}</pre>
		</div>
	{/if}
</div>

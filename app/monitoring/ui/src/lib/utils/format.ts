/**
 * Format bytes to human-readable string
 */
export function formatBytes(bytes: number | undefined | null): string {
	if (bytes === undefined || bytes === null || isNaN(bytes)) return '0 B';
	if (bytes === 0) return '0 B';
	const k = 1024;
	const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	if (i < 0 || i >= sizes.length) return '0 B';
	return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

/**
 * Format milliseconds to human-readable duration
 */
export function formatDuration(ms: number): string {
	if (ms < 1000) return `${ms}ms`;
	if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
	if (ms < 3600000) return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
	return `${Math.floor(ms / 3600000)}h ${Math.floor((ms % 3600000) / 60000)}m`;
}

/**
 * Format a number with commas
 */
export function formatNumber(num: number): string {
	return num.toLocaleString();
}

/**
 * Format relative time (e.g., "2 minutes ago")
 */
export function formatRelativeTime(date: Date | string): string {
	const now = new Date();
	const then = typeof date === 'string' ? new Date(date) : date;
	const diffMs = now.getTime() - then.getTime();
	const diffSec = Math.floor(diffMs / 1000);
	const diffMin = Math.floor(diffSec / 60);
	const diffHour = Math.floor(diffMin / 60);
	const diffDay = Math.floor(diffHour / 24);

	if (diffSec < 5) return 'just now';
	if (diffSec < 60) return `${diffSec}s ago`;
	if (diffMin < 60) return `${diffMin}m ago`;
	if (diffHour < 24) return `${diffHour}h ago`;
	if (diffDay < 7) return `${diffDay}d ago`;

	return then.toLocaleDateString();
}

/**
 * Format absolute datetime
 */
export function formatDateTime(date: Date | string): string {
	const d = typeof date === 'string' ? new Date(date) : date;
	return d.toLocaleString();
}

/**
 * Format date only
 */
export function formatDate(date: Date | string): string {
	const d = typeof date === 'string' ? new Date(date) : date;
	return d.toLocaleDateString();
}

/**
 * Truncate string with ellipsis
 */
export function truncate(str: string, maxLen: number): string {
	if (str.length <= maxLen) return str;
	return str.slice(0, maxLen - 3) + '...';
}

/**
 * Truncate task ID for display
 */
export function truncateId(id: string, maxLen: number = 12): string {
	if (id.length <= maxLen) return id;
	return id.slice(0, maxLen) + '...';
}

/**
 * Parse and validate JSON, returning null if invalid
 */
export function tryParseJson(str: string): unknown | null {
	try {
		return JSON.parse(str);
	} catch {
		return null;
	}
}

/**
 * Pretty print JSON
 */
export function prettyJson(obj: unknown): string {
	return JSON.stringify(obj, null, 2);
}

/**
 * Get status color class
 */
export function getStatusColor(status: string): string {
	const colors: Record<string, string> = {
		running: 'variant-filled-success',
		active: 'variant-filled-success',
		healthy: 'variant-filled-success',
		pending: 'variant-filled-warning',
		paused: 'variant-filled-warning',
		scheduled: 'variant-filled-secondary',
		retry: 'variant-filled-warning',
		failed: 'variant-filled-error',
		archived: 'variant-filled-surface',
		completed: 'variant-filled-tertiary',
		stopped: 'variant-filled-surface',
		idle: 'variant-filled-surface'
	};
	return colors[status.toLowerCase()] || 'variant-filled-surface';
}

/**
 * Get state badge style
 */
export function getStateBadgeClass(state: string): string {
	const styles: Record<string, string> = {
		active: 'bg-success-500 text-white',
		pending: 'bg-warning-500 text-white',
		scheduled: 'bg-secondary-500 text-white',
		retry: 'bg-warning-600 text-white',
		archived: 'bg-surface-500 text-white',
		completed: 'bg-tertiary-500 text-white',
		aggregating: 'bg-primary-500 text-white'
	};
	return styles[state.toLowerCase()] || 'bg-surface-500 text-white';
}

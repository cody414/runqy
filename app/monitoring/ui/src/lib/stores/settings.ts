import { writable, derived } from 'svelte/store';
import { browser } from '$app/environment';

// Types
export type Theme = 'light' | 'dark' | 'system';
export type ViewDensity = 'compact' | 'comfortable';

interface Settings {
	theme: Theme;
	pollInterval: number;
	viewDensity: ViewDensity;
	sidebarCollapsed: boolean;
}

const DEFAULT_SETTINGS: Settings = {
	theme: 'system',
	pollInterval: 5,
	viewDensity: 'comfortable',
	sidebarCollapsed: false
};

function loadSettings(): Settings {
	if (!browser) return DEFAULT_SETTINGS;

	try {
		const stored = localStorage.getItem('runqy-settings');
		if (stored) {
			return { ...DEFAULT_SETTINGS, ...JSON.parse(stored) };
		}
	} catch {
		// Ignore parse errors
	}
	return DEFAULT_SETTINGS;
}

function createSettingsStore() {
	const { subscribe, set, update } = writable<Settings>(loadSettings());

	if (browser) {
		subscribe((value) => {
			localStorage.setItem('runqy-settings', JSON.stringify(value));
		});
	}

	return {
		subscribe,
		setTheme: (theme: Theme) => update((s) => ({ ...s, theme })),
		setPollInterval: (pollInterval: number) => update((s) => ({ ...s, pollInterval })),
		setViewDensity: (viewDensity: ViewDensity) => update((s) => ({ ...s, viewDensity })),
		toggleSidebar: () => update((s) => ({ ...s, sidebarCollapsed: !s.sidebarCollapsed })),
		reset: () => set(DEFAULT_SETTINGS)
	};
}

export const settings = createSettingsStore();

export const effectiveTheme = derived(settings, ($settings) => {
	if ($settings.theme === 'system') {
		if (browser) {
			return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
		}
		return 'dark';
	}
	return $settings.theme;
});

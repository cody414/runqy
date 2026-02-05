import { writable, get } from 'svelte/store';
import { browser } from '$app/environment';
import { base } from '$app/paths';

export interface AuthState {
	authenticated: boolean;
	setupRequired: boolean;
	email: string | null;
	loading: boolean;
	error: string | null;
	checked: boolean;
}

const DEFAULT_STATE: AuthState = {
	authenticated: false,
	setupRequired: false,
	email: null,
	loading: false,
	error: null,
	checked: false
};

function createAuthStore() {
	const { subscribe, set, update } = writable<AuthState>(DEFAULT_STATE);

	async function checkStatus(): Promise<void> {
		if (!browser) return;

		update((s) => ({ ...s, loading: true, error: null }));

		try {
			const response = await fetch(`${base}/api/auth/status`, {
				credentials: 'include'
			});

			if (!response.ok) {
				throw new Error('Failed to check auth status');
			}

			const data = await response.json();
			update((s) => ({
				...s,
				authenticated: data.authenticated,
				setupRequired: data.setup_required,
				email: data.email || null,
				loading: false,
				checked: true
			}));
		} catch (err) {
			update((s) => ({
				...s,
				loading: false,
				error: err instanceof Error ? err.message : 'Unknown error',
				checked: true
			}));
		}
	}

	async function login(email: string, password: string): Promise<boolean> {
		if (!browser) return false;

		update((s) => ({ ...s, loading: true, error: null }));

		try {
			const response = await fetch(`${base}/api/auth/login`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				credentials: 'include',
				body: JSON.stringify({ email, password })
			});

			if (!response.ok) {
				const text = await response.text();
				throw new Error(text || 'Login failed');
			}

			const data = await response.json();
			update((s) => ({
				...s,
				authenticated: true,
				email: data.email,
				loading: false
			}));
			return true;
		} catch (err) {
			update((s) => ({
				...s,
				loading: false,
				error: err instanceof Error ? err.message : 'Login failed'
			}));
			return false;
		}
	}

	async function setup(email: string, password: string, confirmPassword: string): Promise<boolean> {
		if (!browser) return false;

		update((s) => ({ ...s, loading: true, error: null }));

		try {
			const response = await fetch(`${base}/api/auth/setup`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				credentials: 'include',
				body: JSON.stringify({ email, password, confirm_password: confirmPassword })
			});

			if (!response.ok) {
				const text = await response.text();
				throw new Error(text || 'Setup failed');
			}

			const data = await response.json();
			update((s) => ({
				...s,
				authenticated: true,
				setupRequired: false,
				email: data.email,
				loading: false
			}));
			return true;
		} catch (err) {
			update((s) => ({
				...s,
				loading: false,
				error: err instanceof Error ? err.message : 'Setup failed'
			}));
			return false;
		}
	}

	async function logout(): Promise<void> {
		if (!browser) return;

		try {
			await fetch(`${base}/api/auth/logout`, {
				method: 'POST',
				credentials: 'include'
			});
		} catch {
			// Ignore errors - we're logging out anyway
		}

		set({
			...DEFAULT_STATE,
			checked: true
		});
	}

	function clearError(): void {
		update((s) => ({ ...s, error: null }));
	}

	return {
		subscribe,
		checkStatus,
		login,
		setup,
		logout,
		clearError
	};
}

export const authStore = createAuthStore();

import { writable, derived } from 'svelte/store';
import type { VaultSummary } from '$lib/api/types';
import { getVaults } from '$lib/api/client';

interface VaultsState {
	vaults: VaultSummary[];
	loading: boolean;
	error: string | null;
	lastUpdated: Date | null;
	featureDisabled: boolean;
}

function createVaultsStore() {
	const { subscribe, set, update } = writable<VaultsState>({
		vaults: [],
		loading: false,
		error: null,
		lastUpdated: null,
		featureDisabled: false
	});

	return {
		subscribe,
		fetch: async () => {
			update((s) => ({ ...s, loading: true, error: null }));
			try {
				const response = await getVaults();
				update((s) => ({
					...s,
					vaults: response.vaults || [],
					loading: false,
					lastUpdated: new Date(),
					featureDisabled: false
				}));
			} catch (e) {
				const errorMessage = e instanceof Error ? e.message : 'Failed to fetch vaults';
				// Check if vaults feature is disabled (503 response)
				const isDisabled = errorMessage.includes('not configured') ||
					errorMessage.includes('RUNQY_VAULT_MASTER_KEY');
				update((s) => ({
					...s,
					loading: false,
					error: errorMessage,
					featureDisabled: isDisabled
				}));
			}
		},
		clear: () => {
			set({
				vaults: [],
				loading: false,
				error: null,
				lastUpdated: null,
				featureDisabled: false
			});
		}
	};
}

export const vaultsStore = createVaultsStore();

// Derived store for vault count
export const vaultCount = derived(vaultsStore, ($state) => $state.vaults.length);

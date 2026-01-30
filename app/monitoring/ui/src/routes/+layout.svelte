<script lang="ts">
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import '../app.css';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import Toast from '$lib/components/Toast.svelte';
	import { settings, effectiveTheme } from '$lib/stores/settings';

	let { children } = $props();

	$effect(() => {
		if (browser) {
			document.documentElement.classList.toggle('dark', $effectiveTheme === 'dark');
		}
	});

	onMount(() => {
		if (!browser) return;

		const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
		const handleChange = () => {
			if ($settings.theme === 'system') {
				document.documentElement.classList.toggle('dark', mediaQuery.matches);
			}
		};

		mediaQuery.addEventListener('change', handleChange);
		return () => mediaQuery.removeEventListener('change', handleChange);
	});
</script>

<div class="flex h-screen overflow-hidden">
	<Sidebar />

	<main class="main-content flex-1 overflow-y-auto">
		{@render children()}
	</main>
</div>

<Toast />

<script lang="ts">
	import { QueryClientProvider } from '@tanstack/svelte-query';
	import favicon from '#lib/assets/favicon.svg';
	import { installSessionExpiryHandler } from '#lib/auth/auth.svelte';
	import { queryClient } from '#lib/query/queryClient';
	import './layout.css';

	let { children } = $props();

	// A refresh failure anywhere in the app must land the user on the login
	// screen, so the queue's handler is installed once at the root.
	installSessionExpiryHandler(() => queryClient.clear());
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>همسا — مدیریت ساختمان</title>
</svelte:head>

<QueryClientProvider client={queryClient}>
	{@render children()}
</QueryClientProvider>

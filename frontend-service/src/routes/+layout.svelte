<script>
	import favicon from '$lib/assets/favicon.svg';
	import { page } from '$app/state';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<nav class="global-nav">
	<a href="/" class:active={page.url.pathname === '/'}>💬 Chat</a>
	<a href="/benchmark" class:active={page.url.pathname.startsWith('/benchmark')}>📊 Benchmark</a>
</nav>

{@render children()}

<style>
	.global-nav {
		display: flex;
		gap: 4px;
		padding: 8px 16px;
		background: #1e293b;
		border-bottom: 1px solid #334155;
		position: sticky;
		top: 0;
		z-index: 50;
	}
	.global-nav a {
		color: #94a3b8;
		text-decoration: none;
		font-size: 13px;
		font-weight: 500;
		font-family: 'Plus Jakarta Sans', system-ui, sans-serif;
		padding: 6px 14px;
		border-radius: 6px;
		transition: all 0.15s;
	}
	.global-nav a:hover {
		color: #f1f5f9;
		background: #334155;
	}
	.global-nav a.active {
		color: #ffffff;
		background: #3b82f6;
	}
</style>

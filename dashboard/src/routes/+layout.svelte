<script>
	import '../app.css';
	import favicon from '$lib/assets/lantern.png';
	import { page } from '$app/stores';
	import { Activity, ChartNoAxesCombined, Link2, Route, Waypoints } from 'lucide-svelte';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="min-h-screen">
	<nav
		class="sticky top-0 z-50 border-b border-white/10 bg-slate-950 text-white shadow-[0_8px_30px_rgba(15,23,42,0.14)]"
	>
		<div
			class="mx-auto flex max-w-[1440px] flex-wrap items-center gap-x-6 px-4 py-3 sm:flex-nowrap sm:px-6 lg:px-8"
		>
			<a
				href="/dashboard"
				class="group flex shrink-0 items-center gap-3"
				aria-label="Siraaj dashboard"
			>
				<span
					class="grid size-10 place-items-center rounded-xl border border-amber-300/25 bg-amber-300/10 shadow-inner shadow-amber-100/5 transition group-hover:bg-amber-300/15"
				>
					<img src={favicon} alt="" class="size-8" />
				</span>
				<span class="leading-none">
					<span class="display-type block text-lg font-semibold tracking-tight">Siraaj</span>
					<span
						class="mt-1 block text-[9px] font-semibold tracking-[0.2em] text-slate-400 uppercase"
						>Private analytics</span
					>
				</span>
			</a>

			<div
				class="order-3 mt-3 flex w-full gap-1 overflow-x-auto rounded-xl bg-white/[0.04] p-1 sm:order-none sm:mt-0 sm:w-auto"
			>
				<a
					href="/dashboard"
					class="flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-all {$page
						.url.pathname === '/'
						? 'bg-white text-slate-950 shadow-sm'
						: 'text-slate-300 hover:bg-white/10 hover:text-white'}"
				>
					<ChartNoAxesCombined class="size-4" /> Overview
				</a>
				<a
					href="/dashboard/channels"
					class="flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-all {$page
						.url.pathname === '/channels'
						? 'bg-white text-slate-950 shadow-sm'
						: 'text-slate-300 hover:bg-white/10 hover:text-white'}"
				>
					<Waypoints class="size-4" /> Channels
				</a>
				<a
					href="/dashboard/funnel"
					class="flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-all {$page
						.url.pathname === '/funnel'
						? 'bg-white text-slate-950 shadow-sm'
						: 'text-slate-300 hover:bg-white/10 hover:text-white'}"
				>
					<Route class="size-4" /> Funnels
				</a>
				<a
					href="/dashboard/links"
					class="flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-all {$page
						.url.pathname === '/links'
						? 'bg-white text-slate-950 shadow-sm'
						: 'text-slate-300 hover:bg-white/10 hover:text-white'}"
				>
					<Link2 class="size-4" /> Links
				</a>
			</div>

			<div
				class="ml-auto hidden items-center gap-2 rounded-full border border-emerald-300/15 bg-emerald-300/10 px-3 py-1.5 text-xs text-emerald-100 lg:flex"
			>
				<Activity class="size-3.5" />
				<span>Self-hosted</span>
				<span class="size-1.5 rounded-full bg-emerald-300 shadow-[0_0_8px_rgba(110,231,183,0.8)]"
				></span>
			</div>
		</div>
	</nav>

	{@render children?.()}
</div>

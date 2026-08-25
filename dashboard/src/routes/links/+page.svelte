<script lang="ts">
	import { onMount } from 'svelte';
	import { format, subDays } from 'date-fns';
	import {
		ArrowUpRight,
		Check,
		Copy,
		Globe2,
		Link2,
		LoaderCircle,
		MapPin,
		MousePointerClick
	} from 'lucide-svelte';
	import { Button } from '$lib/components/ui/button';
	import { createShortLink, fetchShortLinks, fetchShortLinkStats } from '$lib/api';

	type ShortLink = {
		id: number;
		slug: string;
		destination_url: string;
		project_id: string;
		created_at: string;
		short_url: string;
		click_count: number;
		last_clicked_at: string | null;
	};

	type Breakdown = { name: string; count: number };
	type LinkStats = {
		total_clicks: number;
		countries: Breakdown[];
		referrers: Breakdown[];
		timeline: { date: string; count: number }[];
	};

	let links: ShortLink[] = $state([]);
	let selectedLink: ShortLink | null = $state(null);
	let stats: LinkStats | null = $state(null);
	let destinationURL = $state('');
	let customSlug = $state('');
	let projectID = $state('default');
	let loading = $state(true);
	let creating = $state(false);
	let statsLoading = $state(false);
	let error = $state<string | null>(null);
	let copiedSlug = $state<string | null>(null);
	let startDate = $state(format(subDays(new Date(), 29), 'yyyy-MM-dd'));
	let endDate = $state(format(new Date(), 'yyyy-MM-dd'));

	const maxTimelineClicks = $derived.by(() => {
		if (!stats) return 1;
		return Math.max(...stats.timeline.map((point) => point.count), 1);
	});

	onMount(loadLinks);

	async function loadLinks() {
		loading = true;
		error = null;
		try {
			links = await fetchShortLinks();
			if (selectedLink) {
				selectedLink = links.find((link) => link.slug === selectedLink?.slug) ?? null;
			}
		} catch (caughtError: any) {
			error = caughtError?.message || 'Failed to load short links';
		} finally {
			loading = false;
		}
	}

	async function createLink() {
		creating = true;
		error = null;
		try {
			const created = await createShortLink({
				destination_url: destinationURL,
				custom_slug: customSlug,
				project_id: projectID
			});
			destinationURL = '';
			customSlug = '';
			await loadLinks();
			selectedLink = links.find((link) => link.slug === created.slug) ?? created;
			await loadStats();
		} catch (caughtError: any) {
			error = caughtError?.message || 'Failed to create short link';
		} finally {
			creating = false;
		}
	}

	async function selectLink(link: ShortLink) {
		selectedLink = link;
		await loadStats();
	}

	async function loadStats() {
		if (!selectedLink) return;
		statsLoading = true;
		error = null;
		try {
			stats = await fetchShortLinkStats(selectedLink.slug, startDate, endDate);
		} catch (caughtError: any) {
			error = caughtError?.message || 'Failed to load link analytics';
		} finally {
			statsLoading = false;
		}
	}

	async function copyShortURL(link: ShortLink) {
		await navigator.clipboard.writeText(link.short_url);
		copiedSlug = link.slug;
		setTimeout(() => {
			if (copiedSlug === link.slug) copiedSlug = null;
		}, 1600);
	}

	function displayHost(rawURL: string) {
		try {
			return new URL(rawURL).hostname;
		} catch {
			return rawURL;
		}
	}
</script>

<svelte:head>
	<title>Short links · Siraaj</title>
</svelte:head>

<main class="mx-auto max-w-7xl px-6 py-10">
	<header class="mb-10 grid gap-6 lg:grid-cols-[1fr_auto] lg:items-end">
		<div>
			<div
				class="mb-3 flex items-center gap-2 text-xs font-semibold tracking-[0.18em] text-muted-foreground uppercase"
			>
				<Link2 class="size-4" /> Short-link analytics
			</div>
			<h1 class="max-w-3xl text-4xl font-semibold tracking-tight sm:text-5xl">
				A shorter route to the signal.
			</h1>
			<p class="mt-4 max-w-2xl text-base leading-7 text-muted-foreground">
				Create a memorable Siraaj link, then see when it is opened and which countries and sites
				sent the traffic.
			</p>
		</div>
		<div class="rounded-full border border-border bg-muted/45 px-4 py-2 text-sm">
			<span class="font-semibold"
				>{links.reduce((total, link) => total + link.click_count, 0).toLocaleString()}</span
			>
			<span class="ml-1 text-muted-foreground">all-time clicks</span>
		</div>
	</header>

	{#if error}
		<div
			class="mb-6 rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
			role="alert"
		>
			{error}
		</div>
	{/if}

	<section class="mb-10 overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
		<div class="grid lg:grid-cols-[0.8fr_1.2fr]">
			<div class="flex min-h-56 flex-col justify-between bg-foreground p-7 text-background">
				<div class="flex size-11 items-center justify-center rounded-xl bg-background/10">
					<Link2 class="size-5" />
				</div>
				<div>
					<p class="mb-2 text-xs font-medium tracking-widest text-background/60 uppercase">
						New redirect
					</p>
					<h2 class="text-2xl font-semibold tracking-tight">Make a link worth sharing.</h2>
				</div>
			</div>
			<form
				class="grid gap-5 p-7 sm:grid-cols-2"
				onsubmit={(event) => {
					event.preventDefault();
					createLink();
				}}
			>
				<label class="sm:col-span-2">
					<span class="mb-2 block text-sm font-medium">Destination URL</span>
					<input
						type="url"
						bind:value={destinationURL}
						placeholder="https://example.com/campaign"
						required
						class="h-11 w-full rounded-lg border border-input bg-background px-3 text-sm outline-none focus:border-foreground focus:ring-4 focus:ring-foreground/10"
					/>
				</label>
				<label>
					<span class="mb-2 block text-sm font-medium"
						>Custom ending <span class="text-muted-foreground">optional</span></span
					>
					<div
						class="flex h-11 items-center rounded-lg border border-input focus-within:border-foreground focus-within:ring-4 focus-within:ring-foreground/10"
					>
						<span class="border-r border-input px-3 text-sm text-muted-foreground">/s/</span>
						<input
							bind:value={customSlug}
							pattern={'[A-Za-z0-9_-]{3,64}'}
							placeholder="summer-26"
							class="min-w-0 flex-1 bg-transparent px-3 text-sm outline-none"
						/>
					</div>
				</label>
				<label>
					<span class="mb-2 block text-sm font-medium">Project</span>
					<input
						bind:value={projectID}
						required
						class="h-11 w-full rounded-lg border border-input bg-background px-3 text-sm outline-none focus:border-foreground focus:ring-4 focus:ring-foreground/10"
					/>
				</label>
				<div class="flex items-center justify-between gap-4 sm:col-span-2">
					<p class="text-xs text-muted-foreground">
						Only the country and referring domain are stored for each click.
					</p>
					<Button type="submit" disabled={creating} class="min-w-32">
						{#if creating}<LoaderCircle class="animate-spin" />{/if}
						{creating ? 'Creating…' : 'Create link'}
					</Button>
				</div>
			</form>
		</div>
	</section>

	<div class="grid gap-8 xl:grid-cols-[0.95fr_1.45fr]">
		<section>
			<div class="mb-4 flex items-center justify-between">
				<h2 class="text-lg font-semibold">Your links</h2>
				<span class="text-sm text-muted-foreground">{links.length} total</span>
			</div>
			<div class="space-y-3">
				{#if loading}
					<div
						class="flex h-40 items-center justify-center rounded-xl border border-border text-muted-foreground"
					>
						<LoaderCircle class="mr-2 size-4 animate-spin" /> Loading links
					</div>
				{:else if links.length === 0}
					<div class="rounded-xl border border-dashed border-border bg-muted/20 p-10 text-center">
						<Link2 class="mx-auto mb-3 size-6 text-muted-foreground" />
						<p class="font-medium">No short links yet</p>
						<p class="mt-1 text-sm text-muted-foreground">Your first link will appear here.</p>
					</div>
				{:else}
					{#each links as link}
						<article
							class="rounded-xl border border-border p-4 transition hover:border-foreground/30 {selectedLink?.slug ===
							link.slug
								? 'bg-muted/60 ring-2 ring-foreground/10'
								: 'bg-card'}"
						>
							<button type="button" onclick={() => selectLink(link)} class="w-full text-left">
								<div class="mb-3 flex items-start justify-between gap-4">
									<div class="min-w-0">
										<p class="truncate font-semibold">{link.short_url}</p>
										<p class="mt-1 truncate text-xs text-muted-foreground">
											{displayHost(link.destination_url)}
										</p>
									</div>
									<div class="text-right">
										<p class="text-xl font-semibold tabular-nums">
											{link.click_count.toLocaleString()}
										</p>
										<p class="text-[11px] text-muted-foreground uppercase">clicks</p>
									</div>
								</div>
							</button>
							<div class="flex items-center justify-between border-t border-border pt-3">
								<span class="text-xs text-muted-foreground"
									>{link.project_id} · {format(new Date(link.created_at), 'd MMM yyyy')}</span
								>
								<button
									type="button"
									onclick={() => copyShortURL(link)}
									aria-label="Copy short URL"
									class="rounded-md p-1.5 transition hover:bg-accent"
								>
									{#if copiedSlug === link.slug}<Check class="size-4" />{:else}<Copy
											class="size-4"
										/>{/if}
								</button>
							</div>
						</article>
					{/each}
				{/if}
			</div>
		</section>

		<section class="min-w-0">
			<div class="mb-4 flex flex-wrap items-end justify-between gap-3">
				<div>
					<h2 class="text-lg font-semibold">Click origins</h2>
					<p class="text-sm text-muted-foreground">
						{selectedLink ? selectedLink.short_url : 'Select a link to inspect'}
					</p>
				</div>
				{#if selectedLink}
					<div class="flex items-end gap-2">
						<label class="text-xs"
							><span class="mb-1 block text-muted-foreground">From</span><input
								type="date"
								bind:value={startDate}
								class="rounded-md border border-input bg-background px-2 py-1.5"
							/></label
						>
						<label class="text-xs"
							><span class="mb-1 block text-muted-foreground">To</span><input
								type="date"
								bind:value={endDate}
								class="rounded-md border border-input bg-background px-2 py-1.5"
							/></label
						>
						<Button variant="outline" size="sm" onclick={loadStats}>Apply</Button>
					</div>
				{/if}
			</div>

			{#if !selectedLink}
				<div
					class="flex h-96 flex-col items-center justify-center rounded-2xl border border-dashed border-border bg-muted/15 text-center"
				>
					<MousePointerClick class="mb-3 size-7 text-muted-foreground" />
					<p class="font-medium">Choose a link</p>
					<p class="mt-1 max-w-xs text-sm text-muted-foreground">
						Its click timeline, countries, and referring sites will appear here.
					</p>
				</div>
			{:else if statsLoading}
				<div
					class="flex h-96 items-center justify-center rounded-2xl border border-border text-muted-foreground"
				>
					<LoaderCircle class="mr-2 size-4 animate-spin" /> Loading analytics
				</div>
			{:else if stats}
				<div class="space-y-5">
					<div class="grid gap-3 sm:grid-cols-3">
						<div class="rounded-xl border border-border bg-card p-4">
							<MousePointerClick class="mb-5 size-4 text-muted-foreground" />
							<p class="text-3xl font-semibold tabular-nums">
								{stats.total_clicks.toLocaleString()}
							</p>
							<p class="mt-1 text-xs text-muted-foreground uppercase">Clicks in range</p>
						</div>
						<div class="rounded-xl border border-border bg-card p-4">
							<MapPin class="mb-5 size-4 text-muted-foreground" />
							<p class="text-3xl font-semibold tabular-nums">{stats.countries.length}</p>
							<p class="mt-1 text-xs text-muted-foreground uppercase">Countries</p>
						</div>
						<div class="rounded-xl border border-border bg-card p-4">
							<Globe2 class="mb-5 size-4 text-muted-foreground" />
							<p class="text-3xl font-semibold tabular-nums">{stats.referrers.length}</p>
							<p class="mt-1 text-xs text-muted-foreground uppercase">Sources</p>
						</div>
					</div>

					<div class="rounded-xl border border-border bg-card p-5">
						<div class="mb-5 flex items-center justify-between">
							<h3 class="font-semibold">Click rhythm</h3>
							<a
								href={selectedLink.destination_url}
								target="_blank"
								rel="noreferrer"
								class="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
								>Destination <ArrowUpRight class="size-3" /></a
							>
						</div>
						{#if stats.timeline.length === 0}
							<p class="py-10 text-center text-sm text-muted-foreground">
								No clicks in this date range.
							</p>
						{:else}
							<div class="flex h-36 items-end gap-1" aria-label="Daily clicks chart">
								{#each stats.timeline as point}
									<div class="group relative flex h-full min-w-1 flex-1 items-end">
										<div
											class="w-full rounded-t-sm bg-foreground/80 transition group-hover:bg-foreground"
											style={`height: ${Math.max((point.count / maxTimelineClicks) * 100, 6)}%`}
										></div>
										<span
											class="pointer-events-none absolute bottom-full left-1/2 z-10 mb-2 hidden -translate-x-1/2 rounded bg-foreground px-2 py-1 text-[10px] whitespace-nowrap text-background group-hover:block"
											>{point.date}: {point.count}</span
										>
									</div>
								{/each}
							</div>
						{/if}
					</div>

					<div class="grid gap-5 sm:grid-cols-2">
						{#each [{ title: 'Countries', entries: stats.countries, icon: MapPin }, { title: 'Referring sites', entries: stats.referrers, icon: Globe2 }] as panel}
							<div class="rounded-xl border border-border bg-card p-5">
								<div class="mb-4 flex items-center gap-2">
									<panel.icon class="size-4 text-muted-foreground" />
									<h3 class="font-semibold">{panel.title}</h3>
								</div>
								<div class="space-y-3">
									{#each panel.entries as entry}
										<div>
											<div class="mb-1 flex justify-between gap-4 text-sm">
												<span class="truncate">{entry.name}</span><span
													class="font-medium tabular-nums">{entry.count}</span
												>
											</div>
											<div class="h-1 overflow-hidden rounded-full bg-muted">
												<div
													class="h-full rounded-full bg-foreground"
													style={`width: ${(entry.count / Math.max(stats.total_clicks, 1)) * 100}%`}
												></div>
											</div>
										</div>
									{:else}<p class="text-muted-foreground py-5 text-center text-sm">
											No origin data yet.
										</p>{/each}
								</div>
							</div>
						{/each}
					</div>
				</div>
			{/if}
		</section>
	</div>
</main>

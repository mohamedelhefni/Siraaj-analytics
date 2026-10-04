<script lang="ts">
	import { onDestroy, onMount, tick } from 'svelte';
	import { format, formatDistanceToNow } from 'date-fns';
	import {
		ChevronLeft,
		ChevronRight,
		ExternalLink,
		LoaderCircle,
		MonitorPlay,
		MousePointerClick,
		Search,
		Timer,
		Trash2
	} from 'lucide-svelte';
	import { deleteReplay, fetchReplayEvents, fetchReplays } from '$lib/api';

	type Replay = {
		project_id: string;
		recording_id: string;
		session_id: string;
		url: string;
		started_at: string;
		ended_at: string;
		chunks: number;
		bytes: number;
		clicks: number;
	};
	type Visit = { session_id: string; project_id: string; started_at: string; pages: Replay[] };
	type Player = {
		$destroy(): void;
		$set(props: Record<string, number>): void;
		triggerResize(): void;
	};

	let replays: Replay[] = $state([]);
	let selected: Replay | null = $state(null);
	let viewport = $state({ width: 0, height: 0 });
	let loading = $state(true);
	let playerLoading = $state(false);
	let error = $state<string | null>(null);
	let projectFilter = $state('');
	let query = $state('');
	let stage: HTMLDivElement | undefined = $state();
	let stageWidth = $state(0);
	let player: Player | null = $state.raw(null);

	const projects = $derived([...new Set(replays.map((replay) => replay.project_id))].sort());
	const visits = $derived.by(() => {
		const needle = query.trim().toLowerCase();
		const grouped = new Map<string, Visit>();
		// Replays arrive newest first; keep that order for visits and play pages oldest first.
		for (const replay of replays) {
			if (projectFilter && replay.project_id !== projectFilter) continue;
			if (needle && !replay.url.toLowerCase().includes(needle)) continue;
			const key = `${replay.project_id}/${replay.session_id}`;
			const visit = grouped.get(key) ?? { session_id: replay.session_id, project_id: replay.project_id, started_at: replay.started_at, pages: [] };
			visit.pages.unshift(replay);
			visit.started_at = replay.started_at;
			grouped.set(key, visit);
		}
		return [...grouped.values()];
	});
	const selectedVisit = $derived(visits.find((visit) => visit.pages.includes(selected!)) ?? null);
	const selectedIndex = $derived(selectedVisit?.pages.indexOf(selected!) ?? -1);

	onMount(async () => {
		try {
			replays = await fetchReplays();
		} catch (caughtError: any) {
			error = caughtError?.message || 'Failed to load replays';
		} finally {
			loading = false;
		}
		// Open the newest recording once the player stage has rendered.
		await tick();
		if (replays.length) void play(replays[0], false);
	});

	onDestroy(() => player?.$destroy());

	// Keep the player fitted to the column as the window resizes.
	$effect(() => {
		const size = playerSize(stageWidth);
		if (!player || !size.width) return;
		player.$set(size);
		player.triggerResize();
	});

	function playerSize(width: number) {
		const ratio = viewport.width ? viewport.height / viewport.width : 0.6;
		const maxHeight = typeof window === 'undefined' ? 600 : window.innerHeight * 0.62;
		return { width, height: Math.round(Math.min(width * ratio, maxHeight)) };
	}

	async function play(replay: Replay, autoPlay = true) {
		selected = replay;
		playerLoading = true;
		error = null;
		player?.$destroy();
		player = null;
		try {
			// The player is ~150KB, so it only loads when a replay is opened.
			const [events, { default: Player }] = await Promise.all([
				fetchReplayEvents(replay.project_id, replay.recording_id),
				import('rrweb-player'),
				import('rrweb-player/dist/style.css')
			]);
			if (selected !== replay || !stage) return;
			if (events.length < 2) throw new Error('This recording has not captured enough to replay yet.');
			const meta = events.find((event: any) => event.type === 4)?.data;
			viewport = { width: meta?.width ?? 0, height: meta?.height ?? 0 };
			player = new Player({
				target: stage,
				props: { events, ...playerSize(stageWidth), autoPlay, skipInactive: true, speedOption: [1, 2, 4, 8] }
			}) as unknown as Player;
		} catch (caughtError: any) {
			error = caughtError?.message || 'Failed to load replay';
		} finally {
			if (selected === replay) playerLoading = false;
		}
	}

	async function remove(replay: Replay) {
		if (!confirm(`Delete the recording of ${path(replay.url)}? This cannot be undone.`)) return;
		try {
			await deleteReplay(replay.project_id, replay.recording_id);
			const index = replays.indexOf(replay);
			replays = replays.filter((item) => item !== replay);
			player?.$destroy();
			player = null;
			selected = null;
			const next = replays[Math.min(index, replays.length - 1)];
			if (next) void play(next, false);
		} catch (caughtError: any) {
			error = caughtError?.message || 'Failed to delete replay';
		}
	}

	function seconds(from: string, to: string) {
		return Math.max(0, (Date.parse(to) - Date.parse(from)) / 1000);
	}

	function duration(total: number) {
		if (total < 60) return `${Math.round(total)}s`;
		const minutes = Math.floor(total / 60);
		return minutes < 60 ? `${minutes}m ${Math.round(total % 60)}s` : `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
	}

	function visitDuration(visit: Visit) {
		return duration(seconds(visit.pages[0].started_at, visit.pages[visit.pages.length - 1].ended_at));
	}

	function path(url: string) {
		try {
			const parsed = new URL(url);
			return parsed.pathname + parsed.search;
		} catch {
			return url || 'Unknown page';
		}
	}

	function host(url: string) {
		try {
			return new URL(url).host;
		} catch {
			return '';
		}
	}
</script>

<svelte:head>
	<title>Replays · Siraaj</title>
</svelte:head>

<main class="mx-auto max-w-[1500px] px-6 py-8">
	<header class="mb-6 flex flex-wrap items-end justify-between gap-4">
		<div>
			<div
				class="mb-2 flex items-center gap-2 text-xs font-semibold tracking-[0.18em] text-muted-foreground uppercase"
			>
				<MonitorPlay class="size-4" /> Replays
			</div>
			<h1 class="text-3xl font-semibold tracking-tight">Session replays</h1>
			<p class="mt-1 text-sm text-muted-foreground">
				Every page load is one recording, grouped by visit. Inputs are always masked.
			</p>
		</div>
		{#if replays.length}
			<div class="flex flex-wrap gap-2">
				<label class="relative">
					<span class="sr-only">Search pages</span>
					<Search class="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
					<input
						bind:value={query}
						placeholder="Filter by page"
						class="h-10 w-56 rounded-lg border border-input bg-background pr-3 pl-9 text-sm outline-none focus:border-foreground focus:ring-4 focus:ring-foreground/10"
					/>
				</label>
				{#if projects.length > 1}
					<select
						bind:value={projectFilter}
						aria-label="Project"
						class="h-10 rounded-lg border border-input bg-background px-3 text-sm outline-none focus:border-foreground focus:ring-4 focus:ring-foreground/10"
					>
						<option value="">All projects</option>
						{#each projects as project}<option value={project}>{project}</option>{/each}
					</select>
				{/if}
			</div>
		{/if}
	</header>

	{#if error}
		<div
			class="mb-6 rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
			role="alert"
		>
			{error}
		</div>
	{/if}

	{#if loading}
		<div class="flex h-80 items-center justify-center rounded-2xl border border-border text-muted-foreground">
			<LoaderCircle class="mr-2 size-4 animate-spin" /> Loading replays
		</div>
	{:else if replays.length === 0}
		<section class="rounded-2xl border border-dashed border-border bg-muted/15 p-10">
			<MonitorPlay class="mb-4 size-8 text-muted-foreground" />
			<h2 class="text-lg font-semibold">No recordings yet</h2>
			<p class="mt-1 max-w-xl text-sm leading-6 text-muted-foreground">
				Load the replay script next to the SDK. Recordings appear here a few seconds after a visitor lands. Add
				<code>siraaj-block</code> to hide an element or <code>siraaj-mask</code> to mask its text.
			</p>
			<pre class="mt-5 max-w-2xl overflow-x-auto rounded-xl bg-foreground p-4 text-xs leading-5 text-background"><code
					>&lt;script src="analytics.min.js"&gt;&lt;/script&gt;
&lt;script src="replay.min.js"&gt;&lt;/script&gt;
&lt;script&gt;
  SiraajAnalytics.analytics.init(&#123; apiUrl: '…', trackingToken: 'siraaj_trk_…' &#125;);
  SiraajReplay.startReplay(SiraajAnalytics.analytics, &#123; sampleRate: 1 &#125;);
&lt;/script&gt;</code
				></pre>
		</section>
	{:else}
		<div class="grid gap-6 lg:grid-cols-[22rem_1fr]">
			<aside class="min-w-0">
				<p class="mb-3 text-sm text-muted-foreground">
					{visits.length} visits · {visits.reduce((total, visit) => total + visit.pages.length, 0)} recordings
				</p>
				<div class="max-h-[calc(100vh-14rem)] space-y-3 overflow-y-auto pr-1">
					{#each visits as visit (visit.project_id + visit.session_id)}
						<article class="overflow-hidden rounded-xl border border-border bg-card">
							<div class="flex items-center justify-between gap-3 border-b border-border bg-muted/40 px-3 py-2 text-xs">
								<span class="truncate font-medium" title={format(new Date(visit.started_at), 'PPpp')}>
									{formatDistanceToNow(new Date(visit.started_at), { addSuffix: true })}
								</span>
								<span class="shrink-0 text-muted-foreground">
									{visit.pages.length} {visit.pages.length === 1 ? 'page' : 'pages'} · {visitDuration(visit)}
									{#if projects.length > 1}· {visit.project_id}{/if}
								</span>
							</div>
							<ol>
								{#each visit.pages as replay (replay.recording_id)}
									<li>
										<button
											type="button"
											onclick={() => play(replay)}
											aria-current={selected === replay}
											class="flex w-full items-center gap-3 px-3 py-2.5 text-left text-sm transition hover:bg-accent {selected === replay
												? 'bg-accent shadow-[inset_3px_0_0_var(--color-foreground)]'
												: ''}"
										>
											<MonitorPlay class="size-4 shrink-0 text-muted-foreground" />
											<span class="min-w-0 flex-1 truncate font-medium" title={replay.url}>{path(replay.url)}</span>
											<span class="shrink-0 text-xs text-muted-foreground tabular-nums">
												{duration(seconds(replay.started_at, replay.ended_at))}
											</span>
										</button>
									</li>
								{/each}
							</ol>
						</article>
					{:else}
						<p class="rounded-xl border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
							No recordings match these filters.
						</p>
					{/each}
				</div>
			</aside>

			<section class="min-w-0">
				<div class="overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
					{#if selected}
						<div class="flex flex-wrap items-center justify-between gap-3 border-b border-border px-4 py-3">
							<div class="min-w-0">
								<a
									href={selected.url}
									target="_blank"
									rel="noreferrer"
									class="flex items-center gap-1.5 truncate font-semibold hover:underline"
								>
									<span class="truncate">{path(selected.url)}</span>
									<ExternalLink class="size-3.5 shrink-0 text-muted-foreground" />
								</a>
								<p class="mt-0.5 truncate text-xs text-muted-foreground">
									{host(selected.url)} · {format(new Date(selected.started_at), 'PPp')}
								</p>
							</div>
							<div class="flex items-center gap-1">
								{#if selectedVisit && selectedVisit.pages.length > 1}
									<button
										type="button"
										onclick={() => play(selectedVisit.pages[selectedIndex - 1])}
										disabled={selectedIndex <= 0}
										aria-label="Previous page in this visit"
										class="rounded-md p-1.5 transition hover:bg-accent disabled:opacity-30"
									>
										<ChevronLeft class="size-4" />
									</button>
									<span class="px-1 text-xs text-muted-foreground tabular-nums">
										Page {selectedIndex + 1} of {selectedVisit.pages.length}
									</span>
									<button
										type="button"
										onclick={() => play(selectedVisit.pages[selectedIndex + 1])}
										disabled={selectedIndex >= selectedVisit.pages.length - 1}
										aria-label="Next page in this visit"
										class="rounded-md p-1.5 transition hover:bg-accent disabled:opacity-30"
									>
										<ChevronRight class="size-4" />
									</button>
								{/if}
								<button
									type="button"
									onclick={() => remove(selected!)}
									aria-label="Delete recording"
									class="ml-1 rounded-md p-1.5 text-muted-foreground transition hover:bg-destructive/10 hover:text-destructive"
								>
									<Trash2 class="size-4" />
								</button>
							</div>
						</div>
					{/if}

					<div bind:clientWidth={stageWidth} class="replay-stage relative bg-muted/30">
						<div bind:this={stage} class:invisible={playerLoading}></div>
						{#if !selected || playerLoading}
							<div class="absolute inset-0 flex min-h-80 flex-col items-center justify-center text-muted-foreground">
								{#if playerLoading}
									<LoaderCircle class="mb-2 size-6 animate-spin" /> Loading recording
								{:else}
									<MonitorPlay class="mb-2 size-7" /> Choose a recording
								{/if}
							</div>
						{/if}
						{#if !selected || (playerLoading && !player)}<div class="h-80"></div>{/if}
					</div>

					{#if selected}
						<dl class="grid grid-cols-2 gap-px border-t border-border bg-border text-sm sm:grid-cols-4">
							<div class="bg-card px-4 py-3">
								<dt class="flex items-center gap-1.5 text-xs text-muted-foreground"><Timer class="size-3.5" /> Duration</dt>
								<dd class="mt-0.5 font-semibold tabular-nums">{duration(seconds(selected.started_at, selected.ended_at))}</dd>
							</div>
							<div class="bg-card px-4 py-3">
								<dt class="flex items-center gap-1.5 text-xs text-muted-foreground"><MousePointerClick class="size-3.5" /> Clicks</dt>
								<dd class="mt-0.5 font-semibold tabular-nums">{selected.clicks}</dd>
							</div>
							<div class="bg-card px-4 py-3">
								<dt class="text-xs text-muted-foreground">Viewport</dt>
								<dd class="mt-0.5 font-semibold tabular-nums">
									{viewport.width && !playerLoading ? `${viewport.width}×${viewport.height}` : '—'}
								</dd>
							</div>
							<div class="bg-card px-4 py-3">
								<dt class="text-xs text-muted-foreground">Stored</dt>
								<dd class="mt-0.5 font-semibold tabular-nums">{(selected.bytes / 1024).toFixed(0)} KB · {selected.project_id}</dd>
							</div>
						</dl>
					{/if}
				</div>
			</section>
		</div>
	{/if}
</main>

<style>
	/* Blend rrweb-player into the card instead of its default floating panel. */
	.replay-stage :global(.rr-player) {
		box-shadow: none;
		border-radius: 0;
		background: transparent;
		float: none;
	}
	.replay-stage :global(.rr-player__frame) {
		background: var(--color-background);
	}
	.replay-stage :global(.rr-controller) {
		background: var(--color-card);
		border-top: 1px solid var(--color-border);
		border-radius: 0;
	}
</style>

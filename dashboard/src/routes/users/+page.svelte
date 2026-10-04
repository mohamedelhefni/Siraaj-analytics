<script lang="ts">
	import PageHeader from '$lib/components/PageHeader.svelte';
	import { onMount } from 'svelte';
	import { base } from '$app/paths';
	import { page } from '$app/stores';
	import { format, formatDistanceToNow, subDays } from 'date-fns';
	import {
		Eye,
		Monitor,
		Smartphone,
		Tablet,
		LoaderCircle,
		MessageSquareText,
		MonitorPlay,
		MousePointerClick,
		Search,
		Users,
		Zap
	} from 'lucide-svelte';
	import { fetchEvents, fetchReplays } from '$lib/api';
	import { getBrowserIcon, getCountryFlag, getOSIcon } from '$lib/utils/icons';

	type Ev = {
		id: number;
		timestamp: string;
		event_name: string;
		user_id: string;
		session_id: string;
		url: string;
		country: string;
		browser: string;
		os: string;
		device: string;
		channel: string;
		referrer: string;
	};
	type Answer = { survey_id: number; survey_name: string; answers: string[]; created_at: string };
	type Visitor = {
		id: string;
		events: number;
		first: string;
		last: string;
		country: string;
		device: string;
		browser: string;
		os: string;
	};
	type Item = { at: string; session?: string; ev?: Ev; survey?: Answer };

	const fmt = (d: Date) => format(d, 'yyyy-MM-dd');
	const today = () => new Date();

	let visitors: Visitor[] = $state([]);
	let listLoading = $state(true);
	let query = $state('');
	let selectedId = $state<string | null>(null);
	let detailLoading = $state(false);
	let events: Ev[] = $state([]);
	let surveys: Answer[] = $state([]);
	let error = $state<string | null>(null);
	type Recording = {
		project_id: string;
		recording_id: string;
		session_id: string;
		url: string;
		started_at: string;
	};
	let recordings: Recording[] = $state([]);
	const replaySessions = $derived(new Set(recordings.map((replay) => replay.session_id)));

	// ponytail: no users endpoint yet, so the list groups the newest 1000 events of the last 30 days.
	// Add a GROUP BY user_id endpoint when traffic outgrows that window.
	onMount(async () => {
		// Replays are optional context; the journey still works without them.
		fetchReplays()
			.then((list: Recording[]) => (recordings = list))
			.catch(() => {});
		try {
			const data = await fetchEvents(fmt(subDays(today(), 30)), fmt(today()), 1000);
			const map = new Map<string, Visitor>();
			for (const e of (data.events ?? []) as Ev[]) {
				if (!e.user_id) continue;
				const v = map.get(e.user_id);
				if (!v) {
					map.set(e.user_id, {
						id: e.user_id,
						events: 1,
						first: e.timestamp,
						last: e.timestamp,
						country: e.country,
						device: e.device,
						browser: e.browser,
						os: e.os
					});
				} else {
					v.events++;
					if (e.timestamp < v.first) v.first = e.timestamp;
				}
			}
			visitors = [...map.values()].sort((a, b) => b.last.localeCompare(a.last));
		} catch (e: any) {
			error = e?.message || 'Failed to load users';
		} finally {
			listLoading = false;
		}
		const id = $page.url.searchParams.get('id');
		if (id) select(id);
	});

	async function select(id: string) {
		selectedId = id;
		detailLoading = true;
		error = null;
		try {
			const data = await fetchEvents(fmt(subDays(today(), 90)), fmt(today()), 1000, 0, id);
			if (selectedId !== id) return;
			events = data.events ?? [];
			surveys = data.survey_responses ?? [];
		} catch (e: any) {
			error = e?.message || 'Failed to load user';
		} finally {
			if (selectedId === id) detailLoading = false;
		}
	}

	const shown = $derived(
		visitors.filter((v) => v.id.toLowerCase().includes(query.trim().toLowerCase()))
	);

	// Events and survey answers merged newest-first, then bucketed by day.
	const days = $derived.by(() => {
		const items: Item[] = [
			...events.map((ev) => ({ at: ev.timestamp, session: ev.session_id, ev })),
			...surveys.map((survey) => ({ at: survey.created_at, survey }))
		].sort((a, b) => b.at.localeCompare(a.at));
		const out: { day: string; items: Item[] }[] = [];
		for (const item of items) {
			const day = format(new Date(item.at), 'EEEE, MMM d, yyyy');
			if (out.at(-1)?.day !== day) out.push({ day, items: [] });
			out.at(-1)!.items.push(item);
		}
		return out;
	});

	const summary = $derived.by(() => {
		if (!events.length) return null;
		const last = events[0];
		return {
			first: events[events.length - 1].timestamp,
			last: last.timestamp,
			sessions: new Set(events.map((e) => e.session_id)).size,
			replays: new Set(events.map((e) => e.session_id).filter((id) => replaySessions.has(id))).size,
			pages: events.filter((e) => e.event_name === 'page_view').length,
			last_event: last,
			channel: events[events.length - 1].channel
		};
	});

	// A page view's recording is the same session and URL loaded closest in time.
	function recordingFor(ev: Ev) {
		if (ev.event_name !== 'page_view' || !replaySessions.has(ev.session_id)) return null;
		const at = Date.parse(ev.timestamp);
		let best: Recording | null = null;
		for (const replay of recordings) {
			if (replay.session_id !== ev.session_id || replay.url !== ev.url) continue;
			const gap = Math.abs(Date.parse(replay.started_at) - at);
			if (!best || gap < Math.abs(Date.parse(best.started_at) - at)) best = replay;
		}
		return best;
	}

	// Icon helpers return either an image URL or an emoji fallback.
	const isImage = (icon: string) => /^(\/|http|data:)/.test(icon);
	function deviceIcon(device: string) {
		const name = (device || '').toLowerCase();
		if (name.includes('tablet')) return Tablet;
		if (name.includes('mobile') || name.includes('phone')) return Smartphone;
		return Monitor;
	}

	function path(url: string) {
		try {
			const u = new URL(url);
			return u.pathname + u.search;
		} catch {
			return url || '—';
		}
	}
	function label(e: Ev) {
		if (e.event_name === 'page_view') return `Viewed ${path(e.url)}`;
		return e.event_name.replaceAll('_', ' ');
	}
	function short(id: string) {
		return id.length > 18 ? id.slice(0, 8) + '…' + id.slice(-4) : id;
	}
	function hue(id: string) {
		let h = 0;
		for (const c of id) h = (h * 31 + c.charCodeAt(0)) % 360;
		return h;
	}
</script>

{#snippet environment(where: { country: string; device: string; browser: string; os: string })}
	{@const DeviceIcon = deviceIcon(where.device)}
	<span class="flex flex-wrap items-center gap-x-3 gap-y-1">
		{#if where.country}
			<span class="flex items-center gap-1" title="Country">
				<span aria-hidden="true">{getCountryFlag(where.country)}</span>{where.country}
			</span>
		{/if}
		{#if where.device}
			<span class="flex items-center gap-1 capitalize" title="Device">
				<DeviceIcon class="size-3.5" />{where.device}
			</span>
		{/if}
		{#each [['Browser', where.browser, getBrowserIcon(where.browser)], ['OS', where.os, getOSIcon(where.os)]] as [kind, name, icon] (kind)}
			{#if name}
				<span class="flex items-center gap-1" title={kind}>
					{#if isImage(icon)}
						<img src={icon} alt="" class="size-3.5" />
					{:else}
						<span aria-hidden="true">{icon}</span>
					{/if}
					{name}
				</span>
			{/if}
		{/each}
	</span>
{/snippet}

<svelte:head>
	<title>Users · Siraaj</title>
</svelte:head>

<main class="mx-auto max-w-[1440px] px-4 py-6 sm:px-6 lg:px-8">
	<PageHeader
		class="mb-6"
		title="Users"
		description="Follow one visitor from first visit onward. Call identify(&quot;your-user-id&quot;) in the SDK to tie visits to your own users."
	/>

	{#if error}
		<div
			class="mb-6 rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
			role="alert"
		>
			{error}
		</div>
	{/if}

	<div class="grid gap-8 xl:grid-cols-[0.8fr_1.6fr]">
		<section>
			<div class="relative mb-4">
				<Search class="absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
				<input
					bind:value={query}
					placeholder="Search by user id"
					aria-label="Search users"
					class="w-full rounded-xl border border-border bg-card py-2.5 pr-3 pl-9 text-sm outline-none focus:ring-2 focus:ring-foreground/10"
				/>
			</div>
			<div class="mb-3 flex items-center justify-between text-sm text-muted-foreground">
				<span class="font-semibold text-foreground">Recent visitors</span>
				<span>{shown.length} in last 30 days</span>
			</div>
			<div class="max-h-[70vh] space-y-2 overflow-y-auto">
				{#if listLoading}
					<div
						class="flex h-40 items-center justify-center rounded-xl border border-border text-muted-foreground"
					>
						<LoaderCircle class="mr-2 size-4 animate-spin" /> Loading users
					</div>
				{:else if shown.length === 0}
					<div class="rounded-xl border border-dashed border-border bg-muted/20 p-10 text-center">
						<Users class="mx-auto mb-3 size-6 text-muted-foreground" />
						<p class="font-medium">{query ? 'No matching users' : 'No users yet'}</p>
						{#if query}
							<button class="mt-2 text-sm underline" onclick={() => select(query.trim())}
								>Look up “{query.trim()}” anyway</button
							>
						{/if}
					</div>
				{:else}
					{#each shown as v (v.id)}
						<button
							type="button"
							onclick={() => select(v.id)}
							class="flex w-full items-center gap-3 rounded-xl border border-border p-3 text-left transition hover:border-foreground/30 {selectedId ===
							v.id
								? 'bg-muted/60 ring-2 ring-foreground/10'
								: 'bg-card'}"
						>
							<span
								class="flex size-9 shrink-0 items-center justify-center rounded-full text-xs font-semibold text-white"
								style="background:hsl({hue(v.id)} 55% 45%)"
								aria-hidden="true">{v.id.slice(0, 2).toUpperCase()}</span
							>
							<span class="min-w-0 flex-1">
								<span class="block truncate font-mono text-sm font-medium">{short(v.id)}</span>
								<span class="block truncate text-xs text-muted-foreground">
									Last seen {formatDistanceToNow(new Date(v.last), { addSuffix: true })}
								</span>
								<span class="mt-1 block text-xs text-muted-foreground">
									{@render environment(v)}
								</span>
							</span>
							<span class="text-right">
								<span class="block text-lg font-semibold tabular-nums">{v.events}</span>
								<span class="block text-[11px] text-muted-foreground uppercase">events</span>
							</span>
						</button>
					{/each}
				{/if}
			</div>
		</section>

		<section class="min-w-0">
			{#if !selectedId}
				<div
					class="flex h-96 flex-col items-center justify-center rounded-2xl border border-border bg-card text-center text-muted-foreground"
				>
					<Users class="mb-3 size-7" />
					<p class="font-medium text-foreground">Choose a visitor</p>
					<p class="text-sm">Their full journey shows up here.</p>
				</div>
			{:else if detailLoading}
				<div
					class="flex h-96 items-center justify-center rounded-2xl border border-border bg-card text-muted-foreground"
				>
					<LoaderCircle class="mr-2 size-5 animate-spin" /> Loading journey
				</div>
			{:else}
				<h2 class="mb-4 truncate font-mono text-lg font-semibold" title={selectedId}>
					{selectedId}
				</h2>
				{#if summary}
					<div class="mb-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
						{#each [['First seen', formatDistanceToNow( new Date(summary.first), { addSuffix: true } )], ['Last seen', formatDistanceToNow( new Date(summary.last), { addSuffix: true } )], ['Sessions', summary.sessions], ['Page views', summary.pages]] as [name, value]}
							<div class="rounded-xl border border-border bg-card p-4">
								<p class="text-[11px] text-muted-foreground uppercase">{name}</p>
								<p class="mt-1 text-xl font-semibold tabular-nums">{value}</p>
							</div>
						{/each}
					</div>
					<div
						class="mb-6 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted-foreground"
					>
						{@render environment(summary.last_event)}
						{#if summary.channel}<span>· arrived via {summary.channel}</span>{/if}
						{#if summary.replays}
							<span
								>· {summary.replays} recorded {summary.replays === 1 ? 'session' : 'sessions'}</span
							>
						{/if}
					</div>
				{/if}

				{#if days.length === 0}
					<div class="rounded-xl border border-dashed border-border bg-muted/20 p-10 text-center">
						<p class="font-medium">No activity in the last 90 days</p>
					</div>
				{/if}

				{#each days as d (d.day)}
					<h3
						class="mt-6 mb-3 text-xs font-semibold tracking-wider text-muted-foreground uppercase first:mt-0"
					>
						{d.day}
					</h3>
					<ol class="relative ml-4 space-y-1 border-l border-border pl-6">
						{#each d.items as item, i (item.ev?.id ?? 's' + item.at + i)}
							{@const newSession = item.ev && item.session !== d.items[i + 1]?.session}
							<li class="relative">
								<span
									class="absolute top-2.5 -left-[2.1rem] flex size-6 items-center justify-center rounded-full border border-border bg-background"
								>
									{#if item.survey}
										<MessageSquareText class="size-3 text-violet-500" />
									{:else if item.ev?.event_name === 'page_view'}
										<Eye class="size-3 text-muted-foreground" />
									{:else if item.ev?.event_name === 'click'}
										<MousePointerClick class="size-3 text-sky-500" />
									{:else}
										<Zap class="size-3 text-slate-500" />
									{/if}
								</span>
								<div
									class="flex items-baseline justify-between gap-4 rounded-lg px-3 py-2 hover:bg-muted/40"
								>
									<div class="min-w-0">
										{#if item.survey}
											<p class="text-sm font-medium">Answered “{item.survey.survey_name}”</p>
											<ul class="mt-1 space-y-0.5 text-sm text-muted-foreground">
												{#each item.survey.answers as a, n}
													{#if a}<li><span class="text-xs">Q{n + 1}</span> {a}</li>{/if}
												{/each}
											</ul>
										{:else if item.ev}
											<p class="truncate text-sm font-medium">{label(item.ev)}</p>
											{#if item.ev.event_name !== 'page_view'}
												<p class="truncate text-xs text-muted-foreground">on {path(item.ev.url)}</p>
											{/if}
										{/if}
									</div>
									<span class="flex shrink-0 items-center gap-3">
										{#if item.ev}
											{@const recording = recordingFor(item.ev)}
											{#if recording}
												<a
													href="{base}/replays?project={encodeURIComponent(
														recording.project_id
													)}&recording={encodeURIComponent(recording.recording_id)}"
													class="flex items-center gap-1 rounded-md border border-border px-2 py-0.5 text-xs font-medium transition hover:bg-accent"
												>
													<MonitorPlay class="size-3.5" /> Play recording
												</a>
											{/if}
										{/if}
										<time class="text-xs text-muted-foreground tabular-nums" datetime={item.at}
											>{format(new Date(item.at), 'HH:mm:ss')}</time
										>
									</span>
								</div>
								{#if newSession}
									<p
										class="my-2 ml-3 text-[11px] font-semibold tracking-wider text-muted-foreground uppercase"
									>
										— session started
									</p>
								{/if}
							</li>
						{/each}
					</ol>
				{/each}
			{/if}
		</section>
	</div>
</main>

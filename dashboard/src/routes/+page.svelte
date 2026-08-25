<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { format, subDays, startOfMonth, startOfYear, subMonths } from 'date-fns';
	import {
		fetchOnlineUsers,
		fetchProjects,
		fetchTopStats,
		fetchTimeline,
		fetchTopPages,
		fetchEntryExitPages,
		fetchTopCountries,
		fetchTopSources,
		fetchTopEvents,
		fetchBrowsersDevicesOS
	} from '$lib/api';
	import { Activity, CalendarDays, Filter, RefreshCw, SlidersHorizontal, X } from 'lucide-svelte';
	import {
		Card,
		CardContent,
		CardDescription,
		CardHeader,
		CardTitle
	} from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import TimelineChart from '$lib/components/TimelineChart.svelte';
	import TopItemsList from '$lib/components/TopItemsList.svelte';
	import CountriesPanel from '$lib/components/CountriesPanel.svelte';
	import BrowserPanel from '$lib/components/BrowserPanel.svelte';
	import MetricCard from '$lib/components/MetricCard.svelte';

	let stats: any = $state({
		total_events: 0,
		unique_users: 0,
		total_visits: 0,
		page_views: 0,
		bounce_rate: 0,
		avg_session_duration: 0,
		bot_events: 0,
		human_events: 0,
		bot_users: 0,
		human_users: 0,
		bot_percentage: 0,
		events_change: 0,
		users_change: 0,
		visits_change: 0,
		page_views_change: 0
	});

	let timeline: any = $state({
		timeline: [],
		timeline_format: 'day'
	});

	let comparisonTimeline: any = $state({
		timeline: [],
		timeline_format: 'day'
	});

	let topPages: any = $state({
		top_pages: []
	});

	let entryExitPages: any = $state({
		entry_pages: [],
		exit_pages: []
	});

	let topCountries: any[] = $state([]);
	let topSources: any[] = $state([]);
	let topEvents: any[] = $state([]);

	let browsersDevicesOS: any = $state({
		browsers: [],
		devices: [],
		os: []
	});

	let comparisonStats: any = $state({
		unique_users: 0,
		total_visits: 0,
		page_views: 0,
		total_events: 0,
		bounce_rate: 0,
		avg_session_duration: 0,
		bot_percentage: 0
	});

	let onlineData = $state({
		online_users: 0,
		active_sessions: 0
	});

	let projects: string[] = $state([]);
	let loading = $state(true);
	let statsLoading = $state(false);
	let timelineLoading = $state(false);
	let pagesLoading = $state(false);
	let entryExitLoading = $state(false);
	let countriesLoading = $state(false);
	let sourcesLoading = $state(false);
	let eventsLoading = $state(false);
	let devicesLoading = $state(false);
	let onlineUsersLoading = $state(false);
	let error = $state<string | null>(null);
	let autoRefresh = $state(true);
	let refreshInterval = $state<number | null>(null);
	let refreshIntervalTime = $state(30000); // 30 seconds default
	let lastRefresh = $state(new Date());

	// Pages tab state
	let pagesTab = $state<'all' | 'entry' | 'exit'>('all');

	// Date range presets
	let dateRangePreset = $state('last_7_days');
	const dateRangePresets = [
		{ value: 'today', label: 'Today' },
		{ value: 'yesterday', label: 'Yesterday' },
		{ value: 'last_7_days', label: 'Last 7 days' },
		{ value: 'last_30_days', label: 'Last 30 days' },
		{ value: 'this_month', label: 'This month' },
		{ value: 'last_month', label: 'Last month' },
		{ value: 'last_3_months', label: 'Last 3 months' },
		{ value: 'last_6_months', label: 'Last 6 months' },
		{ value: 'this_year', label: 'This year' },
		{ value: 'custom', label: 'Custom range' }
	];

	// Filter states
	let activeFilters = $state<{
		source: string | null;
		country: string | null;
		browser: string | null;
		device: string | null;
		os: string | null;
		event: string | null;
		project: string | null;
		metric: string | null;
		propertyKey: string | null;
		propertyValue: string | null;
		botFilter: string | null; // 'human', 'bot', or null for all
		page: string | null;
	}>({
		source: null,
		country: null,
		browser: null,
		device: null,
		os: null,
		event: null,
		project: null,
		metric: null, // For filtering by clicked metric card
		propertyKey: null,
		propertyValue: null,
		botFilter: null,
		page: null
	});

	// Default to last 7 days
	let startDate = $state(format(subDays(new Date(), 7), 'yyyy-MM-dd'));
	let endDate = $state(format(new Date(), 'yyyy-MM-dd'));
	let showCustomDateInputs = $state(false);

	// Computed period labels for display
	const currentPeriodLabel = $derived(() => {
		const start = new Date(startDate);
		const end = new Date(endDate);
		return `${format(start, 'd MMM')} - ${format(end, 'd MMM')}`;
	});

	const previousPeriodLabel = $derived(() => {
		const start = new Date(startDate);
		const end = new Date(endDate);
		const duration = end.getTime() - start.getTime();
		const prevEnd = new Date(start.getTime() - 24 * 60 * 60 * 1000);
		const prevStart = new Date(prevEnd.getTime() - duration);
		return `${format(prevStart, 'd MMM')} - ${format(prevEnd, 'd MMM')}`;
	});

	// Apply date range preset
	function applyDateRangePreset(preset: string) {
		applyDateRangePresetWithoutLoad(preset);
		loadStats();
	}

	// URL param helpers
	function updateURLParams() {
		if (typeof window === 'undefined') return;

		const params = new URLSearchParams();
		params.set('range', dateRangePreset);
		if (dateRangePreset === 'custom') {
			params.set('start', startDate);
			params.set('end', endDate);
		}
		if (activeFilters.project) params.set('project', activeFilters.project);
		if (activeFilters.source) params.set('source', activeFilters.source);
		if (activeFilters.country) params.set('country', activeFilters.country);
		if (activeFilters.browser) params.set('browser', activeFilters.browser);
		if (activeFilters.device) params.set('device', activeFilters.device);
		if (activeFilters.os) params.set('os', activeFilters.os);
		if (activeFilters.event) params.set('event', activeFilters.event);
		if (activeFilters.metric) params.set('metric', activeFilters.metric);
		if (activeFilters.propertyKey) params.set('propKey', activeFilters.propertyKey);
		if (activeFilters.propertyValue) params.set('propValue', activeFilters.propertyValue);
		if (activeFilters.botFilter) params.set('botFilter', activeFilters.botFilter);
		if (activeFilters.page) params.set('page', activeFilters.page);
		if (refreshIntervalTime !== 30000) params.set('interval', refreshIntervalTime.toString());

		const newURL = `${window.location.pathname}?${params.toString()}`;
		window.history.replaceState({}, '', newURL);
	}
	function loadFromURLParams() {
		if (typeof window === 'undefined') return;

		const params = new URLSearchParams(window.location.search);

		if (params.has('range')) {
			const range = params.get('range');
			if (range) dateRangePreset = range;
			if (dateRangePreset === 'custom') {
				const start = params.get('start');
				const end = params.get('end');
				if (start) startDate = start;
				if (end) endDate = end;
				showCustomDateInputs = true;
			} else {
				// Don't call loadStats() here - will be called in onMount
				applyDateRangePresetWithoutLoad(dateRangePreset);
			}
		}
		const project = params.get('project');
		const source = params.get('source');
		const country = params.get('country');
		const browser = params.get('browser');
		const device = params.get('device');
		const os = params.get('os');
		const event = params.get('event');
		const metric = params.get('metric');
		const propKey = params.get('propKey');
		const propValue = params.get('propValue');
		const botFilter = params.get('botFilter');
		const page = params.get('page');
		const interval = params.get('interval');

		if (project) activeFilters.project = project;
		if (source) activeFilters.source = source;
		if (country) activeFilters.country = country;
		if (browser) activeFilters.browser = browser;
		if (device) activeFilters.device = device;
		if (os) activeFilters.os = os;
		if (event) activeFilters.event = event;
		if (metric) activeFilters.metric = metric;
		if (propKey) activeFilters.propertyKey = propKey;
		if (propValue) activeFilters.propertyValue = propValue;
		if (botFilter) activeFilters.botFilter = botFilter;
		if (page) activeFilters.page = page;
		if (interval) {
			refreshIntervalTime = parseInt(interval);
		}
	}

	// Apply date range preset without triggering loadStats (for initial load)
	function applyDateRangePresetWithoutLoad(preset: string) {
		const now = new Date();
		const today = format(now, 'yyyy-MM-dd');

		switch (preset) {
			case 'today':
				startDate = today;
				endDate = today;
				break;
			case 'yesterday':
				const yesterday = subDays(now, 1);
				startDate = format(yesterday, 'yyyy-MM-dd');
				endDate = format(yesterday, 'yyyy-MM-dd');
				break;
			case 'last_7_days':
				startDate = format(subDays(now, 7), 'yyyy-MM-dd');
				endDate = today;
				break;
			case 'last_30_days':
				startDate = format(subDays(now, 30), 'yyyy-MM-dd');
				endDate = today;
				break;
			case 'this_month':
				startDate = format(startOfMonth(now), 'yyyy-MM-dd');
				endDate = today;
				break;
			case 'last_month':
				const lastMonthStart = startOfMonth(subMonths(now, 1));
				const lastMonthEnd = subDays(startOfMonth(now), 1);
				startDate = format(lastMonthStart, 'yyyy-MM-dd');
				endDate = format(lastMonthEnd, 'yyyy-MM-dd');
				break;
			case 'last_3_months':
				startDate = format(subMonths(now, 3), 'yyyy-MM-dd');
				endDate = today;
				break;
			case 'last_6_months':
				startDate = format(subMonths(now, 6), 'yyyy-MM-dd');
				endDate = today;
				break;
			case 'this_year':
				startDate = format(startOfYear(now), 'yyyy-MM-dd');
				endDate = today;
				break;
			case 'custom':
				showCustomDateInputs = true;
				return;
		}
		showCustomDateInputs = false;
	}

	async function loadStats() {
		if (loading) {
			// First time loading - show full page loader
			loading = true;
		} else {
			// Subsequent loads - show component-level loaders
			statsLoading = true;
			timelineLoading = true;
			pagesLoading = true;
			entryExitLoading = true;
			countriesLoading = true;
			sourcesLoading = true;
			eventsLoading = true;
			devicesLoading = true;
			onlineUsersLoading = true;
		}

		error = null;

		try {
			// Calculate comparison period once
			const start = new Date(startDate);
			const end = new Date(endDate);
			const duration = end.getTime() - start.getTime();
			const prevEnd = new Date(start.getTime() - 24 * 60 * 60 * 1000);
			const prevStart = new Date(prevEnd.getTime() - duration);

			// Load stats and comparison together (they're displayed together in metric cards)
			Promise.all([
				fetchTopStats(startDate, endDate, activeFilters).catch((err) => {
					console.error('Failed to load top stats:', err);
					return null;
				}),
				fetchTopStats(
					format(prevStart, 'yyyy-MM-dd'),
					format(prevEnd, 'yyyy-MM-dd'),
					activeFilters
				).catch((err) => {
					console.error('Failed to load comparison stats:', err);
					return null;
				})
			]).then(([topStatsData, comparisonStatsData]) => {
				if (topStatsData) stats = topStatsData;
				if (comparisonStatsData) comparisonStats = comparisonStatsData;
				statsLoading = false;
			});

			// Load timeline and comparison timeline together (they render in the same chart)
			Promise.all([
				fetchTimeline(startDate, endDate, activeFilters).catch((err) => {
					console.error('Failed to load timeline:', err);
					return { timeline: [], timeline_format: 'day' };
				}),
				fetchTimeline(
					format(prevStart, 'yyyy-MM-dd'),
					format(prevEnd, 'yyyy-MM-dd'),
					activeFilters
				).catch((err) => {
					console.error('Failed to load comparison timeline:', err);
					return { timeline: [], timeline_format: 'day' };
				})
			]).then(([timelineData, comparisonTimelineData]) => {
				if (timelineData) timeline = timelineData;
				if (comparisonTimelineData) comparisonTimeline = comparisonTimelineData;
				timelineLoading = false;
			});

			// Load pages data
			fetchTopPages(startDate, endDate, 10, activeFilters)
				.then((data) => {
					topPages = data;
					pagesLoading = false;
				})
				.catch((err) => {
					console.error('Failed to load pages:', err);
					pagesLoading = false;
				});

			// Load entry/exit pages
			fetchEntryExitPages(startDate, endDate, 10, activeFilters)
				.then((data) => {
					entryExitPages = data;
					entryExitLoading = false;
				})
				.catch((err) => {
					console.error('Failed to load entry/exit pages:', err);
					entryExitLoading = false;
				});

			// Load countries
			fetchTopCountries(startDate, endDate, 10, activeFilters)
				.then((data) => {
					topCountries = data;
					countriesLoading = false;
				})
				.catch((err) => {
					console.error('Failed to load countries:', err);
					countriesLoading = false;
				});

			// Load sources
			fetchTopSources(startDate, endDate, 10, activeFilters)
				.then((data) => {
					topSources = data;
					sourcesLoading = false;
				})
				.catch((err) => {
					console.error('Failed to load sources:', err);
					sourcesLoading = false;
				});

			// Load events
			fetchTopEvents(startDate, endDate, 10, activeFilters)
				.then((data) => {
					topEvents = data;
					eventsLoading = false;
				})
				.catch((err) => {
					console.error('Failed to load events:', err);
					eventsLoading = false;
				});

			// Load devices
			fetchBrowsersDevicesOS(startDate, endDate, 10, activeFilters)
				.then((data) => {
					browsersDevicesOS = data;
					devicesLoading = false;
				})
				.catch((err) => {
					console.error('Failed to load devices:', err);
					devicesLoading = false;
				});

			// Load online users
			fetchOnlineUsers(5)
				.then((data) => {
					onlineData = data as { online_users: number; active_sessions: number };
					onlineUsersLoading = false;
				})
				.catch((err) => {
					console.error('Failed to load online users:', err);
					onlineUsersLoading = false;
				});

			// Wait for critical data to complete before updating lastRefresh
			await Promise.all([
				new Promise((resolve) => {
					const check = () => {
						if (!statsLoading && !timelineLoading) resolve(true);
						else setTimeout(check, 50);
					};
					check();
				})
			]);

			lastRefresh = new Date();
			updateURLParams();
		} catch (err: any) {
			error = err?.message || 'Failed to load stats';
			// Reset all loading states
			statsLoading = false;
			timelineLoading = false;
			pagesLoading = false;
			entryExitLoading = false;
			countriesLoading = false;
			sourcesLoading = false;
			eventsLoading = false;
			devicesLoading = false;
			onlineUsersLoading = false;
		} finally {
			loading = false;
		}
	}

	function setupAutoRefresh() {
		if (refreshInterval) {
			clearInterval(refreshInterval);
		}

		if (autoRefresh && refreshIntervalTime > 0) {
			refreshInterval = setInterval(() => {
				loadStats();
			}, refreshIntervalTime);
		}
	}

	function handleRefreshIntervalChange(event: Event) {
		const target = event.target as HTMLSelectElement;
		refreshIntervalTime = parseInt(target.value);
		setupAutoRefresh();
		updateURLParams();
	}

	function addFilter(type: keyof typeof activeFilters, value: string) {
		activeFilters[type] = value;
		updateURLParams();
		loadStats();
	}

	function removeFilter(type: keyof typeof activeFilters) {
		activeFilters[type] = null;
		updateURLParams();
		loadStats();
	}

	function clearAllFilters() {
		activeFilters = {
			source: null,
			country: null,
			browser: null,
			device: null,
			os: null,
			event: null,
			project: null,
			metric: null,
			propertyKey: null,
			propertyValue: null,
			botFilter: null,
			page: null
		};
		updateURLParams();
		loadStats();
	}

	onMount(() => {
		loadFromURLParams();
		loadProjects();
		loadStats();
		setupAutoRefresh();
	});

	onDestroy(() => {
		if (refreshInterval) {
			clearInterval(refreshInterval);
		}
	});

	async function loadProjects() {
		try {
			projects = await fetchProjects();
		} catch (err) {
			// Failed to load projects - silently fail, not critical
		}
	}

	function handleDateChange(event: CustomEvent) {
		startDate = event.detail.startDate;
		endDate = event.detail.endDate;
		updateURLParams();
		loadStats();
	}

	// Handle metric card clicks for filtering timeline
	function handleMetricClick(metricType: string) {
		if (activeFilters.metric === metricType) {
			// Toggle off if already selected
			activeFilters.metric = null;
		} else {
			activeFilters.metric = metricType;
		}
		updateURLParams();
		loadStats(); // Reload data with metric filter
	}

	// Check if a metric is selected
	function isMetricSelected(metricType: string) {
		return activeFilters.metric === metricType;
	}

	// Comparison visibility state
	let showComparison = $state(true);
	const hasActiveFilters = $derived(Object.values(activeFilters).some((filter) => filter !== null));
</script>

<div class="dashboard-shell min-h-[calc(100vh-65px)]">
	<main class="mx-auto max-w-[1440px] space-y-6 px-4 py-8 sm:px-6 lg:px-8 lg:py-10">
		<header class="grid gap-6 lg:grid-cols-[1fr_auto] lg:items-end">
			<div>
				<div
					class="mb-3 flex items-center gap-2 text-[11px] font-semibold tracking-[0.18em] text-amber-700 uppercase"
				>
					<Activity class="size-4" /> Analytics observatory
				</div>
				<h1
					class="display-type max-w-4xl text-4xl font-semibold tracking-[-0.035em] text-slate-950 sm:text-5xl"
				>
					{activeFilters.project || 'Your audience'}, in focus.
				</h1>
				<p class="mt-3 max-w-2xl text-sm leading-6 text-slate-600 sm:text-base">
					A quiet, current view of who arrived, what they explored, and where momentum changed.
				</p>
			</div>
			{#if !loading}
				<div
					class="flex items-center gap-3 rounded-2xl border border-slate-200/80 bg-white/85 px-4 py-3 shadow-sm backdrop-blur"
				>
					<span class="relative flex size-3">
						{#if onlineData.online_users > 0}<span
								class="absolute inline-flex size-full animate-ping rounded-full bg-emerald-400 opacity-50 motion-reduce:animate-none"
							></span>{/if}
						<span
							class="relative inline-flex size-3 rounded-full {onlineData.online_users > 0
								? 'bg-emerald-500'
								: 'bg-slate-300'}"
						></span>
					</span>
					<div>
						<p class="text-lg leading-none font-semibold text-slate-950 tabular-nums">
							{onlineData.online_users?.toLocaleString() || '0'}
						</p>
						<p class="mt-1 text-[10px] font-semibold tracking-wider text-slate-500 uppercase">
							Visitors now
						</p>
					</div>
				</div>
			{/if}
		</header>

		<section
			class="rounded-2xl border border-slate-200/90 bg-white/90 p-3 shadow-[0_16px_45px_rgba(15,23,42,0.06)] backdrop-blur"
		>
			<div class="flex flex-wrap items-end gap-3">
				<div
					class="mr-1 hidden size-10 place-items-center rounded-xl bg-slate-950 text-white sm:grid"
				>
					<SlidersHorizontal class="size-4" />
				</div>
				<label class="min-w-40 flex-1 sm:max-w-48">
					<span
						class="mb-1.5 flex items-center gap-1.5 text-[10px] font-semibold tracking-wider text-slate-500 uppercase"
						><CalendarDays class="size-3" /> Period</span
					>
					<select
						class="h-10 w-full rounded-lg border border-slate-200 bg-slate-50 px-3 text-sm font-medium text-slate-800 transition outline-none focus:border-slate-400 focus:bg-white focus:ring-4 focus:ring-slate-900/5"
						bind:value={dateRangePreset}
						onchange={(e: Event) => {
							const target = e.target as HTMLSelectElement;
							applyDateRangePreset(target.value);
						}}
					>
						{#each dateRangePresets as preset}
							<option value={preset.value}>{preset.label}</option>
						{/each}
					</select>
				</label>

				{#if showCustomDateInputs}
					<div
						class="flex h-10 items-center gap-2 rounded-lg border border-slate-200 bg-slate-50 px-2"
					>
						<input
							type="date"
							bind:value={startDate}
							class="bg-transparent text-sm text-slate-700 outline-none"
							onchange={() => loadStats()}
						/>
						<span class="text-xs text-slate-400">to</span>
						<input
							type="date"
							bind:value={endDate}
							class="bg-transparent text-sm text-slate-700 outline-none"
							onchange={() => loadStats()}
						/>
					</div>
				{/if}

				{#if projects.length > 0}
					<label class="min-w-40 flex-1 sm:max-w-48">
						<span
							class="mb-1.5 block text-[10px] font-semibold tracking-wider text-slate-500 uppercase"
							>Project</span
						>
						<select
							class="h-10 w-full rounded-lg border border-slate-200 bg-slate-50 px-3 text-sm font-medium text-slate-800 transition outline-none focus:border-slate-400 focus:bg-white focus:ring-4 focus:ring-slate-900/5"
							value={activeFilters.project || ''}
							onchange={(e: Event) => {
								const target = e.target as HTMLSelectElement;
								if (target.value) {
									addFilter('project', target.value);
								} else {
									removeFilter('project');
								}
							}}
						>
							<option value="">All Projects</option>
							{#each projects as project}
								<option value={project}>{project}</option>
							{/each}
						</select>
					</label>
				{/if}

				<label class="min-w-40 flex-1 sm:max-w-48">
					<span
						class="mb-1.5 block text-[10px] font-semibold tracking-wider text-slate-500 uppercase"
						>Traffic</span
					>
					<select
						class="h-10 w-full rounded-lg border border-slate-200 bg-slate-50 px-3 text-sm font-medium text-slate-800 transition outline-none focus:border-slate-400 focus:bg-white focus:ring-4 focus:ring-slate-900/5"
						value={activeFilters.botFilter || ''}
						onchange={(e: Event) => {
							const target = e.target as HTMLSelectElement;
							if (target.value) {
								addFilter('botFilter', target.value);
							} else {
								removeFilter('botFilter');
							}
						}}
					>
						<option value="">All Traffic</option>
						<option value="human">👤 Human Only</option>
						<option value="bot">🤖 Bots Only</option>
					</select>
				</label>

				<div class="ml-auto flex items-end gap-2">
					<label class="hidden sm:block">
						<span
							class="mb-1.5 block text-[10px] font-semibold tracking-wider text-slate-500 uppercase"
							>Refresh</span
						>
						<select
							class="h-10 rounded-lg border border-slate-200 bg-slate-50 px-3 text-sm font-medium text-slate-800 outline-none"
							value={refreshIntervalTime}
							onchange={handleRefreshIntervalChange}
							aria-label="Auto-refresh interval"
						>
							<option value="0">Manual</option><option value="10000">10 sec</option><option
								value="30000">30 sec</option
							><option value="60000">1 min</option><option value="300000">5 min</option>
						</select>
					</label>
					<Button
						onclick={() => loadStats()}
						class="h-10 rounded-lg bg-slate-950 px-4 text-white hover:bg-slate-800"
					>
						<RefreshCw class="size-4" /> Refresh
					</Button>
				</div>
			</div>
			{#if !loading}
				<div
					class="mt-3 flex items-center justify-end border-t border-slate-100 pt-2 text-[11px] text-slate-400"
				>
					Updated {format(lastRefresh, 'HH:mm:ss')}
				</div>
			{/if}
		</section>

		{#if hasActiveFilters}
			<div
				class="flex flex-wrap items-center gap-2 rounded-xl border border-amber-200/70 bg-amber-50/80 px-4 py-3"
			>
				<span class="mr-1 flex items-center gap-1.5 text-xs font-semibold text-amber-900"
					><Filter class="size-3.5" /> Filtered view</span
				>
				{#if activeFilters.project}
					<Badge variant="secondary" class="gap-1">
						Project: {activeFilters.project}
						<button onclick={() => removeFilter('project')} class="ml-1 hover:text-destructive">
							<X class="h-3 w-3" />
						</button>
					</Badge>
				{/if}
				{#if activeFilters.source}
					<Badge variant="secondary" class="gap-1">
						Source: {activeFilters.source}
						<button onclick={() => removeFilter('source')} class="ml-1 hover:text-destructive">
							<X class="h-3 w-3" />
						</button>
					</Badge>
				{/if}
				{#if activeFilters.country}
					<Badge variant="secondary" class="gap-1">
						Country: {activeFilters.country}
						<button onclick={() => removeFilter('country')} class="ml-1 hover:text-destructive">
							<X class="h-3 w-3" />
						</button>
					</Badge>
				{/if}
				{#if activeFilters.browser}
					<Badge variant="secondary" class="gap-1">
						Browser: {activeFilters.browser}
						<button onclick={() => removeFilter('browser')} class="ml-1 hover:text-destructive">
							<X class="h-3 w-3" />
						</button>
					</Badge>
				{/if}
				{#if activeFilters.device}
					<Badge variant="secondary" class="gap-1">
						Device: {activeFilters.device}
						<button onclick={() => removeFilter('device')} class="ml-1 hover:text-destructive">
							<X class="h-3 w-3" />
						</button>
					</Badge>
				{/if}
				{#if activeFilters.os}
					<Badge variant="secondary" class="gap-1">
						OS: {activeFilters.os}
						<button onclick={() => removeFilter('os')} class="ml-1 hover:text-destructive">
							<X class="h-3 w-3" />
						</button>
					</Badge>
				{/if}
				{#if activeFilters.event}
					<Badge variant="secondary" class="gap-1">
						Event: {activeFilters.event}
						<button onclick={() => removeFilter('event')} class="ml-1 hover:text-destructive">
							<X class="h-3 w-3" />
						</button>
					</Badge>
				{/if}
				{#if activeFilters.metric}
					<Badge variant="secondary" class="gap-1">
						Metric: {activeFilters.metric}
						<button onclick={() => removeFilter('metric')} class="ml-1 hover:text-destructive">
							<X class="h-3 w-3" />
						</button>
					</Badge>
				{/if}
				{#if activeFilters.propertyKey && activeFilters.propertyValue}
					<Badge variant="secondary" class="gap-1">
						Property: {activeFilters.propertyKey}={activeFilters.propertyValue}
						<button
							onclick={() => {
								removeFilter('propertyKey');
								removeFilter('propertyValue');
							}}
							class="ml-1 hover:text-destructive"
						>
							<X class="h-3 w-3" />
						</button>
					</Badge>
				{/if}
				{#if activeFilters.botFilter}
					<Badge variant="secondary" class="gap-1">
						Traffic: {activeFilters.botFilter === 'bot' ? '🤖 Bots Only' : '👤 Human Only'}
						<button onclick={() => removeFilter('botFilter')} class="ml-1 hover:text-destructive">
							<X class="h-3 w-3" />
						</button>
					</Badge>
				{/if}
				{#if activeFilters.page}
					<Badge variant="secondary" class="gap-1">
						Page: {activeFilters.page}
						<button onclick={() => removeFilter('page')} class="ml-1 hover:text-destructive">
							<X class="h-3 w-3" />
						</button>
					</Badge>
				{/if}
				<Button
					variant="ghost"
					size="sm"
					onclick={clearAllFilters}
					class="ml-auto text-amber-900 hover:bg-amber-100">Clear all</Button
				>
			</div>
		{/if}

		{#if loading}
			<div
				class="flex min-h-[420px] items-center justify-center rounded-3xl border border-slate-200/80 bg-white/65 py-20 shadow-sm backdrop-blur"
			>
				<div class="text-center">
					<div
						class="inline-block size-9 animate-spin rounded-full border-[3px] border-solid border-slate-200 border-r-amber-600 motion-reduce:animate-none"
					></div>
					<p class="mt-4 text-sm font-medium text-slate-500">Bringing your signal into focus…</p>
				</div>
			</div>
		{:else if error}
			<Card class="border-destructive/40 bg-white/90 shadow-sm">
				<CardHeader>
					<CardTitle class="text-destructive">Error Loading Data</CardTitle>
					<CardDescription>{error}</CardDescription>
				</CardHeader>
			</Card>
		{:else}
			<section
				class="overflow-hidden rounded-3xl border border-slate-200/90 bg-white/90 shadow-[0_22px_60px_rgba(15,23,42,0.07)] backdrop-blur"
			>
				<div
					class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-100 px-5 py-4 sm:px-6"
				>
					<div>
						<p class="text-[10px] font-semibold tracking-[0.16em] text-amber-700 uppercase">
							Performance
						</p>
						<h2 class="display-type mt-1 text-2xl font-semibold tracking-tight text-slate-950">
							Audience pulse
						</h2>
					</div>
					<p class="text-xs text-slate-400">Select a metric to redraw the timeline</p>
				</div>
				<div class="grid grid-cols-2 gap-px bg-slate-200/80 md:grid-cols-4 xl:grid-cols-8">
					<!-- Unique Visitors -->
					<MetricCard
						label="Unique Visitors"
						currentValue={stats.unique_users || 0}
						previousValue={showComparison ? comparisonStats.unique_users || 0 : null}
						currentPeriod={currentPeriodLabel()}
						previousPeriod={previousPeriodLabel()}
						isSelected={isMetricSelected('users')}
						loading={statsLoading}
						onclick={() => handleMetricClick('users')}
					/>

					<!-- Total Visits -->
					<MetricCard
						label="Total Visits"
						currentValue={stats.total_visits || 0}
						previousValue={showComparison ? comparisonStats.total_visits || 0 : null}
						currentPeriod={currentPeriodLabel()}
						previousPeriod={previousPeriodLabel()}
						isSelected={isMetricSelected('visits')}
						loading={statsLoading}
						onclick={() => handleMetricClick('visits')}
					/>

					<!-- Total Pageviews -->
					<MetricCard
						label="Total Pageviews"
						currentValue={stats.page_views || 0}
						previousValue={showComparison ? comparisonStats.page_views || 0 : null}
						currentPeriod={currentPeriodLabel()}
						previousPeriod={previousPeriodLabel()}
						isSelected={isMetricSelected('page_views')}
						loading={statsLoading}
						onclick={() => handleMetricClick('page_views')}
					/>

					<!-- Total Events -->
					<MetricCard
						label="Total Events"
						currentValue={stats.total_events || 0}
						previousValue={showComparison ? comparisonStats.total_events || 0 : null}
						currentPeriod={currentPeriodLabel()}
						previousPeriod={previousPeriodLabel()}
						isSelected={isMetricSelected('events')}
						loading={statsLoading}
						onclick={() => handleMetricClick('events')}
					/>

					<!-- Views per Visit -->
					<MetricCard
						label="Views per Visit"
						currentValue={stats.total_visits > 0 ? stats.page_views / stats.total_visits : 0}
						previousValue={showComparison && comparisonStats.total_visits > 0
							? comparisonStats.page_views / comparisonStats.total_visits
							: null}
						currentPeriod={currentPeriodLabel()}
						previousPeriod={previousPeriodLabel()}
						formatValue={(val: number) => (val ? val.toFixed(2) : '0.00')}
						isSelected={isMetricSelected('views_per_visit')}
						loading={statsLoading}
						onclick={() => handleMetricClick('views_per_visit')}
					/>

					<!-- Bounce Rate -->
					<MetricCard
						label="Bounce Rate"
						currentValue={stats.bounce_rate || 0}
						previousValue={showComparison ? comparisonStats.bounce_rate || 0 : null}
						currentPeriod={currentPeriodLabel()}
						previousPeriod={previousPeriodLabel()}
						formatValue={(val: number) => (val ? val.toFixed(0) + '%' : '0%')}
						isNegativeBetter={true}
						isSelected={isMetricSelected('bounce_rate')}
						loading={statsLoading}
						onclick={() => handleMetricClick('bounce_rate')}
					/>

					<!-- Visit Duration -->
					<MetricCard
						label="Visit Duration"
						currentValue={stats.avg_session_duration || 0}
						previousValue={showComparison ? comparisonStats.avg_session_duration || 0 : null}
						currentPeriod={currentPeriodLabel()}
						previousPeriod={previousPeriodLabel()}
						formatValue={(val: number) => {
							if (!val) return '0s';
							if (val < 60) return Math.floor(val) + 's';
							if (val < 3600) {
								const minutes = Math.floor(val / 60);
								const seconds = Math.floor(val % 60);
								return `${minutes}m ${seconds}s`;
							}
							const hours = Math.floor(val / 3600);
							const minutes = Math.floor((val % 3600) / 60);
							return `${hours}h ${minutes}m`;
						}}
						isSelected={isMetricSelected('visit_duration')}
						loading={statsLoading}
						onclick={() => handleMetricClick('visit_duration')}
					/>

					<!-- Bot Percentage -->
					<MetricCard
						label="🤖 Bot Traffic"
						currentValue={stats.bot_percentage || 0}
						previousValue={showComparison ? comparisonStats.bot_percentage || 0 : null}
						currentPeriod={currentPeriodLabel()}
						previousPeriod={previousPeriodLabel()}
						formatValue={(val: number) => (val ? val.toFixed(0) + '%' : '0%')}
						isNegativeBetter={true}
						isSelected={activeFilters.botFilter === 'bot'}
						loading={statsLoading}
						onclick={() => {
							if (activeFilters.botFilter === 'bot') {
								removeFilter('botFilter');
							} else {
								addFilter('botFilter', 'bot');
							}
						}}
					/>
				</div>

				<div class="px-2 pt-5 pb-4 sm:px-5">
					{#if timelineLoading}
						<div class="flex h-[300px] items-center justify-center text-muted-foreground">
							<div class="flex flex-col items-center gap-2">
								<div
									class="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent motion-reduce:animate-none"
								></div>
								<p class="text-sm">Loading...</p>
							</div>
						</div>
					{:else}
						<TimelineChart
							data={timeline.timeline || []}
							comparisonData={comparisonTimeline.timeline || []}
							format={timeline.timeline_format || 'day'}
							metric={activeFilters.metric || 'users'}
							bind:showComparison
						/>
					{/if}
				</div>
			</section>

			<div class="grid gap-5 lg:grid-cols-2">
				<div class="space-y-4">
					<Card class="border-slate-200/90 bg-white/90 shadow-sm">
						<CardHeader class="pb-3">
							<div class="flex items-center justify-between">
								<CardTitle class="text-base">Pages</CardTitle>
								<div class="flex gap-1 rounded-lg bg-muted p-1">
									<button
										class="rounded px-3 py-1 text-xs font-medium transition-colors {pagesTab ===
										'all'
											? 'bg-background shadow-sm'
											: 'hover:bg-background/50'}"
										onclick={() => (pagesTab = 'all')}
									>
										All
									</button>
									<button
										class="rounded px-3 py-1 text-xs font-medium transition-colors {pagesTab ===
										'entry'
											? 'bg-background shadow-sm'
											: 'hover:bg-background/50'}"
										onclick={() => (pagesTab = 'entry')}
									>
										Entry
									</button>
									<button
										class="rounded px-3 py-1 text-xs font-medium transition-colors {pagesTab ===
										'exit'
											? 'bg-background shadow-sm'
											: 'hover:bg-background/50'}"
										onclick={() => (pagesTab = 'exit')}
									>
										Exit
									</button>
								</div>
							</div>
						</CardHeader>
						<CardContent>
							{#if pagesLoading || entryExitLoading}
								<div class="flex min-h-[200px] items-center justify-center text-muted-foreground">
									<div class="flex flex-col items-center gap-2">
										<div
											class="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent motion-reduce:animate-none"
										></div>
										<p class="text-xs">Loading...</p>
									</div>
								</div>
							{:else if pagesTab === 'all'}
								<TopItemsList
									items={topPages.top_pages || []}
									labelKey="url"
									maxItems={10}
									valueKey="count"
									showMoreTitle="All Pages"
									onclick={(item: any) => addFilter('page', item.url)}
								/>
							{:else if pagesTab === 'entry'}
								<TopItemsList
									items={entryExitPages.entry_pages || []}
									labelKey="url"
									maxItems={10}
									valueKey="count"
									showMoreTitle="All Entry Pages"
									onclick={(item: any) => addFilter('page', item.url)}
								/>
							{:else if pagesTab === 'exit'}
								<TopItemsList
									items={entryExitPages.exit_pages || []}
									labelKey="url"
									maxItems={10}
									valueKey="count"
									showMoreTitle="All Exit Pages"
									onclick={(item: any) => addFilter('page', item.url)}
								/>
							{/if}
						</CardContent>
					</Card>
				</div>
				<Card class="border-slate-200/90 bg-white/90 shadow-sm">
					<CardHeader class="pb-3">
						<CardTitle class="text-base">Locations</CardTitle>
					</CardHeader>
					<CardContent>
						<CountriesPanel
							countries={topCountries || []}
							onclick={(item: any) => addFilter('country', item.name)}
							loading={countriesLoading}
						/>
					</CardContent>
				</Card>
			</div>

			<div class="grid gap-5 lg:grid-cols-3">
				<Card class="border-slate-200/90 bg-white/90 shadow-sm">
					<CardHeader class="pb-3">
						<CardTitle class="text-base">Top Events</CardTitle>
					</CardHeader>
					<CardContent>
						{#if eventsLoading}
							<div class="flex min-h-[150px] items-center justify-center text-muted-foreground">
								<div class="flex flex-col items-center gap-2">
									<div
										class="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent motion-reduce:animate-none"
									></div>
									<p class="text-xs">Loading...</p>
								</div>
							</div>
						{:else}
							<TopItemsList
								items={topEvents || []}
								labelKey="name"
								valueKey="count"
								maxItems={8}
								type="event"
								showMoreTitle="All Events"
								onclick={(item: any) => addFilter('event', item.name)}
							/>
						{/if}
					</CardContent>
				</Card>

				<Card class="border-slate-200/90 bg-white/90 shadow-sm">
					<CardHeader class="pb-3">
						<CardTitle class="text-base">Top Sources</CardTitle>
					</CardHeader>
					<CardContent>
						{#if sourcesLoading}
							<div class="flex min-h-[150px] items-center justify-center text-muted-foreground">
								<div class="flex flex-col items-center gap-2">
									<div
										class="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent motion-reduce:animate-none"
									></div>
									<p class="text-xs">Loading...</p>
								</div>
							</div>
						{:else}
							<TopItemsList
								items={topSources || []}
								labelKey="name"
								valueKey="count"
								maxItems={8}
								type="source"
								showMoreTitle="All Sources"
								onclick={(item: any) => addFilter('source', item.name)}
							/>
						{/if}
					</CardContent>
				</Card>

				<Card class="border-slate-200/90 bg-white/90 shadow-sm">
					<CardHeader class="pb-3">
						<CardTitle class="text-base">Devices</CardTitle>
					</CardHeader>
					<CardContent>
						<BrowserPanel
							browsers={browsersDevicesOS.browsers || []}
							devices={browsersDevicesOS.devices || []}
							operatingSystems={browsersDevicesOS.os || []}
							onBrowserClick={(item: any) => addFilter('browser', item.name)}
							onDeviceClick={(item: any) => addFilter('device', item.name)}
							onOsClick={(item: any) => addFilter('os', item.name)}
							loading={devicesLoading}
						/>
					</CardContent>
				</Card>
			</div>
		{/if}
	</main>
</div>

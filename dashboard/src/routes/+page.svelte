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
	import { Filter, RefreshCw, X } from 'lucide-svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
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

<main class="mx-auto max-w-[1440px] space-y-5 px-4 py-6 sm:px-6 lg:px-8">
	<PageHeader
		title={activeFilters.project || 'Overview'}
		description="Visitors, engagement and traffic sources for the selected period."
	>
		{#if !loading}
			<span
				class="inline-flex h-9 items-center gap-2 rounded-md border border-border bg-card px-3 text-sm shadow-xs"
			>
				<span class="relative flex size-2">
					{#if onlineData.online_users > 0}<span
							class="absolute inline-flex size-full animate-ping rounded-full bg-success opacity-60 motion-reduce:animate-none"
						></span>{/if}
					<span
						class="relative inline-flex size-2 rounded-full {onlineData.online_users > 0
							? 'bg-success'
							: 'bg-slate-300'}"
					></span>
				</span>
				<span class="font-semibold tabular-nums"
					>{onlineData.online_users?.toLocaleString() || '0'}</span
				>
				<span class="text-muted-foreground">online now</span>
			</span>
		{/if}
	</PageHeader>

	<div class="flex flex-wrap items-center gap-2">
		<select
			class="field font-medium"
			aria-label="Period"
			bind:value={dateRangePreset}
			onchange={(e: Event) => applyDateRangePreset((e.target as HTMLSelectElement).value)}
		>
			{#each dateRangePresets as preset}
				<option value={preset.value}>{preset.label}</option>
			{/each}
		</select>

		{#if showCustomDateInputs}
			<div class="field flex items-center gap-2">
				<input
					type="date"
					bind:value={startDate}
					aria-label="Start date"
					class="bg-transparent outline-none"
					onchange={() => loadStats()}
				/>
				<span class="text-muted-foreground">–</span>
				<input
					type="date"
					bind:value={endDate}
					aria-label="End date"
					class="bg-transparent outline-none"
					onchange={() => loadStats()}
				/>
			</div>
		{/if}

		{#if projects.length > 0}
			<select
				class="field"
				aria-label="Project"
				value={activeFilters.project || ''}
				onchange={(e: Event) => {
					const value = (e.target as HTMLSelectElement).value;
					if (value) addFilter('project', value);
					else removeFilter('project');
				}}
			>
				<option value="">All projects</option>
				{#each projects as project}
					<option value={project}>{project}</option>
				{/each}
			</select>
		{/if}

		<select
			class="field"
			aria-label="Traffic type"
			value={activeFilters.botFilter || ''}
			onchange={(e: Event) => {
				const value = (e.target as HTMLSelectElement).value;
				if (value) addFilter('botFilter', value);
				else removeFilter('botFilter');
			}}
		>
			<option value="">All traffic</option>
			<option value="human">Humans only</option>
			<option value="bot">Bots only</option>
		</select>

		<div class="ml-auto flex items-center gap-2">
			{#if !loading}<span class="hidden text-xs text-muted-foreground sm:inline"
					>Updated {format(lastRefresh, 'HH:mm:ss')}</span
				>{/if}
			<select
				class="field hidden sm:block"
				value={refreshIntervalTime}
				onchange={handleRefreshIntervalChange}
				aria-label="Auto-refresh interval"
			>
				<option value="0">Auto-refresh off</option><option value="10000">Every 10s</option><option
					value="30000">Every 30s</option
				><option value="60000">Every 1m</option><option value="300000">Every 5m</option>
			</select>
			<Button variant="outline" onclick={() => loadStats()} aria-label="Refresh"
				><RefreshCw class="size-4" /><span class="hidden sm:inline">Refresh</span></Button
			>
		</div>
	</div>

	{#if hasActiveFilters}
		<div class="flex flex-wrap items-center gap-2">
			<span class="flex items-center gap-1.5 text-xs font-medium text-muted-foreground"
				><Filter class="size-3.5" /> Filters</span
			>
			{#each [['project', 'Project'], ['source', 'Source'], ['country', 'Country'], ['browser', 'Browser'], ['device', 'Device'], ['os', 'OS'], ['event', 'Event'], ['metric', 'Metric'], ['botFilter', 'Traffic'], ['page', 'Page']] as [key, label]}
				{@const value = activeFilters[key as keyof typeof activeFilters]}
				{#if value}
					<span
						class="inline-flex h-7 items-center gap-1 rounded-md border border-border bg-card pr-1 pl-2.5 text-xs shadow-xs"
					>
						<span class="text-muted-foreground">{label}</span>
						<span class="max-w-60 truncate font-medium"
							>{key === 'botFilter' ? (value === 'bot' ? 'Bots only' : 'Humans only') : value}</span
						>
						<button
							onclick={() => removeFilter(key as any)}
							aria-label="Remove {label} filter"
							class="grid size-5 place-items-center rounded text-muted-foreground hover:bg-muted hover:text-foreground"
							><X class="size-3" /></button
						>
					</span>
				{/if}
			{/each}
			{#if activeFilters.propertyKey && activeFilters.propertyValue}
				<span
					class="inline-flex h-7 items-center gap-1 rounded-md border border-border bg-card pr-1 pl-2.5 text-xs shadow-xs"
				>
					<span class="text-muted-foreground">Property</span>
					<span class="max-w-60 truncate font-medium"
						>{activeFilters.propertyKey}={activeFilters.propertyValue}</span
					>
					<button
						onclick={() => {
							removeFilter('propertyKey');
							removeFilter('propertyValue');
						}}
						aria-label="Remove property filter"
						class="grid size-5 place-items-center rounded text-muted-foreground hover:bg-muted hover:text-foreground"
						><X class="size-3" /></button
					>
				</span>
			{/if}
			<button
				onclick={clearAllFilters}
				class="text-xs font-medium text-muted-foreground hover:text-foreground">Clear all</button
			>
		</div>
	{/if}

	{#if loading}
		<div class="panel flex min-h-[420px] items-center justify-center">
			<div class="flex flex-col items-center gap-3 text-sm text-muted-foreground">
				<div
					class="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent motion-reduce:animate-none"
				></div>
				 Loading analytics…
			</div>
		</div>
	{:else if error}
		<div
			class="rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
			role="alert"
		>
			<p class="font-medium">Couldn't load analytics</p>
			<p class="mt-0.5">{error}</p>
		</div>
	{:else}
		<section class="panel overflow-hidden">
			<div
				class="grid grid-cols-2 gap-px border-b border-border bg-border md:grid-cols-4 xl:grid-cols-8"
			>
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
					label="Bot Traffic"
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

			<div class="px-2 pt-4 pb-6 sm:px-5">
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

		<div class="grid gap-4 lg:grid-cols-2">
			<div class="space-y-4">
				<Card>
					<CardHeader class="pb-3">
						<div class="flex items-center justify-between">
							<CardTitle>Pages</CardTitle>
							<div class="flex gap-0.5 rounded-md bg-muted p-0.5">
								<button
									class="rounded px-3 py-1 text-xs font-medium transition-colors {pagesTab === 'all'
										? 'bg-card text-foreground shadow-xs'
										: 'text-muted-foreground hover:text-foreground'}"
									onclick={() => (pagesTab = 'all')}
								>
									All
								</button>
								<button
									class="rounded px-3 py-1 text-xs font-medium transition-colors {pagesTab ===
									'entry'
										? 'bg-card text-foreground shadow-xs'
										: 'text-muted-foreground hover:text-foreground'}"
									onclick={() => (pagesTab = 'entry')}
								>
									Entry
								</button>
								<button
									class="rounded px-3 py-1 text-xs font-medium transition-colors {pagesTab ===
									'exit'
										? 'bg-card text-foreground shadow-xs'
										: 'text-muted-foreground hover:text-foreground'}"
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
			<Card>
				<CardHeader class="pb-3">
					<CardTitle>Locations</CardTitle>
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

		<div class="grid gap-4 lg:grid-cols-3">
			<Card>
				<CardHeader class="pb-3">
					<CardTitle>Top Events</CardTitle>
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

			<Card>
				<CardHeader class="pb-3">
					<CardTitle>Top Sources</CardTitle>
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

			<Card>
				<CardHeader class="pb-3">
					<CardTitle>Devices</CardTitle>
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

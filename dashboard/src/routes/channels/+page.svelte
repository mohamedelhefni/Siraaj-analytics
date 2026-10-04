<script lang="ts">
	import PageHeader from '$lib/components/PageHeader.svelte';
	import { onMount, onDestroy } from 'svelte';
	import { format, subDays, startOfMonth, startOfYear, subMonths } from 'date-fns';
	import { fetchChannels, fetchProjects, fetchOnlineUsers } from '$lib/api';
	import { RefreshCw, X, BarChart3, PieChart } from 'lucide-svelte';
	import {
		Card,
		CardContent,
		CardDescription,
		CardHeader,
		CardTitle
	} from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import ChannelsChart from '$lib/components/ChannelsChart.svelte';

	// Channel colors - modern palette matching the chart
	const channelColors: Record<string, string> = {
		Direct: '#3b82f6', // Blue
		Paid: '#ef4444', // Red
		'google.com': '#4285f4', // Google Blue
		'facebook.com': '#1877f2', // Facebook Blue
		'twitter.com': '#1da1f2', // Twitter Blue
		'linkedin.com': '#0077b5', // LinkedIn Blue
		'youtube.com': '#ff0000', // YouTube Red
		'instagram.com': '#e4405f', // Instagram Pink
		'github.com': '#24292e', // GitHub Dark
		'reddit.com': '#ff4500', // Reddit Orange
		'pinterest.com': '#bd081c', // Pinterest Red
		'tiktok.com': '#000000', // TikTok Black
		Unknown: '#6b7280' // Gray
	};

	// Get color for any domain or fallback to a hash-based color
	function getChannelColor(channel: string): string {
		if (channelColors[channel]) {
			return channelColors[channel];
		}

		// Generate a consistent color based on channel name
		let hash = 0;
		for (let i = 0; i < channel.length; i++) {
			const char = channel.charCodeAt(i);
			hash = (hash << 5) - hash + char;
			hash = hash & hash; // Convert to 32bit integer
		}

		const hue = Math.abs(hash) % 360;
		return `hsl(${hue}, 65%, 50%)`;
	}

	let channels: any[] = $state([]);
	let projects: string[] = $state([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let autoRefresh = $state(false);
	let refreshInterval = $state<number | null>(null);
	let refreshIntervalTime = $state(30000); // 30 seconds default
	let lastRefresh = $state(new Date());
	let chartType = $state<'bar' | 'pie'>('pie');

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
		{ value: 'this_year', label: 'This year' },
		{ value: 'custom', label: 'Custom range' }
	];

	// Filter states
	let activeFilters = $state<{
		project: string | null;
		country: string | null;
		browser: string | null;
		device: string | null;
		os: string | null;
		botFilter: string | null;
	}>({
		project: null,
		country: null,
		browser: null,
		device: null,
		os: null,
		botFilter: null
	});

	// Default to last 7 days
	let startDate = $state(format(subDays(new Date(), 7), 'yyyy-MM-dd'));
	let endDate = $state(format(new Date(), 'yyyy-MM-dd'));
	let showCustomDateInputs = $state(false);

	// Apply date range preset
	function applyDateRangePreset(preset: string) {
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
			case 'this_year':
				startDate = format(startOfYear(now), 'yyyy-MM-dd');
				endDate = today;
				break;
			case 'custom':
				showCustomDateInputs = true;
				return;
		}
		showCustomDateInputs = false;
		loadChannels();
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
		if (activeFilters.country) params.set('country', activeFilters.country);
		if (activeFilters.browser) params.set('browser', activeFilters.browser);
		if (activeFilters.device) params.set('device', activeFilters.device);
		if (activeFilters.os) params.set('os', activeFilters.os);
		if (activeFilters.botFilter) params.set('botFilter', activeFilters.botFilter);
		if (chartType !== 'pie') params.set('chart', chartType);

		const newURL = `${window.location.pathname}?${params.toString()}`;
		window.history.replaceState({}, '', newURL);
	}

	function loadFromURLParams() {
		if (typeof window === 'undefined') return;

		const params = new URLSearchParams(window.location.search);

		if (params.has('range')) {
			const range = params.get('range');
			if (range) dateRangePreset = range;
		}

		const project = params.get('project');
		const country = params.get('country');
		const browser = params.get('browser');
		const device = params.get('device');
		const os = params.get('os');
		const botFilter = params.get('botFilter');
		const chart = params.get('chart');

		if (project) activeFilters.project = project;
		if (country) activeFilters.country = country;
		if (browser) activeFilters.browser = browser;
		if (device) activeFilters.device = device;
		if (os) activeFilters.os = os;
		if (botFilter) activeFilters.botFilter = botFilter;
		if (chart === 'bar') chartType = 'bar';
	}

	async function loadChannels() {
		loading = true;
		error = null;

		try {
			const data = await fetchChannels(startDate, endDate, activeFilters);
			channels = data || [];
			lastRefresh = new Date();
			updateURLParams();
		} catch (err: any) {
			error = err?.message || 'Failed to load channels';
		} finally {
			loading = false;
		}
	}

	async function loadProjects() {
		try {
			projects = await fetchProjects();
		} catch (err) {
			// Failed to load projects - silently fail
		}
	}

	function setupAutoRefresh() {
		if (refreshInterval) {
			clearInterval(refreshInterval);
		}

		if (autoRefresh && refreshIntervalTime > 0) {
			refreshInterval = setInterval(() => {
				loadChannels();
			}, refreshIntervalTime);
		}
	}

	function handleRefreshIntervalChange(event: Event) {
		const target = event.target as HTMLSelectElement;
		refreshIntervalTime = parseInt(target.value);
		setupAutoRefresh();
	}

	function addFilter(type: keyof typeof activeFilters, value: string) {
		activeFilters[type] = value;
		loadChannels();
	}

	function removeFilter(type: keyof typeof activeFilters) {
		activeFilters[type] = null;
		loadChannels();
	}

	function clearAllFilters() {
		activeFilters = {
			project: null,
			country: null,
			browser: null,
			device: null,
			os: null,
			botFilter: null
		};
		loadChannels();
	}

	// Calculate totals
	let totals = $derived(
		channels.reduce(
			(acc, channel) => ({
				total_events: acc.total_events + (channel.total_events || 0),
				unique_users: acc.unique_users + (channel.unique_users || 0),
				total_visits: acc.total_visits + (channel.total_visits || 0),
				page_views: acc.page_views + (channel.page_views || 0)
			}),
			{ total_events: 0, unique_users: 0, total_visits: 0, page_views: 0 }
		)
	);

	onMount(() => {
		loadFromURLParams();
		loadProjects();
		loadChannels();
		setupAutoRefresh();
	});

	onDestroy(() => {
		if (refreshInterval) {
			clearInterval(refreshInterval);
		}
	});
</script>

<div class="mx-auto max-w-[1440px] space-y-5 px-4 py-6 sm:px-6 lg:px-8">
	<PageHeader
		title="Channels"
		description="Where your visitors come from, grouped by acquisition channel and source."
	/>
	<!-- Controls Row -->
	<div class="flex flex-wrap items-center gap-3">
		<!-- Date Range Selector -->
		<div class="flex items-center gap-2">
			<span class="text-sm font-medium">Period:</span>
			<select
				class="flex h-9 rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm focus-visible:ring-1 focus-visible:ring-ring focus-visible:outline-none"
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
		</div>

		<!-- Custom Date Inputs -->
		{#if showCustomDateInputs}
			<div class="flex items-center gap-2 rounded-lg border p-2">
				<input
					type="date"
					bind:value={startDate}
					class="border-none text-sm focus:outline-none"
					onchange={() => loadChannels()}
				/>
				<span class="text-muted-foreground">to</span>
				<input
					type="date"
					bind:value={endDate}
					class="border-none text-sm focus:outline-none"
					onchange={() => loadChannels()}
				/>
			</div>
		{/if}

		<!-- Project Selector -->
		{#if projects.length > 0}
			<div class="flex items-center gap-2">
				<span class="text-sm font-medium">Project:</span>
				<select
					class="flex h-9 rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm focus-visible:ring-1 focus-visible:ring-ring focus-visible:outline-none"
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
			</div>
		{/if}

		<!-- Bot Filter -->
		<div class="flex items-center gap-2">
			<span class="text-sm font-medium">Traffic:</span>
			<select
				class="flex h-9 rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm focus-visible:ring-1 focus-visible:ring-ring focus-visible:outline-none"
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
		</div>

		<!-- Chart Type Toggle -->
		<div class="ml-auto flex items-center gap-2">
			<span class="text-sm font-medium">Chart:</span>
			<div class="flex gap-1 rounded-lg bg-muted p-1">
				<button
					class="rounded px-3 py-1 text-xs font-medium transition-colors {chartType === 'pie'
						? 'bg-background shadow-sm'
						: 'hover:bg-background/50'}"
					onclick={() => {
						chartType = 'pie';
						updateURLParams();
					}}
				>
					<PieChart class="inline h-4 w-4" />
				</button>
				<button
					class="rounded px-3 py-1 text-xs font-medium transition-colors {chartType === 'bar'
						? 'bg-background shadow-sm'
						: 'hover:bg-background/50'}"
					onclick={() => {
						chartType = 'bar';
						updateURLParams();
					}}
				>
					<BarChart3 class="inline h-4 w-4" />
				</button>
			</div>
		</div>

		<!-- Refresh Controls -->
		<div class="flex items-center gap-2">
			<Button variant="outline" size="sm" onclick={() => loadChannels()} class="gap-2">
				<RefreshCw class="h-4 w-4" />
				Refresh
			</Button>

			{#if !loading}
				<span class="text-xs whitespace-nowrap text-muted-foreground">
					Updated {format(lastRefresh, 'HH:mm:ss')}
				</span>
			{/if}
		</div>
	</div>

	<!-- Active Filters -->
	{#if Object.values(activeFilters).some((v) => v !== null)}
		<div class="flex flex-wrap items-center gap-2">
			<span class="text-sm text-muted-foreground">Active Filters:</span>
			{#if activeFilters.project}
				<Badge variant="secondary" class="gap-1">
					Project: {activeFilters.project}
					<button onclick={() => removeFilter('project')} class="ml-1 hover:text-destructive">
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
			<Button variant="ghost" size="sm" onclick={clearAllFilters}>Clear All</Button>
		</div>
	{/if}

	{#if loading}
		<div class="flex items-center justify-center py-20">
			<div class="text-center">
				<div
					class="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-current border-r-transparent motion-reduce:animate-[spin_1.5s_linear_infinite]"
				></div>
				<p class="mt-4 text-muted-foreground">Loading channel data...</p>
			</div>
		</div>
	{:else if error}
		<Card class="border-destructive">
			<CardHeader>
				<CardTitle class="text-destructive">Error Loading Data</CardTitle>
				<CardDescription>{error}</CardDescription>
			</CardHeader>
		</Card>
	{:else}
		<!-- Summary Cards -->
		<div class="grid gap-4 md:grid-cols-4">
			<Card>
				<CardHeader class="pb-2">
					<CardDescription>Total Events</CardDescription>
					<CardTitle class="text-2xl">{totals.total_events.toLocaleString()}</CardTitle>
				</CardHeader>
			</Card>
			<Card>
				<CardHeader class="pb-2">
					<CardDescription>Unique Users</CardDescription>
					<CardTitle class="text-2xl">{totals.unique_users.toLocaleString()}</CardTitle>
				</CardHeader>
			</Card>
			<Card>
				<CardHeader class="pb-2">
					<CardDescription>Total Visits</CardDescription>
					<CardTitle class="text-2xl">{totals.total_visits.toLocaleString()}</CardTitle>
				</CardHeader>
			</Card>
			<Card>
				<CardHeader class="pb-2">
					<CardDescription>Page Views</CardDescription>
					<CardTitle class="text-2xl">{totals.page_views.toLocaleString()}</CardTitle>
				</CardHeader>
			</Card>
		</div>

		<!-- Main Chart -->
		<Card>
			<CardHeader>
				<CardTitle>Channel Distribution</CardTitle>
				<CardDescription>
					Traffic breakdown by acquisition channel for {format(new Date(startDate), 'd MMM')} - {format(
						new Date(endDate),
						'd MMM'
					)}
				</CardDescription>
			</CardHeader>
			<CardContent>
				{#if channels.length === 0}
					<div class="flex h-[400px] items-center justify-center text-muted-foreground">
						<p>No channel data available for the selected period.</p>
					</div>
				{:else}
					<ChannelsChart data={channels} {chartType} />
				{/if}
			</CardContent>
		</Card>

		<!-- Detailed Table -->
		<Card>
			<CardHeader>
				<CardTitle>Channel Breakdown</CardTitle>
				<CardDescription>Detailed metrics for each traffic channel</CardDescription>
			</CardHeader>
			<CardContent>
				<div class="overflow-x-auto">
					<table class="w-full text-sm">
						<thead>
							<tr class="border-b">
								<th class="pb-2 text-left font-medium">Channel</th>
								<th class="pb-2 text-right font-medium">Events</th>
								<th class="pb-2 text-right font-medium">Users</th>
								<th class="pb-2 text-right font-medium">Visits</th>
								<th class="pb-2 text-right font-medium">Page Views</th>
								<th class="pb-2 text-right font-medium">Views/Visit</th>
								<th class="pb-2 text-right font-medium">% of Total</th>
							</tr>
						</thead>
						<tbody>
							{#each channels as channel, i}
								<tr class="border-b last:border-0">
									<td class="py-3 font-medium">
										<div class="flex items-center gap-2">
											<div
												class="h-3 w-3 rounded-full"
												style="background-color: {getChannelColor(channel.channel || 'Unknown')};"
											></div>
											{channel.channel || 'Unknown'}
										</div>
									</td>
									<td class="py-3 text-right">{channel.total_events?.toLocaleString() || 0}</td>
									<td class="py-3 text-right">{channel.unique_users?.toLocaleString() || 0}</td>
									<td class="py-3 text-right">{channel.total_visits?.toLocaleString() || 0}</td>
									<td class="py-3 text-right">{channel.page_views?.toLocaleString() || 0}</td>
									<td class="py-3 text-right">
										{channel.conversion_rate ? channel.conversion_rate.toFixed(2) : '0.00'}
									</td>
									<td class="py-3 text-right">
										{((channel.total_events / totals.total_events) * 100).toFixed(1)}%
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</CardContent>
		</Card>
	{/if}
</div>

<script>
	import { TrendingUp, TrendingDown, Minus } from 'lucide-svelte';
	import { Card } from '$lib/components/ui/card';
	import { formatCompactNumber as formatNum } from '$lib/utils/formatters.js';

	let { currentStats, previousStats, loading = false } = $props();

	// Calculate percentage change
	function calculateChange(current, previous) {
		if (!previous || previous === 0) return 0;
		return ((current - previous) / previous) * 100;
	}

	// Format number for display
	function formatNumber(num) {
		if (!num && num !== 0) return '0';
		return formatNum(num);
	}

	// Format duration (seconds to readable format)
	function formatDuration(seconds) {
		if (!seconds) return '0s';
		if (seconds < 60) {
			return Math.floor(seconds) + 's';
		} else if (seconds < 3600) {
			const minutes = Math.floor(seconds / 60);
			const secs = Math.floor(seconds % 60);
			return `${minutes}m ${secs}s`;
		} else {
			const hours = Math.floor(seconds / 3600);
			const minutes = Math.floor((seconds % 3600) / 60);
			return `${hours}h ${minutes}m`;
		}
	}

	// Get trend icon based on change
	function getTrendIcon(change) {
		if (change > 0) return TrendingUp;
		if (change < 0) return TrendingDown;
		return Minus;
	}

	// Get trend color based on change and metric type
	function getTrendColor(change, isNegativeBetter = false) {
		if (change === 0) return 'text-slate-600';

		if (isNegativeBetter) {
			// For metrics like bounce rate, lower is better
			return change < 0 ? 'text-green-600' : 'text-red-600';
		} else {
			// For most metrics, higher is better
			return change > 0 ? 'text-green-600' : 'text-red-600';
		}
	}

	// Prepare comparison data
	const metrics = $derived([
		{
			label: 'Unique Visitors',
			current: currentStats.unique_users || 0,
			previous: previousStats.unique_users || 0,
			change: calculateChange(currentStats.unique_users, previousStats.unique_users),
			format: formatNumber
		},
		{
			label: 'Total Visits',
			current: currentStats.total_visits || 0,
			previous: previousStats.total_visits || 0,
			change: calculateChange(currentStats.total_visits, previousStats.total_visits),
			format: formatNumber
		},
		{
			label: 'Page Views',
			current: currentStats.page_views || 0,
			previous: previousStats.page_views || 0,
			change: calculateChange(currentStats.page_views, previousStats.page_views),
			format: formatNumber
		},
		{
			label: 'Total Events',
			current: currentStats.total_events || 0,
			previous: previousStats.total_events || 0,
			change: calculateChange(currentStats.total_events, previousStats.total_events),
			format: formatNumber
		},
		{
			label: 'Bounce Rate',
			current: currentStats.bounce_rate || 0,
			previous: previousStats.bounce_rate || 0,
			change: calculateChange(currentStats.bounce_rate, previousStats.bounce_rate),
			format: (val) => (val ? val.toFixed(1) + '%' : '0%'),
			isNegativeBetter: true
		},
		{
			label: 'Avg. Session',
			current: currentStats.avg_session_duration || 0,
			previous: previousStats.avg_session_duration || 0,
			change: calculateChange(
				currentStats.avg_session_duration,
				previousStats.avg_session_duration
			),
			format: formatDuration
		},
		{
			label: 'Views/Visit',
			current:
				currentStats.total_visits > 0 ? currentStats.page_views / currentStats.total_visits : 0,
			previous:
				previousStats.total_visits > 0 ? previousStats.page_views / previousStats.total_visits : 0,
			change: calculateChange(
				currentStats.total_visits > 0 ? currentStats.page_views / currentStats.total_visits : 0,
				previousStats.total_visits > 0 ? previousStats.page_views / previousStats.total_visits : 0
			),
			format: (val) => (val ? val.toFixed(2) : '0.00')
		},
		{
			label: 'Bot Traffic',
			current: currentStats.bot_percentage || 0,
			previous: previousStats.bot_percentage || 0,
			change: calculateChange(currentStats.bot_percentage, previousStats.bot_percentage),
			format: (val) => (val ? val.toFixed(1) + '%' : '0%'),
			isNegativeBetter: true
		}
	]);
</script>

{#if loading}
	<Card class="mb-4">
		<div class="text-muted-foreground flex items-center justify-center p-6">
			<div class="flex flex-col items-center gap-2">
				<div
					class="border-primary h-6 w-6 animate-spin rounded-full border-2 border-t-transparent"
				></div>
				<p class="text-sm">Loading comparison data...</p>
			</div>
		</div>
	</Card>
{:else}
	<Card class="mb-4">
		<div class="p-4">
			<div class="mb-3 flex items-center justify-between">
				<h3 class="text-sm font-semibold text-slate-700">Period Comparison</h3>
				<span class="text-xs text-slate-500">Current vs Previous Period</span>
			</div>
			<div class="grid grid-cols-2 gap-3 md:grid-cols-4 lg:grid-cols-8">
				{#each metrics as metric}
					<div class="border-border rounded-lg border bg-white p-3">
						<div class="text-muted-foreground mb-1 text-xs font-medium uppercase tracking-wide">
							{metric.label}
						</div>
						<div class="mb-1 text-lg font-bold">{metric.format(metric.current)}</div>

						<!-- Previous Period Value -->
						<div class="text-muted-foreground mb-1 text-xs">
							Was: {metric.format(metric.previous)}
						</div>

						<!-- Change Indicator -->
						{#if metric.change !== 0}
							{@const TrendIcon = getTrendIcon(metric.change)}
							<div
								class="text-xs {getTrendColor(
									metric.change,
									metric.isNegativeBetter
								)} flex items-center gap-1"
							>
								<TrendIcon class="h-3 w-3" />
								<span class="font-medium">{Math.abs(metric.change).toFixed(1)}%</span>
							</div>
						{:else}
							<div class="flex items-center gap-1 text-xs text-slate-600">
								<Minus class="h-3 w-3" />
								<span class="font-medium">No change</span>
							</div>
						{/if}
					</div>
				{/each}
			</div>
		</div>
	</Card>
{/if}

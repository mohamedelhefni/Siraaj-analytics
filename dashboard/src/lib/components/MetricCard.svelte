<script lang="ts">
	import { TrendingUp, TrendingDown, Minus } from 'lucide-svelte';
	import { formatCompactNumber } from '$lib/utils/formatters.js';

	let {
		label,
		currentValue,
		previousValue = null,
		currentPeriod = '',
		previousPeriod = '',
		formatValue = (val: number) => formatCompactNumber(val),
		isSelected = false,
		isNegativeBetter = false,
		loading = false,
		onclick = () => {}
	}: {
		label: string;
		currentValue: number;
		previousValue?: number | null;
		currentPeriod?: string;
		previousPeriod?: string;
		formatValue?: (val: number) => string;
		isSelected?: boolean;
		isNegativeBetter?: boolean;
		loading?: boolean;
		onclick?: () => void;
	} = $props();

	const change = $derived(() => {
		if (!previousValue || previousValue === 0) return 0;
		return ((currentValue - previousValue) / previousValue) * 100;
	});

	function getTrendIcon(changeVal: number) {
		if (changeVal > 0) return TrendingUp;
		if (changeVal < 0) return TrendingDown;
		return Minus;
	}

	function getTrendColor(changeVal: number) {
		if (changeVal === 0) return 'text-muted-foreground';
		const isImprovement = isNegativeBetter ? changeVal < 0 : changeVal > 0;
		return isImprovement ? 'text-success' : 'text-destructive';
	}
</script>

<button
	class="relative flex min-h-28 flex-col bg-card px-4 py-4 text-left transition-colors focus-visible:z-10 focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none focus-visible:ring-inset sm:px-5 {isSelected
		? 'bg-muted/60 after:absolute after:inset-x-0 after:top-0 after:h-0.5 after:bg-primary'
		: 'hover:bg-muted/40'}"
	{onclick}
	disabled={loading}
	aria-pressed={isSelected}
>
	<span
		class="truncate text-xs font-medium {isSelected ? 'text-foreground' : 'text-muted-foreground'}"
		>{label}</span
	>

	{#if loading}
		<div class="mt-2 h-7 w-20 animate-pulse rounded bg-muted"></div>
		<div class="mt-2 h-3 w-14 animate-pulse rounded bg-muted"></div>
	{:else}
		<span class="mt-1.5 text-2xl font-semibold tracking-tight text-foreground tabular-nums"
			>{formatValue(currentValue)}</span
		>
		{#if previousValue !== null}
			<span
				class="mt-auto flex items-center gap-1.5 pt-2 text-xs"
				title={previousPeriod ? `${previousPeriod}: ${formatValue(previousValue)}` : undefined}
			>
				{#if change() !== 0}
					{@const TrendIcon = getTrendIcon(change())}
					<span
						class="inline-flex items-center gap-0.5 font-medium tabular-nums {getTrendColor(
							change()
						)}"><TrendIcon class="size-3" />{Math.abs(change()).toFixed(0)}%</span
					>
				{/if}
				<span class="truncate text-muted-foreground tabular-nums"
					>vs {formatValue(previousValue)}</span
				>
			</span>
		{:else if currentPeriod}
			<span class="mt-auto truncate pt-2 text-xs text-muted-foreground">{currentPeriod}</span>
		{/if}
	{/if}
</button>

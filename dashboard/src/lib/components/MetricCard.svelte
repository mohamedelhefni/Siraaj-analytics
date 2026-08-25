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
		if (changeVal === 0) return isSelected ? 'text-slate-400' : 'text-slate-500';
		const isImprovement = isNegativeBetter ? changeVal < 0 : changeVal > 0;
		if (isImprovement) return isSelected ? 'text-emerald-300' : 'text-emerald-700';
		return isSelected ? 'text-rose-300' : 'text-rose-700';
	}
</script>

<button
	class="group relative flex min-h-48 flex-col overflow-hidden p-4 text-left transition-all focus-visible:z-10 focus-visible:ring-2 focus-visible:ring-amber-500 focus-visible:outline-none focus-visible:ring-inset sm:p-5 {isSelected
		? 'bg-slate-950 text-white shadow-inner'
		: 'bg-white text-slate-950 hover:z-10 hover:bg-amber-50'}"
	{onclick}
	disabled={loading}
>
	{#if isSelected}<span
			class="absolute top-3 right-3 size-1.5 rounded-full bg-amber-300 shadow-[0_0_8px_rgba(252,211,77,0.8)]"
		></span>{/if}
	<div
		class="mb-3 min-h-6 w-full text-[10px] leading-4 font-semibold tracking-[0.14em] uppercase {isSelected
			? 'text-amber-200'
			: 'text-slate-500'}"
	>
		{label}
	</div>

	{#if loading}
		<div class="h-9 w-24 animate-pulse rounded bg-slate-200"></div>
		<div class="mt-3 h-3 w-16 animate-pulse rounded bg-slate-100"></div>
	{:else}
		<div class="w-full">
			<div class="text-3xl font-semibold tracking-[-0.04em] tabular-nums">
				{formatValue(currentValue)}
			</div>
			{#if currentPeriod}
				<div
					class="mt-1 text-[10px] leading-4 whitespace-nowrap {isSelected
						? 'text-slate-400'
						: 'text-slate-500'}"
				>
					{currentPeriod}
				</div>
			{/if}
		</div>

		{#if previousValue !== null}
			<div class="mt-auto flex w-full items-end justify-between gap-2 border-t pt-3 text-[11px]">
				<div class="min-w-0 {isSelected ? 'text-slate-400' : 'text-slate-500'}">
					<span class="block font-medium">Prev. {formatValue(previousValue)}</span>
					{#if previousPeriod}<span class="mt-0.5 block text-[9px] whitespace-nowrap"
							>{previousPeriod}</span
						>{/if}
				</div>
				{#if change() !== 0}
					{@const TrendIcon = getTrendIcon(change())}
					<div class="flex shrink-0 items-center gap-1 font-semibold {getTrendColor(change())}">
						<TrendIcon class="size-3" />
						{Math.abs(change()).toFixed(0)}%
					</div>
				{/if}
			</div>
		{/if}
	{/if}
</button>

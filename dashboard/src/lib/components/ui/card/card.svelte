<script>
	import { cn } from '$lib/utils.js';

	let { ref = $bindable(null), class: className = '', children, ...restProps } = $props();
</script>

<div
	bind:this={ref}
	data-slot="card"
	class={cn(
		'bg-card text-card-foreground flex flex-col gap-4 rounded-xl border py-5 shadow-xs',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</div>

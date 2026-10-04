<script>
	/** @type {{ title: string, description?: string, class?: string, children?: import('svelte').Snippet }} */
	let { title, description = '', class: className = '', children } = $props();
</script>

<header
	class="flex flex-wrap items-end justify-between gap-4 border-b border-border pb-5 {className}"
>
	<div class="min-w-0">
		<h1 class="text-xl font-semibold tracking-tight text-foreground">{title}</h1>
		{#if description}<p class="mt-1 max-w-2xl text-sm text-muted-foreground">{description}</p>{/if}
	</div>
	{#if children}<div class="flex flex-wrap items-center gap-2">{@render children()}</div>{/if}
</header>

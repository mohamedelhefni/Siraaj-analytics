<script lang="ts">
	import { base } from '$app/paths';
	import {
		ArrowRight,
		Bot,
		ChartNoAxesCombined,
		Check,
		Globe,
		Link2,
		MessageSquareText,
		MonitorPlay,
		Route,
		Users,
		Waypoints
	} from 'lucide-svelte';
	import favicon from '$lib/assets/lantern.png';
	import MetricCard from '$lib/components/MetricCard.svelte';
	import TimelineChart from '$lib/components/TimelineChart.svelte';
	import TopItemsList from '$lib/components/TopItemsList.svelte';
	import BrowserPanel from '$lib/components/BrowserPanel.svelte';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';

	// Dummy data for the product preview: 30 days with a weekly rhythm and steady growth.
	const day = 24 * 60 * 60 * 1000;
	const end = Date.UTC(2026, 8, 30);
	const series = (scale: number, growth: number, offset = 0) =>
		Array.from({ length: 30 }, (_, i) => ({
			date: new Date(end - (29 - i) * day).toISOString().slice(0, 10),
			count: Math.round(scale * (1 + growth * (i / 29)) * (1 + 0.18 * Math.sin((i + offset) / 1.1)))
		}));

	const metrics = [
		{ key: 'users', label: 'Unique Visitors', value: 24812, prev: 22140, scale: 700, growth: 0.35 },
		{ key: 'visits', label: 'Total Visits', value: 31205, prev: 28630, scale: 900, growth: 0.3 },
		{ key: 'page_views', label: 'Pageviews', value: 86410, prev: 75120, scale: 2500, growth: 0.4 },
		{
			key: 'bounce_rate',
			label: 'Bounce Rate',
			value: 41,
			prev: 44,
			scale: 44,
			growth: -0.1,
			percent: true
		}
	];
	let selected = $state('users');
	let showComparison = $state(true);
	const current = $derived(metrics.find((m) => m.key === selected)!);
	const timeline = $derived(series(current.scale, current.growth));
	const previous = $derived(series(current.scale * 0.9, current.growth * 0.8, 2));

	const pages = [
		{ url: '/', count: 21340 },
		{ url: '/pricing', count: 13210 },
		{ url: '/blog/self-hosting-analytics', count: 8730 },
		{ url: '/docs/quick-start', count: 5120 },
		{ url: '/changelog', count: 2980 }
	];
	const countries = [
		{ name: 'United States', count: 7420 },
		{ name: 'Germany', count: 3910 },
		{ name: 'Saudi Arabia', count: 3150 },
		{ name: 'United Kingdom', count: 2280 },
		{ name: 'Egypt', count: 1940 }
	];
	const browsers = [
		{ name: 'Chrome', count: 14210 },
		{ name: 'Safari', count: 5630 },
		{ name: 'Firefox', count: 2470 },
		{ name: 'Edge', count: 1580 }
	];
	const devices = [
		{ name: 'Desktop', count: 15120 },
		{ name: 'Mobile', count: 8940 },
		{ name: 'Tablet', count: 750 }
	];
	const os = [
		{ name: 'Windows', count: 8120 },
		{ name: 'Mac OS', count: 6040 },
		{ name: 'iOS', count: 5210 },
		{ name: 'Android', count: 3730 }
	];

	const features = [
		{
			icon: ChartNoAxesCombined,
			title: 'Real-time overview',
			text: 'Visitors, visits, pageviews, bounce rate and duration, compared against the previous period.'
		},
		{
			icon: Waypoints,
			title: 'Channel attribution',
			text: 'Traffic grouped into direct, search, social, referral and paid, with per-source drill-down.'
		},
		{
			icon: Route,
			title: 'Funnels',
			text: 'Build multi-step funnels from pages and events and see exactly where visitors drop off.'
		},
		{
			icon: MonitorPlay,
			title: 'Session replays',
			text: 'Watch real sessions to debug UX issues. Form inputs are masked before they leave the browser.'
		},
		{
			icon: Users,
			title: 'User timelines',
			text: "Call identify() to follow one user's pageviews, events and survey answers on one timeline."
		},
		{
			icon: MessageSquareText,
			title: 'In-app surveys',
			text: 'Show a short survey when a chosen event fires, once per visitor. Answers sit next to your analytics.'
		},
		{
			icon: Link2,
			title: 'Short links',
			text: 'Create short links and see clicks by country and referring site.'
		},
		{
			icon: Bot,
			title: 'Bot detection',
			text: 'Crawlers and headless browsers are flagged automatically, so numbers reflect real people.'
		},
		{
			icon: Globe,
			title: 'Geo & devices',
			text: 'Countries from a local GeoIP database plus browsers, OS and devices, with no external lookups.'
		}
	];
	const privacy = [
		[
			'No cookies, no cross-site tracking',
			'Visitors are counted without third-party cookies or cross-site identifiers.'
		],
		[
			'Your server, your database',
			'Everything lives in a single DuckDB file you can back up, inspect or delete.'
		],
		[
			'Per-project, revocable keys',
			'Each site sends events with its own tracking key. Revoke one and that site stops reporting.'
		],
		['Isolated accounts', 'Every dashboard query is scoped to the projects you own.']
	];
	const github = 'https://github.com/mohamedelhefni/siraaj';
	const docs = 'https://docs.siraaj.live';
	const btn =
		'inline-flex h-10 items-center justify-center gap-2 rounded-md px-4 text-sm font-medium transition whitespace-nowrap';
	const primary = `${btn} bg-primary text-primary-foreground hover:bg-primary/90`;
	const ghost = `${btn} border border-border bg-card shadow-xs hover:bg-muted`;
</script>

<svelte:head>
	<title>Siraaj — Privacy-first, self-hosted web analytics</title>
	<meta
		name="description"
		content="Siraaj is open-source web analytics you run on your own server. No cookies, a 7 KB script, and one binary powered by Go and DuckDB."
	/>
</svelte:head>

<header class="sticky top-0 z-40 border-b border-border bg-background/85 backdrop-blur">
	<div class="mx-auto flex h-15 max-w-6xl items-center gap-6 px-4 sm:px-6">
		<a href="/" class="flex items-center gap-2.5 font-semibold"
			><img src={favicon} alt="" class="size-7" />Siraaj</a
		>
		<nav class="hidden gap-6 text-sm text-muted-foreground md:flex" aria-label="Primary">
			<a href="#features" class="hover:text-foreground">Features</a>
			<a href="#privacy" class="hover:text-foreground">Privacy</a>
			<a href="#install" class="hover:text-foreground">Install</a>
			<a href={docs} class="hover:text-foreground">Docs</a>
		</nav>
		<div class="ml-auto flex gap-2">
			<a href={github} class="{ghost} h-8 px-3">GitHub</a>
			<a href="{base}/" class="{primary} h-8 px-3">Dashboard</a>
		</div>
	</div>
</header>

<main>
	<section class="mx-auto max-w-6xl px-4 pt-16 pb-14 text-center sm:px-6 sm:pt-20">
		<span
			class="inline-flex items-center gap-2 rounded-full border border-border bg-card py-1 pr-3 pl-1 text-xs text-muted-foreground"
		>
			<span class="rounded-full bg-muted px-2 py-0.5 font-medium text-foreground">Open source</span>
			Self-hosted analytics in one binary
		</span>
		<h1 class="mx-auto mt-6 max-w-3xl text-4xl leading-[1.05] font-bold tracking-tight sm:text-6xl">
			Understand your traffic. <span class="text-muted-foreground">Keep your visitors' data.</span>
		</h1>
		<p class="mx-auto mt-5 max-w-xl text-lg text-muted-foreground">
			Siraaj is privacy-first web analytics you run on your own server. Pageviews, channels,
			funnels, session replays and surveys, without cookies or third parties.
		</p>
		<div class="mt-8 flex flex-wrap justify-center gap-2.5">
			<a href="{base}/" class={primary}>Open dashboard <ArrowRight class="size-4" /></a>
			<a href="#install" class={ghost}>Self-host in 2 minutes</a>
		</div>
		<div class="mt-5 flex justify-center">
			<code
				class="max-w-full min-w-0 overflow-x-auto rounded-md border border-border bg-muted px-3 py-1 font-mono text-[13px] whitespace-nowrap"
			>
				docker run -d -p 8080:8080 -v $(pwd)/data:/data mohamedelhefni/siraaj
			</code>
		</div>

		<!-- Live preview built from the real dashboard components with dummy data. -->
		<div
			class="mx-auto mt-14 max-w-5xl overflow-hidden rounded-2xl border border-border bg-background text-left shadow-[0_24px_60px_-24px_rgb(15_23_42/0.3)]"
		>
			<div class="flex items-center gap-1.5 border-b border-border bg-card px-4 py-2.5">
				<span class="size-2.5 rounded-full bg-border"></span><span
					class="size-2.5 rounded-full bg-border"
				></span><span class="size-2.5 rounded-full bg-border"></span>
				<span class="ml-3 text-xs text-muted-foreground">your-server.com/dashboard</span>
				<span class="ml-auto inline-flex items-center gap-1.5 text-xs text-muted-foreground"
					><span class="size-1.5 rounded-full bg-success"></span>Live demo · dummy data</span
				>
			</div>
			<div class="space-y-4 p-3 sm:p-5">
				<section class="panel overflow-hidden">
					<div class="grid grid-cols-2 gap-px border-b border-border bg-border md:grid-cols-4">
						{#each metrics as m}
							<MetricCard
								label={m.label}
								currentValue={m.value}
								previousValue={m.prev}
								formatValue={m.percent ? (v: number) => `${v.toFixed(0)}%` : undefined}
								isNegativeBetter={m.key === 'bounce_rate'}
								isSelected={selected === m.key}
								onclick={() => (selected = m.key)}
							/>
						{/each}
					</div>
					<div class="px-2 pt-4 pb-6 sm:px-5">
						<TimelineChart
							data={timeline}
							comparisonData={previous}
							metric={selected}
							bind:showComparison
						/>
					</div>
				</section>
				<div class="grid gap-4 md:grid-cols-3">
					<Card>
						<CardHeader class="pb-0"><CardTitle>Pages</CardTitle></CardHeader>
						<CardContent><TopItemsList items={pages} labelKey="url" maxItems={5} /></CardContent>
					</Card>
					<Card>
						<CardHeader class="pb-0"><CardTitle>Countries</CardTitle></CardHeader>
						<CardContent><TopItemsList items={countries} type="country" maxItems={5} /></CardContent
						>
					</Card>
					<Card>
						<CardHeader class="pb-0"><CardTitle>Devices</CardTitle></CardHeader>
						<CardContent><BrowserPanel {browsers} {devices} operatingSystems={os} /></CardContent>
					</Card>
				</div>
			</div>
		</div>
	</section>

	<section class="border-y border-border bg-card" aria-label="At a glance">
		<dl class="mx-auto grid max-w-6xl grid-cols-2 px-4 sm:px-6 md:grid-cols-4">
			{#each [['~7 KB', 'gzipped tracking script'], ['0 cookies', 'no third-party trackers'], ['1 binary', 'Go server with embedded DuckDB'], ['100%', 'of your data stays on your server']] as [value, label]}
				<div class="py-6 pr-4">
					<dt class="text-2xl font-semibold tracking-tight">{value}</dt>
					<dd class="text-sm text-muted-foreground">{label}</dd>
				</div>
			{/each}
		</dl>
	</section>

	<section id="features" class="mx-auto max-w-6xl scroll-mt-16 px-4 py-20 sm:px-6">
		<p class="eyebrow">Features</p>
		<h2 class="mt-2 max-w-2xl text-3xl font-bold tracking-tight sm:text-4xl">
			Everything you need to answer “what's working?”
		</h2>
		<p class="mt-3 max-w-xl text-muted-foreground">
			One dashboard for traffic, conversion and user feedback, instead of three tools that each copy
			your visitors' data.
		</p>
		<div
			class="mt-10 grid gap-px overflow-hidden rounded-xl border border-border bg-border sm:grid-cols-2 lg:grid-cols-3"
		>
			{#each features as f}
				<div class="bg-card p-6">
					<f.icon class="size-5" strokeWidth={1.75} />
					<h3 class="mt-3.5 text-[15px] font-semibold">{f.title}</h3>
					<p class="mt-1 text-sm text-muted-foreground">{f.text}</p>
				</div>
			{/each}
		</div>
	</section>

	<section
		id="privacy"
		class="mx-auto grid max-w-6xl scroll-mt-16 items-start gap-10 px-4 pb-20 sm:px-6 lg:grid-cols-2 lg:gap-16"
	>
		<div>
			<p class="eyebrow">Privacy by construction</p>
			<h2 class="mt-2 text-3xl font-bold tracking-tight sm:text-4xl">
				Analytics your visitors don't have to trust a third party for.
			</h2>
			<p class="mt-3 text-muted-foreground">
				Siraaj runs on your infrastructure. There is no Siraaj cloud receiving copies of your data:
				events go from the browser to your server and stop there.
			</p>
		</div>
		<ul class="grid gap-4">
			{#each privacy as [title, text]}
				<li class="flex gap-3">
					<Check class="mt-0.5 size-5 shrink-0 text-success" />
					<div>
						<p class="font-semibold">{title}</p>
						<p class="text-sm text-muted-foreground">{text}</p>
					</div>
				</li>
			{/each}
		</ul>
	</section>

	<section
		id="install"
		class="mx-auto grid max-w-6xl scroll-mt-16 items-start gap-10 px-4 pb-20 sm:px-6 lg:grid-cols-2 lg:gap-16"
	>
		<div>
			<p class="eyebrow">Get started</p>
			<h2 class="mt-2 text-3xl font-bold tracking-tight sm:text-4xl">Running in two steps.</h2>
			<p class="mt-3 text-muted-foreground">
				Start the server, create a project in the dashboard to get a tracking key, then add the SDK
				to your site.
			</p>
			<div class="mt-6 flex flex-wrap gap-2.5">
				<a href={docs} class={primary}>Read the docs</a><a href={github} class={ghost}
					>View source</a
				>
			</div>
		</div>
		<div class="grid gap-4">
			{#each [['Run the server', '# Dashboard at http://localhost:8080/dashboard\ndocker run -d -p 8080:8080 \\\n  -v $(pwd)/data:/data \\\n  mohamedelhefni/siraaj'], ['Add the SDK', "// npm install @hefni101/siraaj\nimport { analytics } from '@hefni101/siraaj';\n\nanalytics.init({\n  apiUrl: 'https://analytics.example.com',\n  projectId: 'my-site',\n  trackingToken: 'siraaj_trk_…'\n});"]] as [title, code], i}
				<div class="panel overflow-hidden">
					<p class="flex items-center gap-2.5 border-b border-border px-4 py-3 text-sm font-medium">
						<span class="grid size-5.5 place-items-center rounded-full bg-muted text-xs"
							>{i + 1}</span
						>{title}
					</p>
					<pre class="overflow-x-auto p-4 font-mono text-[13px] leading-relaxed">{code}</pre>
				</div>
			{/each}
		</div>
	</section>

	<section class="mx-auto max-w-6xl px-4 pb-20 sm:px-6">
		<div class="rounded-2xl bg-primary px-6 py-16 text-center text-primary-foreground">
			<h2 class="text-3xl font-bold tracking-tight sm:text-4xl">Own your analytics.</h2>
			<p class="mx-auto mt-3 max-w-md text-primary-foreground/70">
				Free, open source, and yours to run. Set it up once and stop sending your visitors' data
				elsewhere.
			</p>
			<div class="mt-8 flex flex-wrap justify-center gap-2.5">
				<a
					href="{base}/"
					class="{btn} bg-primary-foreground text-primary hover:bg-primary-foreground/90"
					>Open dashboard</a
				>
				<a
					href={github}
					class="{btn} border border-primary-foreground/25 hover:bg-primary-foreground/10"
					>Star on GitHub</a
				>
			</div>
		</div>
	</section>
</main>

<footer class="border-t border-border">
	<div
		class="mx-auto flex max-w-6xl flex-wrap items-center gap-x-6 gap-y-4 px-4 py-8 text-sm text-muted-foreground sm:px-6"
	>
		<a href="/" class="flex items-center gap-2 font-semibold text-foreground"
			><img src={favicon} alt="" class="size-6" />Siraaj</a
		>
		<span>Open-source, self-hosted web analytics.</span>
		<nav class="flex flex-wrap gap-5 sm:ml-auto" aria-label="Footer">
			<a href={docs} class="hover:text-foreground">Docs</a>
			<a href={github} class="hover:text-foreground">GitHub</a>
			<a href="/api/health" class="hover:text-foreground">Status</a>
			<a href="{base}/" class="hover:text-foreground">Dashboard</a>
		</nav>
	</div>
</footer>

<script>
	import '../app.css';
	import favicon from '$lib/assets/lantern.png';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { base } from '$app/paths';
	import {
		ChartNoAxesCombined,
		KeyRound,
		Link2,
		LogOut,
		Menu,
		MessageSquareText,
		MonitorPlay,
		Route,
		Settings2,
		Users,
		Waypoints,
		X
	} from 'lucide-svelte';
	import {
		bootstrap,
		clearAccessToken,
		fetchCurrentUser,
		getAccessToken,
		login,
		signup
	} from '$lib/api.js';

	let { children } = $props();
	/** @type {any} */
	let user = $state(null);
	let loading = $state(true);
	let authMode = $state('login');
	let email = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let error = $state('');
	let submitting = $state(false);
	let navOpen = $state(false);

	const navGroups = [
		{
			label: 'Analytics',
			items: [
				{ path: '/', label: 'Overview', icon: ChartNoAxesCombined },
				{ path: '/channels', label: 'Channels', icon: Waypoints },
				{ path: '/funnel', label: 'Funnels', icon: Route },
				{ path: '/users', label: 'Users', icon: Users },
				{ path: '/replays', label: 'Replays', icon: MonitorPlay }
			]
		},
		{
			label: 'Engage',
			items: [
				{ path: '/links', label: 'Links', icon: Link2 },
				{ path: '/surveys', label: 'Surveys', icon: MessageSquareText }
			]
		},
		{
			label: 'Workspace',
			items: [{ path: '/settings', label: 'Projects & keys', icon: Settings2 }]
		}
	];

	/** @param {string} path */
	function isActive(path) {
		const current = $page.url.pathname.slice(base.length).replace(/\/$/, '') || '/';
		return current === path;
	}

	// The public landing page renders without the auth gate or app shell. Go also serves it at /,
	// outside the base path, where the router can't match a route id but still hydrates the page.
	const isLanding = $derived($page.route.id === '/welcome' || !$page.url.pathname.startsWith(base));

	// Close the mobile drawer after navigating.
	$effect(() => {
		$page.url.pathname;
		navOpen = false;
	});

	onMount(() => {
		window.addEventListener('siraaj:unauthorized', signOut);
		void (async () => {
			if (getAccessToken()) {
				try {
					user = await fetchCurrentUser();
				} catch {
					clearAccessToken();
				}
			}
			loading = false;
		})();
		return () => window.removeEventListener('siraaj:unauthorized', signOut);
	});

	/** @param {SubmitEvent} event */
	async function submitAuth(event) {
		event.preventDefault();
		error = '';
		submitting = true;
		if (authMode !== 'login' && password !== confirmPassword) {
			error = 'Passwords do not match';
			submitting = false;
			return;
		}
		try {
			if (authMode === 'signup') user = await signup(email, password);
			else if (authMode === 'bootstrap') user = await bootstrap(email, password);
			else user = await login(email, password);
		} catch (reason) {
			error = reason instanceof Error ? reason.message : 'Unable to authenticate';
		} finally {
			submitting = false;
		}
	}

	function signOut() {
		clearAccessToken();
		user = null;
	}
	/** @param {'login' | 'signup' | 'bootstrap'} mode */
	function selectAuthMode(mode) {
		authMode = mode;
		error = '';
		password = '';
		confirmPassword = '';
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{#if isLanding}
	{@render children?.()}
{:else if loading}
	<div class="grid min-h-screen place-items-center bg-background text-muted-foreground">
		<KeyRound class="size-6 animate-pulse" />
	</div>
{:else if !user}
	<main class="grid min-h-screen bg-background lg:grid-cols-2">
		<section
			class="hidden flex-col justify-between border-r border-border bg-slate-900 p-12 text-slate-100 lg:flex"
		>
			<div class="flex items-center gap-2.5">
				<img src={favicon} alt="" class="size-8" /><span class="text-lg font-semibold">Siraaj</span>
			</div>
			<div class="max-w-md">
				<h1 class="text-3xl font-semibold tracking-tight">
					Privacy-first analytics you host yourself.
				</h1>
				<p class="mt-4 text-sm leading-6 text-slate-400">
					Traffic, funnels, replays and surveys — scoped to your account, with revocable keys per
					project.
				</p>
				<dl class="mt-10 grid grid-cols-3 gap-6 border-t border-slate-800 pt-6 text-sm">
					<div>
						<dt class="text-slate-500">Cookies</dt>
						<dd class="mt-1 font-medium">None</dd>
					</div>
					<div>
						<dt class="text-slate-500">Hosting</dt>
						<dd class="mt-1 font-medium">Self-hosted</dd>
					</div>
					<div>
						<dt class="text-slate-500">Access</dt>
						<dd class="mt-1 font-medium">Token scoped</dd>
					</div>
				</dl>
			</div>
			<p class="text-xs text-slate-500">Your data never leaves your server.</p>
		</section>
		<section class="grid place-items-center px-4 py-12">
			<form onsubmit={submitAuth} class="w-full max-w-sm">
				<div class="mb-8 flex items-center gap-2.5 lg:hidden">
					<img src={favicon} alt="" class="size-8" /><span class="text-lg font-semibold"
						>Siraaj</span
					>
				</div>
				<h2 class="text-2xl font-semibold tracking-tight">
					{authMode === 'bootstrap'
						? 'Create the administrator'
						: authMode === 'signup'
							? 'Create your account'
							: 'Sign in'}
				</h2>
				<p class="mt-1.5 text-sm text-muted-foreground">
					{authMode === 'bootstrap'
						? 'This endpoint locks permanently after the first account.'
						: authMode === 'signup'
							? 'Your account starts empty. Only projects you create will appear.'
							: 'Welcome back. Sign in to your workspace.'}
				</p>
				{#if authMode !== 'bootstrap'}
					<div class="mt-6 grid grid-cols-2 rounded-lg bg-muted p-1 text-sm">
						<button
							type="button"
							onclick={() => selectAuthMode('login')}
							class="rounded-md px-3 py-1.5 font-medium transition {authMode === 'login'
								? 'bg-card text-foreground shadow-xs'
								: 'text-muted-foreground hover:text-foreground'}">Sign in</button
						>
						<button
							type="button"
							onclick={() => selectAuthMode('signup')}
							class="rounded-md px-3 py-1.5 font-medium transition {authMode === 'signup'
								? 'bg-card text-foreground shadow-xs'
								: 'text-muted-foreground hover:text-foreground'}">Create account</button
						>
					</div>
				{/if}
				<label class="mt-6 block text-sm font-medium"
					>Email<input
						bind:value={email}
						type="email"
						autocomplete="email"
						required
						class="field mt-1.5 h-10 w-full"
					/></label
				>
				<label class="mt-4 block text-sm font-medium"
					>Password<input
						bind:value={password}
						type="password"
						autocomplete={authMode === 'login' ? 'current-password' : 'new-password'}
						minlength="10"
						maxlength="128"
						required
						class="field mt-1.5 h-10 w-full"
					/></label
				>
				{#if authMode !== 'login'}<label class="mt-4 block text-sm font-medium"
						>Confirm password<input
							bind:value={confirmPassword}
							type="password"
							autocomplete="new-password"
							minlength="10"
							maxlength="128"
							required
							class="field mt-1.5 h-10 w-full"
						/></label
					>{/if}
				{#if error}<p
						class="mt-4 rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive"
						role="alert"
					>
						{error}
					</p>{/if}
				<button
					disabled={submitting}
					class="mt-6 h-10 w-full rounded-md bg-primary text-sm font-medium text-primary-foreground transition hover:bg-primary/90 disabled:opacity-60"
					>{submitting
						? 'Please wait…'
						: authMode === 'bootstrap'
							? 'Create administrator'
							: authMode === 'signup'
								? 'Create account'
								: 'Sign in'}</button
				>
				<button
					type="button"
					onclick={() => selectAuthMode(authMode === 'bootstrap' ? 'login' : 'bootstrap')}
					class="mt-4 w-full text-xs text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
					>{authMode === 'bootstrap' ? 'Back to sign in' : 'Server owner: first-time setup'}</button
				>
			</form>
		</section>
	</main>
{:else}
	<div class="min-h-screen bg-background lg:pl-60">
		<header
			class="sticky top-0 z-40 flex items-center gap-3 border-b border-border bg-card/90 px-4 py-2.5 backdrop-blur lg:hidden"
		>
			<button
				onclick={() => (navOpen = true)}
				aria-label="Open navigation"
				class="grid size-9 place-items-center rounded-md text-muted-foreground hover:bg-muted"
				><Menu class="size-5" /></button
			>
			<img src={favicon} alt="" class="size-6" /><span class="font-semibold">Siraaj</span>
		</header>

		{#if navOpen}<button
				class="fixed inset-0 z-40 bg-slate-950/40 lg:hidden"
				aria-label="Close navigation"
				onclick={() => (navOpen = false)}
			></button>{/if}

		<aside
			class="fixed inset-y-0 left-0 z-50 flex w-60 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground transition-transform lg:translate-x-0 {navOpen
				? 'translate-x-0'
				: '-translate-x-full'}"
		>
			<div class="flex h-14 items-center gap-2.5 border-b border-sidebar-border px-4">
				<a href="{base}/" class="flex items-center gap-2.5" aria-label="Siraaj dashboard">
					<img src={favicon} alt="" class="size-7" />
					<span class="font-semibold text-foreground">Siraaj</span>
				</a>
				<button
					onclick={() => (navOpen = false)}
					aria-label="Close navigation"
					class="ml-auto grid size-8 place-items-center rounded-md text-muted-foreground hover:bg-muted lg:hidden"
					><X class="size-4" /></button
				>
			</div>

			<nav class="flex-1 space-y-5 overflow-y-auto px-3 py-4">
				{#each navGroups as group}
					<div>
						<p class="eyebrow px-2.5 pb-1.5">{group.label}</p>
						<div class="space-y-px">
							{#each group.items as item}
								{@const active = isActive(item.path)}
								<a
									href="{base}{item.path}"
									aria-current={active ? 'page' : undefined}
									class="flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm transition {active
										? 'bg-sidebar-accent font-medium text-sidebar-accent-foreground'
										: 'hover:bg-sidebar-accent/60 hover:text-foreground'}"
								>
									<item.icon
										class="size-4 {active ? 'text-foreground' : 'text-muted-foreground'}"
									/>
									{item.label}
								</a>
							{/each}
						</div>
					</div>
				{/each}
			</nav>

			<div class="border-t border-sidebar-border p-3">
				<div class="flex items-center gap-2.5 rounded-md px-1.5 py-1">
					<span
						class="grid size-8 shrink-0 place-items-center rounded-full bg-muted text-xs font-semibold text-foreground uppercase"
						>{user.email?.[0] ?? '?'}</span
					>
					<div class="min-w-0 flex-1 leading-tight">
						<p class="truncate text-sm font-medium text-foreground">{user.email}</p>
						<p class="text-xs text-muted-foreground capitalize">{user.role}</p>
					</div>
					<button
						onclick={signOut}
						aria-label="Sign out"
						title="Sign out"
						class="grid size-8 place-items-center rounded-md text-muted-foreground transition hover:bg-muted hover:text-foreground"
						><LogOut class="size-4" /></button
					>
				</div>
			</div>
		</aside>

		{@render children?.()}
	</div>
{/if}

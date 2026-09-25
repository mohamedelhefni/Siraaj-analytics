<script>
	import '../app.css';
	import favicon from '$lib/assets/lantern.png';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { Activity, ChartNoAxesCombined, KeyRound, Link2, LogOut, Route, Settings2, Waypoints } from 'lucide-svelte';
	import { bootstrap, clearAccessToken, fetchCurrentUser, getAccessToken, login, signup } from '$lib/api.js';

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

	onMount(() => {
		window.addEventListener('siraaj:unauthorized', signOut);
		void (async () => {
			if (getAccessToken()) {
				try { user = await fetchCurrentUser(); } catch { clearAccessToken(); }
			}
			loading = false;
		})();
		return () => window.removeEventListener('siraaj:unauthorized', signOut);
	});

	/** @param {SubmitEvent} event */
	async function submitAuth(event) {
		event.preventDefault(); error = ''; submitting = true;
		if (authMode !== 'login' && password !== confirmPassword) {
			error = 'Passwords do not match'; submitting = false; return;
		}
		try {
			if (authMode === 'signup') user = await signup(email, password);
			else if (authMode === 'bootstrap') user = await bootstrap(email, password);
			else user = await login(email, password);
		}
		catch (reason) { error = reason instanceof Error ? reason.message : 'Unable to authenticate'; }
		finally { submitting = false; }
	}

	function signOut() { clearAccessToken(); user = null; }
	/** @param {'login' | 'signup' | 'bootstrap'} mode */
	function selectAuthMode(mode) { authMode = mode; error = ''; password = ''; confirmPassword = ''; }
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{#if loading}
	<div class="grid min-h-screen place-items-center bg-slate-950 text-amber-200"><KeyRound class="size-7 animate-pulse" /></div>
{:else if !user}
	<main class="relative grid min-h-screen overflow-hidden bg-slate-950 px-6 py-12 text-white lg:grid-cols-[1.15fr_0.85fr]">
		<div class="pointer-events-none absolute inset-0 opacity-60 [background-image:radial-gradient(circle_at_20%_20%,rgba(251,191,36,.16),transparent_32%),linear-gradient(rgba(255,255,255,.025)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,.025)_1px,transparent_1px)] [background-size:auto,48px_48px,48px_48px]"></div>
		<section class="relative hidden flex-col justify-between p-12 lg:flex">
			<div class="flex items-center gap-3"><img src={favicon} alt="" class="size-12" /><span class="display-type text-2xl font-semibold">Siraaj</span></div>
			<div class="max-w-xl"><p class="mb-5 text-xs font-bold tracking-[.28em] text-amber-300 uppercase">Private by construction</p><h1 class="display-type text-6xl leading-[.95] font-semibold tracking-tight">Your signals.<br />Under your lock.</h1><p class="mt-7 max-w-md text-lg leading-8 text-slate-400">Analytics access now belongs to people, while every incoming event carries a revocable project key.</p></div>
			<p class="text-xs tracking-widest text-slate-600 uppercase">Self-hosted · Token scoped · No public data</p>
		</section>
		<section class="relative grid place-items-center">
			<form onsubmit={submitAuth} class="w-full max-w-md border border-white/10 bg-white/[.045] p-8 shadow-2xl shadow-black/30 backdrop-blur-xl sm:p-10">
				<div class="mb-9 flex size-12 items-center justify-center rounded-full bg-amber-300 text-slate-950"><KeyRound class="size-5" /></div>
				<p class="text-xs font-bold tracking-[.22em] text-amber-300 uppercase">{authMode === 'bootstrap' ? 'First run' : authMode === 'signup' ? 'Your private workspace' : 'Restricted access'}</p>
				<h2 class="display-type mt-3 text-3xl font-semibold">{authMode === 'bootstrap' ? 'Create the administrator' : authMode === 'signup' ? 'Start using Siraaj' : 'Enter the dashboard'}</h2>
				<p class="mt-2 text-sm leading-6 text-slate-400">{authMode === 'bootstrap' ? 'This endpoint locks permanently after the first account.' : authMode === 'signup' ? 'Your account starts empty. Only projects you create will appear.' : 'Sign in to your isolated analytics workspace.'}</p>
				<div class="mt-7 grid grid-cols-2 border border-white/10 p-1 text-sm"><button type="button" onclick={() => selectAuthMode('login')} class="px-3 py-2.5 font-semibold transition {authMode === 'login' ? 'bg-white text-slate-950' : 'text-slate-400 hover:text-white'}">Sign in</button><button type="button" onclick={() => selectAuthMode('signup')} class="px-3 py-2.5 font-semibold transition {authMode === 'signup' ? 'bg-white text-slate-950' : 'text-slate-400 hover:text-white'}">Create account</button></div>
				<label class="mt-8 block text-xs font-semibold tracking-wider text-slate-300 uppercase">Email<input bind:value={email} type="email" autocomplete="email" required class="mt-2 w-full border border-white/10 bg-slate-900/80 px-4 py-3.5 text-base text-white outline-none transition focus:border-amber-300/70" /></label>
				<label class="mt-5 block text-xs font-semibold tracking-wider text-slate-300 uppercase">Password<input bind:value={password} type="password" autocomplete={authMode === 'login' ? 'current-password' : 'new-password'} minlength="10" maxlength="128" required class="mt-2 w-full border border-white/10 bg-slate-900/80 px-4 py-3.5 text-base text-white outline-none transition focus:border-amber-300/70" /></label>
				{#if authMode !== 'login'}<label class="mt-5 block text-xs font-semibold tracking-wider text-slate-300 uppercase">Confirm password<input bind:value={confirmPassword} type="password" autocomplete="new-password" minlength="10" maxlength="128" required class="mt-2 w-full border border-white/10 bg-slate-900/80 px-4 py-3.5 text-base text-white outline-none transition focus:border-amber-300/70" /></label>{/if}
				{#if error}<p class="mt-4 border-l-2 border-red-400 bg-red-400/10 px-3 py-2 text-sm text-red-200">{error}</p>{/if}
				<button disabled={submitting} class="mt-7 w-full bg-amber-300 px-4 py-3.5 font-bold text-slate-950 transition hover:bg-amber-200 disabled:opacity-60">{submitting ? 'Please wait…' : authMode === 'bootstrap' ? 'Create administrator' : authMode === 'signup' ? 'Create my account' : 'Sign in'}</button>
				<button type="button" onclick={() => selectAuthMode(authMode === 'bootstrap' ? 'login' : 'bootstrap')} class="mt-5 w-full text-xs text-slate-500 underline decoration-slate-700 underline-offset-4 hover:text-white">{authMode === 'bootstrap' ? 'Back to sign in' : 'Server owner: first-time setup'}</button>
			</form>
		</section>
	</main>
{:else}
<div class="min-h-screen">
	<nav
		class="sticky top-0 z-50 border-b border-white/10 bg-slate-950 text-white shadow-[0_8px_30px_rgba(15,23,42,0.14)]"
	>
		<div
			class="mx-auto flex max-w-[1440px] flex-wrap items-center gap-x-6 px-4 py-3 sm:flex-nowrap sm:px-6 lg:px-8"
		>
			<a
				href="/dashboard"
				class="group flex shrink-0 items-center gap-3"
				aria-label="Siraaj dashboard"
			>
				<span
					class="grid size-10 place-items-center rounded-xl border border-amber-300/25 bg-amber-300/10 shadow-inner shadow-amber-100/5 transition group-hover:bg-amber-300/15"
				>
					<img src={favicon} alt="" class="size-8" />
				</span>
				<span class="leading-none">
					<span class="display-type block text-lg font-semibold tracking-tight">Siraaj</span>
					<span
						class="mt-1 block text-[9px] font-semibold tracking-[0.2em] text-slate-400 uppercase"
						>Private analytics</span
					>
				</span>
			</a>

			<div
				class="order-3 mt-3 flex w-full gap-1 overflow-x-auto rounded-xl bg-white/[0.04] p-1 sm:order-none sm:mt-0 sm:w-auto"
			>
				<a
					href="/dashboard"
					class="flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-all {$page
						.url.pathname === '/'
						? 'bg-white text-slate-950 shadow-sm'
						: 'text-slate-300 hover:bg-white/10 hover:text-white'}"
				>
					<ChartNoAxesCombined class="size-4" /> Overview
				</a>
				<a
					href="/dashboard/channels"
					class="flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-all {$page
						.url.pathname === '/channels'
						? 'bg-white text-slate-950 shadow-sm'
						: 'text-slate-300 hover:bg-white/10 hover:text-white'}"
				>
					<Waypoints class="size-4" /> Channels
				</a>
				<a
					href="/dashboard/funnel"
					class="flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-all {$page
						.url.pathname === '/funnel'
						? 'bg-white text-slate-950 shadow-sm'
						: 'text-slate-300 hover:bg-white/10 hover:text-white'}"
				>
					<Route class="size-4" /> Funnels
				</a>
				<a
					href="/dashboard/links"
					class="flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-all {$page
						.url.pathname === '/links'
						? 'bg-white text-slate-950 shadow-sm'
						: 'text-slate-300 hover:bg-white/10 hover:text-white'}"
				>
					<Link2 class="size-4" /> Links
				</a>
			</div>


			<a href="/dashboard/settings" aria-label="Users and tracking tokens" class="ml-auto grid size-9 place-items-center rounded-lg text-slate-400 transition hover:bg-white/10 hover:text-white"><Settings2 class="size-4" /></a>
			<button onclick={signOut} aria-label="Sign out" class="grid size-9 place-items-center rounded-lg text-slate-400 transition hover:bg-white/10 hover:text-white"><LogOut class="size-4" /></button>
		</div>
	</nav>

	{@render children?.()}
</div>
{/if}

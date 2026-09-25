<script>
	import { onMount } from 'svelte';
	import { Copy, KeyRound, ShieldCheck, Trash2, UserPlus } from 'lucide-svelte';
	import { createTrackingToken, createUser, fetchCurrentUser, fetchTrackingTokens, fetchUsers, revokeTrackingToken } from '$lib/api.js';

	/** @type {any} */ let currentUser = $state(null);
	/** @type {any[]} */ let tokens = $state([]);
	/** @type {any[]} */ let users = $state([]);
	let error = $state(''), issuedToken = $state('');
		let projectID = $state(''), tokenName = $state('Website'), email = $state(''), password = $state(''), role = $state('user');

	onMount(load);
	async function load() {
		try {
			currentUser = await fetchCurrentUser(); tokens = await fetchTrackingTokens();
			if (currentUser.role === 'admin') users = await fetchUsers();
		} catch (reason) { error = reason instanceof Error ? reason.message : 'Unable to load access settings'; }
	}
	/** @param {SubmitEvent} event */
	async function issueToken(event) {
		event.preventDefault(); error = '';
		try { const result = await createTrackingToken({ project_id: projectID, name: tokenName }); issuedToken = result.token; await load(); }
		catch (reason) { error = reason instanceof Error ? reason.message : 'Unable to create token'; }
	}
	/** @param {SubmitEvent} event */
	async function addUser(event) {
		event.preventDefault(); error = '';
		try { await createUser({ email, password, role }); email = ''; password = ''; await load(); }
		catch (reason) { error = reason instanceof Error ? reason.message : 'Unable to add user'; }
	}
	/** @param {string} id */
	async function revoke(id) { if (confirm('Revoke this tracking token? Existing SDK clients will stop sending events.')) { await revokeTrackingToken(id); await load(); } }
</script>

<svelte:head><title>Access control · Siraaj</title></svelte:head>

<main class="mx-auto max-w-6xl px-4 py-10 sm:px-6 lg:px-8">
	<header class="mb-10 border-b border-slate-200 pb-8"><p class="text-xs font-bold tracking-[.22em] text-amber-700 uppercase">Control room</p><h1 class="display-type mt-2 text-4xl font-semibold tracking-tight text-slate-950">Users & tracking keys</h1><p class="mt-3 max-w-2xl text-slate-500">Each key creates or targets one of your projects. Even administrators cannot view another account's projects or tokens.</p></header>
	{#if error}<p class="mb-6 border-l-4 border-red-500 bg-red-50 p-4 text-sm text-red-800">{error}</p>{/if}
	<div class="grid gap-8 lg:grid-cols-2">
		<section class="border border-slate-200 bg-white p-6 shadow-sm sm:p-8">
			<div class="flex items-center gap-3"><KeyRound class="size-5 text-amber-700" /><h2 class="display-type text-2xl font-semibold">Tracking tokens</h2></div>
				<form onsubmit={issueToken} class="mt-6 grid gap-4 sm:grid-cols-2"><input bind:value={projectID} required placeholder="my-site" aria-label="Unique project ID" class="border border-slate-200 px-3 py-2.5 outline-none focus:border-amber-600" /><input bind:value={tokenName} required placeholder="Token name" aria-label="Tracking token name" class="border border-slate-200 px-3 py-2.5 outline-none focus:border-amber-600" /><button class="bg-slate-950 px-4 py-2.5 font-semibold text-white sm:col-span-2">Issue project token</button></form>
			{#if issuedToken}<div class="mt-5 bg-amber-50 p-4"><p class="text-xs font-bold tracking-wider text-amber-900 uppercase">Copy now — shown once</p><div class="mt-2 flex gap-2"><code class="min-w-0 flex-1 overflow-x-auto text-xs">{issuedToken}</code><button onclick={() => navigator.clipboard.writeText(issuedToken)} aria-label="Copy token"><Copy class="size-4" /></button></div></div>{/if}
			<div class="mt-6 divide-y divide-slate-100">{#each tokens as token}<div class="flex items-center gap-3 py-4"><div class="min-w-0 flex-1"><p class="font-semibold text-slate-900">{token.name}</p><p class="truncate text-xs text-slate-500">{token.project_id} · {token.prefix}… {token.revoked_at ? '· revoked' : ''}</p></div>{#if !token.revoked_at}<button onclick={() => revoke(token.id)} aria-label="Revoke token" class="text-slate-400 hover:text-red-600"><Trash2 class="size-4" /></button>{/if}</div>{/each}</div>
		</section>
			{#if currentUser?.role === 'admin'}<section class="border border-slate-200 bg-white p-6 shadow-sm sm:p-8"><div class="flex items-center gap-3"><UserPlus class="size-5 text-amber-700" /><h2 class="display-type text-2xl font-semibold">Dashboard users</h2></div><form onsubmit={addUser} class="mt-6 grid gap-4"><input bind:value={email} type="email" required placeholder="person@example.com" aria-label="New user email" class="border border-slate-200 px-3 py-2.5 outline-none focus:border-amber-600" /><input bind:value={password} type="password" minlength="10" maxlength="128" required placeholder="Temporary password" aria-label="Temporary password" class="border border-slate-200 px-3 py-2.5 outline-none focus:border-amber-600" /><select bind:value={role} aria-label="New user role" class="border border-slate-200 px-3 py-2.5"><option value="user">User</option><option value="admin">Administrator</option></select><button class="bg-slate-950 px-4 py-2.5 font-semibold text-white">Add user</button></form><div class="mt-6 divide-y divide-slate-100">{#each users as member}<div class="flex items-center gap-3 py-4"><ShieldCheck class="size-4 text-emerald-600" /><div><p class="font-semibold text-slate-900">{member.email}</p><p class="text-xs text-slate-500">{member.role} · {member.active ? 'active' : 'disabled'}</p></div></div>{/each}</div></section>{/if}
	</div>
	<section class="mt-8 bg-slate-950 p-6 text-white sm:p-8"><p class="text-xs font-bold tracking-wider text-amber-300 uppercase">SDK setup</p><pre class="mt-4 overflow-x-auto text-sm text-slate-300"><code>{`analytics.init({
  apiUrl: 'https://analytics.example.com',
  projectId: '${projectID}',
  trackingToken: 'siraaj_trk_…'
})`}</code></pre></section>
</main>

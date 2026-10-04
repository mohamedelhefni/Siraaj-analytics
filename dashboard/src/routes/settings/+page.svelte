<script>
	import { onMount } from 'svelte';
	import { formatDistanceToNow } from 'date-fns';
	import { Check, ChevronDown, Code, Copy, FolderPlus, KeyRound, Plus, RefreshCw, ShieldCheck, Trash2, UserPlus, Users, X } from 'lucide-svelte';
	import { createTrackingToken, createUser, fetchCurrentUser, fetchProjects, fetchTrackingTokens, fetchUsers, revokeTrackingToken } from '$lib/api.js';

	/** @type {any} */ let currentUser = $state(null);
	/** @type {any[]} */ let tokens = $state([]);
	/** @type {string[]} */ let projectIDs = $state([]);
	/** @type {any[]} */ let users = $state([]);
	let loading = $state(true), error = $state('');

	// Key issuing: `formProject === null` hides the form, '' means "new project".
	/** @type {string | null} */ let formProject = $state(null);
	let projectID = $state(''), tokenName = $state('Website'), submitting = $state(false);
	/** @type {{token: string, project_id: string, name: string} | null} */ let issued = $state(null);

	/** @type {Record<string, boolean>} */ let collapsed = $state({});
	/** @type {Record<string, boolean>} */ let showRevoked = $state({});
	let copied = $state('');

	let email = $state(''), password = $state(''), role = $state('user'), addingUser = $state(false);

	const apiUrl = typeof window === 'undefined' ? '' : window.location.origin;

	let projects = $derived(
		[...new Set([...projectIDs, ...tokens.map((t) => t.project_id)])].sort().map((id) => {
			const keys = tokens.filter((t) => t.project_id === id);
			const active = keys.filter((t) => !t.revoked_at);
			const lastUsed = active.map((t) => t.last_used_at).filter(Boolean).sort().at(-1);
			return { id, active, revoked: keys.filter((t) => t.revoked_at), lastUsed };
		})
	);
	let activeKeyCount = $derived(tokens.filter((t) => !t.revoked_at).length);

	onMount(load);
	async function load() {
		try {
			[currentUser, tokens, projectIDs] = await Promise.all([fetchCurrentUser(), fetchTrackingTokens(), fetchProjects().catch(() => [])]);
			if (currentUser.role === 'admin') users = await fetchUsers();
		} catch (reason) { error = reason instanceof Error ? reason.message : 'Unable to load access settings'; }
		finally { loading = false; }
	}

	/** @param {string} project */
	function openForm(project) {
		formProject = project; projectID = project; tokenName = project ? '' : 'Website'; error = '';
	}

	/** @param {SubmitEvent} event */
	async function issueToken(event) {
		event.preventDefault(); error = ''; submitting = true;
		try {
			const result = await createTrackingToken({ project_id: projectID.trim(), name: tokenName.trim() });
			issued = { token: result.token, project_id: result.project_id, name: result.name };
			collapsed[result.project_id] = false; formProject = null;
			await load();
		} catch (reason) { error = reason instanceof Error ? reason.message : 'Unable to create token'; }
		finally { submitting = false; }
	}

	/** @param {any} token */
	async function revoke(token) {
		if (!confirm(`Revoke "${token.name}" for ${token.project_id}? SDK clients using it will stop sending events.`)) return;
		try { await revokeTrackingToken(token.id); await load(); }
		catch (reason) { error = reason instanceof Error ? reason.message : 'Unable to revoke token'; }
	}

	/** @param {any} token */
	async function regenerate(token) {
		if (!confirm(`Replace "${token.name}" with a new copyable key? The old key stops working, so update your SDK config with the new one.`)) return;
		error = '';
		try {
			const result = await createTrackingToken({ project_id: token.project_id, name: token.name });
			await revokeTrackingToken(token.id);
			issued = { token: result.token, project_id: result.project_id, name: result.name };
			await load();
		} catch (reason) { error = reason instanceof Error ? reason.message : 'Unable to regenerate token'; }
	}

	/** @param {SubmitEvent} event */
	async function addUser(event) {
		event.preventDefault(); error = ''; addingUser = true;
		try { await createUser({ email, password, role }); email = ''; password = ''; role = 'user'; await load(); }
		catch (reason) { error = reason instanceof Error ? reason.message : 'Unable to add user'; }
		finally { addingUser = false; }
	}

	/** @param {string} text @param {string} id */
	async function copy(text, id) {
		await navigator.clipboard.writeText(text);
		copied = id; setTimeout(() => { if (copied === id) copied = ''; }, 1500);
	}

	/** @param {string} project @param {string} token */
	const snippet = (project, token = 'siraaj_trk_…') => `analytics.init({\n  apiUrl: '${apiUrl}',\n  projectId: '${project}',\n  trackingToken: '${token}'\n})`;

	/** @param {string | undefined} date */
	const ago = (date) => (date ? formatDistanceToNow(new Date(date), { addSuffix: true }) : 'never');
</script>

<svelte:head><title>Projects & keys · Siraaj</title></svelte:head>

<main class="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:px-10 lg:py-10">
	<header class="flex flex-wrap items-end justify-between gap-6 border-b border-slate-200 pb-8">
		<div>
			<p class="text-xs font-bold tracking-[.22em] text-amber-700 uppercase">Control room</p>
			<h1 class="display-type mt-2 text-4xl font-semibold tracking-tight text-slate-950">Projects & keys</h1>
			<p class="mt-3 max-w-2xl text-slate-500">Every project is private to your account. Tracking keys let the SDK send events to a project and can be revoked at any time.</p>
		</div>
		<button onclick={() => openForm('')} class="inline-flex items-center gap-2 rounded-lg bg-slate-950 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition hover:bg-slate-800"><FolderPlus class="size-4" /> New project</button>
	</header>

	<dl class="mt-6 grid grid-cols-3 gap-3 sm:max-w-xl">
		{#each [['Projects', projects.length], ['Active keys', activeKeyCount], ['Users', currentUser?.role === 'admin' ? users.length : '—']] as [label, value]}
			<div class="rounded-xl border border-slate-200 bg-white px-4 py-3"><dt class="text-xs font-medium text-slate-500">{label}</dt><dd class="display-type mt-1 text-2xl font-semibold text-slate-950">{value}</dd></div>
		{/each}
	</dl>

	{#if error}<p class="mt-6 rounded-lg border-l-4 border-red-500 bg-red-50 p-4 text-sm text-red-800">{error}</p>{/if}

	{#if issued}
		<section class="mt-6 rounded-xl border border-amber-300 bg-amber-50 p-5 sm:p-6">
			<div class="flex items-start gap-3">
				<KeyRound class="mt-0.5 size-5 shrink-0 text-amber-700" />
				<div class="min-w-0 flex-1">
					<p class="font-semibold text-amber-950">Key "{issued.name}" created for <span class="font-mono">{issued.project_id}</span></p>
					<p class="mt-1 text-sm text-amber-900/80">This key is public — it ships in your site's JavaScript. You can copy it again from the project anytime; revoke it to cut access.</p>
					<div class="mt-4 flex items-center gap-2 rounded-lg border border-amber-200 bg-white p-2 pl-3">
						<code class="min-w-0 flex-1 overflow-x-auto font-mono text-sm whitespace-nowrap text-slate-900">{issued.token}</code>
						<button onclick={() => copy(issued?.token ?? '', 'issued')} class="inline-flex shrink-0 items-center gap-1.5 rounded-md bg-slate-950 px-3 py-1.5 text-xs font-semibold text-white hover:bg-slate-800">{#if copied === 'issued'}<Check class="size-3.5" /> Copied{:else}<Copy class="size-3.5" /> Copy key{/if}</button>
					</div>
					<div class="relative mt-3 rounded-lg bg-slate-950 p-4">
						<pre class="overflow-x-auto pr-10 text-xs leading-5 text-slate-300"><code>{snippet(issued.project_id, issued.token)}</code></pre>
						<button onclick={() => copy(snippet(issued?.project_id ?? '', issued?.token), 'issued-snippet')} aria-label="Copy setup snippet" class="absolute top-3 right-3 grid size-7 place-items-center rounded-md text-slate-400 hover:bg-white/10 hover:text-white">{#if copied === 'issued-snippet'}<Check class="size-3.5" />{:else}<Copy class="size-3.5" />{/if}</button>
					</div>
				</div>
				<button onclick={() => (issued = null)} aria-label="Dismiss" class="grid size-8 place-items-center rounded-lg text-amber-800 hover:bg-amber-100"><X class="size-4" /></button>
			</div>
		</section>
	{/if}

	{#if formProject === ''}
		<form onsubmit={issueToken} class="mt-6 rounded-xl border border-slate-200 bg-white p-5 shadow-sm sm:p-6">
			<h2 class="font-semibold text-slate-950">Create a project</h2>
			<p class="mt-1 text-sm text-slate-500">The project ID goes in your SDK config. Its first tracking key is created with it.</p>
			<div class="mt-5 grid gap-4 sm:grid-cols-2">
				<label class="text-sm font-medium text-slate-700">Project ID
					<input bind:value={projectID} required pattern="[A-Za-z0-9][A-Za-z0-9._\-]{'{0,127}'}" placeholder="my-site" class="mt-1.5 w-full rounded-lg border border-slate-200 px-3 py-2.5 font-mono text-sm outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-600/20" />
					<span class="mt-1 block text-xs font-normal text-slate-400">Letters, numbers, dots, underscores, hyphens.</span>
				</label>
				<label class="text-sm font-medium text-slate-700">First key name
					<input bind:value={tokenName} required maxlength="100" placeholder="Website" class="mt-1.5 w-full rounded-lg border border-slate-200 px-3 py-2.5 text-sm outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-600/20" />
				</label>
			</div>
			<div class="mt-5 flex justify-end gap-2">
				<button type="button" onclick={() => (formProject = null)} class="rounded-lg px-4 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100">Cancel</button>
				<button disabled={submitting} class="rounded-lg bg-slate-950 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800 disabled:opacity-60">{submitting ? 'Creating…' : 'Create project'}</button>
			</div>
		</form>
	{/if}

	<div class="mt-8 grid gap-8 {currentUser?.role === 'admin' ? 'xl:grid-cols-[1fr_22rem]' : ''}">
		<section>
			<h2 class="mb-4 text-sm font-semibold tracking-wide text-slate-500 uppercase">Your projects</h2>
			{#if loading}
				<div class="space-y-3">{#each [1, 2] as _}<div class="h-20 animate-pulse rounded-xl bg-slate-100"></div>{/each}</div>
			{:else if projects.length === 0}
				<div class="rounded-xl border border-dashed border-slate-300 bg-white px-6 py-14 text-center">
					<FolderPlus class="mx-auto size-8 text-slate-300" />
					<p class="mt-3 font-semibold text-slate-900">No projects yet</p>
					<p class="mt-1 text-sm text-slate-500">Create a project to get a tracking key for the SDK.</p>
					<button onclick={() => openForm('')} class="mt-5 inline-flex items-center gap-2 rounded-lg bg-slate-950 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800"><Plus class="size-4" /> New project</button>
				</div>
			{:else}
				<div class="space-y-3">
					{#each projects as project (project.id)}
						<article class="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
							<button onclick={() => (collapsed[project.id] = !collapsed[project.id])} aria-expanded={!collapsed[project.id]} class="flex w-full items-center gap-4 px-5 py-4 text-left hover:bg-slate-50">
								<span class="grid size-10 shrink-0 place-items-center rounded-lg bg-amber-100 font-mono text-sm font-bold text-amber-800 uppercase">{project.id[0]}</span>
								<span class="min-w-0 flex-1">
									<span class="block truncate font-mono font-semibold text-slate-950">{project.id}</span>
									<span class="mt-0.5 block text-xs text-slate-500">{project.active.length} active {project.active.length === 1 ? 'key' : 'keys'} · last event {ago(project.lastUsed)}</span>
								</span>
								{#if project.active.length === 0}<span class="hidden rounded-full bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-600 sm:inline">No active keys</span>
								{:else if project.lastUsed}<span class="hidden items-center gap-1.5 rounded-full bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-700 sm:inline-flex"><span class="size-1.5 rounded-full bg-emerald-500"></span>Receiving</span>
								{:else}<span class="hidden rounded-full bg-amber-50 px-2.5 py-1 text-xs font-medium text-amber-700 sm:inline">Awaiting first event</span>{/if}
								<ChevronDown class="size-4 shrink-0 text-slate-400 transition {collapsed[project.id] ? '' : 'rotate-180'}" />
							</button>

							{#if !collapsed[project.id]}
								<div class="border-t border-slate-100 px-5 py-4">
									<div class="flex items-center justify-between gap-3">
										<p class="text-xs font-semibold tracking-wide text-slate-500 uppercase">Tracking keys</p>
										{#if formProject !== project.id}<button onclick={() => openForm(project.id)} class="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold text-slate-700 hover:bg-slate-50"><Plus class="size-3.5" /> Add key</button>{/if}
									</div>

									{#if formProject === project.id}
										<form onsubmit={issueToken} class="mt-3 flex flex-wrap gap-2 rounded-lg bg-slate-50 p-3">
											<!-- svelte-ignore a11y_autofocus -->
											<input bind:value={tokenName} required maxlength="100" placeholder="Key name, e.g. Staging site" aria-label="Key name" autofocus class="min-w-0 flex-1 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-600/20" />
											<button type="button" onclick={() => (formProject = null)} class="rounded-lg px-3 py-2 text-sm text-slate-600 hover:bg-slate-200">Cancel</button>
											<button disabled={submitting} class="rounded-lg bg-slate-950 px-3 py-2 text-sm font-semibold text-white hover:bg-slate-800 disabled:opacity-60">{submitting ? 'Creating…' : 'Create key'}</button>
										</form>
									{/if}

									<ul class="mt-3 divide-y divide-slate-100">
										{#each project.active as token (token.id)}
											<li class="flex items-center gap-3 py-3">
												<KeyRound class="size-4 shrink-0 text-slate-400" />
												<div class="min-w-0 flex-1">
													<p class="truncate text-sm font-medium text-slate-900">{token.name}</p>
													<p class="truncate text-xs text-slate-500">created {ago(token.created_at)} · used {ago(token.last_used_at)}</p>
													<code class="mt-1.5 block rounded bg-slate-50 px-2 py-1 font-mono text-xs break-all text-slate-700 select-all">{token.token || `${token.prefix}…`}</code>
												</div>
												{#if token.token}<button onclick={() => copy(token.token, token.id)} class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs font-medium text-slate-500 hover:bg-slate-100 hover:text-slate-900">{#if copied === token.id}<Check class="size-3.5" /><span class="hidden sm:inline">Copied</span>{:else}<Copy class="size-3.5" /><span class="hidden sm:inline">Copy</span>{/if}</button>
												{:else}<button onclick={() => regenerate(token)} title="This key was issued before full keys were stored, so it can't be copied. Regenerate creates a copyable replacement and revokes this one." class="inline-flex items-center gap-1.5 rounded-lg bg-amber-50 px-2.5 py-1.5 text-xs font-medium text-amber-800 hover:bg-amber-100"><RefreshCw class="size-3.5" /><span class="hidden sm:inline">Regenerate</span></button>{/if}
												<button onclick={() => revoke(token)} class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs font-medium text-slate-500 hover:bg-red-50 hover:text-red-600"><Trash2 class="size-3.5" /><span class="hidden sm:inline">Revoke</span></button>
											</li>
										{:else}
											<li class="py-3 text-sm text-slate-500">No active keys. Add one to start tracking again.</li>
										{/each}
									</ul>

									{#if project.revoked.length}
										<button onclick={() => (showRevoked[project.id] = !showRevoked[project.id])} class="mt-1 text-xs font-medium text-slate-500 hover:text-slate-800">{showRevoked[project.id] ? 'Hide' : 'Show'} {project.revoked.length} revoked</button>
										{#if showRevoked[project.id]}
											<ul class="mt-2 divide-y divide-slate-100 opacity-60">
												{#each project.revoked as token (token.id)}
													<li class="py-2 text-xs text-slate-500"><span class="font-medium text-slate-700 line-through">{token.name}</span> · <span class="font-mono">{token.prefix}…</span> · revoked {ago(token.revoked_at)}</li>
												{/each}
											</ul>
										{/if}
									{/if}

									<details class="group mt-4 rounded-lg border border-slate-200">
										<summary class="flex cursor-pointer list-none items-center gap-2 px-3 py-2 text-xs font-semibold text-slate-600"><Code class="size-3.5" /> SDK setup <ChevronDown class="ml-auto size-3.5 transition group-open:rotate-180" /></summary>
										<div class="relative bg-slate-950 p-4">
											<pre class="overflow-x-auto pr-10 text-xs leading-5 text-slate-300"><code>{snippet(project.id, project.active.find((t) => t.token)?.token)}</code></pre>
											<button onclick={() => copy(snippet(project.id, project.active.find((t) => t.token)?.token), `snippet-${project.id}`)} aria-label="Copy setup snippet" class="absolute top-3 right-3 grid size-7 place-items-center rounded-md text-slate-400 hover:bg-white/10 hover:text-white">{#if copied === `snippet-${project.id}`}<Check class="size-3.5" />{:else}<Copy class="size-3.5" />{/if}</button>
										</div>
									</details>
								</div>
							{/if}
						</article>
					{/each}
				</div>
			{/if}
		</section>

		{#if currentUser?.role === 'admin'}
			<aside>
				<h2 class="mb-4 flex items-center gap-2 text-sm font-semibold tracking-wide text-slate-500 uppercase"><Users class="size-4" /> Team</h2>
				<div class="rounded-xl border border-slate-200 bg-white shadow-sm">
					<ul class="divide-y divide-slate-100">
						{#each users as member (member.id ?? member.email)}
							<li class="flex items-center gap-3 px-4 py-3">
								<span class="grid size-8 shrink-0 place-items-center rounded-full bg-slate-100 text-xs font-bold text-slate-600 uppercase">{member.email[0]}</span>
								<div class="min-w-0 flex-1"><p class="truncate text-sm font-medium text-slate-900">{member.email}</p><p class="text-xs text-slate-500">{member.active ? 'Active' : 'Disabled'}</p></div>
								<span class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium {member.role === 'admin' ? 'bg-amber-50 text-amber-700' : 'bg-slate-100 text-slate-600'}">{#if member.role === 'admin'}<ShieldCheck class="size-3" />{/if}{member.role}</span>
							</li>
						{/each}
					</ul>
					<form onsubmit={addUser} class="space-y-3 border-t border-slate-100 p-4">
						<p class="flex items-center gap-2 text-sm font-semibold text-slate-900"><UserPlus class="size-4 text-amber-700" /> Invite a user</p>
						<input bind:value={email} type="email" required placeholder="person@example.com" aria-label="New user email" class="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-600/20" />
						<input bind:value={password} type="password" minlength="10" maxlength="128" required placeholder="Temporary password (10+ chars)" aria-label="Temporary password" class="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm outline-none focus:border-amber-600 focus:ring-2 focus:ring-amber-600/20" />
						<div class="flex gap-2">
							<select bind:value={role} aria-label="New user role" class="flex-1 rounded-lg border border-slate-200 px-3 py-2 text-sm"><option value="user">User</option><option value="admin">Administrator</option></select>
							<button disabled={addingUser} class="rounded-lg bg-slate-950 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-800 disabled:opacity-60">{addingUser ? 'Adding…' : 'Add'}</button>
						</div>
						<p class="text-xs text-slate-400">Users only see their own projects — admins included.</p>
					</form>
				</div>
			</aside>
		{/if}
	</div>
</main>

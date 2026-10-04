<script lang="ts">
	import PageHeader from '$lib/components/PageHeader.svelte';
	import UserAvatar, { userName } from '$lib/components/UserAvatar.svelte';
	import { onMount } from 'svelte';
	import { base } from '$app/paths';
	import { format, subDays } from 'date-fns';
	import {
		LoaderCircle,
		MessageSquareText,
		Pause,
		Pencil,
		Play,
		Plus,
		Trash2,
		X
	} from 'lucide-svelte';
	import { Button } from '$lib/components/ui/button';
	import {
		createSurvey,
		deleteSurvey,
		fetchProjects,
		fetchSurveyResponses,
		fetchSurveys,
		fetchStats,
		setSurveyActive,
		updateSurvey
	} from '$lib/api';

	type QuestionType = 'rating' | 'choice' | 'text';
	type Question = { type: QuestionType; text: string; options?: string[] };
	type Frequency = 'once' | 'until_answered' | 'recurring';
	type Survey = {
		id: number;
		project_id: string;
		name: string;
		trigger_event: string;
		delay_seconds: number;
		frequency: Frequency;
		repeat_days: number;
		sample_percent: number;
		questions: Question[];
		active: boolean;
		created_at: string;
		response_count: number;
		shown_count: number;
		dismissed_count: number;
	};
	type SurveyResponse = { id: number; user_id: string; answers: string[]; created_at: string };

	let surveys: Survey[] = $state([]);
	let projects: string[] = $state([]);
	let knownEvents: string[] = $state([]);
	let selected: Survey | null = $state(null);
	let responses: SurveyResponse[] = $state([]);
	let loading = $state(true);
	let creating = $state(false);
	let responsesLoading = $state(false);
	let error = $state<string | null>(null);

	let name = $state('');
	let projectID = $state('');
	let triggerEvent = $state('');
	let customEvent = $state(false);
	let delaySeconds = $state(0);
	let frequency: Frequency = $state('once');
	let repeatDays = $state(30);
	let samplePercent = $state(100);
	let editing: Survey | null = $state(null);
	const CUSTOM = '__custom__';
	const starterQuestion = () => ({
		type: 'rating' as QuestionType,
		text: 'How would you rate your experience?',
		options: ''
	});
	let draft: { type: QuestionType; text: string; options: string }[] = $state([starterQuestion()]);
	const frequencyLabel = (survey: Survey) =>
		survey.frequency === 'recurring'
			? `every ${survey.repeat_days}d`
			: survey.frequency === 'until_answered'
				? `every ${survey.repeat_days}d until answered`
				: 'once';

	const inputClass =
		'h-10 w-full rounded-lg border border-input bg-background px-3 text-sm outline-none focus:border-foreground focus:ring-4 focus:ring-foreground/10';

	onMount(() => {
		void loadPage();
	});

	async function loadPage() {
		try {
			projects = await fetchProjects();
			projectID = projects[0] ?? '';
			fetchStats(
				format(subDays(new Date(), 30), 'yyyy-MM-dd'),
				format(new Date(), 'yyyy-MM-dd'),
				100,
				{}
			)
				.then((stats: any) => (knownEvents = (stats.top_events ?? []).map((e: any) => e.name)))
				.catch((err) => console.error('Failed to load events:', err));
			await loadSurveys();
		} catch (caughtError: any) {
			error = caughtError?.message || 'Failed to load surveys';
			loading = false;
		}
	}

	async function loadSurveys() {
		loading = true;
		try {
			surveys = await fetchSurveys();
			selected = surveys.find((survey) => survey.id === selected?.id) ?? null;
		} catch (caughtError: any) {
			error = caughtError?.message || 'Failed to load surveys';
		} finally {
			loading = false;
		}
	}

	function resetForm() {
		editing = null;
		name = '';
		triggerEvent = '';
		customEvent = false;
		delaySeconds = 0;
		frequency = 'once';
		repeatDays = 30;
		samplePercent = 100;
		draft = [starterQuestion()];
	}

	function edit(survey: Survey) {
		editing = survey;
		name = survey.name;
		projectID = survey.project_id;
		triggerEvent = survey.trigger_event;
		customEvent = !knownEvents.includes(survey.trigger_event);
		delaySeconds = survey.delay_seconds;
		frequency = survey.frequency;
		repeatDays = survey.repeat_days || 30;
		samplePercent = survey.sample_percent;
		draft = survey.questions.map((question) => ({
			type: question.type,
			text: question.text,
			options: (question.options ?? []).join(', ')
		}));
		window.scrollTo({ top: 0, behavior: 'smooth' });
	}

	async function save() {
		creating = true;
		error = null;
		try {
			const body = {
				project_id: projectID,
				name,
				trigger_event: triggerEvent,
				delay_seconds: delaySeconds,
				frequency,
				repeat_days: frequency === 'once' ? 0 : repeatDays,
				sample_percent: samplePercent,
				questions: draft.map((question) => ({
					type: question.type,
					text: question.text,
					...(question.type === 'choice' && {
						options: question.options
							.split(',')
							.map((option) => option.trim())
							.filter(Boolean)
					})
				}))
			};
			const id = editing ? editing.id : (await createSurvey(body)).id;
			if (editing) await updateSurvey(id, body);
			resetForm();
			await loadSurveys();
			const saved = surveys.find((survey) => survey.id === id);
			if (saved) await select(saved);
		} catch (caughtError: any) {
			error = caughtError?.message || 'Failed to save survey';
		} finally {
			creating = false;
		}
	}

	async function select(survey: Survey) {
		selected = survey;
		responsesLoading = true;
		try {
			responses = await fetchSurveyResponses(survey.id);
		} catch (caughtError: any) {
			error = caughtError?.message || 'Failed to load responses';
		} finally {
			responsesLoading = false;
		}
	}

	async function toggle(survey: Survey) {
		try {
			await setSurveyActive(survey.id, !survey.active);
			await loadSurveys();
		} catch (caughtError: any) {
			error = caughtError?.message || 'Failed to update survey';
		}
	}

	async function remove(survey: Survey) {
		if (!confirm(`Delete "${survey.name}" and all ${survey.response_count} responses?`)) return;
		try {
			await deleteSurvey(survey.id);
			if (selected?.id === survey.id) selected = null;
			await loadSurveys();
		} catch (caughtError: any) {
			error = caughtError?.message || 'Failed to delete survey';
		}
	}

	// Outcome split of everyone who saw the survey. Responses recorded before
	// impressions were counted can exceed shown_count, so the base never drops below them.
	function outcomes(survey: Survey) {
		const shown = Math.max(survey.shown_count, survey.response_count + survey.dismissed_count);
		const pct = (count: number) => (shown ? (count / shown) * 100 : 0);
		const rows = [
			{ label: 'Submitted', count: survey.response_count, color: 'bg-emerald-500' },
			{ label: 'Dismissed', count: survey.dismissed_count, color: 'bg-rose-500' },
			{
				label: 'No action',
				count: shown - survey.response_count - survey.dismissed_count,
				color: 'bg-muted-foreground/40'
			}
		].map((row) => ({ ...row, pct: pct(row.count) }));
		return { shown, rows };
	}

	// Per-question tallies: counts for rating/choice, recent answers for text.
	function summarize(question: Question, index: number) {
		const answers = responses.map((response) => response.answers[index]).filter(Boolean);
		const values =
			question.type === 'rating' ? ['1', '2', '3', '4', '5'] : (question.options ?? []);
		const counts = values.map((value) => ({
			value,
			count: answers.filter((a) => a === value).length
		}));
		const average =
			question.type === 'rating' && answers.length
				? answers.reduce((sum, answer) => sum + Number(answer), 0) / answers.length
				: null;
		return { answers, counts, average };
	}
</script>

<svelte:head>
	<title>Surveys · Siraaj</title>
</svelte:head>

<main class="mx-auto max-w-[1440px] px-4 py-6 sm:px-6 lg:px-8">
	<PageHeader
		class="mb-6"
		title="Surveys"
		description="Show a short survey when an event fires, once or on a repeating schedule per visitor. Requires the SDK configured with a trackingToken."
	/>

	{#if error}
		<div
			class="mb-6 rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
			role="alert"
		>
			{error}
		</div>
	{/if}

	<form
		class="mb-10 grid gap-5 rounded-2xl border border-border bg-card p-7 shadow-sm sm:grid-cols-3"
		onsubmit={(event) => {
			event.preventDefault();
			save();
		}}
	>
		{#if editing}
			<p
				class="rounded-lg border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-sm sm:col-span-3"
			>
				Editing <strong>{editing.name}</strong>. Answers are stored by question position, so
				reordering or removing questions changes how earlier responses line up.
			</p>
		{/if}
		<label>
			<span class="mb-2 block text-sm font-medium">Name</span>
			<input
				bind:value={name}
				required
				maxlength="500"
				placeholder="Post-checkout NPS"
				class={inputClass}
			/>
		</label>
		<div>
			<span class="mb-2 block text-sm font-medium">Show when event fires</span>
			<select
				value={customEvent ? CUSTOM : triggerEvent}
				onchange={(e) => {
					const v = e.currentTarget.value;
					customEvent = v === CUSTOM;
					triggerEvent = customEvent ? '' : v;
				}}
				required
				class={inputClass}
			>
				<option value="">Select event...</option>
				{#each knownEvents as event}<option value={event}>{event}</option>{/each}
				<option value={CUSTOM}>Custom event…</option>
			</select>
			{#if customEvent}
				<input
					bind:value={triggerEvent}
					required
					maxlength="500"
					placeholder="Custom event name"
					class="{inputClass} mt-2"
				/>
			{/if}
		</div>
		<label>
			<span class="mb-2 block text-sm font-medium">Delay after event (seconds)</span>
			<input
				type="number"
				bind:value={delaySeconds}
				min="0"
				max="3600"
				step="1"
				required
				class={inputClass}
			/>
		</label>
		<label>
			<span class="mb-2 block text-sm font-medium">Project</span>
			<select bind:value={projectID} required disabled={!!editing} class={inputClass}>
				<option value="" disabled
					>{projects.length ? 'Choose a project' : 'Create a tracking key first'}</option
				>
				{#each projects as project}<option value={project}>{project}</option>{/each}
			</select>
		</label>
		<div>
			<span class="mb-2 block text-sm font-medium">Show to each visitor</span>
			<select bind:value={frequency} class={inputClass}>
				<option value="once">Once, never again</option>
				<option value="until_answered">Repeatedly until they answer</option>
				<option value="recurring">Repeatedly, even after answering</option>
			</select>
			{#if frequency !== 'once'}
				<label class="mt-2 flex items-center gap-2 text-sm text-muted-foreground">
					Every
					<input
						type="number"
						bind:value={repeatDays}
						min="1"
						max="365"
						step="1"
						required
						aria-label="Days between showings"
						class="{inputClass} w-24"
					/>
					days
				</label>
			{/if}
		</div>
		<label>
			<span class="mb-2 block text-sm font-medium">Visitors who can see it (%)</span>
			<input
				type="number"
				bind:value={samplePercent}
				min="1"
				max="100"
				step="1"
				required
				class={inputClass}
			/>
		</label>

		<fieldset class="space-y-3 sm:col-span-3">
			<legend class="mb-2 text-sm font-medium">Questions</legend>
			{#each draft as question, i}
				<div class="grid gap-2 rounded-lg border border-border p-3 sm:grid-cols-[9rem_1fr_auto]">
					<select bind:value={question.type} aria-label="Question {i + 1} type" class={inputClass}>
						<option value="rating">Rating 1–5</option>
						<option value="choice">Single choice</option>
						<option value="text">Free text</option>
					</select>
					<input
						bind:value={question.text}
						required
						maxlength="500"
						placeholder="Question"
						aria-label="Question {i + 1} text"
						class={inputClass}
					/>
					<button
						type="button"
						onclick={() => draft.splice(i, 1)}
						disabled={draft.length === 1}
						aria-label="Remove question {i + 1}"
						class="grid size-10 place-items-center rounded-lg transition hover:bg-accent disabled:opacity-30"
					>
						<X class="size-4" />
					</button>
					{#if question.type === 'choice'}
						<input
							bind:value={question.options}
							required
							placeholder="Options, comma separated: Price, Speed, Support"
							aria-label="Question {i + 1} options"
							class="{inputClass} sm:col-start-2"
						/>
					{/if}
				</div>
			{/each}
		</fieldset>

		<div class="flex items-center justify-between gap-4 sm:col-span-3">
			<Button
				type="button"
				variant="outline"
				size="sm"
				disabled={draft.length >= 10}
				onclick={() => draft.push({ type: 'text', text: '', options: '' })}
			>
				<Plus /> Add question
			</Button>
			<div class="flex gap-2">
				{#if editing}
					<Button type="button" variant="outline" onclick={resetForm}>Cancel</Button>
				{/if}
				<Button type="submit" disabled={creating} class="min-w-32">
					{#if creating}<LoaderCircle class="animate-spin" />{/if}
					{creating ? 'Saving…' : editing ? 'Save changes' : 'Create survey'}
				</Button>
			</div>
		</div>
	</form>

	<div class="grid gap-8 xl:grid-cols-[0.95fr_1.45fr]">
		<section>
			<div class="mb-4 flex items-center justify-between">
				<h2 class="text-lg font-semibold">Your surveys</h2>
				<span class="text-sm text-muted-foreground">{surveys.length} total</span>
			</div>
			<div class="space-y-3">
				{#if loading}
					<div
						class="flex h-40 items-center justify-center rounded-xl border border-border text-muted-foreground"
					>
						<LoaderCircle class="mr-2 size-4 animate-spin" /> Loading surveys
					</div>
				{:else if surveys.length === 0}
					<div class="rounded-xl border border-dashed border-border bg-muted/20 p-10 text-center">
						<MessageSquareText class="mx-auto mb-3 size-6 text-muted-foreground" />
						<p class="font-medium">No surveys yet</p>
					</div>
				{:else}
					{#each surveys as survey}
						<article
							class="rounded-xl border border-border p-4 transition hover:border-foreground/30 {selected?.id ===
							survey.id
								? 'bg-muted/60 ring-2 ring-foreground/10'
								: 'bg-card'}"
						>
							<button type="button" onclick={() => select(survey)} class="w-full text-left">
								<div class="mb-3 flex items-start justify-between gap-4">
									<div class="min-w-0">
										<p class="truncate font-semibold">{survey.name}</p>
										<p class="mt-1 truncate text-xs text-muted-foreground">
											on <code>{survey.trigger_event}</code>{survey.delay_seconds
												? ` after ${survey.delay_seconds}s`
												: ''} · {frequencyLabel(survey)}{survey.sample_percent < 100
												? ` · ${survey.sample_percent}% of visitors`
												: ''} · {survey.questions.length} questions
										</p>
									</div>
									<div class="text-right">
										<p class="text-xl font-semibold tabular-nums">
											{survey.response_count.toLocaleString()}
										</p>
										<p class="text-[11px] text-muted-foreground uppercase">responses</p>
									</div>
								</div>
							</button>
							<div class="flex items-center justify-between border-t border-border pt-3">
								<span class="text-xs text-muted-foreground">
									{survey.project_id} · {survey.active ? 'Live' : 'Paused'} · {outcomes(
										survey
									).rows[0].pct.toFixed(0)}% response rate
								</span>
								<div class="flex gap-1">
									<button
										type="button"
										onclick={() => toggle(survey)}
										aria-label={survey.active ? 'Pause survey' : 'Resume survey'}
										class="rounded-md p-1.5 transition hover:bg-accent"
									>
										{#if survey.active}<Pause class="size-4" />{:else}<Play class="size-4" />{/if}
									</button>
									<button
										type="button"
										onclick={() => edit(survey)}
										aria-label="Edit survey"
										class="rounded-md p-1.5 transition hover:bg-accent"
									>
										<Pencil class="size-4" />
									</button>
									<button
										type="button"
										onclick={() => remove(survey)}
										aria-label="Delete survey"
										class="rounded-md p-1.5 transition hover:bg-accent"
									>
										<Trash2 class="size-4" />
									</button>
								</div>
							</div>
						</article>
					{/each}
				{/if}
			</div>
		</section>

		<section class="min-w-0">
			<h2 class="mb-4 text-lg font-semibold">{selected ? selected.name : 'Results'}</h2>
			{#if !selected}
				<div
					class="flex h-96 flex-col items-center justify-center rounded-2xl border border-dashed border-border bg-muted/15 text-center"
				>
					<MessageSquareText class="mb-3 size-7 text-muted-foreground" />
					<p class="font-medium">Choose a survey</p>
					<p class="mt-1 max-w-xs text-sm text-muted-foreground">
						Per-question results will appear here.
					</p>
				</div>
			{:else if responsesLoading}
				<div
					class="flex h-96 items-center justify-center rounded-2xl border border-border text-muted-foreground"
				>
					<LoaderCircle class="mr-2 size-4 animate-spin" /> Loading responses
				</div>
			{:else}
				{@const stats = outcomes(selected)}
				<div class="space-y-5">
					<div class="rounded-xl border border-border bg-card p-5">
						<div class="mb-4 flex items-baseline justify-between gap-4">
							<h3 class="font-semibold">Engagement</h3>
							<span class="text-xs text-muted-foreground"
								>{stats.shown.toLocaleString()} times shown</span
							>
						</div>
						<div class="grid gap-3 sm:grid-cols-3">
							{#each stats.rows as row}
								<div>
									<p class="flex items-center gap-2 text-xs text-muted-foreground uppercase">
										<span class="size-2 rounded-full {row.color}"></span>{row.label}
									</p>
									<p class="mt-1 text-2xl font-semibold tabular-nums">{row.pct.toFixed(1)}%</p>
									<p class="text-xs text-muted-foreground tabular-nums">
										{row.count.toLocaleString()}
									</p>
								</div>
							{/each}
						</div>
						<div
							class="mt-4 flex h-2 overflow-hidden rounded-full bg-muted"
							role="img"
							aria-label={stats.rows.map((row) => `${row.label} ${row.pct.toFixed(1)}%`).join(', ')}
						>
							{#each stats.rows as row}
								<div class={row.color} style={`width: ${row.pct}%`}></div>
							{/each}
						</div>
					</div>
					{#each selected.questions as question, i}
						{@const summary = summarize(question, i)}
						<div class="rounded-xl border border-border bg-card p-5">
							<div class="mb-4 flex items-start justify-between gap-4">
								<h3 class="font-semibold">{question.text}</h3>
								<span class="shrink-0 text-xs text-muted-foreground">
									{summary.answers.length} answers{summary.average !== null
										? ` · avg ${summary.average.toFixed(1)}`
										: ''}
								</span>
							</div>
							{#if question.type === 'text'}
								<ul class="max-h-72 space-y-2 overflow-y-auto">
									{#each summary.answers.slice(0, 100) as answer}
										<li class="rounded-lg bg-muted/40 px-3 py-2 text-sm">{answer}</li>
									{:else}<li class="py-5 text-center text-sm text-muted-foreground">
											No answers yet.
										</li>{/each}
								</ul>
							{:else}
								<div class="space-y-3">
									{#each summary.counts as entry}
										<div>
											<div class="mb-1 flex justify-between gap-4 text-sm">
												<span class="truncate">{entry.value}</span>
												<span class="font-medium tabular-nums">{entry.count}</span>
											</div>
											<div class="h-1 overflow-hidden rounded-full bg-muted">
												<div
													class="h-full rounded-full bg-foreground"
													style={`width: ${(entry.count / Math.max(summary.answers.length, 1)) * 100}%`}
												></div>
											</div>
										</div>
									{/each}
								</div>
							{/if}
						</div>
					{/each}
					<div class="rounded-xl border border-border bg-card p-5">
						<div class="mb-4 flex items-baseline justify-between gap-4">
							<h3 class="font-semibold">Responses by user</h3>
							<span class="text-xs text-muted-foreground"
								>{new Set(responses.map((r) => r.user_id).filter(Boolean)).size} users</span
							>
						</div>
						<ul class="max-h-96 space-y-2 overflow-y-auto">
							{#each responses.slice(0, 100) as response (response.id)}
								<li class="rounded-lg bg-muted/40 px-3 py-2 text-sm">
									<div class="flex items-baseline justify-between gap-4">
										{#if response.user_id}
											<a
												href="{base}/users?project={encodeURIComponent(
													selected.project_id
												)}&id={encodeURIComponent(response.user_id)}"
												class="flex min-w-0 items-center gap-2 text-xs font-medium underline-offset-2 hover:underline"
												title={response.user_id}
												><UserAvatar id={response.user_id} size={22} /><span class="truncate"
													>{userName(response.user_id)}</span
												></a
											>
										{:else}
											<span class="text-xs text-muted-foreground">Anonymous</span>
										{/if}
										<time class="shrink-0 text-xs text-muted-foreground tabular-nums"
											>{format(new Date(response.created_at), 'MMM d, HH:mm')}</time
										>
									</div>
									<p class="mt-1 truncate text-muted-foreground">
										{response.answers.filter(Boolean).join(' · ')}
									</p>
								</li>
							{:else}<li class="py-5 text-center text-sm text-muted-foreground">
									No responses yet.
								</li>{/each}
						</ul>
					</div>
				</div>
			{/if}
		</section>
	</div>
</main>

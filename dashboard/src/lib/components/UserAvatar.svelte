<script module lang="ts">
	const ADJECTIVES = [
		'Amber',
		'Brave',
		'Calm',
		'Clever',
		'Cosmic',
		'Coral',
		'Crimson',
		'Dusty',
		'Eager',
		'Fuzzy',
		'Gentle',
		'Golden',
		'Happy',
		'Indigo',
		'Jolly',
		'Lucky',
		'Lunar',
		'Mellow',
		'Misty',
		'Noble',
		'Olive',
		'Quiet',
		'Rapid',
		'Rusty',
		'Silver',
		'Sleepy',
		'Snowy',
		'Sunny',
		'Swift',
		'Teal',
		'Velvet',
		'Witty'
	];
	const ANIMALS = [
		'Badger',
		'Bear',
		'Beaver',
		'Bison',
		'Camel',
		'Crane',
		'Dolphin',
		'Eagle',
		'Falcon',
		'Ferret',
		'Fox',
		'Gazelle',
		'Heron',
		'Ibex',
		'Koala',
		'Lemur',
		'Lynx',
		'Marten',
		'Moose',
		'Narwhal',
		'Otter',
		'Owl',
		'Panda',
		'Penguin',
		'Puffin',
		'Raven',
		'Seal',
		'Sparrow',
		'Tiger',
		'Turtle',
		'Walrus',
		'Wolf'
	];
	const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

	function hash(id: string) {
		let h = 2166136261;
		for (const c of id) h = Math.imul(h ^ c.charCodeAt(0), 16777619) >>> 0;
		return h;
	}

	// The SDK generates a UUID until identify() is called, so anything else came from the site.
	export const isIdentified = (id: string) => !UUID.test(id);

	// Anonymous visitors get a stable friendly name; identified ones keep their own id.
	export function userName(id: string) {
		if (isIdentified(id)) return id;
		const h = hash(id);
		return `${ADJECTIVES[h % ADJECTIVES.length]} ${ANIMALS[(h >>> 8) % ANIMALS.length]}`;
	}
</script>

<script lang="ts">
	let { id, size = 36 }: { id: string; size?: number } = $props();

	const h = $derived(hash(id));
	const initials = $derived(
		userName(id)
			.split(/[^\p{L}\p{N}]+/u)
			.filter(Boolean)
			.slice(0, 2)
			.map((word, _, words) => (words.length === 1 ? word.slice(0, 2) : word[0]))
			.join('')
			.toUpperCase()
	);
</script>

<span
	class="inline-flex shrink-0 items-center justify-center rounded-full font-semibold text-white shadow-sm ring-2 ring-background"
	style="width:{size}px;height:{size}px;font-size:{size *
		0.36}px;background:linear-gradient(135deg, hsl({h % 360} 70% 55%), hsl({(h >>> 9) %
		360} 65% 40%))"
	aria-hidden="true">{initials}</span
>

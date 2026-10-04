<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import * as d3 from 'd3';
	import * as topojson from 'topojson-client';
	import { ZoomIn, ZoomOut, RotateCcw } from 'lucide-svelte';

	let { data = [], onclick = null } = $props();
	let svgElement: SVGSVGElement | undefined = $state();
	let mapContainer: HTMLDivElement | undefined = $state();
	let zoomBehavior: any = $state();
	let svgSelection: any = $state();
	let tooltipElement: any = null;
	let cleanupTooltip: (() => void) | null = null;

	// World map TopoJSON URL (using Natural Earth data)
	const WORLD_MAP_URL = 'https://cdn.jsdelivr.net/npm/world-atlas@2/countries-110m.json';

	// Helper function to normalize country names
	function normalizeCountryName(name: string | undefined): string | undefined {
		if (!name) return name;
		// Replace Israel with Palestine
		if (name.toLowerCase() === 'israel') {
			return 'Palestine';
		}
		return name;
	}

	function handleZoomIn() {
		if (svgSelection && zoomBehavior) {
			svgSelection.transition().duration(300).call(zoomBehavior.scaleBy, 1.5);
		}
	}

	function handleZoomOut() {
		if (svgSelection && zoomBehavior) {
			svgSelection.transition().duration(300).call(zoomBehavior.scaleBy, 0.67);
		}
	}

	function handleResetZoom() {
		if (svgSelection && zoomBehavior) {
			svgSelection.transition().duration(500).call(zoomBehavior.transform, d3.zoomIdentity);
		}
	}

	async function drawMap() {
		if (!svgElement || !mapContainer) return;

		try {
			const isFirstDraw = d3.select(svgElement).selectAll('g').empty();

			// Only clear on first draw
			if (isFirstDraw) {
				d3.select(svgElement).selectAll('*').remove();
			}

			// Clean up old tooltip if exists
			if (cleanupTooltip) {
				cleanupTooltip();
				cleanupTooltip = null;
			}

			const width = mapContainer.clientWidth;
			const height = 400;

			if (width === 0) {
				console.warn('Map container has zero dimensions');
				return;
			}

			svgSelection = d3
				.select(svgElement)
				.attr('width', width)
				.attr('height', height)
				.attr('viewBox', [0, 0, width, height]);

			// Create a container group for zoom (only once)
			let g = svgSelection.select('g.map-group');
			if (g.empty()) {
				g = svgSelection.append('g').attr('class', 'map-group');
			}

			// Create projection
			const projection = d3
				.geoNaturalEarth1()
				.scale(width / 6)
				.translate([width / 2, height / 2]);

			const path = d3.geoPath(projection);

			// Setup zoom behavior
			zoomBehavior = d3
				.zoom()
				.scaleExtent([1, 8]) // Min and max zoom levels
				.on('zoom', (event: any) => {
					g.attr('transform', event.transform);
				});

			svgSelection.call(zoomBehavior);

			// Create color scale based on data
			const maxCount = d3.max(data, (d: any) => d.count) || 1;
			const colorScale = d3.scaleSequential(d3.interpolateRgb('#e2e8f0', '#1e293b')).domain([0, maxCount]);

			// Create a map of country data for quick lookup
			// Normalize country names (replace Israel with Palestine)
			const countryDataMap = new Map(
				data.map((d: any) => [normalizeCountryName(d.country)?.toLowerCase(), d.count])
			);

			// Load world map data
			const worldData: any = await d3.json(WORLD_MAP_URL);
			const countries: any = topojson.feature(worldData, worldData.objects.countries);

			// Create or reuse tooltip
			if (!tooltipElement) {
				tooltipElement = d3
					.select('body')
					.append('div')
					.style('position', 'absolute')
					.style('background', 'rgba(0, 0, 0, 0.8)')
					.style('color', 'white')
					.style('padding', '8px 12px')
					.style('border-radius', '4px')
					.style('font-size', '14px')
					.style('pointer-events', 'none')
					.style('opacity', 0)
					.style('z-index', '1000')
					.style('transition', 'opacity 0.2s ease');
			}

			const tooltip = tooltipElement;

			// Draw countries in the zoomable group with smooth data updates
			const countries_paths = g.selectAll('path').data(countries.features, (d: any) => d.id);

			// Enter new countries
			countries_paths
				.enter()
				.append('path')
				.attr('d', path)
				.attr('stroke', '#fff')
				.attr('stroke-width', 0.5)
				.style('cursor', 'pointer')
				.style('opacity', 0)
				.merge(countries_paths)
				.attr('fill', (d: any) => {
					let countryName = normalizeCountryName(d.properties.name);
					const count = countryDataMap.get(countryName?.toLowerCase());
					return count ? colorScale(count) : '#f1f5f9';
				})
				.on('mouseover', function (this: any, event: any, d: any) {
					const countryName = normalizeCountryName(d.properties.name);
					const count = countryDataMap.get(countryName?.toLowerCase());

					if (count) {
						d3.select(this)
							.transition()
							.duration(200)
							.attr('stroke-width', 2)
							.attr('stroke', '#0f172a');

						tooltip.transition().duration(200).style('opacity', 1);

						tooltip
							.html(`<strong>${countryName}</strong><br/>Visitors: <strong>${count}</strong>`)
							.style('left', event.pageX + 10 + 'px')
							.style('top', event.pageY - 10 + 'px');
					}
				})
				.on('mouseout', function (this: any) {
					d3.select(this)
						.transition()
						.duration(200)
						.attr('stroke-width', 0.5)
						.attr('stroke', '#fff');

					tooltip.transition().duration(200).style('opacity', 0);
				})
				.on('click', function (event: any, d: any) {
					const countryName = normalizeCountryName(d.properties.name);
					const count = countryDataMap.get(countryName?.toLowerCase());

					if (count && onclick) {
						onclick({ name: countryName, count });
					}
				})
				.transition()
				.duration(isFirstDraw ? 800 : 500)
				.delay(isFirstDraw ? (_d: any, i: number) => i * 2 : 0)
				.style('opacity', 1);

			// Remove old countries
			countries_paths.exit().transition().duration(300).style('opacity', 0).remove();

			// Add legend
			const legendWidth = 200;
			const legendHeight = 10;
			const legendX = width - legendWidth - 20;
			const legendY = height - 40;

			const legendScale = d3.scaleLinear().domain([0, maxCount]).range([0, legendWidth]);

			const legendAxis = d3.axisBottom(legendScale).ticks(5).tickFormat(d3.format('.0f'));

			// Create gradient for legend
			const defs = svgSelection.append('defs');
			const linearGradient = defs
				.append('linearGradient')
				.attr('id', 'legend-gradient')
				.attr('x1', '0%')
				.attr('x2', '100%');

			linearGradient
				.selectAll('stop')
				.data(d3.range(0, 1.1, 0.1))
				.join('stop')
				.attr('offset', (d: number) => d * 100 + '%')
				.attr('stop-color', (d: number) => colorScale(d * maxCount));

			// Draw legend
			const legend = svgSelection.append('g').attr('transform', `translate(${legendX},${legendY})`);

			legend
				.append('rect')
				.attr('width', legendWidth)
				.attr('height', legendHeight)
				.style('fill', 'url(#legend-gradient)')
				.attr('stroke', '#ccc')
				.attr('stroke-width', 1);

			legend
				.append('g')
				.attr('transform', `translate(0,${legendHeight})`)
				.call(legendAxis)
				.selectAll('text')
				.style('font-size', '10px')
				.style('fill', '#64748b');

			legend
				.append('text')
				.attr('x', legendWidth / 2)
				.attr('y', -5)
				.attr('text-anchor', 'middle')
				.style('font-size', '12px')
				.style('fill', '#334155')
				.text('Visitors');

			// Cleanup function for tooltip
			cleanupTooltip = () => {
				if (tooltipElement) {
					tooltipElement.remove();
					tooltipElement = null;
				}
			};
		} catch (error) {
			// Error loading map data - show error message
			if (svgSelection && mapContainer) {
				const containerWidth = mapContainer.clientWidth || 800;
				const containerHeight = 400;
				svgSelection
					.append('text')
					.attr('x', containerWidth / 2)
					.attr('y', containerHeight / 2)
					.attr('text-anchor', 'middle')
					.attr('fill', '#64748b')
					.text('Error loading map data');
			}
		}
	}

	onMount(() => {
		// Use setTimeout to ensure container is fully rendered
		const timeoutId = setTimeout(() => {
			drawMap();
		}, 100);

		// Handle window resize
		const handleResize = () => {
			drawMap();
		};

		window.addEventListener('resize', handleResize);

		return () => {
			clearTimeout(timeoutId);
			window.removeEventListener('resize', handleResize);
			if (cleanupTooltip) {
				cleanupTooltip();
			}
		};
	});

	// Watch for data changes - redraw when data changes
	$effect(() => {
		// Track the data dependency
		data;

		if (mapContainer && data.length > 0) {
			untrack(() => {
				drawMap();
			});
		}
	});
</script>

<div bind:this={mapContainer} class="relative h-[400px] w-full">
	{#if data.length === 0}
		<div class="text-muted-foreground flex h-full items-center justify-center">
			<p>No geographical data available for the selected period</p>
		</div>
	{:else}
		<svg bind:this={svgElement} class="w-full"></svg>

		<!-- Zoom Controls -->
		<div class="absolute right-4 top-4 flex flex-col gap-1 rounded-md bg-white shadow-md">
			<button
				onclick={handleZoomIn}
				class="hover:bg-accent p-2 transition-colors"
				title="Zoom In"
				type="button"
			>
				<ZoomIn class="h-4 w-4" />
			</button>
			<div class="bg-border h-px"></div>
			<button
				onclick={handleZoomOut}
				class="hover:bg-accent p-2 transition-colors"
				title="Zoom Out"
				type="button"
			>
				<ZoomOut class="h-4 w-4" />
			</button>
			<div class="bg-border h-px"></div>
			<button
				onclick={handleResetZoom}
				class="hover:bg-accent p-2 transition-colors"
				title="Reset Zoom"
				type="button"
			>
				<RotateCcw class="h-4 w-4" />
			</button>
		</div>
	{/if}
</div>

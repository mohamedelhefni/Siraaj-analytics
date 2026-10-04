<script>
	import { onMount, untrack } from 'svelte';
	import * as d3 from 'd3';
	import { Button } from '$lib/components/ui/button';
	import { Eye, EyeOff } from 'lucide-svelte';
	import { formatCompactNumber } from '$lib/utils/formatters.js';

	let {
		data = [],
		comparisonData = [],
		format = 'day',
		metric = 'events',
		showComparison = $bindable(true)
	} = $props();
	let svgElement = $state();
	let chartContainer = $state();

	// Get label based on metric type
	const metricLabels = {
		events: 'Events',
		users: 'Unique Visitors',
		visits: 'Total Visits',
		page_views: 'Page Views',
		views_per_visit: 'Views per Visit',
		bounce_rate: 'Bounce Rate',
		visit_duration: 'Visit Duration'
	};

	// Format the count value based on metric type
	function formatCount(count, metricType) {
		if (metricType === 'bounce_rate') {
			return count.toFixed(1) + '%';
		} else if (metricType === 'visit_duration') {
			// Format seconds into readable duration
			if (count < 60) {
				return Math.floor(count) + 's';
			} else if (count < 3600) {
				const minutes = Math.floor(count / 60);
				const seconds = Math.floor(count % 60);
				return `${minutes}m ${seconds}s`;
			} else {
				const hours = Math.floor(count / 3600);
				const minutes = Math.floor((count % 3600) / 60);
				return `${hours}h ${minutes}m`;
			}
		} else if (metricType === 'views_per_visit') {
			return count.toFixed(2);
		} else {
			return formatCompactNumber(Math.round(count));
		}
	}

	// Format date labels based on granularity
	function formatLabel(dateStr) {
		const date = new Date(dateStr);

		if (format === 'hour') {
			return d3.timeFormat('%H:%M')(date);
		} else if (format === 'month') {
			return d3.timeFormat('%b %Y')(date);
		} else {
			return d3.timeFormat('%b %d')(date);
		}
	}

	function formatTooltipDate(dateStr) {
		const date = new Date(dateStr);

		if (format === 'hour') {
			return d3.timeFormat('%b %d, %H:%M')(date);
		} else if (format === 'month') {
			return d3.timeFormat('%B %Y')(date);
		} else {
			return d3.timeFormat('%B %d, %Y')(date);
		}
	}

	let tooltipElement = $state(null);
	let cleanupTooltip = $state(null);

	function drawChart() {
		if (!svgElement || !chartContainer || data.length === 0) return;

		// Remove old tooltip if it exists
		if (cleanupTooltip) {
			cleanupTooltip();
			cleanupTooltip = null;
		}

		const isFirstDraw = d3.select(svgElement).selectAll('g').empty();

		// Only clear on first draw, otherwise update
		if (isFirstDraw) {
			d3.select(svgElement).selectAll('*').remove();
		}

		const margin = { top: 20, right: 30, bottom: 30, left: 50 };
		const width = chartContainer.clientWidth - margin.left - margin.right;
		const height = 300 - margin.top - margin.bottom;

		const svgSelection = d3
			.select(svgElement)
			.attr('width', width + margin.left + margin.right)
			.attr('height', height + margin.top + margin.bottom);

		let svg = svgSelection.select('g.chart-group');
		if (svg.empty()) {
			svg = svgSelection
				.append('g')
				.attr('class', 'chart-group')
				.attr('transform', `translate(${margin.left},${margin.top})`);
		}

		// Parse dates and prepare data
		const parsedData = data.map((d) => ({
			date: new Date(d.date),
			count: d.count
		}));

		// Parse comparison data if available
		const parsedComparisonData = comparisonData.map((d) => ({
			date: new Date(d.date),
			count: d.count
		}));

		// Create scales
		const x = d3
			.scaleTime()
			.domain(d3.extent(parsedData, (d) => d.date))
			.range([0, width]);

		// Calculate max value from both datasets for consistent y-axis
		const maxValue = d3.max([
			d3.max(parsedData, (d) => d.count),
			showComparison && parsedComparisonData.length > 0
				? d3.max(parsedComparisonData, (d) => d.count)
				: 0
		]);

		const y = d3.scaleLinear().domain([0, maxValue]).nice().range([height, 0]);

		// Create line generator
		const line = d3
			.line()
			.x((d) => x(d.date))
			.y((d) => y(d.count))
			.curve(d3.curveMonotoneX);

		// Create area generator for fill
		const area = d3
			.area()
			.x((d) => x(d.date))
			.y0(height)
			.y1((d) => y(d.count))
			.curve(d3.curveMonotoneX);

		// Update or create X axis
		const xAxisGroup = svg.selectAll('.x-axis').data([null]);
		const xAxisEnter = xAxisGroup
			.enter()
			.append('g')
			.attr('class', 'x-axis')
			.attr('transform', `translate(0,${height})`);

		xAxisGroup
			.merge(xAxisEnter)
			.transition()
			.duration(isFirstDraw ? 0 : 500)
			.call(
				d3
					.axisBottom(x)
					.ticks(Math.min(parsedData.length, 7))
					.tickFormat((d) => formatLabel(d))
			)
			.selectAll('text')
			.style('font-size', '12px')
			.style('fill', '#64748b');

		// Update or create Y axis
		const yAxis = d3.axisLeft(y).ticks(5);

		// Format Y axis based on metric type
		if (metric === 'bounce_rate') {
			yAxis.tickFormat((d) => d.toFixed(0) + '%');
		} else if (metric === 'visit_duration') {
			yAxis.tickFormat((d) => {
				if (d < 60) return Math.floor(d) + 's';
				const minutes = Math.floor(d / 60);
				return minutes + 'm';
			});
		} else if (metric === 'views_per_visit') {
			yAxis.tickFormat((d) => d.toFixed(1));
		} else {
			yAxis.tickFormat((d) => formatCompactNumber(d));
		}

		const yAxisGroup = svg.selectAll('.y-axis').data([null]);
		const yAxisEnter = yAxisGroup.enter().append('g').attr('class', 'y-axis');

		yAxisGroup
			.merge(yAxisEnter)
			.transition()
			.duration(isFirstDraw ? 0 : 500)
			.call(yAxis)
			.selectAll('text')
			.style('font-size', '12px')
			.style('fill', '#64748b');

		// Update or create grid lines
		const gridGroup = svg.selectAll('.grid').data([null]);
		const gridEnter = gridGroup.enter().append('g').attr('class', 'grid').attr('opacity', 0.1);

		gridGroup
			.merge(gridEnter)
			.transition()
			.duration(isFirstDraw ? 0 : 500)
			.call(d3.axisLeft(y).ticks(5).tickSize(-width).tickFormat(''));

		// Define linear gradient for area fill (only once)
		let defs = svgSelection.select('defs');
		if (defs.empty()) {
			defs = svgSelection.append('defs');
		}

		let gradient = defs.select('#area-gradient');
		if (gradient.empty()) {
			gradient = defs
				.append('linearGradient')
				.attr('id', 'area-gradient')
				.attr('x1', '0%')
				.attr('x2', '0%')
				.attr('y1', '0%')
				.attr('y2', '100%');

			gradient
				.append('stop')
				.attr('offset', '0%')
				.attr('stop-color', 'rgb(51, 65, 85)')
				.attr('stop-opacity', 0.1);

			gradient
				.append('stop')
				.attr('offset', '100%')
				.attr('stop-color', 'rgb(51, 65, 85)')
				.attr('stop-opacity', 0);
		}

		// Update or create area with smooth transition
		const areaPath = svg.selectAll('.area-path').data([parsedData]);

		areaPath
			.enter()
			.append('path')
			.attr('class', 'area-path')
			.attr('fill', 'url(#area-gradient)')
			.attr('d', area)
			.style('opacity', 0)
			.transition()
			.duration(750)
			.ease(d3.easeQuadInOut)
			.style('opacity', 1);

		areaPath
			.transition()
			.duration(isFirstDraw ? 0 : 750)
			.ease(d3.easeQuadInOut)
			.attr('d', area);

		areaPath.exit().transition().duration(300).style('opacity', 0).remove();

		// Update or create main line with smooth transition
		const mainPath = svg.selectAll('.main-line').data([parsedData]);

		const mainPathEnter = mainPath
			.enter()
			.append('path')
			.attr('class', 'main-line')
			.attr('fill', 'none')
			.attr('stroke', 'rgb(51, 65, 85)')
			.attr('stroke-width', 2)
			.attr('d', line);

		if (isFirstDraw) {
			const totalLength = mainPathEnter.node()?.getTotalLength() || 0;
			mainPathEnter
				.attr('stroke-dasharray', totalLength + ' ' + totalLength)
				.attr('stroke-dashoffset', totalLength)
				.transition()
				.duration(750)
				.ease(d3.easeQuadInOut)
				.attr('stroke-dashoffset', 0)
				.on('end', function () {
					d3.select(this).attr('stroke-dasharray', null);
				});
		}

		mainPath
			.transition()
			.duration(isFirstDraw ? 0 : 750)
			.ease(d3.easeQuadInOut)
			.attr('d', line);

		mainPath.exit().transition().duration(300).style('opacity', 0).remove();

		// Add comparison line (dotted, previous period)
		if (showComparison && parsedComparisonData.length > 0) {
			// Shift comparison dates to align with current period
			const timeDiff = parsedData[0].date.getTime() - parsedComparisonData[0].date.getTime();
			const alignedComparisonData = parsedComparisonData.map((d) => ({
				date: new Date(d.date.getTime() + timeDiff),
				count: d.count,
				originalDate: d.date
			}));

			const comparisonLine = d3
				.line()
				.x((d) => x(d.date))
				.y((d) => y(d.count))
				.curve(d3.curveMonotoneX);

			// Update or create comparison line
			const compPath = svg.selectAll('.comparison-line').data([alignedComparisonData]);

			const compPathEnter = compPath
				.enter()
				.append('path')
				.attr('class', 'comparison-line')
				.attr('fill', 'none')
				.attr('stroke', 'rgb(148, 163, 184)')
				.attr('stroke-width', 2)
				.attr('stroke-dasharray', '5,5')
				.attr('opacity', 0.6)
				.attr('d', comparisonLine);

			if (isFirstDraw) {
				const comparisonLength = compPathEnter.node()?.getTotalLength() || 0;
				compPathEnter
					.attr('stroke-dasharray', comparisonLength + ' ' + comparisonLength)
					.attr('stroke-dashoffset', comparisonLength)
					.transition()
					.duration(750)
					.ease(d3.easeQuadInOut)
					.attr('stroke-dashoffset', 0)
					.on('end', function () {
						d3.select(this).attr('stroke-dasharray', '5,5');
					});
			}

			compPath
				.transition()
				.duration(isFirstDraw ? 0 : 750)
				.ease(d3.easeQuadInOut)
				.attr('d', comparisonLine);

			compPath.exit().transition().duration(300).style('opacity', 0).remove();

			// Update or create comparison dots
			const comparisonDots = svg.selectAll('.comparison-dot').data(alignedComparisonData);

			comparisonDots
				.enter()
				.append('circle')
				.attr('class', 'comparison-dot')
				.attr('cx', (d) => x(d.date))
				.attr('cy', (d) => y(d.count))
				.attr('r', 0)
				.attr('fill', 'rgb(148, 163, 184)')
				.attr('opacity', 0.6)
				.style('cursor', 'pointer')
				.on('mouseover', function (event, d) {
					d3.select(this).transition().duration(200).attr('r', 5);

					if (tooltipElement) {
						d3.select(tooltipElement).transition().duration(200).style('opacity', 1);

						d3.select(tooltipElement)
							.html(
								`<strong>Previous Period</strong><br/>${formatTooltipDate(d.originalDate.toISOString())}<br/>${metricLabels[metric] || 'Count'}: <strong>${formatCount(d.count, metric)}</strong>`
							)
							.style('left', event.pageX + 10 + 'px')
							.style('top', event.pageY - 10 + 'px');
					}
				})
				.on('mouseout', function () {
					d3.select(this).transition().duration(200).attr('r', 3);

					if (tooltipElement) {
						d3.select(tooltipElement).transition().duration(200).style('opacity', 0);
					}
				})
				.transition()
				.delay((d, i) => i * 50)
				.duration(300)
				.attr('r', 3);

			comparisonDots
				.transition()
				.duration(isFirstDraw ? 0 : 500)
				.attr('cx', (d) => x(d.date))
				.attr('cy', (d) => y(d.count));

			comparisonDots.exit().transition().duration(200).attr('r', 0).remove();
		} else {
			// Remove comparison elements if not showing
			svg.selectAll('.comparison-line').transition().duration(300).style('opacity', 0).remove();
			svg.selectAll('.comparison-dot').transition().duration(200).attr('r', 0).remove();
		}

		// Update or create main data dots
		const dots = svg.selectAll('.main-dot').data(parsedData);

		dots
			.enter()
			.append('circle')
			.attr('class', 'main-dot')
			.attr('cx', (d) => x(d.date))
			.attr('cy', (d) => y(d.count))
			.attr('r', 0)
			.attr('fill', 'rgb(51, 65, 85)')
			.style('cursor', 'pointer')
			.on('mouseover', function (event, d) {
				d3.select(this).transition().duration(200).attr('r', 6);

				if (tooltipElement) {
					d3.select(tooltipElement).transition().duration(200).style('opacity', 1);

					d3.select(tooltipElement)
						.html(
							`<strong>${formatTooltipDate(d.date.toISOString())}</strong><br/>${metricLabels[metric] || 'Count'}: <strong>${formatCount(d.count, metric)}</strong>`
						)
						.style('left', event.pageX + 10 + 'px')
						.style('top', event.pageY - 10 + 'px');
				}
			})
			.on('mouseout', function () {
				d3.select(this).transition().duration(200).attr('r', 4);

				if (tooltipElement) {
					d3.select(tooltipElement).transition().duration(200).style('opacity', 0);
				}
			})
			.transition()
			.delay((d, i) => i * 50)
			.duration(300)
			.attr('r', 4);

		dots
			.transition()
			.duration(isFirstDraw ? 0 : 500)
			.attr('cx', (d) => x(d.date))
			.attr('cy', (d) => y(d.count));

		dots.exit().transition().duration(200).attr('r', 0).remove();

		// Create or reuse tooltip (only once)
		if (!tooltipElement) {
			tooltipElement = d3
				.select('body')
				.append('div')
				.attr('class', 'chart-tooltip')
				.style('position', 'absolute')
				.style('background', 'transparent')
				.style('padding', '0')
				.style('border-radius', '0')
				.style('font-size', '14px')
				.style('pointer-events', 'none')
				.style('opacity', 0)
				.style('z-index', '1000')
				.style('transition', 'opacity 0.1s ease')
				.node();
		}

		// Add invisible overlay for hover-anywhere tooltip
		const bisect = d3.bisector((d) => d.date).left;

		// Create vertical line and circles for hover indicator (only once)
		let hoverLine = svg.select('.hover-line');
		if (hoverLine.empty()) {
			hoverLine = svg
				.append('line')
				.attr('class', 'hover-line')
				.attr('stroke', '#94a3b8')
				.attr('stroke-width', 1)
				.attr('stroke-dasharray', '3,3')
				.attr('opacity', 0);
		}

		let hoverCircleCurrent = svg.select('.hover-circle-current');
		if (hoverCircleCurrent.empty()) {
			hoverCircleCurrent = svg
				.append('circle')
				.attr('class', 'hover-circle-current')
				.attr('r', 5)
				.attr('fill', 'rgb(51, 65, 85)')
				.attr('stroke', 'white')
				.attr('stroke-width', 2)
				.attr('opacity', 0);
		}

		let hoverCirclePrev = svg.select('.hover-circle-prev');
		if (hoverCirclePrev.empty()) {
			hoverCirclePrev = svg
				.append('circle')
				.attr('class', 'hover-circle-prev')
				.attr('r', 5)
				.attr('fill', 'rgb(148, 163, 184)')
				.attr('stroke', 'white')
				.attr('stroke-width', 2)
				.attr('opacity', 0);
		}

		// Align comparison data for lookup
		let alignedComparisonLookup = new Map();
		if (showComparison && parsedComparisonData.length > 0) {
			const timeDiff = parsedData[0].date.getTime() - parsedComparisonData[0].date.getTime();
			parsedComparisonData.forEach((d) => {
				const alignedDate = new Date(d.date.getTime() + timeDiff);
				alignedComparisonLookup.set(alignedDate.getTime(), {
					date: d.date,
					count: d.count,
					alignedDate: alignedDate
				});
			});
		}

		// Update or create overlay for hover
		let overlay = svg.select('.hover-overlay');
		if (overlay.empty()) {
			overlay = svg
				.append('rect')
				.attr('class', 'hover-overlay')
				.attr('fill', 'none')
				.attr('pointer-events', 'all')
				.style('cursor', 'crosshair');
		}

		overlay
			.attr('width', width)
			.attr('height', height)
			.on('mousemove', function (event) {
				const [mouseX] = d3.pointer(event);
				const xDate = x.invert(mouseX);
				const index = bisect(parsedData, xDate);

				// Get closest data point
				let d0 = parsedData[index - 1];
				let d1 = parsedData[index];
				let d = d0;

				if (d1 && d0) {
					d = xDate - d0.date > d1.date - xDate ? d1 : d0;
				} else if (d1) {
					d = d1;
				}

				if (!d) return;

				// Find corresponding comparison data
				let compD = null;
				if (showComparison && alignedComparisonLookup.size > 0) {
					// Find closest comparison point
					let closestDiff = Infinity;
					alignedComparisonLookup.forEach((value) => {
						const diff = Math.abs(value.alignedDate.getTime() - d.date.getTime());
						if (diff < closestDiff) {
							closestDiff = diff;
							compD = value;
						}
					});
				}

				// Update hover indicators
				const xPos = x(d.date);
				const yPos = y(d.count);

				hoverLine
					.attr('x1', xPos)
					.attr('x2', xPos)
					.attr('y1', 0)
					.attr('y2', height)
					.attr('opacity', 0.5);

				hoverCircleCurrent.attr('cx', xPos).attr('cy', yPos).attr('opacity', 1);

				if (compD && showComparison) {
					const yPosComp = y(compD.count);
					hoverCirclePrev.attr('cx', xPos).attr('cy', yPosComp).attr('opacity', 1);
				} else {
					hoverCirclePrev.attr('opacity', 0);
				}

				// Calculate percentage change
				let changePercent = null;
				let changeIcon = '';
				if (compD) {
					changePercent = ((d.count - compD.count) / compD.count) * 100;
					changeIcon = changePercent >= 0 ? '↗' : '↘';
				}

				// Build tooltip HTML
				let tooltipHTML = `
					<div style="background: rgba(30, 41, 59, 0.95); backdrop-filter: blur(8px); border-radius: 8px; padding: 12px; min-width: 200px; box-shadow: 0 4px 6px rgba(0,0,0,0.3);">
						<div style="font-weight: 600; font-size: 13px; color: #e2e8f0; margin-bottom: 8px;">${metricLabels[metric] || 'Metric'}</div>
						${changePercent !== null ? `<div style="color: ${changePercent >= 0 ? '#10b981' : '#ef4444'}; font-size: 12px; margin-bottom: 8px; display: flex; align-items: center; gap: 4px;"><span>${changeIcon}</span><span>${Math.abs(changePercent).toFixed(1)}%</span></div>` : ''}
						<div style="display: flex; align-items: center; gap: 8px; margin-bottom: 6px;">
							<div style="width: 8px; height: 8px; border-radius: 50%; background: rgb(51, 65, 85);"></div>
							<div style="color: #cbd5e1; font-size: 12px;">${formatTooltipDate(d.date.toISOString())}</div>
							<div style="color: white; font-weight: 600; font-size: 14px; margin-left: auto;">${formatCount(d.count, metric)}</div>
						</div>
						${
							compD
								? `
							<div style="display: flex; align-items: center; gap: 8px; opacity: 0.7;">
								<div style="width: 8px; height: 8px; border-radius: 50%; background: rgb(148, 163, 184);"></div>
								<div style="color: #cbd5e1; font-size: 12px;">${formatTooltipDate(compD.date.toISOString())}</div>
								<div style="color: white; font-weight: 600; font-size: 14px; margin-left: auto;">${formatCount(compD.count, metric)}</div>
							</div>
						`
								: ''
						}
					</div>
				`;

				if (tooltipElement) {
					d3.select(tooltipElement).transition().duration(100).style('opacity', 1);
					d3.select(tooltipElement)
						.html(tooltipHTML)
						.style('left', event.pageX + 15 + 'px')
						.style('top', event.pageY - 15 + 'px');
				}
			})
			.on('mouseout', function () {
				hoverLine.attr('opacity', 0);
				hoverCircleCurrent.attr('opacity', 0);
				hoverCirclePrev.attr('opacity', 0);
				if (tooltipElement) {
					d3.select(tooltipElement).transition().duration(200).style('opacity', 0);
				}
			});

		// Cleanup function for tooltip
		cleanupTooltip = () => {
			if (tooltipElement) {
				d3.select(tooltipElement).remove();
				tooltipElement = null;
			}
		};
	}

	onMount(() => {
		// Handle window resize
		const handleResize = () => {
			drawChart();
		};

		window.addEventListener('resize', handleResize);

		return () => {
			window.removeEventListener('resize', handleResize);
			if (cleanupTooltip) {
				cleanupTooltip();
			}
		};
	});

	// Watch for changes and redraw when necessary
	$effect(() => {
		// Track the dependencies
		const currentData = data;
		const currentComparisonData = comparisonData;
		const currentMetric = metric;
		const currentShowComparison = showComparison;

		// Only redraw if we have a container and data
		// Use untrack to prevent the drawChart function from creating new dependencies
		if (chartContainer && currentData && currentData.length > 0) {
			untrack(() => {
				drawChart();
			});
		}
	});
</script>

<div class="relative">
	{#if data.length > 0 && comparisonData.length > 0}
		<div class="absolute -bottom-5 right-0 z-10 flex items-center gap-4">
			<div class="flex items-center gap-3 text-sm">
				<div class="flex items-center gap-2">
					<div class="h-0.5 w-6 bg-slate-700"></div>
					<span class="text-muted-foreground">Current Period</span>
				</div>
				<div class="flex items-center gap-2">
					<div class="h-0.5 w-6 border-t-2 border-dashed border-slate-400"></div>
					<span class="text-muted-foreground">Previous Period</span>
				</div>
			</div>
			<Button
				variant="ghost"
				size="sm"
				onclick={() => (showComparison = !showComparison)}
				class="gap-2"
			>
				{#if showComparison}
					<Eye class="h-4 w-4" />
					Hide Comparison
				{:else}
					<EyeOff class="h-4 w-4" />
					Show Comparison
				{/if}
			</Button>
		</div>
	{/if}

	<div bind:this={chartContainer} class="h-auto w-full">
		{#if data.length === 0}
			<div class="text-muted-foreground flex h-full items-center justify-center">
				<p>No data available for the selected period</p>
			</div>
		{:else}
			<svg bind:this={svgElement} class="w-full"></svg>
		{/if}
	</div>
</div>

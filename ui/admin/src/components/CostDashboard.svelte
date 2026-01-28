<script lang="ts">
	import { onMount } from 'svelte';
	import { Chart } from 'chart.js/auto';

	interface CostReport {
		totalCostUSD: number;
		breakdown: {
			modelProvider: string;
			modelName: string;
			totalTokens: number;
			costUSD: number;
			requestCount: number;
		}[];
		byUser: {
			userId: string;
			userName: string;
			costUSD: number;
		}[];
		byWorkspace: {
			workspaceId: string;
			workspaceName: string;
			costUSD: number;
		}[];
		timeline: {
			date: string;
			costUSD: number;
		}[];
	}

	let costReport: CostReport | null = null;
	let loading = true;
	let error = '';
	let chartCanvas: HTMLCanvasElement;
	let chart: Chart | null = null;
	let timeRange = '7d'; // 7d, 30d, 90d

	onMount(async () => {
		await loadCostReport();
	});

	async function loadCostReport() {
		try {
			loading = true;
			const response = await fetch(`/api/costs/report?range=${timeRange}`);
			if (!response.ok) {
				throw new Error('Failed to load cost report');
			}
			costReport = await response.json();
			
			// Update chart after data loads
			if (costReport) {
				updateChart();
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Unknown error';
		} finally {
			loading = false;
		}
	}

	function updateChart() {
		if (!costReport || !chartCanvas) return;

		// Destroy existing chart
		if (chart) {
			chart.destroy();
		}

		// Create new chart
		const ctx = chartCanvas.getContext('2d');
		if (!ctx) return;

		chart = new Chart(ctx, {
			type: 'line',
			data: {
				labels: costReport.timeline.map(t => t.date),
				datasets: [{
					label: 'Daily Cost (USD)',
					data: costReport.timeline.map(t => t.costUSD),
					borderColor: 'rgb(59, 130, 246)',
					backgroundColor: 'rgba(59, 130, 246, 0.1)',
					tension: 0.3,
					fill: true
				}]
			},
			options: {
				responsive: true,
				maintainAspectRatio: false,
				plugins: {
					legend: {
						display: true,
						position: 'top'
					},
					tooltip: {
						mode: 'index',
						intersect: false,
						callbacks: {
							label: function(context) {
								return `Cost: $${context.parsed.y.toFixed(2)}`;
							}
						}
					}
				},
				scales: {
					y: {
						beginAtZero: true,
						ticks: {
							callback: function(value) {
								return '$' + value;
							}
						}
					}
				}
			}
		});
	}

	async function handleTimeRangeChange(range: string) {
		timeRange = range;
		await loadCostReport();
	}

	function formatCurrency(amount: number): string {
		return new Intl.NumberFormat('en-US', {
			style: 'currency',
			currency: 'USD',
			minimumFractionDigits: 2,
			maximumFractionDigits: 2
		}).format(amount);
	}

	function formatNumber(num: number): string {
		return new Intl.NumberFormat().format(num);
	}
</script>

<div class="cost-dashboard">
	<div class="flex justify-between items-center mb-6">
		<h2 class="text-2xl font-bold">Cost Analytics</h2>
		<div class="flex gap-2">
			<button 
				class="px-3 py-1 rounded {timeRange === '7d' ? 'bg-blue-500 text-white' : 'bg-gray-200'}"
				on:click={() => handleTimeRangeChange('7d')}
			>
				7 Days
			</button>
			<button 
				class="px-3 py-1 rounded {timeRange === '30d' ? 'bg-blue-500 text-white' : 'bg-gray-200'}"
				on:click={() => handleTimeRangeChange('30d')}
			>
				30 Days
			</button>
			<button 
				class="px-3 py-1 rounded {timeRange === '90d' ? 'bg-blue-500 text-white' : 'bg-gray-200'}"
				on:click={() => handleTimeRangeChange('90d')}
			>
				90 Days
			</button>
		</div>
	</div>

	{#if loading}
		<div class="flex justify-center items-center h-64">
			<div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
		</div>
	{:else if error}
		<div class="bg-red-50 border border-red-200 rounded-lg p-4 text-red-800">
			<p>Error: {error}</p>
			<button on:click={loadCostReport} class="mt-2 text-sm underline">
				Try again
			</button>
		</div>
	{:else if costReport}
		<!-- Total Cost -->
		<div class="bg-white rounded-lg border border-gray-200 p-6 mb-6">
			<div class="text-center">
				<h3 class="text-lg font-semibold text-gray-600 mb-2">Total Cost</h3>
				<p class="text-4xl font-bold text-blue-600">{formatCurrency(costReport.totalCostUSD)}</p>
			</div>
		</div>

		<!-- Timeline Chart -->
		<div class="bg-white rounded-lg border border-gray-200 p-6 mb-6">
			<h3 class="text-lg font-semibold mb-4">Cost Trend</h3>
			<div style="height: 300px;">
				<canvas bind:this={chartCanvas}></canvas>
			</div>
		</div>

		<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
			<!-- Cost by Model -->
			<div class="bg-white rounded-lg border border-gray-200 p-6">
				<h3 class="text-lg font-semibold mb-4">Cost by Model</h3>
				<div class="space-y-3">
					{#each costReport.breakdown as item}
						<div class="flex justify-between items-center">
							<div>
								<p class="font-medium">{item.modelProvider} - {item.modelName}</p>
								<p class="text-sm text-gray-500">
									{formatNumber(item.totalTokens)} tokens • {formatNumber(item.requestCount)} requests
								</p>
							</div>
							<p class="font-semibold text-blue-600">{formatCurrency(item.costUSD)}</p>
						</div>
					{/each}
				</div>
			</div>

			<!-- Cost by User (Top 10) -->
			<div class="bg-white rounded-lg border border-gray-200 p-6">
				<h3 class="text-lg font-semibold mb-4">Top Users by Cost</h3>
				<div class="space-y-3">
					{#each costReport.byUser.slice(0, 10) as user}
						<div class="flex justify-between items-center">
							<p class="font-medium">{user.userName || user.userId}</p>
							<p class="font-semibold text-blue-600">{formatCurrency(user.costUSD)}</p>
						</div>
					{/each}
				</div>
			</div>

			<!-- Cost by Workspace (Top 10) -->
			<div class="bg-white rounded-lg border border-gray-200 p-6 lg:col-span-2">
				<h3 class="text-lg font-semibold mb-4">Top Workspaces by Cost</h3>
				<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
					{#each costReport.byWorkspace.slice(0, 9) as workspace}
						<div class="border border-gray-200 rounded p-4">
							<p class="font-medium mb-2">{workspace.workspaceName || workspace.workspaceId}</p>
							<p class="text-2xl font-bold text-blue-600">{formatCurrency(workspace.costUSD)}</p>
						</div>
					{/each}
				</div>
			</div>
		</div>

		<div class="mt-6 flex justify-end">
			<button 
				on:click={loadCostReport}
				class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 transition"
			>
				Refresh
			</button>
		</div>
	{/if}
</div>

<style>
	.cost-dashboard {
		padding: 1.5rem;
	}
</style>

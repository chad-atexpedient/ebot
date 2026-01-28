<script lang="ts">
	import { onMount } from 'svelte';
	
	interface QuotaUsage {
		userId: string;
		workspaceId: string;
		mcpServers: { used: number; limit: number };
		threads: { used: number; limit: number };
		knowledgeSets: { used: number; limit: number };
		knowledgeFiles: { used: number; limit: number };
		storageBytes: { used: number; limit: number };
		cpuCores: { used: number; limit: number };
		memoryBytes: { used: number; limit: number };
		llmTokensPerDay: { used: number; limit: number };
		llmCostPerMonth: { used: number; limit: number };
	}

	let quotaUsage: QuotaUsage | null = null;
	let loading = true;
	let error = '';

	onMount(async () => {
		await loadQuotaUsage();
	});

	async function loadQuotaUsage() {
		try {
			loading = true;
			const response = await fetch('/api/quota/usage');
			if (!response.ok) {
				throw new Error('Failed to load quota usage');
			}
			quotaUsage = await response.json();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Unknown error';
		} finally {
			loading = false;
		}
	}

	function getPercentage(used: number, limit: number): number {
		if (limit === 0) return 0;
		return Math.min((used / limit) * 100, 100);
	}

	function getColorClass(percentage: number): string {
		if (percentage >= 90) return 'text-red-600 bg-red-100';
		if (percentage >= 75) return 'text-yellow-600 bg-yellow-100';
		return 'text-green-600 bg-green-100';
	}

	function formatBytes(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
	}

	function formatNumber(num: number): string {
		return new Intl.NumberFormat().format(num);
	}
</script>

<div class="quota-dashboard">
	<h2 class="text-2xl font-bold mb-6">Resource Quota Usage</h2>

	{#if loading}
		<div class="flex justify-center items-center h-64">
			<div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
		</div>
	{:else if error}
		<div class="bg-red-50 border border-red-200 rounded-lg p-4 text-red-800">
			<p>Error: {error}</p>
			<button on:click={loadQuotaUsage} class="mt-2 text-sm underline">
				Try again
			</button>
		</div>
	{:else if quotaUsage}
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
			<!-- MCP Servers -->
			<div class="quota-card">
				<h3 class="text-lg font-semibold mb-2">MCP Servers</h3>
				<div class="mb-2">
					<div class="flex justify-between text-sm mb-1">
						<span>{quotaUsage.mcpServers.used} / {quotaUsage.mcpServers.limit}</span>
						<span>{getPercentage(quotaUsage.mcpServers.used, quotaUsage.mcpServers.limit).toFixed(1)}%</span>
					</div>
					<div class="w-full bg-gray-200 rounded-full h-2">
						<div 
							class="h-2 rounded-full transition-all {getColorClass(getPercentage(quotaUsage.mcpServers.used, quotaUsage.mcpServers.limit))}"
							style="width: {getPercentage(quotaUsage.mcpServers.used, quotaUsage.mcpServers.limit)}%"
						></div>
					</div>
				</div>
			</div>

			<!-- Threads -->
			<div class="quota-card">
				<h3 class="text-lg font-semibold mb-2">Threads</h3>
				<div class="mb-2">
					<div class="flex justify-between text-sm mb-1">
						<span>{quotaUsage.threads.used} / {quotaUsage.threads.limit}</span>
						<span>{getPercentage(quotaUsage.threads.used, quotaUsage.threads.limit).toFixed(1)}%</span>
					</div>
					<div class="w-full bg-gray-200 rounded-full h-2">
						<div 
							class="h-2 rounded-full transition-all {getColorClass(getPercentage(quotaUsage.threads.used, quotaUsage.threads.limit))}"
							style="width: {getPercentage(quotaUsage.threads.used, quotaUsage.threads.limit)}%"
						></div>
					</div>
				</div>
			</div>

			<!-- Knowledge Sets -->
			<div class="quota-card">
				<h3 class="text-lg font-semibold mb-2">Knowledge Sets</h3>
				<div class="mb-2">
					<div class="flex justify-between text-sm mb-1">
						<span>{quotaUsage.knowledgeSets.used} / {quotaUsage.knowledgeSets.limit}</span>
						<span>{getPercentage(quotaUsage.knowledgeSets.used, quotaUsage.knowledgeSets.limit).toFixed(1)}%</span>
					</div>
					<div class="w-full bg-gray-200 rounded-full h-2">
						<div 
							class="h-2 rounded-full transition-all {getColorClass(getPercentage(quotaUsage.knowledgeSets.used, quotaUsage.knowledgeSets.limit))}"
							style="width: {getPercentage(quotaUsage.knowledgeSets.used, quotaUsage.knowledgeSets.limit)}%"
						></div>
					</div>
				</div>
			</div>

			<!-- Storage -->
			<div class="quota-card">
				<h3 class="text-lg font-semibold mb-2">Storage</h3>
				<div class="mb-2">
					<div class="flex justify-between text-sm mb-1">
						<span>{formatBytes(quotaUsage.storageBytes.used)} / {formatBytes(quotaUsage.storageBytes.limit)}</span>
						<span>{getPercentage(quotaUsage.storageBytes.used, quotaUsage.storageBytes.limit).toFixed(1)}%</span>
					</div>
					<div class="w-full bg-gray-200 rounded-full h-2">
						<div 
							class="h-2 rounded-full transition-all {getColorClass(getPercentage(quotaUsage.storageBytes.used, quotaUsage.storageBytes.limit))}"
							style="width: {getPercentage(quotaUsage.storageBytes.used, quotaUsage.storageBytes.limit)}%"
						></div>
					</div>
				</div>
			</div>

			<!-- LLM Tokens (Daily) -->
			<div class="quota-card">
				<h3 class="text-lg font-semibold mb-2">LLM Tokens (Today)</h3>
				<div class="mb-2">
					<div class="flex justify-between text-sm mb-1">
						<span>{formatNumber(quotaUsage.llmTokensPerDay.used)} / {formatNumber(quotaUsage.llmTokensPerDay.limit)}</span>
						<span>{getPercentage(quotaUsage.llmTokensPerDay.used, quotaUsage.llmTokensPerDay.limit).toFixed(1)}%</span>
					</div>
					<div class="w-full bg-gray-200 rounded-full h-2">
						<div 
							class="h-2 rounded-full transition-all {getColorClass(getPercentage(quotaUsage.llmTokensPerDay.used, quotaUsage.llmTokensPerDay.limit))}"
							style="width: {getPercentage(quotaUsage.llmTokensPerDay.used, quotaUsage.llmTokensPerDay.limit)}%"
						></div>
					</div>
				</div>
			</div>

			<!-- LLM Cost (Monthly) -->
			<div class="quota-card">
				<h3 class="text-lg font-semibold mb-2">LLM Cost (This Month)</h3>
				<div class="mb-2">
					<div class="flex justify-between text-sm mb-1">
						<span>${quotaUsage.llmCostPerMonth.used.toFixed(2)} / ${quotaUsage.llmCostPerMonth.limit.toFixed(2)}</span>
						<span>{getPercentage(quotaUsage.llmCostPerMonth.used, quotaUsage.llmCostPerMonth.limit).toFixed(1)}%</span>
					</div>
					<div class="w-full bg-gray-200 rounded-full h-2">
						<div 
							class="h-2 rounded-full transition-all {getColorClass(getPercentage(quotaUsage.llmCostPerMonth.used, quotaUsage.llmCostPerMonth.limit))}"
							style="width: {getPercentage(quotaUsage.llmCostPerMonth.used, quotaUsage.llmCostPerMonth.limit)}%"
						></div>
					</div>
				</div>
			</div>
		</div>

		<div class="mt-8 flex justify-end">
			<button 
				on:click={loadQuotaUsage}
				class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 transition"
			>
				Refresh
			</button>
		</div>
	{/if}
</div>

<style>
	.quota-dashboard {
		padding: 1.5rem;
	}

	.quota-card {
		background: white;
		border: 1px solid #e5e7eb;
		border-radius: 0.5rem;
		padding: 1.5rem;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
	}
</style>

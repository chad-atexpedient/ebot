<script lang="ts">
	import { onMount } from 'svelte';

	interface Region {
		id: string;
		name: string;
		displayName: string;
		location: string;
		status: 'active' | 'degraded' | 'maintenance' | 'offline';
		capabilities: string[];
		metadata: Record<string, string>;
	}

	interface DataResidencyPolicy {
		id: string;
		name: string;
		workspaceId?: string;
		userId?: string;
		primaryRegion: string;
		allowedRegions: string[];
		crossRegionReplication: boolean;
		replicationRegions: string[];
		complianceFrameworks: string[];
	}

	let regions: Region[] = [];
	let policies: DataResidencyPolicy[] = [];
	let selectedRegion: Region | null = null;
	let loading = true;
	let error: string | null = null;

	// Fetch regions
	async function fetchRegions() {
		try {
			const response = await fetch('/api/regions');
			const data = await response.json();
			regions = data.regions || [];
		} catch (err) {
			error = 'Failed to load regions';
			console.error(err);
		}
	}

	// Fetch policies
	async function fetchPolicies() {
		try {
			const response = await fetch('/api/data-residency/policies');
			const data = await response.json();
			policies = data.policies || [];
		} catch (err) {
			console.error('Failed to load policies:', err);
		}
	}

	onMount(async () => {
		await Promise.all([fetchRegions(), fetchPolicies()]);
		loading = false;
	});

	function getStatusColor(status: string): string {
		switch (status) {
			case 'active':
				return 'bg-green-100 text-green-800';
			case 'degraded':
				return 'bg-yellow-100 text-yellow-800';
			case 'maintenance':
				return 'bg-blue-100 text-blue-800';
			case 'offline':
				return 'bg-red-100 text-red-800';
			default:
				return 'bg-gray-100 text-gray-800';
		}
	}

	function getStatusIcon(status: string): string {
		switch (status) {
			case 'active':
				return '✓';
			case 'degraded':
				return '⚠';
			case 'maintenance':
				return '🔧';
			case 'offline':
				return '✗';
			default:
				return '?';
		}
	}
</script>

<div class="region-dashboard p-6">
	<div class="header mb-6">
		<h1 class="text-3xl font-bold text-gray-900">Multi-Region Management</h1>
		<p class="text-gray-600 mt-2">Manage regions, data residency policies, and cross-region replication</p>
	</div>

	{#if loading}
		<div class="loading flex items-center justify-center p-12">
			<div class="spinner-border animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
			<span class="ml-3 text-gray-600">Loading regions...</span>
		</div>
	{:else if error}
		<div class="error bg-red-50 border border-red-200 rounded-lg p-4">
			<p class="text-red-800">{error}</p>
		</div>
	{:else}
		<!-- Region Overview -->
		<div class="regions-grid grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 mb-8">
			{#each regions as region}
				<div 
					class="region-card bg-white rounded-lg shadow-sm border border-gray-200 p-4 hover:shadow-md transition-shadow cursor-pointer"
					on:click={() => selectedRegion = region}
					on:keypress={() => selectedRegion = region}
					role="button"
					tabindex="0"
				>
					<div class="flex items-start justify-between mb-3">
						<div>
							<h3 class="font-semibold text-lg text-gray-900">{region.displayName}</h3>
							<p class="text-sm text-gray-600">{region.location}</p>
							<p class="text-xs text-gray-400 mt-1">ID: {region.id}</p>
						</div>
						<span class={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(region.status)}`}>
							{getStatusIcon(region.status)} {region.status}
						</span>
					</div>

					<div class="capabilities mt-3">
						<p class="text-xs font-medium text-gray-700 mb-1">Capabilities:</p>
						<div class="flex flex-wrap gap-1">
							{#each region.capabilities as capability}
								<span class="px-2 py-0.5 bg-blue-50 text-blue-700 text-xs rounded">
									{capability}
								</span>
							{/each}
						</div>
					</div>

					{#if region.metadata && Object.keys(region.metadata).length > 0}
						<div class="metadata mt-3 pt-3 border-t border-gray-100">
							<p class="text-xs font-medium text-gray-700 mb-1">Metadata:</p>
							{#each Object.entries(region.metadata) as [key, value]}
								<p class="text-xs text-gray-600">{key}: {value}</p>
							{/each}
						</div>
					{/if}
				</div>
			{/each}
		</div>

		<!-- Data Residency Policies -->
		<div class="policies-section mb-8">
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-2xl font-bold text-gray-900">Data Residency Policies</h2>
				<button class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors">
					+ Create Policy
				</button>
			</div>

			{#if policies.length === 0}
				<div class="empty-state bg-gray-50 border border-gray-200 rounded-lg p-8 text-center">
					<p class="text-gray-600">No data residency policies configured</p>
					<p class="text-sm text-gray-500 mt-2">Create a policy to enforce regional data storage requirements</p>
				</div>
			{:else}
				<div class="policies-table bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
					<table class="min-w-full divide-y divide-gray-200">
						<thead class="bg-gray-50">
							<tr>
								<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
								<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Scope</th>
								<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Primary Region</th>
								<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Allowed Regions</th>
								<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Replication</th>
								<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Compliance</th>
								<th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
							</tr>
						</thead>
						<tbody class="bg-white divide-y divide-gray-200">
							{#each policies as policy}
								<tr class="hover:bg-gray-50">
									<td class="px-6 py-4 whitespace-nowrap">
										<div class="text-sm font-medium text-gray-900">{policy.name}</div>
										<div class="text-xs text-gray-500">{policy.id}</div>
									</td>
									<td class="px-6 py-4 whitespace-nowrap">
										<div class="text-sm text-gray-900">
											{#if policy.workspaceId}
												<span class="text-blue-600">Workspace</span>
											{:else if policy.userId}
												<span class="text-green-600">User</span>
											{:else}
												<span class="text-gray-600">Global</span>
											{/if}
										</div>
									</td>
									<td class="px-6 py-4 whitespace-nowrap">
										<span class="px-2 py-1 bg-blue-100 text-blue-800 text-xs rounded">
											{policy.primaryRegion}
										</span>
									</td>
									<td class="px-6 py-4">
										<div class="flex flex-wrap gap-1">
											{#each policy.allowedRegions as region}
												<span class="px-2 py-0.5 bg-gray-100 text-gray-700 text-xs rounded">
													{region}
												</span>
											{/each}
										</div>
									</td>
									<td class="px-6 py-4 whitespace-nowrap">
										{#if policy.crossRegionReplication}
											<span class="text-green-600 text-sm">✓ Enabled</span>
											<div class="text-xs text-gray-500 mt-1">
												{policy.replicationRegions.length} replica(s)
											</div>
										{:else}
											<span class="text-gray-400 text-sm">Disabled</span>
										{/if}
									</td>
									<td class="px-6 py-4">
										<div class="flex flex-wrap gap-1">
											{#each policy.complianceFrameworks as framework}
												<span class="px-2 py-0.5 bg-purple-100 text-purple-700 text-xs rounded font-medium">
													{framework}
												</span>
											{/each}
										</div>
									</td>
									<td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
										<button class="text-blue-600 hover:text-blue-900 mr-3">Edit</button>
										<button class="text-red-600 hover:text-red-900">Delete</button>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</div>

		<!-- Region Details Modal -->
		{#if selectedRegion}
			<div 
				class="modal-overlay fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50"
				on:click={() => selectedRegion = null}
				on:keypress={() => selectedRegion = null}
				role="button"
				tabindex="0"
			>
				<div 
					class="modal-content bg-white rounded-lg shadow-xl max-w-2xl w-full max-h-[80vh] overflow-auto p-6"
					on:click|stopPropagation
					on:keypress|stopPropagation
					role="dialog"
					tabindex="0"
				>
					<div class="flex items-start justify-between mb-4">
						<div>
							<h2 class="text-2xl font-bold text-gray-900">{selectedRegion.displayName}</h2>
							<p class="text-gray-600">{selectedRegion.location}</p>
						</div>
						<button 
							class="text-gray-400 hover:text-gray-600"
							on:click={() => selectedRegion = null}
						>
							✕
						</button>
					</div>

					<div class="space-y-4">
						<div>
							<label class="block text-sm font-medium text-gray-700 mb-1">Region ID</label>
							<p class="text-gray-900">{selectedRegion.id}</p>
						</div>

						<div>
							<label class="block text-sm font-medium text-gray-700 mb-1">Status</label>
							<span class={`inline-block px-3 py-1 rounded-full text-sm font-medium ${getStatusColor(selectedRegion.status)}`}>
								{getStatusIcon(selectedRegion.status)} {selectedRegion.status}
							</span>
						</div>

						<div>
							<label class="block text-sm font-medium text-gray-700 mb-2">Capabilities</label>
							<div class="flex flex-wrap gap-2">
								{#each selectedRegion.capabilities as capability}
									<span class="px-3 py-1 bg-blue-50 text-blue-700 text-sm rounded">
										{capability}
									</span>
								{/each}
							</div>
						</div>

						{#if selectedRegion.metadata && Object.keys(selectedRegion.metadata).length > 0}
							<div>
								<label class="block text-sm font-medium text-gray-700 mb-2">Metadata</label>
								<div class="bg-gray-50 rounded p-3 space-y-1">
									{#each Object.entries(selectedRegion.metadata) as [key, value]}
										<div class="flex justify-between">
											<span class="text-sm text-gray-600">{key}:</span>
											<span class="text-sm text-gray-900 font-medium">{value}</span>
										</div>
									{/each}
								</div>
							</div>
						{/if}

						<div class="pt-4 border-t border-gray-200">
							<button class="w-full px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors">
								View Region Health
							</button>
						</div>
					</div>
				</div>
			</div>
		{/if}

		<!-- Statistics Summary -->
		<div class="stats-grid grid grid-cols-1 md:grid-cols-4 gap-4">
			<div class="stat-card bg-white rounded-lg shadow-sm border border-gray-200 p-4">
				<p class="text-sm text-gray-600 mb-1">Total Regions</p>
				<p class="text-3xl font-bold text-gray-900">{regions.length}</p>
			</div>
			<div class="stat-card bg-white rounded-lg shadow-sm border border-gray-200 p-4">
				<p class="text-sm text-gray-600 mb-1">Active Regions</p>
				<p class="text-3xl font-bold text-green-600">
					{regions.filter(r => r.status === 'active').length}
				</p>
			</div>
			<div class="stat-card bg-white rounded-lg shadow-sm border border-gray-200 p-4">
				<p class="text-sm text-gray-600 mb-1">Data Residency Policies</p>
				<p class="text-3xl font-bold text-blue-600">{policies.length}</p>
			</div>
			<div class="stat-card bg-white rounded-lg shadow-sm border border-gray-200 p-4">
				<p class="text-sm text-gray-600 mb-1">Replicated Regions</p>
				<p class="text-3xl font-bold text-purple-600">
					{policies.filter(p => p.crossRegionReplication).reduce((sum, p) => sum + p.replicationRegions.length, 0)}
				</p>
			</div>
		</div>
	{/if}
</div>

<style>
	.region-dashboard {
		font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
	}

	.spinner-border {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>

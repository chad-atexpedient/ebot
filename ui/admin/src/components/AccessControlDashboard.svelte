<script lang="ts">
	import { onMount } from 'svelte';

	interface Policy {
		id: string;
		name: string;
		description: string;
		effect: 'allow' | 'deny';
		enabled: boolean;
		priority: number;
		rules: Rule[];
		created_at: string;
		updated_at: string;
	}

	interface Rule {
		id: string;
		description: string;
		condition: string;
		effect: 'allow' | 'deny';
	}

	interface ServiceAccount {
		id: string;
		name: string;
		description: string;
		workspace_id: string;
		type: 'standard' | 'elevated' | 'read_only';
		status: 'active' | 'suspended' | 'revoked';
		created_at: string;
		last_used_at?: string;
	}

	interface APIKey {
		id: string;
		name: string;
		key_prefix: string;
		status: 'active' | 'deprecated' | 'revoked' | 'expired';
		created_at: string;
		expires_at?: string;
		last_used?: string;
		usage_count: number;
	}

	let activeTab: 'policies' | 'service-accounts' | 'saml' = 'policies';
	let policies: Policy[] = [];
	let serviceAccounts: ServiceAccount[] = [];
	let selectedAccount: ServiceAccount | null = null;
	let apiKeys: APIKey[] = [];
	let loading = false;
	let error = '';

	// ABAC Policies
	let newPolicy: Partial<Policy> = {
		name: '',
		description: '',
		effect: 'allow',
		enabled: true,
		priority: 100,
		rules: [],
	};

	let newRule: Partial<Rule> = {
		description: '',
		condition: '',
		effect: 'allow',
	};

	// Service Accounts
	let newServiceAccount: Partial<ServiceAccount> = {
		name: '',
		description: '',
		workspace_id: '',
		type: 'standard',
	};

	let newAPIKeyOptions = {
		name: '',
		scopes: ['read', 'write'],
		expiresInDays: 90,
		rateLimitRPM: 1000,
	};

	let generatedAPIKey = '';
	let showAPIKeyModal = false;

	// SAML Configuration
	let samlConfig = {
		entityID: '',
		acsUrl: '',
		idpSSOUrl: '',
		idpMetadataURL: '',
		signAuthnRequests: true,
		forceAuthn: false,
	};

	let samlMetadata = '';

	onMount(async () => {
		await loadPolicies();
		await loadServiceAccounts();
	});

	async function loadPolicies() {
		loading = true;
		try {
			const res = await fetch('/api/policies');
			const data = await res.json();
			policies = data.policies || [];
		} catch (e) {
			error = 'Failed to load policies';
		} finally {
			loading = false;
		}
	}

	async function createPolicy() {
		try {
			const res = await fetch('/api/policies', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(newPolicy),
			});

			if (res.ok) {
				await loadPolicies();
				newPolicy = {
					name: '',
					description: '',
					effect: 'allow',
					enabled: true,
					priority: 100,
					rules: [],
				};
			} else {
				error = 'Failed to create policy';
			}
		} catch (e) {
			error = 'Failed to create policy';
		}
	}

	async function deletePolicy(id: string) {
		if (!confirm('Delete this policy?')) return;

		try {
			const res = await fetch(`/api/policies/${id}`, { method: 'DELETE' });
			if (res.ok) {
				await loadPolicies();
			}
		} catch (e) {
			error = 'Failed to delete policy';
		}
	}

	async function togglePolicy(policy: Policy) {
		try {
			const res = await fetch(`/api/policies/${policy.id}`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ ...policy, enabled: !policy.enabled }),
			});

			if (res.ok) {
				await loadPolicies();
			}
		} catch (e) {
			error = 'Failed to update policy';
		}
	}

	function addRule() {
		if (!newRule.condition) return;

		newPolicy.rules = [
			...(newPolicy.rules || []),
			{
				id: `rule_${Date.now()}`,
				description: newRule.description || '',
				condition: newRule.condition || '',
				effect: newRule.effect || 'allow',
			},
		];

		newRule = {
			description: '',
			condition: '',
			effect: 'allow',
		};
	}

	async function loadServiceAccounts() {
		try {
			const res = await fetch('/api/service-accounts');
			const data = await res.json();
			serviceAccounts = data.accounts || [];
		} catch (e) {
			error = 'Failed to load service accounts';
		}
	}

	async function createServiceAccount() {
		try {
			const res = await fetch('/api/service-accounts', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(newServiceAccount),
			});

			if (res.ok) {
				await loadServiceAccounts();
				newServiceAccount = {
					name: '',
					description: '',
					workspace_id: '',
					type: 'standard',
				};
			} else {
				error = 'Failed to create service account';
			}
		} catch (e) {
			error = 'Failed to create service account';
		}
	}

	async function selectServiceAccount(account: ServiceAccount) {
		selectedAccount = account;
		await loadAPIKeys(account.id);
	}

	async function loadAPIKeys(accountId: string) {
		try {
			const res = await fetch(`/api/service-accounts/${accountId}/api-keys`);
			const data = await res.json();
			apiKeys = data.keys || [];
		} catch (e) {
			error = 'Failed to load API keys';
		}
	}

	async function generateAPIKey() {
		if (!selectedAccount) return;

		try {
			const expiresAt = new Date();
			expiresAt.setDate(expiresAt.getDate() + newAPIKeyOptions.expiresInDays);

			const res = await fetch(`/api/service-accounts/${selectedAccount.id}/api-keys`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					name: newAPIKeyOptions.name,
					scopes: newAPIKeyOptions.scopes,
					expiresAt: expiresAt.toISOString(),
					rateLimitRPM: newAPIKeyOptions.rateLimitRPM,
				}),
			});

			if (res.ok) {
				const data = await res.json();
				generatedAPIKey = data.key;
				showAPIKeyModal = true;
				await loadAPIKeys(selectedAccount.id);
			} else {
				error = 'Failed to generate API key';
			}
		} catch (e) {
			error = 'Failed to generate API key';
		}
	}

	async function revokeAPIKey(keyId: string) {
		if (!confirm('Revoke this API key?')) return;

		try {
			const res = await fetch(`/api/api-keys/${keyId}`, { method: 'DELETE' });
			if (res.ok && selectedAccount) {
				await loadAPIKeys(selectedAccount.id);
			}
		} catch (e) {
			error = 'Failed to revoke API key';
		}
	}

	async function loadSAMLMetadata() {
		try {
			const res = await fetch('/auth/saml/metadata');
			samlMetadata = await res.text();
		} catch (e) {
			error = 'Failed to load SAML metadata';
		}
	}

	function copyToClipboard(text: string) {
		navigator.clipboard.writeText(text);
	}

	function getStatusColor(status: string): string {
		switch (status) {
			case 'active':
				return 'green';
			case 'suspended':
			case 'deprecated':
				return 'yellow';
			case 'revoked':
			case 'expired':
				return 'red';
			default:
				return 'gray';
		}
	}
</script>

<div class="access-control-dashboard">
	<h1>Access Control Management</h1>

	{#if error}
		<div class="error-banner">
			{error}
			<button on:click={() => (error = '')}>×</button>
		</div>
	{/if}

	<div class="tabs">
		<button
			class:active={activeTab === 'policies'}
			on:click={() => (activeTab = 'policies')}
		>
			ABAC Policies
		</button>
		<button
			class:active={activeTab === 'service-accounts'}
			on:click={() => (activeTab = 'service-accounts')}
		>
			Service Accounts
		</button>
		<button
			class:active={activeTab === 'saml'}
			on:click={() => (activeTab = 'saml')}
		>
			SAML Configuration
		</button>
	</div>

	{#if activeTab === 'policies'}
		<div class="tab-content">
			<div class="section">
				<h2>Create ABAC Policy</h2>
				<div class="form">
					<input
						type="text"
						placeholder="Policy Name"
						bind:value={newPolicy.name}
					/>
					<textarea
						placeholder="Description"
						bind:value={newPolicy.description}
					></textarea>

					<div class="form-row">
						<label>
							Effect:
							<select bind:value={newPolicy.effect}>
								<option value="allow">Allow</option>
								<option value="deny">Deny</option>
							</select>
						</label>

						<label>
							Priority:
							<input
								type="number"
								bind:value={newPolicy.priority}
								min="1"
								max="1000"
							/>
						</label>

						<label>
							<input type="checkbox" bind:checked={newPolicy.enabled} />
							Enabled
						</label>
					</div>

					<div class="rules-section">
						<h3>Rules</h3>
						<div class="rule-form">
							<input
								type="text"
								placeholder="Rule Description"
								bind:value={newRule.description}
							/>
							<textarea
								placeholder="CEL Condition (e.g., subject.roles.contains('admin'))"
								bind:value={newRule.condition}
							></textarea>
							<button on:click={addRule} class="btn-secondary">Add Rule</button>
						</div>

						{#if newPolicy.rules && newPolicy.rules.length > 0}
							<ul class="rules-list">
								{#each newPolicy.rules as rule, i}
									<li>
										<strong>{rule.description}</strong>
										<code>{rule.condition}</code>
										<span class="badge {rule.effect}">{rule.effect}</span>
									</li>
								{/each}
							</ul>
						{/if}
					</div>

					<button on:click={createPolicy} class="btn-primary">Create Policy</button>
				</div>
			</div>

			<div class="section">
				<h2>Existing Policies ({policies.length})</h2>
				{#if loading}
					<p>Loading...</p>
				{:else if policies.length === 0}
					<p>No policies configured.</p>
				{:else}
					<table>
						<thead>
							<tr>
								<th>Name</th>
								<th>Effect</th>
								<th>Priority</th>
								<th>Rules</th>
								<th>Status</th>
								<th>Actions</th>
							</tr>
						</thead>
						<tbody>
							{#each policies as policy}
								<tr>
									<td>
										<strong>{policy.name}</strong>
										<br />
										<small>{policy.description}</small>
									</td>
									<td>
										<span class="badge {policy.effect}">{policy.effect}</span>
									</td>
									<td>{policy.priority}</td>
									<td>{policy.rules?.length || 0}</td>
									<td>
										<span class="status {policy.enabled ? 'active' : 'inactive'}">
											{policy.enabled ? 'Enabled' : 'Disabled'}
										</span>
									</td>
									<td>
										<button on:click={() => togglePolicy(policy)} class="btn-small">
											{policy.enabled ? 'Disable' : 'Enable'}
										</button>
										<button on:click={() => deletePolicy(policy.id)} class="btn-small btn-danger">
											Delete
										</button>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{/if}
			</div>
		</div>
	{/if}

	{#if activeTab === 'service-accounts'}
		<div class="tab-content">
			<div class="two-column">
				<div class="section">
					<h2>Service Accounts ({serviceAccounts.length})</h2>

					<div class="form">
						<input
							type="text"
							placeholder="Account Name"
							bind:value={newServiceAccount.name}
						/>
						<textarea
							placeholder="Description"
							bind:value={newServiceAccount.description}
						></textarea>
						<input
							type="text"
							placeholder="Workspace ID"
							bind:value={newServiceAccount.workspace_id}
						/>
						<select bind:value={newServiceAccount.type}>
							<option value="standard">Standard</option>
							<option value="elevated">Elevated</option>
							<option value="read_only">Read Only</option>
						</select>
						<button on:click={createServiceAccount} class="btn-primary">
							Create Service Account
						</button>
					</div>

					<ul class="account-list">
						{#each serviceAccounts as account}
							<li
								class:selected={selectedAccount?.id === account.id}
								on:click={() => selectServiceAccount(account)}
							>
								<div class="account-info">
									<strong>{account.name}</strong>
									<span class="badge {account.type}">{account.type}</span>
									<span class="status {getStatusColor(account.status)}">
										{account.status}
									</span>
								</div>
								<small>{account.description}</small>
								{#if account.last_used_at}
									<small>Last used: {new Date(account.last_used_at).toLocaleDateString()}</small>
								{/if}
							</li>
						{/each}
					</ul>
				</div>

				{#if selectedAccount}
					<div class="section">
						<h2>API Keys for {selectedAccount.name}</h2>

						<div class="form">
							<h3>Generate New API Key</h3>
							<input
								type="text"
								placeholder="Key Name"
								bind:value={newAPIKeyOptions.name}
							/>
							<div class="form-row">
								<label>
									Expires in (days):
									<input
										type="number"
										bind:value={newAPIKeyOptions.expiresInDays}
										min="1"
										max="365"
									/>
								</label>
								<label>
									Rate Limit (RPM):
									<input
										type="number"
										bind:value={newAPIKeyOptions.rateLimitRPM}
										min="1"
									/>
								</label>
							</div>
							<button on:click={generateAPIKey} class="btn-primary">
								Generate API Key
							</button>
						</div>

						<table>
							<thead>
								<tr>
									<th>Name</th>
									<th>Prefix</th>
									<th>Status</th>
									<th>Usage</th>
									<th>Expires</th>
									<th>Actions</th>
								</tr>
							</thead>
							<tbody>
								{#each apiKeys as key}
									<tr>
										<td>{key.name}</td>
										<td><code>{key.key_prefix}</code></td>
										<td>
											<span class="status {getStatusColor(key.status)}">
												{key.status}
											</span>
										</td>
										<td>{key.usage_count}</td>
										<td>
											{#if key.expires_at}
												{new Date(key.expires_at).toLocaleDateString()}
											{:else}
												Never
											{/if}
										</td>
										<td>
											<button on:click={() => revokeAPIKey(key.id)} class="btn-small btn-danger">
												Revoke
											</button>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</div>
		</div>
	{/if}

	{#if activeTab === 'saml'}
		<div class="tab-content">
			<div class="section">
				<h2>SAML 2.0 Configuration</h2>

				<div class="form">
					<label>
						Entity ID (SP):
						<input type="text" bind:value={samlConfig.entityID} />
					</label>

					<label>
						Assertion Consumer Service URL:
						<input type="text" bind:value={samlConfig.acsUrl} />
					</label>

					<label>
						IdP SSO URL:
						<input type="text" bind:value={samlConfig.idpSSOUrl} />
					</label>

					<label>
						IdP Metadata URL:
						<input type="text" bind:value={samlConfig.idpMetadataURL} />
					</label>

					<div class="form-row">
						<label>
							<input type="checkbox" bind:checked={samlConfig.signAuthnRequests} />
							Sign Authentication Requests
						</label>

						<label>
							<input type="checkbox" bind:checked={samlConfig.forceAuthn} />
							Force Re-authentication
						</label>
					</div>

					<button class="btn-primary">Save Configuration</button>
				</div>
			</div>

			<div class="section">
				<h2>SAML Metadata</h2>
				<p>Provide this metadata to your identity provider:</p>
				<button on:click={loadSAMLMetadata} class="btn-secondary">
					Load Metadata
				</button>

				{#if samlMetadata}
					<div class="metadata-box">
						<pre>{samlMetadata}</pre>
						<button on:click={() => copyToClipboard(samlMetadata)} class="btn-small">
							Copy to Clipboard
						</button>
					</div>
				{/if}
			</div>

			<div class="section">
				<h2>Test SAML Login</h2>
				<p>Test your SAML configuration:</p>
				<a href="/auth/saml/login" class="btn-primary" target="_blank">
					Initiate SAML Login
				</a>
			</div>
		</div>
	{/if}
</div>

{#if showAPIKeyModal}
	<div class="modal-overlay" on:click={() => (showAPIKeyModal = false)}>
		<div class="modal" on:click|stopPropagation>
			<h2>API Key Generated</h2>
			<p>Save this API key securely. It won't be shown again.</p>
			<div class="key-display">
				<code>{generatedAPIKey}</code>
				<button on:click={() => copyToClipboard(generatedAPIKey)} class="btn-small">
					Copy
				</button>
			</div>
			<button on:click={() => (showAPIKeyModal = false)} class="btn-primary">
				Close
			</button>
		</div>
	</div>
{/if}

<style>
	.access-control-dashboard {
		padding: 2rem;
		max-width: 1400px;
		margin: 0 auto;
	}

	h1 {
		font-size: 2rem;
		margin-bottom: 2rem;
		color: #1a1a1a;
	}

	h2 {
		font-size: 1.5rem;
		margin-bottom: 1rem;
		color: #333;
	}

	h3 {
		font-size: 1.2rem;
		margin-bottom: 0.5rem;
	}

	.tabs {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 2rem;
		border-bottom: 2px solid #e0e0e0;
	}

	.tabs button {
		padding: 0.75rem 1.5rem;
		background: none;
		border: none;
		border-bottom: 3px solid transparent;
		cursor: pointer;
		font-weight: 600;
		color: #666;
		transition: all 0.2s;
	}

	.tabs button.active {
		color: #2563eb;
		border-bottom-color: #2563eb;
	}

	.tabs button:hover {
		color: #1e40af;
	}

	.tab-content {
		animation: fadeIn 0.3s;
	}

	@keyframes fadeIn {
		from {
			opacity: 0;
		}
		to {
			opacity: 1;
		}
	}

	.section {
		background: white;
		border-radius: 8px;
		padding: 1.5rem;
		margin-bottom: 2rem;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
	}

	.two-column {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 2rem;
	}

	.form {
		display: flex;
		flex-direction: column;
		gap: 1rem;
		margin-bottom: 1.5rem;
	}

	.form-row {
		display: flex;
		gap: 1rem;
		align-items: center;
	}

	input[type='text'],
	input[type='number'],
	textarea,
	select {
		padding: 0.75rem;
		border: 1px solid #d1d5db;
		border-radius: 6px;
		font-size: 1rem;
	}

	textarea {
		min-height: 80px;
		resize: vertical;
	}

	.btn-primary,
	.btn-secondary,
	.btn-small,
	.btn-danger {
		padding: 0.75rem 1.5rem;
		border: none;
		border-radius: 6px;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.2s;
	}

	.btn-primary {
		background: #2563eb;
		color: white;
	}

	.btn-primary:hover {
		background: #1e40af;
	}

	.btn-secondary {
		background: #6b7280;
		color: white;
	}

	.btn-secondary:hover {
		background: #4b5563;
	}

	.btn-small {
		padding: 0.375rem 0.75rem;
		font-size: 0.875rem;
	}

	.btn-danger {
		background: #dc2626;
		color: white;
	}

	.btn-danger:hover {
		background: #b91c1c;
	}

	table {
		width: 100%;
		border-collapse: collapse;
		margin-top: 1rem;
	}

	th,
	td {
		padding: 0.75rem;
		text-align: left;
		border-bottom: 1px solid #e5e7eb;
	}

	th {
		background: #f9fafb;
		font-weight: 600;
		color: #374151;
	}

	.badge {
		display: inline-block;
		padding: 0.25rem 0.75rem;
		border-radius: 12px;
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
	}

	.badge.allow {
		background: #d1fae5;
		color: #065f46;
	}

	.badge.deny {
		background: #fee2e2;
		color: #991b1b;
	}

	.status {
		padding: 0.25rem 0.75rem;
		border-radius: 12px;
		font-size: 0.75rem;
		font-weight: 600;
	}

	.status.active {
		background: #d1fae5;
		color: #065f46;
	}

	.status.inactive,
	.status.yellow {
		background: #fef3c7;
		color: #92400e;
	}

	.status.red {
		background: #fee2e2;
		color: #991b1b;
	}

	.status.green {
		background: #d1fae5;
		color: #065f46;
	}

	.account-list {
		list-style: none;
		padding: 0;
		margin-top: 1rem;
	}

	.account-list li {
		padding: 1rem;
		border: 1px solid #e5e7eb;
		border-radius: 6px;
		margin-bottom: 0.5rem;
		cursor: pointer;
		transition: all 0.2s;
	}

	.account-list li:hover {
		background: #f9fafb;
		border-color: #2563eb;
	}

	.account-list li.selected {
		background: #eff6ff;
		border-color: #2563eb;
	}

	.account-info {
		display: flex;
		gap: 0.5rem;
		align-items: center;
		margin-bottom: 0.5rem;
	}

	.rules-list {
		list-style: none;
		padding: 0;
		margin-top: 1rem;
	}

	.rules-list li {
		padding: 0.75rem;
		background: #f9fafb;
		border-radius: 6px;
		margin-bottom: 0.5rem;
	}

	.rules-list code {
		display: block;
		margin: 0.5rem 0;
		padding: 0.5rem;
		background: white;
		border-radius: 4px;
		font-size: 0.875rem;
	}

	.metadata-box {
		position: relative;
		margin-top: 1rem;
	}

	.metadata-box pre {
		background: #f9fafb;
		padding: 1rem;
		border-radius: 6px;
		overflow-x: auto;
		font-size: 0.875rem;
	}

	.modal-overlay {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.5);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
	}

	.modal {
		background: white;
		padding: 2rem;
		border-radius: 12px;
		max-width: 600px;
		width: 90%;
		box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
	}

	.key-display {
		display: flex;
		gap: 1rem;
		align-items: center;
		padding: 1rem;
		background: #f9fafb;
		border-radius: 6px;
		margin: 1rem 0;
	}

	.key-display code {
		flex: 1;
		word-break: break-all;
		font-family: 'Monaco', 'Courier New', monospace;
		font-size: 0.875rem;
	}

	.error-banner {
		background: #fee2e2;
		color: #991b1b;
		padding: 1rem;
		border-radius: 6px;
		margin-bottom: 1rem;
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.error-banner button {
		background: none;
		border: none;
		font-size: 1.5rem;
		cursor: pointer;
		color: #991b1b;
	}
</style>

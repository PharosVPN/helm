<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->
<!-- Copyright (C) 2026 The PharosVPN Authors -->
<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { api, errorMessage, isApiError } from '$lib/api';
	import Modal from '$lib/components/Modal.svelte';
	import type { Node, LiveEvent } from '$lib/types';

	let nodes = $state<Node[]>([]);
	let loading = $state(true);
	let loadError = $state('');

	let events = $state<LiveEvent[]>([]);
	let wsConnected = $state(false);

	let renaming = $state<Node | null>(null);
	let renameValue = $state('');
	let renameError = $state('');
	let renameBusy = $state(false);

	let deleting = $state<Node | null>(null);
	let deleteError = $state('');
	let deleteBusy = $state(false);

	const total = $derived(nodes.length);
	const activeCount = $derived(nodes.filter((n) => n.status === 'active').length);
	const attentionCount = $derived(
		nodes.filter((n) => n.status === 'error' || n.status === 'unreachable').length
	);

	function statusBadge(s: string): string {
		if (s === 'active') return 'badge-success';
		if (s === 'error' || s === 'unreachable') return 'badge-danger';
		if (s === 'provisioning' || s === 'enrolling') return 'badge-warning';
		if (s === 'stopped') return 'badge-gray';
		return 'badge-info';
	}

	function clockTime(iso: string): string {
		const d = new Date(iso);
		return Number.isNaN(d.getTime()) ? '' : d.toLocaleTimeString();
	}

	async function loadNodes() {
		loading = true;
		loadError = '';
		try {
			nodes = await api.get<Node[]>('/api/nodes');
		} catch (e) {
			loadError = errorMessage(e);
		}
		loading = false;
	}

	// ───────── Live events WebSocket ─────────
	let ws: WebSocket | null = null;
	let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
	let destroyed = false;

	function connectEvents() {
		if (destroyed) return;
		const scheme = location.protocol === 'https:' ? 'wss' : 'ws';
		ws = new WebSocket(`${scheme}://${location.host}/ws/events`);
		ws.onopen = () => (wsConnected = true);
		ws.onmessage = (m) => {
			try {
				const ev = JSON.parse(m.data) as LiveEvent;
				events = [ev, ...events].slice(0, 50);
			} catch {
				/* ignore malformed frame */
			}
		};
		ws.onclose = () => {
			wsConnected = false;
			if (!destroyed) reconnectTimer = setTimeout(connectEvents, 3000);
		};
		ws.onerror = () => ws?.close();
	}

	onMount(() => {
		loadNodes();
		connectEvents();
	});
	onDestroy(() => {
		destroyed = true;
		if (reconnectTimer) clearTimeout(reconnectTimer);
		ws?.close();
	});

	// ───────── Rename (optimistic concurrency) ─────────
	function openRename(n: Node) {
		renaming = n;
		renameValue = n.name;
		renameError = '';
	}
	async function submitRename() {
		if (!renaming) return;
		renameBusy = true;
		renameError = '';
		try {
			const updated = await api.patch<Node>(`/api/nodes/${renaming.id}`, {
				version: renaming.version,
				name: renameValue
			});
			nodes = nodes.map((n) => (n.id === updated.id ? updated : n));
			renaming = null;
		} catch (e) {
			renameError = errorMessage(e);
			// On a 409 the row is stale — refresh so a retry uses live data.
			if (isApiError(e) && e.status === 409) loadNodes();
		}
		renameBusy = false;
	}

	// ───────── Delete ─────────
	async function confirmDelete() {
		if (!deleting) return;
		deleteBusy = true;
		deleteError = '';
		try {
			await api.del(`/api/nodes/${deleting.id}`);
			nodes = nodes.filter((n) => n.id !== deleting!.id);
			deleting = null;
		} catch (e) {
			deleteError = errorMessage(e);
		}
		deleteBusy = false;
	}
</script>

<svelte:head><title>Fleet — helm</title></svelte:head>

<h1 class="section-title">Fleet</h1>
<p class="section-subtitle">buoy nodes under this controller.</p>

<!-- Stat cards -->
<div class="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
	{#each [{ label: 'Total nodes', value: total }, { label: 'Active', value: activeCount }, { label: 'Needs attention', value: attentionCount }] as stat (stat.label)}
		<div class="card p-5">
			<div class="tnum text-[28px] font-bold text-ink">{stat.value}</div>
			<div class="mt-1 text-sm text-ink-2">{stat.label}</div>
		</div>
	{/each}
</div>

<!-- Node table -->
<div class="mt-8 card overflow-hidden">
	{#if loading}
		<div class="p-6 text-sm text-ink-3">Loading nodes…</div>
	{:else if loadError}
		<div class="p-6 text-sm" style="color: var(--c-danger)">{loadError}</div>
	{:else if nodes.length === 0}
		<div class="p-8 text-center">
			<div class="text-base font-semibold text-ink">No nodes yet</div>
			<p class="mt-1 text-sm text-ink-2">
				Onboard one with <code>helm nodes add &lt;ssh-host&gt; --region &lt;region&gt;</code>.
			</p>
		</div>
	{:else}
		<table class="dtable">
			<thead>
				<tr>
					<th>Name</th><th>Region</th><th>Status</th><th>SSH host</th>
					<th>Agent</th><th class="text-right">Actions</th>
				</tr>
			</thead>
			<tbody>
				{#each nodes as n (n.id)}
					<tr>
						<td class="font-medium">{n.name}</td>
						<td class="text-ink-2">{n.region}</td>
						<td>
							<span class="badge {statusBadge(n.status)}">
								<span class="dot"></span>{n.status}
							</span>
						</td>
						<td class="text-ink-2">{n.ssh_host || '—'}</td>
						<td class="text-ink-2">{n.agent_version || '—'}</td>
						<td class="text-right whitespace-nowrap">
							<button class="btn btn-text btn-sm" onclick={() => openRename(n)}>Rename</button>
							<button
								class="btn btn-text btn-sm"
								style="color: var(--c-danger)"
								onclick={() => {
									deleting = n;
									deleteError = '';
								}}>Remove</button
							>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</div>

<!-- Live events -->
<div class="mt-8">
	<div class="flex items-center gap-2">
		<h2 class="section-title">Live events</h2>
		<span class="badge {wsConnected ? 'badge-success' : 'badge-gray'}">
			<span class="dot" class:pulse={wsConnected}></span>
			{wsConnected ? 'connected' : 'offline'}
		</span>
	</div>
	<div class="mt-4 card p-4">
		{#if events.length === 0}
			<p class="text-sm text-ink-3">
				No events yet. Node handshakes and peer changes appear here in real time.
			</p>
		{:else}
			<ul class="flex flex-col gap-2">
				{#each events as ev, i (i)}
					<li class="flex items-center gap-3 text-sm">
						<span class="tnum text-xs text-ink-3">{clockTime(ev.at)}</span>
						<span class="badge badge-info">{ev.type}</span>
						<span class="text-ink-2">{ev.node_id}</span>
						{#if ev.message}<span class="text-ink-3">{ev.message}</span>{/if}
					</li>
				{/each}
			</ul>
		{/if}
	</div>
</div>

<!-- Rename dialog -->
{#if renaming}
	<Modal title="Rename node" onclose={() => (renaming = null)}>
		<label class="label" for="rename">Node name</label>
		<input id="rename" class="input" bind:value={renameValue} />
		{#if renameError}<p class="field-error" role="alert">{renameError}</p>{/if}
		<div class="mt-6 flex justify-end gap-3">
			<button class="btn btn-secondary" onclick={() => (renaming = null)}>Cancel</button>
			<button class="btn btn-primary" onclick={submitRename} disabled={renameBusy}>
				{renameBusy ? 'Saving…' : 'Save'}
			</button>
		</div>
	</Modal>
{/if}

<!-- Delete confirmation -->
{#if deleting}
	<Modal title="Remove node" onclose={() => (deleting = null)}>
		<p class="text-sm text-ink-2">
			Remove <span class="font-medium text-ink">{deleting.name}</span> from the fleet
			inventory? This does not touch the VM.
		</p>
		{#if deleteError}<p class="field-error" role="alert">{deleteError}</p>{/if}
		<div class="mt-6 flex justify-end gap-3">
			<button class="btn btn-secondary" onclick={() => (deleting = null)}>Cancel</button>
			<button class="btn btn-danger" onclick={confirmDelete} disabled={deleteBusy}>
				{deleteBusy ? 'Removing…' : 'Remove'}
			</button>
		</div>
	</Modal>
{/if}

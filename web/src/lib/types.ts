// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

export interface User {
	id: string;
	email: string;
	role: string;
	status: string;
	version: number;
}

export interface Node {
	id: string;
	name: string;
	region: string;
	status: string;
	public_ip: string;
	ssh_host: string;
	control_addr: string;
	agent_version: string;
	version: number;
	created_at: string;
	updated_at: string;
}

export interface LiveEvent {
	node_id: string;
	at: string;
	type: string;
	protocol?: string;
	peer_id?: string;
	message?: string;
}

export interface ApiError {
	status: number;
	message: string;
}

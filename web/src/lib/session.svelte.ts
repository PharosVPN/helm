// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 The PharosVPN Authors

import type { User } from './types';

// session holds the authenticated admin. null until login / a successful
// /api/auth/me check.
export const session = $state<{ user: User | null }>({ user: null });

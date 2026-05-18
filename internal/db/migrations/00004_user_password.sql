-- SPDX-License-Identifier: AGPL-3.0-or-later
-- Copyright (C) 2026 The PharosVPN Authors
--
-- Admin/user password authentication for the admin Web UI (DESIGN §8). The
-- hash is an Argon2id PHC string; an empty value means the account has no
-- password set yet.

-- +goose Up

ALTER TABLE users ADD COLUMN password_hash TEXT NOT NULL DEFAULT '';

-- +goose Down

ALTER TABLE users DROP COLUMN password_hash;

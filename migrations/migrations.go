package migrations

import _ "embed"

// SQL contains the idempotent initial schema migration.
//
//go:embed 001_init.sql
var SQL string

// FeedbackSQL contains the idempotent public-beta feedback migration.
//
//go:embed 002_feedback.sql
var FeedbackSQL string

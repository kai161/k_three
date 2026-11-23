CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,
    activated BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS activations (
    code_hash TEXT PRIMARY KEY,
    device_id TEXT REFERENCES devices(id),
    activated_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS entries (
    device_id TEXT NOT NULL REFERENCES devices(id),
    day INTEGER NOT NULL CHECK (day BETWEEN 1 AND 7),
    mood TEXT NOT NULL CHECK (mood IN ('轻松', '平静', '复杂', '疲惫')),
    ritual TEXT NOT NULL CHECK (ritual IN ('揉掉', '封存')),
    completed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (device_id, day)
);

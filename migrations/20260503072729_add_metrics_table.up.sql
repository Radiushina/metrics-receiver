CREATE TABLE IF NOT EXISTS gauges (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name    TEXT NOT NULL UNIQUE,
    value   DOUBLE PRECISION NOT NULL
);

CREATE TABLE IF NOT EXISTS counters (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name    TEXT NOT NULL UNIQUE,
    value   BIGINT  NOT NULL DEFAULT 0
);

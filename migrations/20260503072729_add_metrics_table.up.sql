CREATE TABLE IF NOT EXISTS gauges (
    id      uuid  PRIMARY KEY DEFAULT gen_random_uuid(),
    name    text NOT NULL,
    value   DOUBLE PRECISION,
);

CREATE TABLE IF NOT EXISTS counters (
    id      uuid  PRIMARY KEY DEFAULT gen_random_uuid(),
    name    text NOT NULL,
    value   BIGINT,
);
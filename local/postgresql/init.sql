CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id          UUID PRIMARY KEY,
    email       VARCHAR(255) NOT NULL UNIQUE,
    name        VARCHAR(255) NOT NULL,
    password    TEXT NOT NULL,
    roles       TEXT[] NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO users (id, email, name, password, roles, created_at)
VALUES (
    -- Fixed id: must match the graph user seeded in local/neo4j/init.cypher.
    '519fbd7e-4973-4d08-a605-3f1fd44e2f26',
    'william_cesar_santos@hotmail.com',
    'William',
    '$argon2id$v=19$m=65536,t=3,p=4$qNkRswLidbmSiP0zbdj81g$Y6hkfgo8OaAMoGT0hQLUlfFVWGjH2V2Tsv28qA2M0j4',
    ARRAY['*'],
    NOW()
) ON CONFLICT (email) DO NOTHING;

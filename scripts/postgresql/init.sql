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
    gen_random_uuid(),
    'william_cesar_santos@hotmail.com',
    'William',
    '$argon2id$v=19$m=65536,t=3,p=4$qNkRswLidbmSiP0zbdj81g$Y6hkfgo8OaAMoGT0hQLUlfFVWGjH2V2Tsv28qA2M0j4',
    ARRAY['*'],
    NOW()
) ON CONFLICT (email) DO NOTHING;

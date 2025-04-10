CREATE TABLE problems (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    sample_input TEXT NOT NULL,
    sample_output TEXT NOT NULL,
    time_limit_ms BIGINT NOT NULL CHECK (time_limit_ms > 0),
    memory_limit_kb BIGINT NOT NULL CHECK (memory_limit_kb > 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE test_cases (
    id SERIAL PRIMARY KEY,
    problem_id INTEGER NOT NULL REFERENCES problems (id) ON DELETE CASCADE,
    input TEXT NOT NULL,
    output TEXT NOT NULL
);

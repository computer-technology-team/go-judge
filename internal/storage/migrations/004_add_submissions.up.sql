CREATE TYPE SUBMISSION_STATUS AS ENUM (
    'IN_QUEUE', 'PENDING', 'RUNNING', 'ACCEPTED', 'WRONG_ANSWER',
    'TIME_LIMIT_EXCEEDED', 'MEMORY_LIMIT_EXCEEDED', 'RUNTIME_ERROR',
    'COMPILATION_ERROR', 'INTERNAL_ERROR'
);

CREATE TABLE submissions (
    id UUID DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    problem_id INT REFERENCES problems (id) NOT NULL,
    user_id UUID REFERENCES users (id) NOT NULL,
    solution_code TEXT NOT NULL,
    status SUBMISSION_STATUS NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now() NOT NULL,
    last_modified TIMESTAMP WITH TIME ZONE DEFAULT now() NOT NULL,
    message varchar(256),
    retries INT NOT NULL DEFAULT 0
);

-- Create a function to update the updated_at timestamp
CREATE FUNCTION sync_last_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_modified = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create a trigger to automatically update the updated_at column
CREATE TRIGGER sync_submissions_last_modified
BEFORE UPDATE ON submissions
FOR EACH ROW
EXECUTE FUNCTION sync_last_modified_column();

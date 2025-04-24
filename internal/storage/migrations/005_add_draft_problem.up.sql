ALTER TABLE problems
	ADD COLUMN draft BOOLEAN NOT NULL DEFAULT TRUE, ADD COLUMN published_at TIMESTAMPTZ;

CREATE INDEX problems_published_at_idx on problems (published_at);
CREATE INDEX problems_draft_idx on problems (draft);

ALTER TABLE problems
	DROP COLUMN draft, DROP COLUMN published_at;

DROP INDEX problems_published_at_idx;
DROP INDEX problems_draft_idx;

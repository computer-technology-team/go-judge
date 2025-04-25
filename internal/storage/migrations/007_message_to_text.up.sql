-- Change the message column from varchar(256) to text
ALTER TABLE submissions
ALTER COLUMN message TYPE text;

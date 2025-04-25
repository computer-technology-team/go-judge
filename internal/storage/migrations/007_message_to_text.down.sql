-- Change the message column back from text to varchar(256)
ALTER TABLE submissions
ALTER COLUMN message TYPE varchar(256);

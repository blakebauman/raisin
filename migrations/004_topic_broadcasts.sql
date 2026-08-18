-- Topic-targeted broadcasts
ALTER TABLE broadcasts
    ADD COLUMN IF NOT EXISTS topic_id UUID REFERENCES topics(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS broadcasts_topic_idx ON broadcasts(topic_id);

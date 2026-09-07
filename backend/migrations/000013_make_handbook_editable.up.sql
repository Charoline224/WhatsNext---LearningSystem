ALTER TABLE knowledge_articles
  ADD COLUMN user_edited BOOLEAN NOT NULL DEFAULT FALSE AFTER sort_order;

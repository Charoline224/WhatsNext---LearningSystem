ALTER TABLE exam_question_patterns
  ADD COLUMN tested_knowledge TEXT NOT NULL AFTER description,
  ADD COLUMN common_mistakes TEXT NOT NULL AFTER tested_knowledge,
  ADD COLUMN solving_strategy TEXT NOT NULL AFTER common_mistakes;

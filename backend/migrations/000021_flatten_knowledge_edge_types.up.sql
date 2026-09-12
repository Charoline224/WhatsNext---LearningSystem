UPDATE learning_edges SET relation_type='related' WHERE relation_type='contains';

ALTER TABLE learning_edges
  DROP CHECK chk_edges_type,
  ADD CONSTRAINT chk_edges_type CHECK (relation_type IN ('prerequisite', 'related'));

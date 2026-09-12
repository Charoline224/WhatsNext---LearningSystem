ALTER TABLE learning_edges
  DROP CHECK chk_edges_type,
  ADD CONSTRAINT chk_edges_type CHECK (relation_type IN ('prerequisite', 'related', 'contains'));

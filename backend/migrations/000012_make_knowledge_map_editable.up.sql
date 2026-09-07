ALTER TABLE learning_nodes
  ADD COLUMN position_x DECIMAL(10,2) NULL AFTER sort_order,
  ADD COLUMN position_y DECIMAL(10,2) NULL AFTER position_x,
  ADD COLUMN user_edited BOOLEAN NOT NULL DEFAULT FALSE AFTER position_y;

ALTER TABLE learning_edges
  ADD COLUMN user_edited BOOLEAN NOT NULL DEFAULT FALSE AFTER relation_type;

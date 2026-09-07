package model

import "time"

type ChatLearningSignal struct {
	ID               string    `db:"id" json:"id"`
	Question         string    `db:"question" json:"question"`
	Answer           string    `db:"answer" json:"answer"`
	RelatedNodeID    *string   `db:"related_node_id" json:"related_node_id"`
	IncorporatedPlan *string   `db:"incorporated_plan_id" json:"incorporated_plan_id"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

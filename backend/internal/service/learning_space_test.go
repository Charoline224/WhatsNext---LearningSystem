package service

import (
	"errors"
	"testing"
	"time"

	"whatsnext/backend/internal/dto"
)

func TestValidateSpace(t *testing.T) {
	future := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	valid := dto.CreateSpaceInput{Name: "计算机网络", Mode: "exam", Goal: "达到 80 分", ExamDate: future, DailyMinutes: 120}
	if err := validateSpace(valid); err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}
	cases := []struct {
		name   string
		mutate func(*dto.CreateSpaceInput)
		field  string
	}{{"mode", func(v *dto.CreateSpaceInput) { v.Mode = "growth" }, "mode"}, {"minutes", func(v *dto.CreateSpaceInput) { v.DailyMinutes = 5 }, "daily_minutes"}, {"date", func(v *dto.CreateSpaceInput) { v.ExamDate = "not-a-date" }, "exam_date"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := valid
			tc.mutate(&input)
			err := validateSpace(input)
			var validation ValidationError
			if !errors.As(err, &validation) || validation.Field != tc.field {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

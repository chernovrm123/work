package models

import "time"

// Expense — одна трата.
// приложениях деньги хранят в целых числах.
type Expense struct {
	ID          int
	Amount      float64
	Description string
	Date        time.Time
}

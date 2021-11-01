package bot

import (
	"fmt"
	"time"
)

func getDate() string {
	now := time.Now().In(time.Local)
	return fmt.Sprintf("%d-%d-%d", now.Year(), now.Month(), now.Day())
}

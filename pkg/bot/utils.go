package bot

import (
	"fmt"
	"time"
)

func getDate() string {
	now := time.Now()
	return fmt.Sprint("%d-%d-%d", now.Year(), now.Month(), now.Day)
}

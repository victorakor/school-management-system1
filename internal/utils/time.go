package utils

import "time"

// NowPtr returns a pointer to the current UTC time.
func NowPtr() *time.Time {
	t := time.Now().UTC()
	return &t
}

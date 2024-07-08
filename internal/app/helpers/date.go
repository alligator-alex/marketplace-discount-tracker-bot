package helpers

import "time"

const (
	DateDatabase = "2006-01-02 15:04:05"
	DateHuman    = "02.01.2006 15:04"
)

// Format time instance to human format.
func TimeToHuman(time time.Time) string {
	return time.Format(DateHuman)
}

// Format time instance to database format.
func TimeToDatabase(time time.Time) string {
	return time.Format(DateDatabase)
}

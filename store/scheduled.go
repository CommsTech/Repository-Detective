package store

// ScheduledRepository is a connected repo with an active scan schedule.
type ScheduledRepository struct {
	Repository
	ScheduleCron string
}

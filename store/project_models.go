package store

import "time"

// ProjectGroup groups related repositories for shared reporting context.
type ProjectGroup struct {
	ID                  int64     `json:"id"`
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	PrimaryRepositoryID int64     `json:"primary_repository_id"`
	RepositoryIDs       []int64   `json:"repository_ids"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

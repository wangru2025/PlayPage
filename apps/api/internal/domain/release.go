package domain

import "time"

type Release struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Status      string    `json:"status"`
	ArchivePath string    `json:"archivePath"`
	PublicPath  string    `json:"publicPath"`
	EntryFile   string    `json:"entryFile"`
	ChangeNote  string    `json:"changeNote"`
	Warnings    []string  `json:"warnings,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

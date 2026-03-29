package model

import (
	"errors"
	"strings"
	"time"
)

const SchemaVersion = 1
const NoFileRef = "N/A"

type Record struct {
	Title         string
	Type          string
	Snippet       string
	SchemaVersion int
	FileRef       string
	Fields        map[string]string
	FieldOrder    []string
	CreatedAt     time.Time
}

func NewRecordWithOptions(title string, recordType string) (Record, error) {
	if strings.TrimSpace(title) == "" {
		return Record{}, errors.New("title is required")
	}
	if recordType != "full" && recordType != "snippet" && recordType != "adr" {
		return Record{}, errors.New("record type must be full, snippet or adr")
	}
	return Record{Title: title,
			Type:          recordType,
			CreatedAt:     time.Now(),
			SchemaVersion: SchemaVersion,
			FileRef:       NoFileRef,
			Fields:        map[string]string{}},
		nil
}

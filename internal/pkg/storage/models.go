package storage

import "time"

// Image - is model of how image stored in DB
type Image struct {
	ID                   int64
	Key                  string
	UserID               int32
	URL                  string
	Approved             bool
	TransformsUploaded   bool
	Deleted              bool
	CreateDate           time.Time
	ApproveDate          time.Time
	TransformsUploadDate time.Time
	DeletionDate         time.Time
}

// Transformation - is model of how transforamiotn stored in DB
// json mappings needed to read initial config from file
type Transformation struct {
	ID      int32  `json:"-"`
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Type    string `json:"type"`
	Quality int    `json:"quality"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

type TransformList struct {
	Transformations []Transformation `json:"transformations"`
}

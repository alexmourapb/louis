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
	CreateDate           time.Time
	ApproveDate          time.Time
	TransformsUploadDate time.Time
}

// Transformation - is model of how transforamiotn stored in DB
// json mappings needed to read initial config from file
type Transformation struct {
	ID      int32  `json:"-"`
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Type    string `json:"type"`
	Quality int32  `json:"quality"`
	Width   int32  `json:"width"`
	Height  int32  `json:"height"`
}

type TransformList struct {
	Transformations []Transformation `json:"transformations"`
}

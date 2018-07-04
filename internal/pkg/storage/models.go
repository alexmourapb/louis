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
type Transformation struct {
	ID      int32
	Name    string
	Tag     string
	Type    string
	Quality int32
	Width   int32
	Height  int32
}

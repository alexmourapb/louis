package storage

import "time"

// Image - is model of how image stored in DB
type Image struct {
	ID                   int64
	Key                  string
	UserID               int32
	User                 *User
	URL                  string    `sql:"default:''"`
	Approved             bool      `sql:"default:false"`
	TransformsUploaded   bool      `sql:"default:false"`
	Deleted              bool      `sql:"default:false"`
	CreateDate           time.Time `sql:"default:now()"`
	ApproveDate          time.Time `sql:"default:now()"`
	TransformsUploadDate time.Time `sql:"default:now()"`
	DeletionDate         time.Time `sql:"default:now()"`
	Tags                 []string  `pg:",array"`
}

// Transformation - is model of how transforamiotn stored in DB
// json mappings needed to read initial config from file
type Transformation struct {
	ID      int32  `json:"-"`
	Name    string `json:"name" sql:",unique"`
	Tag     string `json:"tag"`
	Type    string `json:"type"`
	Quality int    `json:"quality"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

type TransformList struct {
	Transformations []Transformation `json:"transformations"`
}

type User struct {
	ID        int32
	PublicKey string
	SecretKey string
}

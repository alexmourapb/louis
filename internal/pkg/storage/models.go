package storage

import (
	"time"

	"github.com/lib/pq"
)

// Image - is model of how image stored in DB
type Image struct {
	ID                   int64
	Key                  string `gorm:"unique"`
	UserID               int32
	User                 *User
	URL                  string         `gorm:"default:''"`
	Approved             bool           `gorm:"default:false"`
	TransformsUploaded   bool           `gorm:"default:false"`
	Deleted              bool           `gorm:"default:false"`
	CreateDate           time.Time      `gorm:"default:now()"`
	ApproveDate          time.Time      `gorm:"default:now()"`
	TransformsUploadDate time.Time      `gorm:"default:now()"`
	DeletionDate         time.Time      `gorm:"default:now()"`
	Tags                 pq.StringArray `gorm:"type:varchar(256)[]"`
	AppliedTags          pq.StringArray `gorm:"type:varchar(256)[]"`
	Progressive          bool           `gorm:"default:false"`
	WithRealCopy         bool           // if "real" transform is applied
}

// Transformation - is model of how transforamiotn stored in DB
// json mappings needed to read initial config from file
type Transformation struct {
	ID      int32  `json:"-"`
	Name    string `json:"name" gorm:"unique"`
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

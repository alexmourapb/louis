package storage

import (
	"fmt"
	"github.com/KazanExpress/louis/internal/pkg/config"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"log"
	"sync"
	"time"
)

const (
	TagLength = 20
)

type DB struct {
	*pg.DB
	driver string
}
type Tx struct {
	*pg.Tx
}

// Open returns a DB reference for a data source.
func Open(cfg *config.Config) (*DB, error) {

	db := pg.Connect(&pg.Options{
		User:     cfg.PostgresUser,
		Password: cfg.PostgresPassword,
		Addr:     cfg.PostgresAddress,
		Database: cfg.PostgresDatabase,
	})

	db.OnQueryProcessed(func(event *pg.QueryProcessedEvent) {
		query, err := event.FormattedQuery()
		if err != nil {
			panic(err)
		}

		log.Printf("POSTGRES: %s %s", time.Since(event.StartTime), query)
	})

	return &DB{db, "pg"}, nil
}

// Begin starts an returns a new transaction.
func (db *DB) Begin() (*Tx, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{tx}, nil
}

var ifNotExist = &orm.CreateTableOptions{
	IfNotExists: true,
}

var created bool
var lock = &sync.Mutex{}

func (db *DB) InitDB() error {

	log.Printf("INITING")
	lock.Lock()
	defer lock.Unlock()
	log.Printf("INITING ->")
	defer log.Printf("INITED ->")
	// if created {
	// 	log.Printf("HEY, DB is already created!")
	// }
	created = true
	err := db.CreateTable(&User{}, &orm.CreateTableOptions{IfNotExists: true})

	if err != nil {
		return err
	}
	err = db.CreateTable(&Image{}, &orm.CreateTableOptions{IfNotExists: true})

	if err != nil {
		return err
	}

	err = db.CreateTable(&Transformation{}, &orm.CreateTableOptions{IfNotExists: true})

	if err != nil {
		return err
	}
	<-time.After(time.Second)
	return err
}

func (db *DB) EnsureTransformations(trans []Transformation) error {
	_, err := db.Model(&trans).
		OnConflict("(name) DO NOTHING").
		// TODO: add update
		Insert()
	return err
}

func (db *DB) DropDB() error {

	log.Printf("DROPINNG->")
	lock.Lock()
	defer lock.Unlock()
	log.Printf("DROPINNG-->>>>")
	defer log.Printf("DROPED-->>>>")
	if db.driver == "pg" {
		opt := &orm.DropTableOptions{IfExists: true, Cascade: true}
		err := db.DropTable(&Image{}, opt)
		if err != nil {
			log.Printf("ERROR: on droping db - %v", err)
			err = nil
		}
		err = db.DropTable(&Transformation{}, opt)
		if err != nil {
			log.Printf("ERROR: on droping db - %v", err)
			err = nil
		}
		err = db.DropTable(&User{}, opt)
		if err != nil {
			log.Printf("ERROR: on droping db - %v", err)
			err = nil
		}
		<-time.After(time.Second * 1)
		// db.Close()
		return err
	}

	return fmt.Errorf("'%s' driver not supported", db.driver)
}

func (db *DB) QueryImageByKey(key string) (*Image, error) {

	img := new(Image)
	return img, db.Model(img).Where("Key = ?", key).Select(img)
}

func (db *DB) AddImage(imageKey string, userID int32, tags ...string) (ImageID int64, err error) {
	var img = &Image{
		UserID: userID,
		Key:    imageKey,
		Tags:   tags,
	}
	err = db.Insert(img)
	return img.ID, err
}

func (db *DB) GetTransformations(imageID int64) ([]Transformation, error) {
	var trans []Transformation
	img := &Image{ID: imageID}
	err := db.Select(img)
	if err != nil {
		return nil, err
	}
	if len(img.Tags) == 0 {
		return nil, nil
	}
	var interfaceSlice []interface{} = make([]interface{}, len(img.Tags))
	for i, d := range img.Tags {
		interfaceSlice[i] = d
	}

	return trans, db.Model((*Transformation)(nil)).WhereIn("Tag IN (?)", interfaceSlice...).Select(&trans)

}

func (db *DB) SetTransformsUploaded(imgID int64) error {

	img := &Image{ID: imgID}
	_, err := db.Model(img).
		Set("Transforms_Uploaded=true, Transforms_Upload_Date=now()").
		WherePK().
		Update(img)

	return err
}

func (db *DB) SetClaimImage(imageKey string, userID int32) error {
	img := &Image{}
	_, err := db.Model(img).
		Set("Approved=true, Approve_Date=now()").
		Where("Key = ? AND User_ID = ?", imageKey, userID).
		Update(img)
	return err
}

func (db *DB) DeleteImage(imageKey string) error {
	img := &Image{}
	_, err := db.Model(img).
		Set("Deleted=true, Deletion_Date=now()").
		Where("Key = ?", imageKey).
		Update(img)
	return err

}

func (db *DB) SetImageURL(key string, userID int32, URL string) error {
	img := &Image{}
	_, err := db.Model(img).
		Set("Url=?", URL).
		Where("Key = ? AND User_ID = ?", key, userID).
		Update(img)
	return err
}

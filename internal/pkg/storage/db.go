package storage

import (
	"fmt"
	"github.com/KazanExpress/louis/internal/pkg/config"
	"strings"
	// "github.com/go-pg/pg"
	// "github.com/go-pg/pg/orm"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"log"
	"sync"
)

const (
	TagLength = 20
)

type DB struct {
	*gorm.DB
	driver string
}

// Open returns a DB reference for a data source.
func Open(cfg *config.Config) (*DB, error) {

	tmp := strings.Split(cfg.PostgresAddress, ":")
	host := tmp[0]
	port := tmp[1]

	db, err := gorm.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable", host, port, cfg.PostgresUser, cfg.PostgresDatabase, cfg.PostgresPassword))

	if err != nil {
		return nil, err
	}

	// db.LogMode(true)

	return &DB{db, "pg"}, nil
}

// Begin starts an returns a new transaction.

var lock = &sync.Mutex{}

func (db *DB) InitDB() error {

	lock.Lock()
	defer lock.Unlock()
	d := db.CreateTable(&User{})
	if d.Error != nil {
		return d.Error
	}
	d = db.CreateTable(&Image{})
	if d.Error != nil {
		return d.Error
	}
	return db.CreateTable(&Transformation{}).Error

}

func (db *DB) EnsureTransformations(trans []Transformation) error {
	for _, tr := range trans {
		err := db.Set("gorm:insert_option", "ON CONFLICT (name) DO NOTHING").Create(&tr).Error
		if err != nil {
			return err
		}
	}
	return nil
	// TODO: add update
	// https://github.com/jinzhu/gorm/issues/721
}

func (db *DB) DropDB() error {

	log.Printf("DROPINNG->")
	lock.Lock()
	defer lock.Unlock()
	log.Printf("DROPINNG-->>>>")
	defer log.Printf("DROPED-->>>>")
	if db.driver == "pg" {
		err := db.DropTableIfExists(&Image{}).Error
		if err != nil {
			log.Printf("ERROR: on droping db - %v", err)
			err = nil
		}
		err = db.DropTableIfExists(&Transformation{}).Error
		if err != nil {
			log.Printf("ERROR: on droping db - %v", err)
			err = nil
		}
		err = db.DropTableIfExists(&User{}).Error
		if err != nil {
			log.Printf("ERROR: on droping db - %v", err)
			err = nil
		}
		// db.Close()
		return err
	}

	return fmt.Errorf("'%s' driver not supported", db.driver)
}

func (db *DB) QueryImageByKey(key string) (*Image, error) {

	img := new(Image)
	return img, db.Where("Key = ?", key).First(img).Error
}

func (db *DB) AddImage(imageKey string, userID int32, tags ...string) (ImageID int64, err error) {
	var img = &Image{
		UserID: userID,
		Key:    imageKey,
		Tags:   tags,
	}
	err = db.Create(img).Error
	return img.ID, err
}

// TODO: cover with test
func (db *DB) GetTransformations(imageID int64) ([]Transformation, error) {
	var trans []Transformation
	img := &Image{ID: imageID}
	err := db.First(img, imageID).Error
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

	return trans, db.Where("Tag IN (?)", interfaceSlice).Find(&trans).Error

}

func (db *DB) SetTransformsUploaded(imgID int64) error {

	img := &Image{ID: imgID}
	err := db.Model(img).
		Updates(map[string]interface{}{"Transforms_Uploaded": true, "Transforms_Upload_Date": "now()"}).Error

	return err
}

func (db *DB) SetClaimImage(imageKey string, userID int32) error {
	img := &Image{}
	err := db.Model(img).
		Where("Key = ? AND User_ID = ?", imageKey, userID).
		Updates(map[string]interface{}{"Approved": true, "Approve_Date": "now()"}).Error

	return err
}

func (db *DB) DeleteImage(imageKey string) error {
	img := &Image{}
	err := db.Model(img).
		Where("Key = ?", imageKey).
		Updates(map[string]interface{}{"Deleted": true, "Deletion_Date": "now()"}).Error
	return err

}

func (db *DB) SetImageURL(key string, userID int32, URL string) error {
	img := &Image{}
	err := db.Model(img).
		Where("Key = ? AND User_ID = ?", key, userID).
		Updates(map[string]interface{}{"Url": URL}).Error
	return err
}

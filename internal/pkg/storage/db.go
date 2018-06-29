package storage

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	//"github.com/mattn/go-sqlite3"
)

const (
	TagLength = 20
)

type DB struct {
	*sql.DB
	driver         string
	dataSourceName string
}
type Tx struct {
	*sql.Tx
}

// Open returns a DB reference for a data source.
func Open(dataSourceName string) (*DB, error) {
	db, err := sql.Open("sqlite3", dataSourceName)

	if err != nil {
		return nil, err
	}
	return &DB{db, "sqlite3", dataSourceName}, nil
}

// Begin starts an returns a new transaction.
func (db *DB) Begin() (*Tx, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{tx}, nil
}

func (db *DB) InitDB() error {

	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS Users
		(
		 ID INTEGER PRIMARY KEY,
		 PublicKey VARCHAR(100),
		 SecretKey VARCHAR(100)
		)`)

	if err != nil {
		return err
	}
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS Images
		(
		 ID INTEGER PRIMARY KEY,
		 Key VARCHAR(20),
		 UserID INTEGER,
		 URL VARCHAR(50),
		 Approved BOOLEAN,
		 TransformsUploaded BOOLEAN,
		 CreateDate DATETIME,
		 ApproveDate DATETIME,
		 TransformsUploadDate DATETIME,

		 FOREIGN KEY(UserID) REFERENCES Users(ID)
		)`)

	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS Transformations
		(
		 ID INTEGER PRIMARY KEY,
		 Name VARCHAR(30),
		 Tag VARCHAR(%v),
		 Type VARCHAR(10),
		 Quality INTEGER,
		 Width INTEGER,
		 Height INTEGER
		)`, TagLength))

	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS ImageTags
		(
		 ImageID INTEGER,
		 Tag VARCHAR(%v),
		 
		 FOREIGN KEY(ImageID) REFERENCES Images(ID)
		)`, TagLength))
	return err
}

func (db *DB) DropDB() error {
	db.Close()
	if db.driver == "sqlite3" {
		os.Remove(db.dataSourceName)
		return os.Remove(db.dataSourceName + "-journal")

	}

	return fmt.Errorf("'%s' driver not supported", db.driver)
}

// CreateImage creates a new image.
// Returns id of newly created image and an error if some shit happened
func (tx *Tx) CreateImage(key string, userID int32) (int64, error) {

	stmt, err := tx.Prepare(`
		INSERT INTO Images(Key, UserID, CreateDate)
			VALUES (?, ?, DATETIME('now', 'localtime') )`)

	if err != nil {
		return 0, err
	}
	res, err := stmt.Exec(key, userID)
	if err != nil {
		return -1, err
	}
	return res.LastInsertId()
}

func (tx *Tx) AddImageTags(imageID int64, tags []string) error {
	query := "INSERT INTO ImageTags(ImageID, Tag) VALUES "
	params := make([]string, len(tags))
	args := make([]interface{}, len(tags)*2)

	for i, tag := range tags {
		params[i] = "(?, ?)"
		args[i*2] = imageID
		args[i*2+1] = tag
	}
	query += strings.Join(params, ",")

	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(args...)
	return err
}

func (tx *Tx) updateImage(query string, args ...interface{}) error {
	stmt, err := tx.Prepare(query)

	if err != nil {
		return err
	}

	res, err := stmt.Exec(args...)

	if err != nil {
		return err
	}
	if ra, er := res.RowsAffected(); er != nil {
		return er
	} else if ra != 1 {
		log.Printf("ERROR: failed to update image: 1 row should be updated but updated %v", ra)
		if err = tx.Rollback(); err != nil {
			return err
		}
		return fmt.Errorf("failed to update image: 1 row should be updated but updated %v", ra)
	}
	return nil
}

func (tx *Tx) ClaimImage(key string, userID int32) error {
	return tx.updateImage(`
		UPDATE Images
		SET Approved=true,
			ApproveDate=DATETIME('now', 'localtime')
		WHERE Key=? AND UserID=?`, key, userID)
}

func (tx *Tx) SetImageURL(key string, userID int32, URL string) error {
	return tx.updateImage(`
		UPDATE Images
		SET URL=?
		WHERE Key=? AND UserID=?`, URL, key, userID)

}

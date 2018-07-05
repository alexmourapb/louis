package storage

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	//"github.com/mattn/go-sqlite3"
)

const (
	TagLength = 20
)

type DB struct {
	*sql.DB
	mutex          *sync.Mutex
	driver         string
	dataSourceName string
}
type Tx struct {
	*sql.Tx
}

// Open returns a DB reference for a data source.
func Open(dataSourceName string) (*DB, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	db.SetMaxOpenConns(1)
	if err != nil {
		return nil, err
	}
	return &DB{db, new(sync.Mutex), "sqlite3", dataSourceName}, nil
}

// Begin starts an returns a new transaction.
func (db *DB) Begin() (*Tx, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{tx}, nil
}

func (db *DB) Lock() {
	db.mutex.Lock()
}

func (db *DB) Unlock() {
	db.mutex.Unlock()
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
		 URL VARCHAR(50) DEFAULT '' NOT NULL,
		 Approved BOOLEAN DEFAULT FALSE,
		 TransformsUploaded BOOLEAN DEFAULT FALSE,
		 CreateDate DATETIME DEFAULT current_timestamp,
		 ApproveDate DATETIME DEFAULT current_timestamp,
		 TransformsUploadDate DATETIME DEFAULT current_timestamp,

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
		 Height INTEGER,
		 -- UserID INTEGER, -- will be needed in future

		 UNIQUE(Name)
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
	if err != nil {
		return err
	}
	// add default transformations
	// db.Exec(`
	// INSERT INTO Transformations(Name, Tag, Type, Quality, Width, Height)
	// VALUES ('thubnail_100x100_20', 'thubnail_small_low', 'fit', 20, 100, 100)`)

	return err
}

func (db *DB) EnsureTransformations(trans []Transformation) error {
	query := "INSERT OR IGNORE INTO Transformations(Name, Tag, Type, Quality, Width, Height) VALUES "
	var args []interface{}

	query += strings.Repeat("(?, ?, ?, ?, ?, ?), ", len(trans)-1) + "(?, ?, ?, ?, ?, ?)"
	for _, tran := range trans {
		args = append(args, tran.Name, tran.Tag, tran.Type, tran.Quality, tran.Width, tran.Height)
	}
	_, err := db.Exec(query, args...)
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

func (db *DB) QueryImageByKey(key string) (*Image, error) {

	rows, err := db.Query(`
		SELECT ID, Key, UserID, URL, Approved, TransformsUploaded, CreateDate, ApproveDate, TransformsUploadDate
		FROM Images
		WHERE Key=?`, key)
	defer rows.Close()

	if err != nil {
		return nil, err
	}

	if !rows.Next() {
		return nil, fmt.Errorf("image not found")
	}
	img := new(Image)
	return img, rows.Scan(&img.ID, &img.Key, &img.UserID, &img.URL, &img.Approved, &img.TransformsUploaded, &img.CreateDate, &img.ApproveDate, &img.TransformsUploadDate)
}

func (db *DB) AddImage(imageKey string, userID int32, tags ...string) (ImageID int64, err error) {
	db.Lock()
	defer db.Unlock()
	ImageID = -1
	tx, err := db.Begin()
	if err != nil {
		return
	}

	ImageID, err = tx.CreateImage(imageKey, userID)
	if err != nil {
		return
	}

	err = tx.Commit()
	if err != nil {
		return
	}

	if tags != nil && len(tags) > 0 {
		tx, err = db.Begin()
		if err != nil {
			return
		}

		err = tx.AddImageTags(ImageID, tags)
		if err != nil {
			return
		}
		err = tx.Commit()
	}
	return
}

func (db *DB) GetTransformations(imageID int64) ([]Transformation, error) {
	rows, err := db.Query(`
		SELECT t.Name, t.Tag, t.Type, t.Quality, t.Width, t.Height
		FROM Transformations t, ImageTags it
		WHERE it.ImageID = ? AND it.Tag = t.Tag`, imageID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trans []Transformation

	for rows.Next() {
		tr := Transformation{}
		err := rows.Scan(&tr.Name, &tr.Tag, &tr.Type, &tr.Quality, &tr.Width, &tr.Height)
		if err != nil {
			return nil, err
		}

		trans = append(trans, tr)
	}
	return trans, nil
}

func (db *DB) SetTransformsUploaded(imgID int64) error {
	db.Lock()
	defer db.Unlock()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	err = tx.updateImage(`
		UPDATE Images
		SET TransformsUploaded=true,
			TransformsUploadDate=DATETIME('now', 'localtime')
		WHERE ID=?`, imgID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) SetClaimImage(imageKey string, userID int32) error {
	db.Lock()
	defer db.Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	err = tx.ClaimImage(imageKey, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
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

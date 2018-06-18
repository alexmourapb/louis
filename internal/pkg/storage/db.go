package storage

import (
	"database/sql"
	"fmt"
	"log"
	//"github.com/mattn/go-sqlite3"
	"time"
)

type DB struct {
	*sql.DB
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
	return &DB{db}, nil
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
		(ID INTEGER PRIMARY KEY,
		 PublicKey VARCHAR(100),
		 SecretKey VARCHAR(100))`)

	if err != nil {
		return err
	}
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS images
		(ID INTEGER PRIMARY KEY,
		 Key VARCHAR(20),
		 UserID INTEGER,
		 URL VARCHAR(50),
		 Approved BOOLEAN,
		 TransformsUploaded BOOLEAN,
		 CreateDate DATETIME,
		 ApproveDate DATETIME,
		 TransformsUploadDate DATETIME,

		 FOREIGN KEY(UserID) REFERENCES Users(ID))`)
	return err
}

// CreateImage creates a new image.
// Returns id of newly created image and an error if some shit happened
func (tx *Tx) CreateImage(key string, userID int32) (int64, error) {

	stmt, err := tx.Prepare(`
		INSERT INTO images(Key, UserID, CreateDate)
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

func (tx *Tx) ClaimImage(key string, userID int32) error {
	stmt, err := tx.Prepare(`
		UPDATE Images
		SET Approved=true,
			ApproveDate=DATETIME('now', 'localtime')
		WHERE Key=? AND UserID=?`)

	if err != nil {
		return err
	}

	res, err := stmt.Exec(key, userID)

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

func maqin() {
	db, err := sql.Open("sqlite3", "./foo.db")
	checkErr(err)

	// insert
	stmt, err := db.Prepare("INSERT INTO userinfo(username, departname, created) values(?,?,?)")
	checkErr(err)

	res, err := stmt.Exec("astaxie", "研发部门", "2012-12-09")
	checkErr(err)

	id, err := res.LastInsertId()
	checkErr(err)

	fmt.Println(id)
	// update
	stmt, err = db.Prepare("update userinfo set username=? where uid=?")
	checkErr(err)

	res, err = stmt.Exec("astaxieupdate", id)
	checkErr(err)

	affect, err := res.RowsAffected()
	checkErr(err)

	fmt.Println(affect)

	// query
	rows, err := db.Query("SELECT * FROM userinfo")
	checkErr(err)
	var uid int
	var username string
	var department string
	var created time.Time

	for rows.Next() {
		err = rows.Scan(&uid, &username, &department, &created)
		checkErr(err)
		fmt.Println(uid)
		fmt.Println(username)
		fmt.Println(department)
		fmt.Println(created)
	}

	rows.Close() //good habit to close

	// delete
	stmt, err = db.Prepare("delete from userinfo where uid=?")
	checkErr(err)

	res, err = stmt.Exec(id)
	checkErr(err)

	affect, err = res.RowsAffected()
	checkErr(err)

	fmt.Println(affect)

	db.Close()

}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

package storage

import (
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"testing"
)

var pathToTestDB = "../../../test/data/test.db"

func failIfError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s - %v", msg, err)
	}
}
func TestInitDB(t *testing.T) {
	var db, err = Open(pathToTestDB)
	defer os.Remove(pathToTestDB)
	defer db.Close()

	failIfError(err, "failed to open db")

	failIfError(db.InitDB(), "failed to create initial tables")

}

func TestCreateImage(t *testing.T) {
	var db, err = Open(pathToTestDB)
	defer os.Remove(pathToTestDB)
	defer os.Remove(pathToTestDB + "-journal")
	defer db.Close()

	failIfError(err, "failed to open db")

	err = db.InitDB()
	failIfError(err, "failed to create tables")

	tx, err := db.Begin()
	failIfError(err, "failed to create transaction")

	id, err := tx.CreateImage("test_image_key", 1)
	failIfError(err, "failed to create image")

	err = tx.Commit()
	failIfError(err, "failed to create image")

	if id != 1 {
		log.Fatalf("expected 1 but get %v", id)
	}

	rows, err := db.Query("SELECT ID, UserID, Key FROM Images WHERE key='test_image_key'")
	failIfError(err, "failed to find created row")

	var (
		rowID  int
		userID int
		key    string
	)
	if rows.Next() {
		failIfError(rows.Scan(&rowID, &userID, &key), "failed to read rowID, userID, key")
	} else {
		log.Fatalf("image not saved")
	}
	if rowID != 1 {
		log.Fatalf("expected image id 1 bug get %v", rowID)
	}
	if key != "test_image_key" {
		log.Fatalf("expected image key 'test_image_key' bug get %v", key)
	}

	if userID != 1 {
		log.Fatalf("expected userID = 1 but get %v", userID)
	}
}

func TestClaimImage(t *testing.T) {
	var db, err = Open(pathToTestDB)
	defer os.Remove(pathToTestDB)
	defer os.Remove(pathToTestDB + "-journal")
	defer db.Close()

	failIfError(err, "failed to open db")

	err = db.InitDB()
	failIfError(err, "failed to create tables")

	tx, err := db.Begin()
	failIfError(err, "failed to create createImage transaction")

	var (
		imageKey = "imageKey"
		userID   = int32(2)
	)
	_, err = tx.CreateImage(imageKey, userID)

	failIfError(err, "failed to create image")

	failIfError(tx.Commit(), "failed to commit create image transaction")

	tx, err = db.Begin()
	failIfError(err, "failed to create claim transaction")

	failIfError(tx.ClaimImage(imageKey, userID), "failed to claim image")

	failIfError(tx.Commit(), "failed to commit claim image transaction")

	rows, err := db.Query("SELECT Approved FROM Images WHERE key=?", imageKey)

	var approved bool

	if rows.Next() {
		failIfError(rows.Scan(&approved), "failed to scan approved column")
	} else {
		log.Fatalf("image with key %s not found", imageKey)
	}

	if !approved {
		log.Fatalf("expected approved = true but get %v", approved)
	}
}

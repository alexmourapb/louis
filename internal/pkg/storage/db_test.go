package storage

import (
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	// "os"
	"testing"
)

var pathToTestDB = "../../../test/data/test.db"

func getDB() (*DB, error) {
	return Open(pathToTestDB)
}

func failIfError(t *testing.T, err error, msg string) {
	if err != nil {
		t.Fatalf("%s - %v", msg, err)
	}
}
func TestInitDB(t *testing.T) {
	var db, err = getDB()
	defer db.DropDB()

	failIfError(t, err, "failed to open db")

	failIfError(t, db.InitDB(), "failed to create initial tables")
}

func TestCreateImage(t *testing.T) {
	var db, err = getDB()
	defer db.DropDB()

	failIfError(t, err, "failed to open db")

	failIfError(t, db.InitDB(), "failed to create tables")

	tx, err := db.Begin()
	failIfError(t, err, "failed to create transaction")

	id, err := tx.CreateImage("test_image_key", 1)
	failIfError(t, err, "failed to create image")

	err = tx.Commit()
	failIfError(t, err, "failed to create image")

	if id != 1 {
		t.Fatalf("expected 1 but get %v", id)
	}

	rows, err := db.Query("SELECT ID, UserID, Key FROM Images WHERE key='test_image_key'")
	defer rows.Close()

	failIfError(t, err, "failed to find created row")

	var (
		rowID  int
		userID int
		key    string
	)
	if rows.Next() {
		failIfError(t, rows.Scan(&rowID, &userID, &key), "failed to read rowID, userID, key")
	} else {
		t.Fatalf("image not saved")
	}
	if rowID != 1 {
		t.Fatalf("expected image id 1 bug get %v", rowID)
	}
	if key != "test_image_key" {
		t.Fatalf("expected image key 'test_image_key' bug get %v", key)
	}

	if userID != 1 {
		t.Fatalf("expected userID = 1 but get %v", userID)
	}
}

func TestClaimImage(t *testing.T) {
	var db, err = getDB()
	defer db.DropDB()

	failIfError(t, err, "failed to open db")

	failIfError(t, db.InitDB(), "failed to create tables")

	tx, err := db.Begin()
	failIfError(t, err, "failed to create createImage transaction")

	var (
		imageKey = "imageKey"
		userID   = int32(2)
	)
	_, err = tx.CreateImage(imageKey, userID)

	failIfError(t, err, "failed to create image")

	failIfError(t, tx.Commit(), "failed to commit create image transaction")

	tx, err = db.Begin()
	failIfError(t, err, "failed to create claim transaction")

	failIfError(t, tx.ClaimImage(imageKey, userID), "failed to claim image")

	failIfError(t, tx.Commit(), "failed to commit claim image transaction")

	rows, err := db.Query("SELECT Approved FROM Images WHERE key=?", imageKey)
	defer rows.Close()

	var approved bool

	if rows.Next() {
		failIfError(t, rows.Scan(&approved), "failed to scan approved column")
	} else {
		t.Fatalf("image with key %s not found", imageKey)
	}

	if !approved {
		t.Fatalf("expected approved = true but get %v", approved)
	}
}

func TestSetImageURL(t *testing.T) {

	var db, err = getDB()
	defer db.DropDB()

	failIfError(t, err, "failed to open db")

	failIfError(t, db.InitDB(), "failed to create tables")

	tx, err := db.Begin()
	failIfError(t, err, "failed to create createImage transaction")

	var (
		imageKey = "imageKey"
		userID   = int32(2)
		imageURL = "https://test.hb.mcs.ru/test.jpg"
	)
	_, err = tx.CreateImage(imageKey, userID)

	failIfError(t, err, "failed to create image")

	failIfError(t, tx.Commit(), "failed to commit create image tx")

	tx, err = db.Begin()

	failIfError(t, tx.SetImageURL(imageKey, userID, imageURL), "failed to set image url")

	failIfError(t, tx.Commit(), "failed to commit set image url tx")

	rows, err := db.Query("SELECT URL FROM Images WHERE key=?", imageKey)
	defer rows.Close()

	var URL string

	if rows.Next() {
		failIfError(t, rows.Scan(&URL), "failed to scan URL column")
	} else {
		t.Fatalf("image with key %s not found", imageKey)
	}

	if URL != imageURL {
		t.Fatalf("expected URL = true but get %v", URL)
	}

}

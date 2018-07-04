package storage

import (
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
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

	assert.Equal(t, int64(1), id)

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

	assert.Equal(t, 1, rowID, "imageID should be 1")
	assert.Equal(t, "test_image_key", key)
	assert.Equal(t, 1, userID)
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

	failIfError(t, db.SetClaimImage(imageKey, userID), "failed to create claim transaction")

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

	var urlFromDatabase string

	if rows.Next() {
		failIfError(t, rows.Scan(&urlFromDatabase), "failed to scan URL column")
	} else {
		t.Fatalf("image with key %s not found", imageKey)
	}

	assert.Equal(t, imageURL, urlFromDatabase)

}

func TestAddImageTags(t *testing.T) {
	var db, err = getDB()
	defer db.DropDB()

	failIfError(t, err, "failed to open db")

	failIfError(t, db.InitDB(), "failed to create tables")

	tx, err := db.Begin()
	failIfError(t, err, "failed to create createImage transaction")

	var (
		imageKey = "imageKey"
		userID   = int32(2)
		tags     = []string{"tag1", "tag2", "super-tag"}
	)

	imageID, err := tx.CreateImage(imageKey, userID)
	failIfError(t, err, "failed to create image")
	failIfError(t, tx.Commit(), "failed to commit create image tx")

	tx, err = db.Begin()
	failIfError(t, err, "failed to create add image tags transaction")
	failIfError(t, tx.AddImageTags(imageID, tags), "failed to add tags")
	failIfError(t, tx.Commit(), "failed to commit add image tags tx")

	rows, err := db.Query("SELECT Tag FROM ImageTags WHERE ImageID=?", imageID)
	defer rows.Close()

	failIfError(t, err, "failed to obtain rows")

	var recivedTags = make([]string, len(tags))
	if rows.Next() {
		failIfError(t, rows.Scan(&recivedTags[0]), "failed to scan")
		rows.Next()
		failIfError(t, rows.Scan(&recivedTags[1]), "failed to scan")
		rows.Next()
		failIfError(t, rows.Scan(&recivedTags[2]), "failed to scan")
	}

	assert.ElementsMatch(t, recivedTags, tags)
}

func TestAddImage(t *testing.T) {
	t.Run("without tags", getAddImageTest("this_is_image_key", 1))
	t.Run("without tags", getAddImageTest("this_is_image_key", 1, "this-is-tag", "tag2", "super-tag"))
}

func getAddImageTest(key string, userID int32, tags ...string) func(*testing.T) {
	return func(t *testing.T) {
		var db, err = getDB()
		defer db.DropDB()

		failIfError(t, err, "failed to open db")

		failIfError(t, db.InitDB(), "failed to create tables")

		imageID, err := db.AddImage(key, userID, tags...)

		assert.Equal(t, int64(1), imageID, "imageID should be 1")

		rows, err := db.Query("SELECT ID, UserID, Key FROM Images WHERE key=?", key)
		defer rows.Close()

		failIfError(t, err, "failed to find created row")

		var (
			imgIDFromDB  int64
			userIDFromDB int32
			keyFromDB    string
		)
		if rows.Next() {
			failIfError(t, rows.Scan(&imgIDFromDB, &userIDFromDB, &keyFromDB), "failed to read rowID, userID, key")
		} else {
			t.Fatalf("image not saved")
		}

		assert.Equal(t, imageID, imgIDFromDB)
		assert.Equal(t, key, keyFromDB)
		assert.Equal(t, userID, userIDFromDB)

		if len(tags) > 0 {
			rows.Close()
			rows, err := db.Query("SELECT Tag FROM ImageTags WHERE ImageID=?", imageID)
			// defer rows.Close()

			failIfError(t, err, "failed to obtain rows")

			var recivedTags []string
			var tag string
			for rows.Next() {
				failIfError(t, rows.Scan(&tag), "failed to scan")
				recivedTags = append(recivedTags, tag)
			}

			assert.ElementsMatch(t, recivedTags, tags)
		}
	}
}

func TestGetTransformations(t *testing.T) {
	assert := assert.New(t)

	var db, err = getDB()
	defer db.DropDB()

	failIfError(t, err, "failed to open db")

	failIfError(t, db.InitDB(), "failed to create tables")

	const (
		imgKey       = "img_key"
		userID int32 = 1
	)
	var tags = []string{"thubnail_small_low"}

	imgID, err := db.AddImage(imgKey, userID, tags...)
	assert.NoError(err)

	trans, err := db.GetTransformations(imgID)
	assert.NoError(err)

	assert.Equal(1, len(trans))
}

func TestQueryImageByKey(t *testing.T) {
	assert := assert.New(t)

	var db, err = getDB()
	defer db.DropDB()

	failIfError(t, err, "failed to open db")

	failIfError(t, db.InitDB(), "failed to create tables")

	const (
		imgKey       = "img_key"
		userID int32 = 1
	)

	imgID, err := db.AddImage(imgKey, userID)

	assert.NoError(err)

	img, err := db.QueryImageByKey(imgKey)

	assert.NoError(err)
	assert.Equal(imgID, img.ID)
	assert.Equal(userID, img.UserID)

}

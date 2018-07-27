package storage

import (
	"github.com/KazanExpress/louis/internal/pkg/config"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	// "os"
	"testing"
)

var pathToTestDB = "../../../test/data/test.db"

var tlist = []Transformation{
	{
		Name:    "super_transform",
		Tag:     "thubnail_small_low",
		Width:   100,
		Height:  100,
		Quality: 40,
		Type:    "fit",
	},
	{
		Name:    "cover",
		Tag:     "cover_wide",
		Type:    "fill",
		Width:   1200,
		Height:  200,
		Quality: 70,
	},
}

func getDB() (*DB, error) {
	cfg := config.InitFrom("../../../.env")
	return Open(cfg)
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

func TestDeleteImage(t *testing.T) {
	assert := assert.New(t)
	var db, err = getDB()
	defer db.DropDB()

	failIfError(t, err, "failed to open db")

	failIfError(t, db.InitDB(), "failed to create tables")

	_, err = db.AddImage("key", 1)

	assert.NoError(err)

	assert.NoError(db.DeleteImage("key"))

	img, err := db.QueryImageByKey("key")

	assert.NoError(err)
	assert.True(img.Deleted)
}

func TestClaimImage(t *testing.T) {
	var db, err = getDB()
	defer db.DropDB()

	failIfError(t, err, "failed to open db")

	failIfError(t, db.InitDB(), "failed to create tables")

	var (
		imageKey = "imageKey"
		userID   = int32(2)
	)
	_, err = db.AddImage(imageKey, userID)

	failIfError(t, err, "failed to create image")

	failIfError(t, db.SetClaimImage(imageKey, userID), "failed to create claim transaction")

	img, err := db.QueryImageByKey(imageKey)

	assert.NoError(t, err)
	assert.True(t, img.Approved)
}

func TestSetImageURL(t *testing.T) {

	var db, err = getDB()
	defer db.DropDB()

	failIfError(t, err, "failed to open db")

	failIfError(t, db.InitDB(), "failed to create tables")

	var (
		imageKey = "imageKey"
		userID   = int32(2)
		imageURL = "https://test.hb.mcs.ru/test.jpg"
	)
	_, err = db.AddImage(imageKey, userID)

	assert.NoError(t, err)

	failIfError(t, db.SetImageURL(imageKey, userID, imageURL), "failed to set image url")

	img, err := db.QueryImageByKey(imageKey)

	assert.Equal(t, imageURL, img.URL)

}

func TestAddImageTags(t *testing.T) {
	var db, err = getDB()
	defer db.DropDB()

	failIfError(t, err, "failed to open db")

	failIfError(t, db.InitDB(), "failed to create tables")

	var (
		imageKey = "imageKey"
		userID   = int32(2)
		tags     = []string{"tag1", "tag2", "super-tag"}
	)

	_, err = db.AddImage(imageKey, userID, tags...)
	failIfError(t, err, "failed to create image")

	img, err := db.QueryImageByKey(imageKey)
	assert.NoError(t, err)

	assert.ElementsMatch(t, img.Tags, tags)
}

func TestAddImage(t *testing.T) {
	t.Run("without tags", getAddImageTest("this_is_image_key", 1))
	t.Run("with tags", getAddImageTest("this_is_image_key", 1, "this-is-tag", "tag2", "super-tag"))
}

func getAddImageTest(key string, userID int32, tags ...string) func(*testing.T) {
	return func(t *testing.T) {
		var db, err = getDB()
		defer db.DropDB()

		failIfError(t, err, "failed to open db")

		failIfError(t, db.InitDB(), "failed to create tables")

		imageID, err := db.AddImage(key, userID, tags...)

		assert.Equal(t, int64(1), imageID, "imageID should be 1")

		img, err := db.QueryImageByKey(key)

		failIfError(t, err, "failed to find created row")

		assert.Equal(t, imageID, img.ID)
		assert.Equal(t, key, img.Key)
		assert.Equal(t, userID, img.UserID)
		assert.ElementsMatch(t, tags, img.Tags)

	}
}

func TestGetTransformations(t *testing.T) {
	assert := assert.New(t)

	var db, err = getDB()
	defer db.DropDB()

	failIfError(t, err, "failed to open db")

	failIfError(t, db.InitDB(), "failed to create tables")

	assert.NoError(db.EnsureTransformations(tlist))
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

func TestEnsureTransformations(t *testing.T) {
	assert := assert.New(t)

	var db, err = getDB()
	defer db.DropDB()

	failIfError(t, err, "failed to open db")

	failIfError(t, db.InitDB(), "failed to create tables")

	assert.NoError(db.EnsureTransformations(tlist))

	count, err := db.Model((*Transformation)(nil)).Count()

	assert.NoError(err)

	assert.Equal(len(tlist), count)
	assert.NoError(db.EnsureTransformations(tlist))

	count, err = db.Model((*Transformation)(nil)).Count()

	assert.Equal(len(tlist), count)

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

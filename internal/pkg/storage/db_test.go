package storage

import (
	"fmt"
	"github.com/KazanExpress/louis/internal/pkg/config"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"log"
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

type Suite struct {
	suite.Suite
	db *DB
}

func (s *Suite) SetupSuite() {
	log.Printf("Executing setup all suite")
	var err error
	s.db, err = getDB()
	if err != nil {
		s.Fail("Failed to open db:%v", err)
	}
}

func (s *Suite) BeforeTest(tn, sn string) {
	err := s.db.InitDB()
	if err != nil {
		s.Fail("failed to initdb: %v", err)
	}
}

func (s *Suite) TearDownSuite() {
	s.db.Close()
}

func (s *Suite) AfterTest(tn, sn string) {
	s.db.DropDB()
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

func (s *Suite) TestDeleteImage() {
	assert := s.Assertions
	var db = s.db

	_, err := db.AddImage("key", 1)

	assert.NoError(err)

	assert.NoError(db.DeleteImage("key"))

	img, err := db.QueryImageByKey("key")

	assert.NoError(err)
	assert.True(img.Deleted)
}

func (s *Suite) TestClaimImage() {
	var db = s.db
	assert := assert.New(s.T())
	var (
		imageKey = "imageKey"
		userID   = int32(2)
	)
	_, err := db.AddImage(imageKey, userID)

	assert.NoError(err, "failed to create image")

	assert.NoError(db.SetClaimImage(imageKey, userID), "failed to create claim transaction")

	img, err := db.QueryImageByKey(imageKey)

	assert.NoError(err)
	assert.True(img.Approved)
}

func (s *Suite) TestSetImageURL() {

	var db = s.db
	assert := assert.New(s.T())
	var (
		imageKey = "imageKey"
		userID   = int32(2)
		imageURL = "https://test.hb.mcs.ru/test.jpg"
	)
	_, err := db.AddImage(imageKey, userID)

	assert.NoError(err)

	assert.NoError(db.SetImageURL(imageKey, userID, imageURL), "failed to set image url")

	img, err := db.QueryImageByKey(imageKey)
	s.NoError(err)
	assert.Equal(imageURL, img.URL)

}

func (s *Suite) TestAddImageTags() {
	var db = s.db
	assert := assert.New(s.T())

	var (
		imageKey = "imageKey"
		userID   = int32(2)
		tags     = []string{"tag1", "tag2", "super-tag"}
	)

	_, err := db.AddImage(imageKey, userID, tags...)
	assert.NoError(err, "failed to create image")

	img, err := db.QueryImageByKey(imageKey)
	assert.NoError(err)

	assert.ElementsMatch(img.Tags, tags)
}

func (s *Suite) TestAddImageWithTags() {
	getAddImageTest("this_is_image_key", 1, "this-is-tag", "tag2", "super-tag")(s)
}
func (s *Suite) TestAddImageWithoutTags() {
	getAddImageTest("this_is_image_key", 1)(s)
}

func getAddImageTest(key string, userID int32, tags ...string) func(*Suite) {
	return func(s *Suite) {
		db := s.db
		assert := assert.New(s.T())
		imageID, err := db.AddImage(key, userID, tags...)
		s.NoError(err)
		assert.Equal(int64(1), imageID, "imageID should be 1")

		img, err := db.QueryImageByKey(key)

		assert.NoError(err, "failed to find created row")

		assert.Equal(imageID, img.ID)
		assert.Equal(key, img.Key)
		assert.Equal(userID, img.UserID)
		assert.ElementsMatch(tags, img.Tags)

	}
}

func (s *Suite) TestGetTransformations() {
	db := s.db
	assert := assert.New(s.T())

	assert.NoError(db.EnsureTransformations(tlist))
	const (
		imgKey       = "img_key"
		userID int32 = 1
	)
	var tags = []string{"thubnail_small_low", "cover_wide"}

	imgID, err := db.AddImage(imgKey, userID, tags...)
	assert.NoError(err)

	trans, err := db.GetTransformations(imgID)
	assert.NoError(err)

	assert.Equal(1, len(trans))
}

func (s *Suite) TestEnsureTransformations(t *testing.T) {
	assert := assert.New(t)
	db := s.db

	assert.NoError(db.EnsureTransformations(tlist))

	var count int
	err := db.Model(&Transformation{}).Count(&count).Error

	assert.NoError(err)

	assert.Equal(len(tlist), count)
	assert.Equal(fmt.Errorf("sql: no rows in result set"), db.EnsureTransformations(tlist))

	err = db.Model(&Transformation{}).Count(&count).Error
	s.NoError(err)
	assert.Equal(len(tlist), count)

}

func (s *Suite) TestQueryImageByKey(t *testing.T) {
	db := s.db
	assert := assert.New(s.T())

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

func (s *Suite) TestGetImagesWithKeys() {
	const userID = 1
	var keys = []string{"key1", "key2", "key3", "key4"}

	for _, key := range keys {
		_, err := s.db.AddImage(key, userID)
		s.NoError(err)
	}

	images, err := s.db.GetImagesWithKeys(keys[:2])
	s.NoError(err)
	s.Equal(2, len(*images))
}

func (s *Suite) TestClaimImages() {
	const userID = 1
	var keys = []string{"key1", "key2", "key3", "key4"}
	for _, key := range keys {
		_, err := s.db.AddImage(key, userID)
		s.NoError(err)
	}

	s.NoError(s.db.SetClaimImages(keys, userID))
	images, err := s.db.GetImagesWithKeys(keys)
	s.NoError(err)
	for _, img := range *images {
		s.True(img.Approved)
	}
}

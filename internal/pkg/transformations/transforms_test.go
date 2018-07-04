package transformations

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/bimg.v1"
	"io/ioutil"
	"path"
	"testing"
)

func testFit(pictureBytes []byte) func(*testing.T) {

	return func(t *testing.T) {

		// assert.
		assert := assert.New(t)

		img := bimg.NewImage(pictureBytes)
		sz, err := img.Size()
		assert.NoError(err)

		side := sz.Width / 2

		newPictureBytes, err := Fit(pictureBytes, side, 80)
		assert.NoError(err)

		newImg := bimg.NewImage(newPictureBytes)

		assert.Condition(func() bool {
			isz, err := newImg.Size()
			assert.NoError(err)

			return isz.Width <= side && isz.Height <= side
		})
	}
}

func testFill(pictureBytes []byte) func(*testing.T) {

	return func(t *testing.T) {

		// assert.
		assert := assert.New(t)

		newPictureBytes, err := Fill(pictureBytes, 1200, 200, 80)
		assert.NoError(err)

		newImg := bimg.NewImage(newPictureBytes)

		assert.Condition(func() bool {
			isz, err := newImg.Size()
			assert.NoError(err)

			return isz.Width == 1200 && isz.Height == 200
		})
	}
}

func TestFit(t *testing.T) {
	const picsDir = "../../../test/data/pics"
	files, err := ioutil.ReadDir(picsDir)
	assert.NoError(t, err)
	for _, file := range files {
		imgpath := path.Join(picsDir, file.Name())
		picture, err := bimg.Read(imgpath)
		assert.NoError(t, err)
		t.Run(fmt.Sprintf("Test Fit on image %v", imgpath), testFit(picture))
	}
}

func TestFill(t *testing.T) {
	const picsDir = "../../../test/data/pics"
	files, err := ioutil.ReadDir(picsDir)
	assert.NoError(t, err)

	for _, file := range files {
		imgpath := path.Join(picsDir, file.Name())
		picture, err := bimg.Read(imgpath)
		assert.NoError(t, err)
		t.Run(fmt.Sprintf("Test Fill on image %v", imgpath), testFill(picture))
	}
}

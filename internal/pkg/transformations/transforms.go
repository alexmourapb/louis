package transformations

import (
	"gopkg.in/h2non/bimg.v1"
)

// Fit - the image is resized so that it takes up as much space as possible
// within a bounding box defined by the given width and height parameters.
// The original aspect ratio is retained and all of the original image is visible.
func Fit(buffer []byte, side, quality int) ([]byte, error) {
	var img = bimg.NewImage(buffer)

	if img.Type() != "jpg" {
		jpg, err := img.Convert(bimg.JPEG)
		if err != nil {
			return nil, err
		}
		img = bimg.NewImage(jpg)
	}

	var sz, err = img.Size()
	if err != nil {
		return nil, err
	}
	if sz.Height > sz.Width {
		return img.Process(bimg.Options{
			Height:        side,
			Quality:       quality,
			StripMetadata: true,
		})
	} else {
		return img.Process(bimg.Options{
			Width:         side,
			Quality:       quality,
			StripMetadata: true,
		})
	}
}

// Fill - fills image to given width & height
func Fill(buffer []byte, width, height, quality int) ([]byte, error) {
	var img = bimg.NewImage(buffer)
	return img.Process(bimg.Options{
		Width:   width,
		Height:  height,
		Enlarge: true,
		Embed:   true,
	})
}

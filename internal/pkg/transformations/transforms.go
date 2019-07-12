package transformations

import (
	"gopkg.in/h2non/bimg.v1"
)

type ImageBuffer = []byte

// Fit - the image is resized so that it takes up as much space as possible
// within a bounding box defined by the given width and height parameters.
// The original aspect ratio is retained and all of the original image is visible.
func Fit(buffer ImageBuffer, side, quality int) (ImageBuffer, error) {
	var img = bimg.NewImage(buffer)

	if img.Type() != "jpeg" {
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
			NoAutoRotate:  false,
			Interlace:     true, // adds progressive jpeg support
		})
	}

	return img.Process(bimg.Options{
		Width:         side,
		Quality:       quality,
		StripMetadata: true,
		NoAutoRotate:  false,
		Interlace:     true, // adds progressive jpeg support
	})
}

// Fill - fills image to given width & height
func Fill(buffer ImageBuffer, width, height, quality int) (ImageBuffer, error) {
	var img = bimg.NewImage(buffer)
	return img.Process(bimg.Options{
		Width:         width,
		Height:        height,
		Enlarge:       true,
		Embed:         true,
		NoAutoRotate:  false,
		StripMetadata: true,
		Interlace:     true, // adds progressive jpeg support
	})
}

// Compress - reduces quality of image
func Compress(buffer ImageBuffer, quality int) (ImageBuffer, error) {
	return bimg.NewImage(buffer).Process(bimg.Options{
		Quality:       quality,
		NoAutoRotate:  false,
		Interlace:     true, // Adds progressive jpeg support
		StripMetadata: true,
	})
}

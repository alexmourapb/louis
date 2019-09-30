package transformations

import (
	"fmt"
	"github.com/KazanExpress/louis/internal/pkg/storage"
	"github.com/KazanExpress/louis/internal/pkg/utils"
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

// Crop - Extracts area image of image between from top left point with given height and width
func Crop(buffer ImageBuffer, x, y, width, height, quality int) (ImageBuffer, error) {
	var img = bimg.NewImage(buffer)
	var tmpImg, err = img.Process(bimg.Options{
		NoAutoRotate:  false,
		StripMetadata: true,
		Interlace:     true, // adds progressive jpeg support
	})
	if err != nil {
		return nil, err
	}

	img = bimg.NewImage(tmpImg)
	return img.Extract(y, x, width, height)
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

func MakeProgressive(image ImageBuffer) (ImageBuffer, error) {
	return bimg.NewImage(image).Process(bimg.Options{
		NoAutoRotate:  false,
		Interlace:     true,
		StripMetadata: true,
	})
}

// TODO: move it to other package
type TransformParams struct {
	Image      ImageBuffer
	CropSquare *utils.Square
}

// ImageTransformer - is shortcut type
type ImageTransformer = func(params TransformParams, trans *storage.Transformation) (ImageBuffer, error)

// GetTransformsMappings - returns map containing transformers for each transform type
func GetTransformsMappings() map[string]ImageTransformer {
	return map[string]ImageTransformer{
		"fill": func(params TransformParams, tran *storage.Transformation) (ImageBuffer, error) {
			return Fill(params.Image, tran.Width, tran.Height, tran.Quality)
		},
		"fit": func(params TransformParams, tran *storage.Transformation) (ImageBuffer, error) {
			return Fit(params.Image, tran.Width, tran.Quality)
		},
		"real": func(params TransformParams, trans *storage.Transformation) (ImageBuffer, error) {
			return params.Image, nil
		},
		"original": func(params TransformParams, trans *storage.Transformation) (ImageBuffer, error) {
			return Compress(params.Image, trans.Quality)
		},
		"crop": func(params TransformParams, trans *storage.Transformation) (ImageBuffer, error) {
			if params.CropSquare == nil {
				return nil, fmt.Errorf("crop requires crop square")
			}
			var width = params.CropSquare.BottomRightPoint.X - params.CropSquare.TopLeftPoint.X
			var height = params.CropSquare.BottomRightPoint.Y - params.CropSquare.TopLeftPoint.Y
			return Crop(params.Image, params.CropSquare.TopLeftPoint.X, params.CropSquare.TopLeftPoint.Y, width, height, trans.Quality)
		},
	}
}

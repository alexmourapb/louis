package louis

import (
	"bytes"
	"context"
	"fmt"
	"github.com/KazanExpress/louis/internal/pkg/storage"
	"github.com/KazanExpress/louis/internal/pkg/transformations"
	"github.com/KazanExpress/louis/internal/pkg/utils"
	"log"
	"strings"
	"sync"
)

var (
	ImageCanNotBeRestoredError = fmt.Errorf("image can not be restored")
	ImageNotArchivedError      = fmt.Errorf("image is not archived, nothing to restore")
)

// ImageService - interface of a service for uploading and transforming image
type ImageService interface {
	// Get()
	Upload(*UploadArgs) (*UploadResults, error)
	// Approve()
	Archive(imageKey string) error
	// Restore(key string)
}

type UploadArgs struct {
	Transformations []storage.Transformation
	ImageKey        string
	Image           ImageBuffer
}

type UploadResults struct {
	TransformURLs map[string]string
}

// LouisService - implementation of ImageService
type LouisService struct {
	ctx *AppContext
}

func NewLouisService(ctx *AppContext) *LouisService {
	return &LouisService{
		ctx: ctx,
	}
}

type ImageBuffer = []byte

type ImageTransformer = func(image ImageBuffer, trans *storage.Transformation) (ImageBuffer, error)

// UploadImage - upload original image and it's transformations
func (svc *LouisService) Upload(args *UploadArgs) (*UploadResults, error) {

	var wg sync.WaitGroup
	var allTransformationsCount = len(args.Transformations) + 2 // 2 additional transformations: original and real
	var errors = make(chan error, allTransformationsCount)
	var transformURLs = utils.NewConcurrentMap()

	var ctx, cancelCtx = context.WithCancel(context.Background())
	defer cancelCtx()

	wg.Add(allTransformationsCount)

	var makeTransformation = func(localCtx context.Context, transformName string, transformer ImageTransformer, trans *storage.Transformation) {
		defer wg.Done()
		var transformedImage, err = transformer(args.Image, trans)
		if err != nil {
			errors <- err
			return
		}
		url, err := svc.ctx.Storage.UploadFileWithContext(
			localCtx,
			bytes.NewReader(transformedImage),
			makePath(transformName, args.ImageKey))
		transformURLs.Set(transformName, url)
		if err != nil {
			errors <- err
		}
	}

	go makeTransformation(ctx, RealTransformName,
		func(image ImageBuffer, trans *storage.Transformation) (ImageBuffer, error) {
			return image, nil
		},
		nil,
	)

	go makeTransformation(ctx, OriginalTransformName,
		func(image ImageBuffer, trans *storage.Transformation) (ImageBuffer, error) {
			return transformations.Compress(image, OriginalTransformQuality)
		},
		nil,
	)

	var fitTransformer = func(image ImageBuffer, tran *storage.Transformation) (ImageBuffer, error) {
		return transformations.Fit(image, tran.Width, tran.Quality)
	}

	var fillTransformer = func(image ImageBuffer, tran *storage.Transformation) (ImageBuffer, error) {
		return transformations.Fill(image, tran.Width, tran.Height, tran.Quality)
	}

	for _, tr := range args.Transformations {
		switch tr.Type {
		case "fit":
			go makeTransformation(ctx, tr.Name, fitTransformer, &tr)
			break
		case "fill":
			go makeTransformation(ctx, tr.Name, fillTransformer, &tr)
			break
		default:
			log.Printf("WARN: unknown transformation type: %v", tr.Type)
			wg.Done()
		}
	}
	select {
	case err := <-errors:
		cancelCtx()
		log.Printf("ERROR: on parallel transforms - %v", err)
		return nil, err

	case <-func() chan bool {
		wg.Wait()
		var channel = make(chan bool, 1)
		channel <- true
		return channel
	}():
		var terr error
		select {
		case kerr := <-errors:
			terr = kerr
			break
		default:
			terr = nil
		}
		return &UploadResults{
			TransformURLs: transformURLs.ToMap(),
		}, terr

	}
}

// Archive - delete all transforms except real
func (svc *LouisService) Archive(imageKey string) error {

	files, err := svc.ctx.Storage.ListFiles(imageKey)
	if err != nil {
		return err
	}
	var filteredFiles = make([]storage.ObjectID, 0)
	for _, file := range files {
		if strings.HasSuffix(*file.Key, RealTransformName+"."+ImageExtension) {
			continue
		}
		filteredFiles = append(filteredFiles, file)
	}

	return svc.ctx.Storage.DeleteFiles(filteredFiles)
}

// func (svc *LouisService) Restore(image *storage.Image) error {
// 	if !image.Deleted {
// 		return ImageNotArchivedError
//     }

//     var imageTransformToUse = OriginalTransformName
//     if image.WithRealCopy {
//         imageTransformToUse = RealTransformName
//     }

//     // svc.ctx.Storage.
// }

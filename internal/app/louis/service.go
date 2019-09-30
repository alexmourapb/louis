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

	realTransformation     = storage.Transformation{Type: "real", Name: RealTransformName}
	originalTransformation = storage.Transformation{Type: "original", Name: OriginalTransformName, Quality: OriginalTransformQuality}
)

// ImageService - interface of a service for uploading and transforming image
type ImageService interface {
	// Get()
	Upload(*UploadArgs) (map[string]string, error)
	// Approve()
	Archive(imageKey string) error
	Restore(key string) error
}

type UploadArgs struct {
	ImageKey string
	ImageID  int64
	Params   transformations.TransformParams
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

type imageTransformer = func(args transformations.TransformParams, trans *storage.Transformation) (ImageBuffer, error)

func (svc *LouisService) upload(transformationsList []storage.Transformation, args transformations.TransformParams, imageKey string) (map[string]string, error) {

	var wg sync.WaitGroup
	var allTransformationsCount = len(transformationsList)
	var errors = make(chan error, allTransformationsCount)
	var transformURLs = utils.NewConcurrentMap()

	var ctx, cancelCtx = context.WithCancel(context.Background())
	defer cancelCtx()

	wg.Add(allTransformationsCount)

	var makeTransformation = func(localCtx context.Context, transformName string, transformer imageTransformer, trans storage.Transformation) {
		defer wg.Done()
		var transformedImage, err = transformer(args, &trans)
		if err != nil {
			errors <- err
			return
		}
		url, err := svc.ctx.Storage.UploadFileWithContext(
			localCtx,
			bytes.NewReader(transformedImage),
			makePath(transformName, imageKey))
		transformURLs.Set(transformName, url)
		if err != nil {
			errors <- err
		}
	}

	var mappings = transformations.GetTransformsMappings()

	for _, tr := range transformationsList {

		var transformer, exists = mappings[tr.Type]
		if exists {
			go makeTransformation(ctx, tr.Name, transformer, tr)
		} else {
			log.Printf("WARN: unkown transform type %v", tr.Type)
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
		return transformURLs.ToMap(), terr

	}
}

// Upload - upload original image and it's transformations
func (svc *LouisService) Upload(args *UploadArgs) (map[string]string, error) {

	var newTransformationsList, err = svc.ctx.DB.GetTransformations(args.ImageID)
	if err != nil {
		return nil, err
	}

	newTransformationsList = append(newTransformationsList,
		realTransformation,
		originalTransformation,
	)

	transformUrls, err := svc.upload(newTransformationsList, args.Params, args.ImageKey)
	if err != nil {
		return nil, err
	}

	err = svc.ctx.DB.SetTransformsUploaded(args.ImageID)

	return transformUrls, err
}

// Archive - delete all transforms except real
func (svc *LouisService) Archive(imageKey string) error {

	files, err := svc.ctx.Storage.ListFiles(imageKey)
	if err != nil {
		return err
	}
	var objectsToDelete = make([]storage.ObjectID, 0)
	var originalKey storage.ObjectID
	var realExists = false
	for _, file := range files {
		if strings.HasSuffix(*file.Key, RealTransformName+"."+ImageExtension) {
			realExists = true
			continue
		}
		if strings.HasSuffix(*file.Key, OriginalTransformName+"."+ImageExtension) {
			originalKey = file
			continue
		}
		objectsToDelete = append(objectsToDelete, file)
	}
	if realExists && originalKey != nil {
		objectsToDelete = append(objectsToDelete, originalKey)
	}

	if len(objectsToDelete) > 0 {
		err = svc.ctx.Storage.DeleteFiles(objectsToDelete)
		if err != nil {
			return err
		}
	}

	return svc.ctx.DB.DeleteImage(imageKey)
}

func (svc *LouisService) Restore(imageKey string) error {

	var image, err = svc.ctx.DB.QueryImageByKey(imageKey)

	if err != nil {
		return err
	}

	if !image.Deleted {
		return ImageNotArchivedError
	}

	var imageTransformToUse = OriginalTransformName
	var additionalTransformation = realTransformation
	if image.WithRealCopy {
		imageTransformToUse = RealTransformName
		additionalTransformation = originalTransformation
	}

	baseImage, err := svc.ctx.Storage.GetObject(makePath(imageTransformToUse, image.Key))

	if err != nil {
		if err == storage.NoSuchKeyError {
			return ImageCanNotBeRestoredError
		}
		return err
	}

	transformationsList, err := svc.ctx.DB.GetTransformations(image.ID)

	if err != nil {
		return err
	}

	transformationsList = append(transformationsList, additionalTransformation)

	_, err = svc.upload(transformationsList, transformations.TransformParams{Image: baseImage}, imageKey)

	if err != nil {
		return err
	}

	err = svc.ctx.DB.SetTransformsUploaded(image.ID)

	if err != nil {
		return err
	}

	return svc.ctx.DB.SetImageRestored(imageKey)

}

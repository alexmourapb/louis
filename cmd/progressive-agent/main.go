package main

import (
	"bytes"
	"github.com/KazanExpress/louis/internal/app/louis"
	"github.com/KazanExpress/louis/internal/pkg/storage"
	"github.com/KazanExpress/louis/internal/pkg/transformations"
	"github.com/KazanExpress/louis/internal/pkg/utils"
	"log"
	"sync"
)

func main() {

	var err error
	var appCtx = new(louis.AppContext)
	appCtx.Config = utils.InitConfig()
	appCtx.DB, err = storage.Open(appCtx.Config)
	appCtx.ImageService = louis.NewLouisService(appCtx)

	if err != nil {
		log.Fatal(err)
	}

	appCtx.Storage, err = storage.InitS3Context(appCtx.Config)
	if err != nil {
		log.Fatalf("kek %v - ", err)
	}

	// TODO: add linter for go in vscode
	// TODO: configure federation grafana and prometheus
	const batchSize = 10
	var cursor = 0
	for {
		var wg sync.WaitGroup
		wg.Add(batchSize)

		var res = new([]storage.Image)
		var err = appCtx.DB.Where("Progressive = false and deleted = false").Offset(cursor).Limit(batchSize).Find(res).Error
		if err != nil {
			log.Fatal(err)
		}

		for i := range *res {
			var img = (*res)[i]
			log.Printf("proccesning %v", img.Key)
			go func(img storage.Image) {
				defer wg.Done()
				if !img.Progressive {
					var images, err = appCtx.Storage.ListFiles(img.Key)
					if err != nil {
						log.Printf("failed to get list of transformations - %s", err)
						return
					}
					for _, id := range images {
						var im, err = appCtx.Storage.GetObject(*id.Key)
						if err != nil {
							log.Printf("failed to get objeet - %s", err)
							return
						}

						transformed, err := transformations.MakeProgressive(im)
						if err != nil {
							log.Printf("failed to make image as proggressive - %s", err)
							return
						}

						_, err = appCtx.Storage.UploadFile(bytes.NewReader(transformed), *id.Key)
						if err != nil {
							log.Printf(" failed to upload transformed image - %s", err)
							return
						}
					}
					err = appCtx.DB.Update(img.Key, map[string]interface{}{
						"Progressive": true,
					})

					if err != nil {
						log.Printf("failed to set progressive - %s", err)
					}

				}
			}(img)
		}

		if len(*res) < batchSize {
			break
		}
		cursor += batchSize
		wg.Wait()
	}

	// var res = new([]storage.Image)
	// 	var err = appCtx.DB.Where("Progressive = false").Offset(cursor).Limit(batchSize).Find(res).Error
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	// appCtx.Storage.ListFiles()
}

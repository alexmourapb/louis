package main

// equalizer is example of louis agent implementation
// equalizer were needed to fix all images used for feedbacks
// some of them were uploaded with feedback but no transformation were applied to them
// others were uploaded even without tags

import (
	"encoding/json"
	"github.com/KazanExpress/louis/internal/app/louis"
	"github.com/KazanExpress/louis/internal/pkg/storage"
	"github.com/KazanExpress/louis/internal/pkg/utils"
	"github.com/lib/pq"
	"io/ioutil"
	"log"
)

type feedbackData struct {
	Images []string `json:"images"`
}

// Equal tells whether a and b contain the same elements.
// A nil argument is equivalent to an empty slice.
func equal(a, b pq.StringArray) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

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
		log.Fatal(err)
	}

	data, err := ioutil.ReadFile("data.json")
	if err != nil {
		log.Fatalf("failed to read from file: %v", err)
	}

	var feedback = new(feedbackData)
	err = json.Unmarshal(data, feedback)
	if err != nil {
		log.Fatalf("failed to parse json: %v", err)
	}

	var total = len(feedback.Images)
	var skipped = 0
	var failed = 0
	for i, imageKey := range feedback.Images {
		log.Printf("proccesing image %v/%v - %v", i, total, imageKey)
		image, err := appCtx.DB.QueryImageByKey(imageKey)
		if err != nil {
			failed++
			log.Printf("failed to get image by key: %v. skipping", imageKey)
			continue
		}

		if image.Deleted {
			skipped++
			log.Printf("image is archived - %v, skipping", imageKey)
			continue
		}

		if len(image.Tags) > 0 {
			log.Printf("there are some tags for %v", imageKey)

			var tag = image.Tags[0]

			if equal(image.Tags, image.AppliedTags) {
				log.Printf("tags are equal for %v", imageKey)
				if len(image.Tags) > 1 {
					skipped++
					log.Printf("image %v has %v tags. unexpected beh. skipping", imageKey, len(image.Tags))
					continue
				}
				if tag == "feedback" {
					skipped++
					log.Printf("image %v is already uploaded correctly. skipping", imageKey)
					// already ok
					continue
				} else {
					err = appCtx.ImageService.Archive(imageKey)
					if err != nil {
						failed++
						log.Printf("failed to archive image %v - %v", imageKey, err)
						continue
					}

					err = appCtx.DB.SetImageTags(imageKey, []string{"feedback"})
					if err != nil {
						failed++
						log.Printf("failed to set tags for image %v - %v", imageKey, err)
						continue
					}

					err = appCtx.ImageService.Restore(imageKey)
					if err != nil {
						failed++
						log.Printf("failed to restore image %v - %v", imageKey, err)
						continue
					}
				}

			} else {
				log.Printf("tags != appliedTags. restoring. %v", imageKey)
				err = appCtx.ImageService.Archive(imageKey)
				if err != nil {
					log.Printf("failed to archive image %v - %v", imageKey, err)
					failed++
					continue
				}

				if tag != "feedback" {
					err = appCtx.DB.SetImageTags(imageKey, []string{"feedback"})
					if err != nil {
						failed++
						log.Printf("failed to set tags for image %v - %v", imageKey, err)
						continue
					}
				}

				err = appCtx.ImageService.Restore(imageKey)
				if err != nil {
					failed++
					log.Printf("failed to restore image %v - %v", imageKey, err)
				}
			}
		} else {
			err = appCtx.DB.DeleteImage(imageKey)
			if err != nil {
				failed++
				log.Printf("failed to mark image as deleted %v - %v", imageKey, err)
				continue
			}

			err = appCtx.DB.SetImageTags(imageKey, []string{"feedback"})
			if err != nil {
				failed++
				log.Printf("failed to set tags for image %v - %v", imageKey, err)
				continue
			}

			err = appCtx.ImageService.Restore(imageKey)
			if err != nil {
				failed++
				log.Printf("failed to restore image %v - %v", imageKey, err)
				continue
			}
		}
	}

	log.Printf("from %v images %v skipped, %v failed", total, skipped, failed)
}

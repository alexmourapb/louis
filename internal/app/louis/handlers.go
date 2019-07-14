package louis

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/KazanExpress/louis/internal/pkg/utils"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/rs/xid"
	"io/ioutil"
	"log"
	"net/http"
	"runtime/debug"
)

var (
	NoKeysProvidedError = errors.New("no keys provided")
)

type requestArgs struct {
	image    ImageBuffer
	tags     []string
	imageKey string
}

type session struct {
	ctx    *AppContext
	userID int32
	args   *requestArgs
}

type sessionHandler = func(*session, http.ResponseWriter, *http.Request)

type handlerFunc = func(*session) (w http.ResponseWriter, req *http.Request)

func withSession(ctx *AppContext) func(sessionHandler) http.HandlerFunc {

	return func(handler sessionHandler) http.HandlerFunc {

		var s = &session{
			ctx: ctx,
		}
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			handler(s, w, req)
		})
	}

}

func (s *session) tryCreateImageRecord(w http.ResponseWriter, r *http.Request) (int64, bool) {

	if s.args.imageKey == "" {
		s.args.imageKey = xid.New().String()
	}

	imgID, err := s.ctx.DB.AddImage(s.args.imageKey, s.userID, s.args.tags...)

	if err != nil {

		if pger, ok := err.(*pq.Error); ok && pger.Constraint == "images_key_key" {
			failOnError(w, err, "image with such key is already exists", http.StatusBadRequest)
		} else {
			failOnError(w, err, "error on creating db record", http.StatusInternalServerError)
		}
		return imgID, false
	}
	return imgID, true
}

func parseClaimRequestBody(r *http.Request) (*imageData, error) {
	var body, err = ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var img = &imageData{}
	err = json.Unmarshal(body, &img)

	if err != nil {
		return nil, err
	}

	if img.Keys == nil || len(img.Keys) == 0 {
		return nil, NoKeysProvidedError
	}

	return img, nil
}

func handleClaim(s *session, w http.ResponseWriter, r *http.Request) {
	var img, err = parseClaimRequestBody(r)

	if err != nil {
		respondWithJSON(w, err.Error(), nil, http.StatusBadRequest)
		return
	}

	images, err := s.ctx.DB.GetImagesWithKeys(img.Keys)
	if failOnError(w, err, "failed to get images with keys", http.StatusBadRequest) {
		return
	}

	if len(*images) != len(img.Keys) {
		log.Printf("ERROR: only %v images found out of %v", len(*images), len(img.Keys))
		respondWithJSON(w, "not all images were found", nil, http.StatusBadRequest)
		return
	}

	for _, image := range *images {

		if image.Deleted {
			respondWithJSON(w, fmt.Sprintf("image with key = %v is deleted", image.Key), "", http.StatusBadRequest)
			log.Printf("INFO: trying to claim deleted image")
			return
		}
	}

	if failOnError(w, s.ctx.DB.SetClaimImages(img.Keys, s.userID), "failed to claim image", http.StatusInternalServerError) {
		return
	}

	log.Printf("INFO: images with keys [%v] claimed", img.Keys)
	respondWithJSON(w, "", "ok", 200)

}

func handleUploadWithClaim(s *session, w http.ResponseWriter, r *http.Request) {
	imgID, created := s.tryCreateImageRecord(w, r)

	if !created {
		// response in prev method
		return
	}

	transformsURLs, err := s.ctx.ImageService.Upload(&UploadArgs{
		ImageID:  imgID,
		ImageKey: s.args.imageKey,
		Image:    s.args.image,
	})

	if failOnError(w, err, "failed to upload transforms", http.StatusInternalServerError) {
		return
	}

	if failOnError(w, s.ctx.DB.SetImageURL(s.args.imageKey, s.userID, transformsURLs[OriginalTransformName]), "failed to set image url", http.StatusInternalServerError) {
		return
	}

	if failOnError(w, s.ctx.DB.SetClaimImage(s.args.imageKey, s.userID), "failed to claim image", http.StatusInternalServerError) {
		return
	}

	log.Printf("INFO: image with key %v and %v transforms uploaded and claimed", s.args.imageKey, len(transformsURLs))
	respondWithJSON(w, "", makeTransformsPayload(s.args.imageKey, transformsURLs), 200)
}

func handleUpload(s *session, w http.ResponseWriter, r *http.Request) {
	imgID, created := s.tryCreateImageRecord(w, r)

	if !created {
		// response in prev method
		return
	}
	transformsURLs, err := s.ctx.ImageService.Upload(&UploadArgs{
		ImageID:  imgID,
		ImageKey: s.args.imageKey,
		Image:    s.args.image,
	})

	if failOnError(w, err, "failed to upload transforms", http.StatusInternalServerError) {
		return
	}

	_, err = s.ctx.Enqueuer.EnqueueUniqueIn(CleanupTask, int64(s.ctx.Config.CleanUpDelay*60), map[string]interface{}{"key": s.args.imageKey})
	if err != nil {
		log.Printf("ERROR: failed to enqueue clean up task: %v", err)
	}

	if failOnError(w, s.ctx.DB.SetImageURL(s.args.imageKey, s.userID, transformsURLs[OriginalTransformName]), "failed to set image url", http.StatusInternalServerError) {
		return
	}

	log.Printf("INFO: image with key %v and %v transforms uploaded", s.args.imageKey, len(transformsURLs))
	respondWithJSON(w, "", makeTransformsPayload(s.args.imageKey, transformsURLs), 200)
}

func handleRestore(s *session, w http.ResponseWriter, r *http.Request) {
	var vars = mux.Vars(r)
	var imageKey, passed = vars["imageKey"]

	if !passed {
		respondWithJSON(w, "imageKey is not passed", nil, http.StatusBadRequest)
		return
	}

	var err = s.ctx.ImageService.Restore(imageKey)
	if err != nil {
		if err == ImageCanNotBeRestoredError {
			respondWithJSON(w, err.Error(), nil, http.StatusPreconditionFailed)
		} else {
			respondWithJSON(w, "failed to restore image: "+err.Error(), nil, http.StatusInternalServerError)
		}
		return
	}

	respondWithJSON(w, "", "ok", http.StatusAccepted)
}

// simple handlers without need of session

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, here is your dashboard")
}

func handleFree(w http.ResponseWriter, req *http.Request) {
	debug.FreeOSMemory()
	w.WriteHeader(200)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	health := utils.GetHealthStats()
	body, _ := json.Marshal(health)
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

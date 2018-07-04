package queue

import (
	"bytes"
	"encoding/json"
	"github.com/KazanExpress/louis/internal/pkg/storage"
	"github.com/KazanExpress/louis/internal/pkg/transformations"
	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/backends/result"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/tasks"
	"io"
	"net/http"
	"path"
)

const QueueName = "transformations_queue"

type TransformData struct {
	Width       int
	Height      int
	Quality     int
	ImageKey    string
	ImageURL    string
	Name        string
	S3Directory string
}

func NewTransformData(img *storage.Image, tran *storage.Transformation) TransformData {
	return TransformData{
		Width:       int(tran.Width),
		Height:      int(tran.Height),
		Quality:     int(tran.Quality),
		ImageKey:    img.Key,
		ImageURL:    img.URL,
		S3Directory: tran.Name,
	}
}

type JobQueue interface {
	PublishFitTransform(td TransformData) (*result.AsyncResult, error)
}

type MachineryQueue struct {
	MachineryServer *machinery.Server
}

func NewMachineryQueue(redisURL string) (*MachineryQueue, error) {

	var cnf = &config.Config{
		Broker:        redisURL,
		DefaultQueue:  QueueName,
		ResultBackend: redisURL,
	}

	server, err := machinery.NewServer(cnf)
	err = server.RegisterTask("fit", fitTransform)
	worker := server.NewWorker("transforms", 10)
	go worker.Launch()
	return &MachineryQueue{server}, err
}

func downloadFile(url string, w io.Writer) error {

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func fitTransform(byteJson string) error {
	bs := []byte(byteJson)
	var dt TransformData
	err := json.Unmarshal(bs, &dt)
	if err != nil {
		return err
	}

	var buffer = bytes.Buffer{}
	err = downloadFile(dt.ImageURL, &buffer)
	if err != nil {
		return err
	}

	result, err := transformations.Fit(buffer.Bytes(), dt.Width, dt.Quality)
	if err != nil {
		return err
	}

	_, err = storage.UploadFile(bytes.NewReader(result), path.Join(dt.S3Directory, dt.ImageKey+".jpg"))

	return err
}

func (mq *MachineryQueue) PublishFitTransform(td TransformData) (*result.AsyncResult, error) {

	jsonBytes, err := json.Marshal(td)
	if err != nil {
		return nil, err
	}
	task := &tasks.Signature{
		Name: "fit",
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: string(jsonBytes),
			},
		},
	}
	return mq.MachineryServer.SendTask(task)
}

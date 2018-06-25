package queue

import (
	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/tasks"
)

const QueueName = "transformations_queue"

type JobQueue interface {
	Publish(data []byte) error
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
	return &MachineryQueue{server}, err
}

func (mq *MachineryQueue) Publish(data []byte) error {
	signature := &tasks.Signature{
		Name: "transform",
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: string(data),
			},
		},
	}

	_, err := mq.MachineryServer.SendTask(signature)
	return err
}

package messenger

import (
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
)

func Client(seeds []string, topics []string) (*kgo.Client, error) {
	cl, err := kgo.NewClient(kgo.SeedBrokers(seeds...), kgo.ConsumerGroup("image-processing-worker"), kgo.ConsumeTopics(topics...))

	if err != nil {
		fmt.Printf("There's an error on start client" + err.Error())
		return nil, err
	}

	return cl, nil
}

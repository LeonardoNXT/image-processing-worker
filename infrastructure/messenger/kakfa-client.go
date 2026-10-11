package messenger

import (
	"github.com/twmb/franz-go/pkg/kgo"
	"time"
)

func Client(seeds, topics []string) (*kgo.Client, error) {
	return kgo.NewClient(kgo.SeedBrokers(seeds...), kgo.ConsumerGroup("image-processing-worker"), kgo.ConsumeTopics(topics...),
		kgo.DisableAutoCommit(), kgo.BlockRebalanceOnPoll(), kgo.RebalanceTimeout(10*time.Minute))
}

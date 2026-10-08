package messenger

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
)

func Consumer(client *kgo.Client, ctx context.Context, fn func(record *kgo.Record)) {
	for {
		fetches := client.PollFetches(ctx)
		fetches.EachRecord(func(record *kgo.Record) {
			fn(record)
		})
	}
}

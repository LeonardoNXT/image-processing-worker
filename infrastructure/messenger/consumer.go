package messenger

import (
	"context"
	"fmt"
	"github.com/twmb/franz-go/pkg/kgo"
)

func Consumer(client *kgo.Client, ctx context.Context, fn func(context.Context, *kgo.Record) error) error {
	defer client.AllowRebalance()
	for {
		fetches := client.PollRecords(ctx, 1)
		if err := ctx.Err(); err != nil {
			return err
		}
		if errs := fetches.Errors(); len(errs) > 0 {
			return fmt.Errorf("Kafka fetch: %w", errs[0].Err)
		}
		for _, record := range fetches.Records() {
			if err := fn(ctx, record); err != nil {
				return fmt.Errorf("topic %s partition %d offset %d: %w", record.Topic, record.Partition, record.Offset, err)
			}
			if err := client.CommitRecords(ctx, record); err != nil {
				return err
			}
		}
		client.AllowRebalance()
	}
}

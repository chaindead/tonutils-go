package ton

import (
	"context"

	"github.com/chaindead/tonutils-go/tl"
	"golang.org/x/time/rate"
)

type RateClient struct {
	rateLimiter *rate.Limiter
	LiteClient
}

func Limit(c LiteClient, r rate.Limit, b int) *RateClient {
	rateLimiter := rate.NewLimiter(r, b)

	return &RateClient{
		rateLimiter: rateLimiter,
		LiteClient:  c,
	}
}

func (c *RateClient) QueryLiteserver(ctx context.Context, request tl.Serializable, result tl.Serializable) error {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	return c.LiteClient.QueryLiteserver(ctx, request, result)
}

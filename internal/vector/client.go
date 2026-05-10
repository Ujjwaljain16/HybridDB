package vector

import "github.com/Ujjwaljain16/hybriddb/internal/common"

type EmbeddingClient interface {
	Embed(text string) ([]float32, error)
}

type HTTPEmbeddingClient struct{}

func NewHTTPEmbeddingClient(baseURL string) *HTTPEmbeddingClient {
	panic("not implemented")
}

func (c *HTTPEmbeddingClient) Embed(text string) ([]float32, error) {
	return nil, common.ErrNotImplemented
}

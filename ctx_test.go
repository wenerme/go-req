package req

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	ctx := context.Background()
	assert.Nil(t, FromContext(ctx))
	ctx = NewContext(ctx, nil)
	assert.Nil(t, FromContext(ctx))
	assert.Equal(t, "req.contextKey(Request)", RequestContextKey.String())
}

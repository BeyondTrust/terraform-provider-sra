package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProductSetting(t *testing.T) {
	// t.Parallel()

	SetProductIsRS(true)
	assert.False(t, IsPRA())
	assert.True(t, IsRS())

	SetProductIsRS(false)
	assert.True(t, IsPRA())
	assert.False(t, IsRS())
}

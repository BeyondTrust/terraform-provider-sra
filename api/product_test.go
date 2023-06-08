package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProductDefault(t *testing.T) {
	t.Parallel()

	assert.True(t, IsPRA())
	assert.False(t, IsRS())
}

func TestProductSetting(t *testing.T) {
	t.Parallel()

	SetProductIsRS(true)
	assert.False(t, IsPRA())
	assert.True(t, IsRS())

	SetProductIsRS(false)
	assert.True(t, IsPRA())
	assert.False(t, IsRS())
}

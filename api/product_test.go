package api

import (
	"context"
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

func TestProductName(t *testing.T) {
	// t.Parallel()

	SetProductIsRS(true)
	assert.Equal(t, ProductRS, ProductName())

	SetProductIsRS(false)
	assert.Equal(t, ProductPRA, ProductName())
}

type noInterface struct{}

type allProducts struct{}

func (p allProducts) AllowPRA() bool {
	return true
}

func (p allProducts) AllowRS() bool {
	return true
}

type productFeature struct {
	product string
}

func (p productFeature) AllowPRA() bool {
	return p.product == ProductPRA
}

func (p productFeature) AllowRS() bool {
	return p.product == ProductRS
}

func TestProductRestriction(t *testing.T) {
	p := &productFeature{ProductPRA}
	assert.True(t, p.AllowPRA())
	assert.False(t, p.AllowRS())

	r := &productFeature{ProductRS}
	assert.False(t, r.AllowPRA())
	assert.True(t, r.AllowRS())

	n := &noInterface{}

	a := &allProducts{}
	assert.True(t, a.AllowPRA())
	assert.True(t, a.AllowRS())

	ctx := context.Background()

	SetProductIsRS(true)
	assert.False(t, IsProductAllowed(ctx, p))
	assert.True(t, IsProductAllowed(ctx, r))
	assert.True(t, IsProductAllowed(ctx, n))
	assert.True(t, IsProductAllowed(ctx, a))

	SetProductIsRS(false)
	assert.True(t, IsProductAllowed(ctx, p))
	assert.False(t, IsProductAllowed(ctx, r))
	assert.True(t, IsProductAllowed(ctx, n))
	assert.True(t, IsProductAllowed(ctx, a))
}

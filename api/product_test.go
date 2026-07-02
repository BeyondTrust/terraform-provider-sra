package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientProductMethods(t *testing.T) {
	t.Parallel()

	c := &APIClient{Product: ProductRS}
	assert.True(t, c.IsRS())
	assert.False(t, c.IsPRA())
	assert.Equal(t, ProductRS, c.ProductName())

	c.Product = ProductPRA
	assert.False(t, c.IsRS())
	assert.True(t, c.IsPRA())
	assert.Equal(t, ProductPRA, c.ProductName())
}

func TestConcurrentProductSafety(t *testing.T) {
	t.Parallel()

	rs := &APIClient{Product: ProductRS}
	pra := &APIClient{Product: ProductPRA}

	assert.True(t, rs.IsRS())
	assert.True(t, pra.IsPRA())
	assert.False(t, rs.IsPRA())
	assert.False(t, pra.IsRS())
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

	assert.False(t, IsProductAllowed(ctx, p, ProductRS))
	assert.True(t, IsProductAllowed(ctx, r, ProductRS))
	assert.True(t, IsProductAllowed(ctx, n, ProductRS))
	assert.True(t, IsProductAllowed(ctx, a, ProductRS))

	assert.True(t, IsProductAllowed(ctx, p, ProductPRA))
	assert.False(t, IsProductAllowed(ctx, r, ProductPRA))
	assert.True(t, IsProductAllowed(ctx, n, ProductPRA))
	assert.True(t, IsProductAllowed(ctx, a, ProductPRA))
}

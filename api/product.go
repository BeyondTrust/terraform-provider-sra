package api

type CtxKey string

const ProductKey CtxKey = "product"
const ProductRS = "rs"
const ProductPRA = "pra"

var (
	product = ProductPRA
)

func SetProductIsRS(isRS bool) {
	if isRS {
		product = ProductRS
	} else {
		product = ProductPRA
	}
}
func IsRS() bool {
	return product == ProductRS
}
func IsPRA() bool {
	return product == ProductPRA
}

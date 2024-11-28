package products

import (
	"context"
	"shop/services/store"
	"strconv"
)

func Serve(ctx context.Context, index, limit string) ([]store.Product, error) {
	i, err := strconv.Atoi(index)
	if err != nil {
		i = 0
		err = nil
	}
	l, err := strconv.Atoi(limit)
	if err != nil {
		l = 30
		err = nil
	}
	products, err := store.Pub.GetProducts(context.Background(), i, l)
	if err != nil {
		return []store.Product{}, err
	}
	return products, nil
}

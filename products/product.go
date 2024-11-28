package products

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"net/http"
	"shop/handlers/render"
	"shop/services/auth"
	"shop/services/store"
	viewProducts "shop/views/products"
)

func SinglePage(w http.ResponseWriter, r *http.Request) error {
	user, _ := auth.GetUserSession(r)
	sku := store.Sku(chi.URLParam(r, "sku"))
	productName := chi.URLParam(r, "name")
	if len(sku) <= 0 {
		http.Redirect(w, r, "/oops", http.StatusSeeOther)
		return errors.New("product id not present")
	}
	if len(productName) <= 0 {
		http.Redirect(w, r, "/oops", http.StatusSeeOther)
		return errors.New("product name not present")
	}
	productId, err := sku.ProductId()
	if err != nil {
		http.Redirect(w, r, "/oops", http.StatusSeeOther)
		return err
	}
	ctx := context.Background()
	ctx = context.WithValue(ctx, "productId", productId)
	ctx = context.WithValue(ctx, "sku", sku)
	product, err := store.Pub.GetProduct(ctx)
	if err != nil {
		http.Redirect(w, r, "/oops", http.StatusSeeOther)
		return errors.New("product not present")
	}
	cartItems, err := store.Pub.CartCountItems(ctx, user.Id)
	if err != nil {
		http.Redirect(w, r, "/oops", http.StatusSeeOther)
		return err
	}
	return render.Template(w, r, viewProducts.SingleProduct(user, product, cartItems))
}

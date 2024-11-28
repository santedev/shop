package checkout

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"shop/handlers"
	"shop/handlers/render"
	"shop/services/auth"
	"shop/services/store"
	"shop/views/checkout"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"strconv"
)

func PaymentPageCart(w http.ResponseWriter, r *http.Request) error {
	user, err := auth.GetUserSession(r)
	if err != nil {
		handlers.Redirect(w, r, "/login")
		return err
	}
	cartItems, err := store.Pub.GetCart(context.Background(), user.Id)
	if err != pgx.ErrNoRows && err != nil {
		handlers.Redirect(w, r, "/oops")
		return err
	}
	countCart, cartTotal, err := store.Pub.CartCountItemsWithTotal(context.Background(), user.Id)
	if err != nil {
		handlers.Redirect(w, r, "/oops")
		return err
	}
	return render.Template(w, r, checkout.Index(user, countCart, cartTotal, true, cartItems...))
}

func PaymentPageBuyNow(w http.ResponseWriter, r *http.Request) error {
	if ok := handlers.HtmxRedirect(w, r); ok {
		return nil
	}
	user, err := auth.GetUserSession(r)
	if err != nil {
		handlers.Redirect(w, r, "/login")
		return err
	}
	sku := store.Sku(chi.URLParam(r, "sku"))
	if len(sku) <= 0 {
		handlers.Redirect(w, r, "/oops")
		return errors.New("sku not present in the params")
	}

	params := r.URL.Query()
	quantityProd, err := strconv.Atoi(params.Get("quantity"))
	if err != nil {
		handlers.Redirect(w, r, "/oops")
		return fmt.Errorf("quantity of product within url params encounter a error: %w", err)
	}
	if quantityProd <= 0 {
		handlers.Redirect(w, r, "/oops")
		return errors.New("quantity for products is below or equals zero")
	}
	productId, err := sku.ProductId()
	if err != nil {
		handlers.Redirect(w, r, "/oops")
		return err
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "productId", productId)
	ctx = context.WithValue(ctx, "sku", sku)
	product, err := store.Pub.GetProduct(ctx)
	if err != nil {
		handlers.Redirect(w, r, "/oops")
		return err
	}
	item := store.Items{
		Id:               product.Id,
		Name:             product.Name,
		Description:      product.Description,
		ShortDescription: product.ShortDescription,
		Variants:         product.Variants,
		Images:           product.Images,
		Quantity:         quantityProd,
	}
	err = item.SetComb(sku, product.Combinations)
	if err != nil {
		handlers.Redirect(w, r, "/oops")
		return err
	}
	countCart, err := store.Pub.CartCountItems(context.Background(), user.Id)
	if err != nil {
		handlers.Redirect(w, r, "/oops")
		return err
	}
	cartTotal, err := store.TotalItems(item.Comb.Currency, item)
	if err != nil {
		handlers.Redirect(w, r, "/oops")
		return err
	}
	return render.Template(w, r, checkout.Index(user, countCart, cartTotal, false, item))
}

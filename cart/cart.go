package cart

import (
	"context"
	"errors"
	"net/http"
	"shop/handlers"
	"shop/handlers/render"
	"shop/services/auth"
	"shop/services/store"
	viewCart "shop/views/cart"
	"shop/views/component"
	"strconv"
)

func Page(w http.ResponseWriter, r *http.Request) error {
	user, err := auth.GetUserSession(r)
	if err != nil {
		handlers.Redirect(w, r, "/login")
		return errors.New("needs user for the cart page")
	}
	cartItems, err := store.Pub.GetCart(context.Background(), user.Id)
	if err != nil {
		handlers.Redirect(w, r, "/oops")
		return err
	}
	countCart, cartBalance, err := store.Pub.CartCountItemsWithTotal(context.Background(), user.Id)
	if err != nil {
		handlers.Redirect(w, r, "/oops")
		return err
	}
	return render.Template(w, r, viewCart.Index(user, countCart, cartBalance, cartItems))
}

func AddToCart(w http.ResponseWriter, r *http.Request) error {
	user, err := auth.GetUserSession(r)
	if err != nil {
		handlers.Redirect(w, r, "/login")
		return errors.New("needs user for adding to the cart")
	}
	params := r.URL.Query()
	sku := store.Sku(params.Get("sku"))
	if len(sku) <= 0 {
		render.Template(w, r, component.ErrorModalCart("An error occurred", errors.New("App error")))
		return errors.New("need sku which is not present in params")
	}
	quantity, err := strconv.Atoi(params.Get("quantity"))
	if err != nil {
		render.Template(w, r, component.ErrorModalCart("An error occurred", errors.New("App error")))
		return err
	}
	cartItemQuantity, err := store.Pub.AddToCart(context.Background(), user.Id, sku, quantity)
	if err == store.ErrNoStock {
		render.Template(w, r, component.ErrorModalCart("Not enough stock", errors.New("Can't add more of this product to the cart, because there is not enough stock")))
		return err
	}
	if err != nil {
		render.Template(w, r, component.ErrorModalCart("An error occurred", errors.New("App error")))
		return err
	}
	return render.Template(w, r, component.AddToCart(cartItemQuantity))
}

func UpdateProductCount(w http.ResponseWriter, r *http.Request) error {
	user, err := auth.GetUserSession(r)
	if err != nil {
		handlers.Redirect(w, r, "/login")
		return errors.New("needs user for adding to the cart")
	}
	params := r.URL.Query()
	sku := store.Sku(params.Get("sku"))
	if len(sku) <= 0 {
		handlers.Redirect(w, r, "/oops")
		return errors.New("need sku which is not present in params")
	}
	quantity, err := strconv.Atoi(params.Get("quantity"))
	if err != nil {
		render.Template(w, r, component.ErrorModalCart("An error occurred", errors.New("App error")))
		return err
	}
	cartCount, err := store.Pub.UpdateCartCount(context.Background(), user.Id, sku, quantity)
	if err != nil {
		handlers.Redirect(w, r, "/oops")
		return err
	}
	return render.Template(w, r, component.UpdateCartPage(
		sku,
		cartCount.CartCount,
		cartCount.ProductCount,
		cartCount.CartBalance,
		cartCount.ProductBalance,
	))
}

func RemoveProduct(w http.ResponseWriter, r *http.Request) error {
	user, err := auth.GetUserSession(r)
	if err != nil {
		handlers.Redirect(w, r, "/login")
		return errors.New("needs user for adding to the cart")
	}
	params := r.URL.Query()
	sku := store.Sku(params.Get("sku"))
	if len(sku) <= 0 {
		render.Template(w, r, component.ErrorModalCart("An error occurred", errors.New("App error")))
		return errors.New("need sku which is not present in params")
	}
	cartCount, err := store.Pub.RemoveProductFromCart(context.Background(), user.Id, sku)
	if err != nil {
		render.Template(w, r, component.ErrorModalCart("An error occurred", errors.New("App error")))
		return err
	}
	return render.Template(w, r, component.CartPageRemoveProduct(
		cartCount.CartCount,
		cartCount.CartBalance,
	))
}

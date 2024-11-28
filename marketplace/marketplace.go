package marketplace

import (
	"context"
	"net/http"
	"shop/handlers/render"
	"shop/products"
	"shop/services/auth"
	"shop/services/store"
	"shop/views/home"
)

func Home(w http.ResponseWriter, r *http.Request) error {
	params := r.URL.Query()
	index := params.Get("index")
	limit := params.Get("limit")
	products, err := products.Serve(context.Background(), limit, index)
	if err != nil {
		http.Redirect(w, r, "/oops", http.StatusSeeOther)
		return err
	}
	user, err := auth.GetUserSession(r)
	if err != nil {
		return render.Template(w, r, home.Index(store.User{}, products, 0))
	}
	cartCountItems, err := store.Pub.CartCountItems(context.Background(), user.Id)
	if err != nil {
		http.Redirect(w, r, "/oops", http.StatusSeeOther)
		return err
	}
	return render.Template(w, r, home.Index(user, products, cartCountItems))
}

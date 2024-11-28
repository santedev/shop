package main

import (
	"encoding/gob"
	"log"
	"log/slog"
	"net/http"

	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"math/rand/v2"
	"shop/cart"
	"shop/checkout"
	"shop/config"
	"shop/gateaways/paypal"
	"shop/handlers"
	m "shop/handlers/middleware"
	"shop/marketplace"
	"shop/products"
	"shop/services/auth"
	"shop/services/store"
	"strconv"
	"time"
)

func randomFloat(limit, truncate int) float64 {
	r := rand.Float64() * float64(limit)
	tr := fmt.Sprintf("%.*f", truncate, r)
	f, _ := strconv.ParseFloat(tr, 64)
	return f
}

func product(l, t int) store.Product {
	return store.Product{
		Id:               0,
		Name:             "T-Shirt",
		Description:      "A comfortable cotton t-shirt.",
		ShortDescription: "Soft and stylish t-shirt.",
		Images:           []string{"image1.jpg", "image2.jpg"},
		Variants: []store.Variant{
			{
				Label: "size",
				Options: []store.Option{
					{Id: 1, VariantId: 1, Option: "Small"},
					{Id: 2, VariantId: 1, Option: "Medium"},
					{Id: 3, VariantId: 1, Option: "Large"},
				},
			},
			{
				Label: "color",
				Options: []store.Option{
					{Id: 4, VariantId: 2, Option: "Red"},
					{Id: 5, VariantId: 2, Option: "Blue"},
					{Id: 6, VariantId: 2, Option: "Green"},
				},
			},
		},
		Combinations: []store.Combination{
			{
				Price:    randomFloat(l, t),
				Currency: store.USD,
				Stock:    100,
				Options: []store.Option{
					{Id: 1, VariantId: 1, Option: "Small"},
					{Id: 4, VariantId: 2, Option: "Red"},
				},
			},
			{
				Price:    randomFloat(l, t),
				Currency: store.USD,
				Stock:    50,
				Options: []store.Option{
					{Id: 2, VariantId: 1, Option: "Medium"},
					{Id: 5, VariantId: 2, Option: "Blue"},
				},
			},
			{
				Price:    randomFloat(l, t),
				Currency: store.USD,
				Stock:    30,
				Options: []store.Option{
					{Id: 3, VariantId: 1, Option: "Large"},
					{Id: 6, VariantId: 2, Option: "Green"},
				},
			},
		},
		CreatedAt: time.Now(),
	}
}
func main() {
	err := initPkgs()
	if err != nil {
		log.Fatalf("init failed with error: %v\n", err)
	}
	r := chi.NewRouter()
	corsConfig := newCors()

	r.Use(corsConfig.Handler)
	r.Use(middleware.Logger)
	r.Handle("/*", public())

	r.Get("/admin/login", m.LogErr(admin.Login))

	r.Get("/", m.LogErr(marketplace.Home))
	r.Get("/{name}/p/{sku}", m.LogErr(products.SinglePage))

	r.Get("/cart", m.LogErr(cart.Page))
	r.Get("/cart/add-to-cart", m.LogErr(cart.AddToCart))
	r.Get("/cart/count/update-product", m.LogErr(cart.UpdateProductCount))
	r.Delete("/cart/count/remove-product", m.LogErr(cart.RemoveProduct))

	r.Get("/checkout/buy", m.LogErr(checkout.PaymentPageCart))
	r.Get("/checkout/buynow/{sku}", m.LogErr(checkout.PaymentPageBuyNow))
	paypalEndpoints(r)

	googleOAuthEndpoints(r)
	r.Get("/login", m.LogErr(handlers.LoginPage))
	r.Get("/auth/logout", m.LogErrAndRedirect(handlers.AuthLogout, "/"))
	listenAddr := ":" + config.Envs.Port

	listenNServe(serverSettings(listenAddr, r), listenAddr)
	cleanUp()
}

func listenNServe(srv *http.Server, listenAddr string) {
	slog.Info("HTTP server started", "address", config.Envs.PublicHost+listenAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %s\n", err)
	}
}

func recoverFromPanic() {
	if r := recover(); r != nil {
		log.Printf("Recovered from panic: %v", r)
		cleanUp()
		log.Println("Server exited due to panic")
	}
}

func paypalEndpoints(r *chi.Mux) {
	r.Post("/create-paypal-order", m.LogErr(paypal.CreatePaypalOrder))
	r.Post("/capture-paypal-order/{order-id}", m.LogErr(paypal.CaptureOrder))
}

func googleOAuthEndpoints(r *chi.Mux) {
	r.Post("/auth/google/idtoken", m.LogErrAndRedirect(handlers.HandleCredentialsGoogle, "/login"))
}

func newCors() *cors.Cors {
	return cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, //CHANGE THIS IN PRODUCTION
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           3600,
	})
}

func newCookieStore() {
	auth.NewUserCookieStore(auth.SessionOptions{
		CookiesKey: config.Envs.CookiesAuthSecret,
		Path:       config.Envs.CookiesPath,
		MaxAge:     config.Envs.CookiesAuthAgeInSeconds,
		HttpOnly:   config.Envs.CookiesAuthIsHttpOnly,
		Secure:     config.Envs.CookiesAuthIsSecure,
		SameSite:   http.SameSiteStrictMode,
	})
}

func cleanUp() {
	store.Pub.Close()
}

func initPkgs() error {
	err := config.LoadEnv()
	if err != nil {
		return err
	}
	if !config.Envs.Production {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}
	gob.Register(store.User{})
	gob.Register(store.Admin{})
	newCookieStore()
	err = paypal.Init()
	if err != nil {
		return err
	}
	s := store.PostgresStore{}
	err = s.Init()
	if err != nil {
		return err
	}
	return nil
}

func serverSettings(listenAddr string, r *chi.Mux) *http.Server {
	return &http.Server{
		Addr:         listenAddr,
		Handler:      r,
		ReadTimeout:  45 * time.Second,
		WriteTimeout: 45 * time.Second,
	}
}

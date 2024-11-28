package paypal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"io"
	"log"
	"net/http"
	"shop/gateaways"
	"shop/handlers"
	"shop/handlers/render"
	"shop/services/auth"
	"shop/services/store"
	"shop/views/checkout"
	"strconv"
	"strings"
)

func CreatePaypalOrder(w http.ResponseWriter, r *http.Request) error {
	user, err := auth.GetUserSession(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return err
	}
	accessToken, err := getAcessToken()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	var cart store.Order
	err = json.NewDecoder(r.Body).Decode(&cart)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return err
	}
	if err := cart.Currency.Valid(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return err
	}
	if cart.FromCart {
		updatedCart, err := store.Pub.CheckStockFromItemsAndUpdateCart(context.Background(), user.Id, cart.Products)
		if !updatedCart && err != nil {
			return err
		}
		if updatedCart {
			if err != nil {
				log.Println(err)
			}
			cartItems, err := store.Pub.GetCart(context.Background(), user.Id)
			if err != nil {
				return err
			}
			countCart, cartTotal, err := store.Pub.CartCountItemsWithTotal(context.Background(), user.Id)
			if err != nil {
				return err
			}
			err = render.Template(w, r, checkout.UpdateItems(cartItems, cart.FromCart, countCart, cartTotal))
			if err != nil {
				return err
			}
		}
	} else {
		err := store.Pub.CheckStockFromItems(context.Background(), cart.Products)
		if err != nil {
			handlers.Redirect(w, r, "/oops")
			return err
		}
	}
	total, err := store.Pub.TotalItems(context.Background(), cart.Products)
	if err != nil {
		return err
	}
	referenceId := uuid.New().String()
	order := CreateOrderRequest{
		PurchaseUnits: []PurchaseUnit{
			{
				Amount: Amount{
					CurrencyCode: string(cart.Currency),
					Value:        fmt.Sprintf("%.*f", cart.Currency.Truncate(), total),
				},
				ReferenceID: referenceId,
			},
		},
		Intent: "CAPTURE",
		PaymentSource: PaymentSource{
			Paypal: PayPalExp{
				ExperienceContext: ExperienceContext{
					PaymentMethodPreference: "IMMEDIATE_PAYMENT_REQUIRED",
					PaymentMethodSelected:   "PAYPAL",
					BrandName:               "EXAMPLE INC",
					Locale:                  "en-US",
					LandingPage:             "LOGIN",
					ShippingPreference:      "GET_FROM_FILE",
					UserAction:              "PAY_NOW",
				},
			},
		},
	}
	orderPayload, err := json.Marshal(order)
	if err != nil {
		return err
	}
	reqBody := bytes.NewReader(orderPayload)

	req, err := http.NewRequest("POST", paypalOrderURL(), reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	authHeaderValue := "Bearer " + accessToken.Token
	req.Header.Set("Authorization", authHeaderValue)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bod, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Authorization", authHeaderValue)
	w.Header().Set("Reference-Id", referenceId)
	_, err = w.Write(bod)
	if err != nil {
		return err
	}
	return nil
}

func CaptureOrder(w http.ResponseWriter, r *http.Request) error {
	user, err := auth.GetUserSession(r)
	if err != nil {
		return err
	}
	var cart store.Order
	err = json.NewDecoder(r.Body).Decode(&cart)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return err
	}
	if err := cart.Currency.Valid(); err != nil {
		return err
	}
	orderId := chi.URLParam(r, "order-id")
	if len(orderId) <= 0 {
		return errors.New("order id from params cant be len zero or below")
	}
	accessToken, err := getAcessToken()
	if err != nil {
		return err
	}
	referenceId := r.Header.Get("Reference-Id")
	if len(referenceId) <= 0 {
		return errors.New("reference id was not found in headers")
	}
	err = checkAuthToken(r, accessToken)
	if err != nil {
		return err
	}
	transaction, bod, err := fetchCaptureOrder(orderId, accessToken)
	if err != nil {
		return err
	}
	if transaction.Status != COMPLETED {
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(bod)
		if err != nil {
			return fmt.Errorf("transaction went wrong, STATUS:%s. err: %w", transaction.Status, err)
		}
		return errors.New(fmt.Sprintf("transaction went wrong, STATUS:%s", transaction.Status))
	}
	go func(user store.User, cart store.Order) {
		if cart.FromCart {
			err := store.Pub.EmptyingCart(context.Background(), user.Id)
			if err != nil {
				log.Println(err)
			}
		}
		err = store.Pub.UpdateStock(context.Background(), cart.Products)
		if err != nil {
			log.Println(err)
		}
	}(user, cart)
	captureIds, err := getCaptureIds(transaction.PurchaseUnits)
	if err != nil {
		panic(fmt.Errorf("something went wrong with the transaction err:%w", err))
	}
	totals, err := getTotalNetAmount(transaction.PurchaseUnits)
	if err != nil {
		return err
	}
	total, err := store.TotalItemsFloat(cart.Currency, totals...)
	if err != nil {
		return err
	}
	id, err := store.Pub.MakeOrder(
		context.Background(),
		gateaways.Paypal,
		user.Id,
		cart.Products,
		total,
		cart.Currency,
		transaction.ID,
		transaction.Payer.Name.GivenName,
		transaction.Payer.EmailAddress,
		transaction.Payer.PayerID,
		[]string{referenceId},
		captureIds)
	if err != nil {
		log.Println(err)
	}
	//func DeliverOrder
	go func(orderId int, user store.User, cart store.Order, transaction Transaction) {
	}(id, user, cart, transaction)

	w.Header().Set("Content-Type", "application/json")
	handlers.Redirect(w, r, "/thanks")
	_, err = w.Write(bod)
	if err != nil {
		return err
	}
	return nil

}

func fetchCaptureOrder(orderId string, accessToken *accessToken) (Transaction, []byte, error) {
	req, err := http.NewRequest("POST", captureOrderURL(orderId), bytes.NewReader([]byte{}))
	if err != nil {
		return Transaction{}, []byte{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	authHeaderValue := "Bearer " + accessToken.Token
	req.Header.Set("Authorization", authHeaderValue)
	resp, err := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	bod, err := io.ReadAll(resp.Body)
	if err != nil {
		return Transaction{}, []byte{}, err
	}
	defer resp.Body.Close()
	var transaction Transaction
	err = json.Unmarshal(bod, &transaction)
	if err != nil {
		return Transaction{}, []byte{}, err
	}
	return transaction, bod, nil
}

func checkAuthToken(r *http.Request, accessToken *accessToken) error {
	hAccessToken := r.Header.Get("Authorization")
	tAccessTArr := strings.SplitN(hAccessToken, "Bearer ", 2)
	if len(tAccessTArr) < 2 {
		return errors.New("header has no access token, strings array len below 2")
	}
	accessTokenH := tAccessTArr[1]
	if accessToken.Token != accessTokenH {
		return errors.New("access token from header doesnt match got access token")
	}
	return nil
}

func getTotalNetAmount(purchaseUnits []CapturePurchaseUnit) ([]float64, error) {
	if len(purchaseUnits) <= 0 {
		return []float64{}, errors.New("invalid length of array purchase units")
	}
	totals := make([]float64, 0, len(purchaseUnits))
	for _, unit := range purchaseUnits {
		for _, capture := range unit.Payments.Captures {
			amount, err := strconv.ParseFloat(capture.SellerReceivableBreakdown.NetAmount.Value, 64)
			if err != nil {
				return []float64{}, err
			}
			totals = append(totals, amount)
		}
	}

	if len(totals) <= 0 {
		return []float64{}, errors.New("invalid length of array totals")
	}
	return totals, nil
}

func getCaptureIds(purchaseUnits []CapturePurchaseUnit) ([]string, error) {
	if len(purchaseUnits) <= 0 {
		return []string{}, errors.New("")
	}
	captureIds := make([]string, 0, len(purchaseUnits))
	for _, unit := range purchaseUnits {
		for _, capture := range unit.Payments.Captures {
			captureIds = append(captureIds, capture.ID)
		}
	}
	if len(captureIds) <= 0 {
		return []string{}, errors.New("invalid length of array capture ids")
	}
	return captureIds, nil
}

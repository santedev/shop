package store

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"shop/gateaways"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

type provider string
type Sku string
type currency string

const (
	Google provider = "google"
	Local  provider = "local"
)
const (
	USD currency = "USD"
	COP currency = "COP"
)

type Store interface {
	Init() error
	Close()

	RestoreUser(context.Context) (User, error)
	GetUser(context.Context) (User, error)
	NewUser(context.Context, User) (User, error)
	DeleteUser(ctx context.Context, id string) error

	GetProducts(ctx context.Context, index, limit int) ([]Product, error)
	GetProduct(context.Context) (Product, error)
	InsertProduct(context.Context, Product) (Product, error)
	UpdateCombinations(context.Context, int, []Combination) error
	UpdateVariants(context.Context, int, []Variant) error
	RemoveProduct(context.Context) error

	EmptyingCart(ctx context.Context, userId int) error
	RemoveProductFromCart(ctx context.Context, userId int, sku Sku) (count, error)
	UpdateCartCount(ctx context.Context, userId int, sku Sku, quantity int) (count, error)
	CartCountItemsWithTotal(ctx context.Context, userId int) (int, float64, error)
	CartCountItems(ctx context.Context, userId int) (int, error)
	AddToCart(ctx context.Context, userId int, sku Sku, quantity int) (int, error)
	AddToCartWithItem(ctx context.Context, userId int, sku Sku, quantity int) (Items, int, error)
	GetCart(ctx context.Context, userId int) ([]Items, error)
	TotalItems(ctx context.Context, items []OrderItems) (float64, error)

	MakeOrder(ctx context.Context, paymentProvider gateaways.PaymentProvider, userId int, cartItems []OrderItems, total float64, currency currency, orderId, payerName, payerEmail, payerId string, referenceIds, captureIds []string) (int, error)

	UpdateStock(ctx context.Context, items []OrderItems) error
	CheckStockFromItemsAndUpdateCart(ctx context.Context, userId int, items []OrderItems) (bool, error)
	CheckStockFromItems(ctx context.Context, items []OrderItems) error
}

type count struct {
	CartCount      int
	ProductCount   int
	ProductBalance float64
	CartBalance    float64
}

type Option struct {
	Id        int
	VariantId int
	Option    string
}

type Items struct {
	Id               int
	Name             string
	Description      string
	ShortDescription string
	Images           []string
	Variants         []Variant
	Comb             Combination
	Quantity         int
	CreatedAt        time.Time
}

type Product struct {
	Id               int
	Name             string
	Description      string
	ShortDescription string
	Images           []string
	Variants         []Variant
	Combinations     []Combination
	CreatedAt        time.Time
}

type Variant struct {
	Label   string
	Options []Option
}

type Combination struct {
	Sku      Sku
	Price    float64
	Currency currency
	Stock    int
	Options  []Option
}

type Account interface {
	Legit() bool
}

type User struct {
	Id          int
	Name        string
	Email       string
	AvatarUrl   string
	CreatedAt   time.Time
	CartId      int
	FavoritesId int
	Provider    provider
}

func (u User) Legit() bool {
	return false
}

type Admin struct {
	Id          int
	Name        string
	Email       string
	AvatarUrl   string
	CreatedAt   time.Time
	CartId      int
	FavoritesId int
	Provider    provider
}

func (a Admin) Legit() bool {
	return true
}

type Order struct {
	Products []OrderItems `json:"products"`
	Currency currency     `json:"currency"`
	FromCart bool         `json:"fromCart"`
}

type OrderItems struct {
	Sku      Sku `json:"sku"`
	Quantity int `json:"quantity"`
}

func ToCurrency(curr string) (currency, error) {
	switch curr {
	case "USD":
		return USD, nil
	case "COP":
		return COP, nil
	default:
		return "", errors.New("not a supported currency")
	}
}

func (curr currency) Valid() error {
	switch curr {
	case USD, COP:
		return nil
	default:
		return errors.New("not a supported currency")
	}
}

func (curr currency) Truncate() int {
	switch curr {
	case USD:
		return 2
	case COP:
		return 3
	}
	return -1
}

func (item *Items) SetComb(sku Sku, combs []Combination) error {
	for _, comb := range combs {
		if sku == comb.Sku {
			item.Comb = comb
			return nil
		}
	}
	return errors.New("could not find match for sku within the slice of combinations")
}

func GetProvider(r *http.Request) (provider, error) {
	provider := provider(chi.URLParam(r, "provider"))
	switch provider {
	case Google:
		return provider, nil
	case Local:
		return provider, nil
	default:
		return "", errors.New("not supported provider")
	}
}

func (sku Sku) ProductId() (int, error) {
	if len(sku) <= 1 {
		return -1, errors.New("invalid len for sku")
	}
	if !(sku[len(sku)-1] >= '0' && sku[len(sku)-1] <= '9') {
		return -1, errors.New(fmt.Sprintf("sku cant end with '%s' char", string(sku[len(sku)-1])))
	}
	var i int
	for i = len(sku) - 1; i >= 0; i-- {
		if sku[i] == '-' {
			break
		}
		if !(sku[i] >= '0' && sku[i] <= '9') {
			return -1, errors.New(fmt.Sprintf("sku suffix needs to be a digit, got: '%s'", string(sku[i])))
		}
		if i == 0 {
			return -1, errors.New("not a valid sku")
		}
	}
	id, err := strconv.Atoi(string(sku[i+1:]))
	if err != nil {
		return -1, err
	}
	return id, nil
}

func TotalItems(currency currency, items ...Items) (float64, error) {
	if len(items) <= 0 {
		return -1, errors.New("incorrect len of items, needs at least one item")
	}
	if err := currency.Valid(); err != nil {
		return -1, err
	}
	truncate := currency.Truncate()
	total := decimal.NewFromInt(0)
	for _, item := range items {
		price, err := truncateFl(truncate, item.Comb.Price)
		if err != nil {
			return -1, err
		}
		if total.LessThanOrEqual(decimal.NewFromInt(0)) {
			total = price.Mul(decimal.NewFromInt(int64(item.Quantity)))
			continue
		}
		total = total.Add(price.Mul(decimal.NewFromInt(int64(item.Quantity))))
	}
	t, _ := total.Float64()
	return t, nil
}

func TotalItemsFloat(currency currency, items ...float64) (float64, error) {
	if len(items) <= 0 {
		return -1, errors.New("incorrect len of items, needs at least one item")
	}
	if err := currency.Valid(); err != nil {
		return -1, err
	}
	truncate := currency.Truncate()
	total := decimal.NewFromInt(0)
	for _, item := range items {
		price, err := truncateFl(truncate, item)
		if err != nil {
			return -1, err
		}
		if total.LessThanOrEqual(decimal.NewFromInt(0)) {
			total = price
			continue
		}
		total = total.Add(price)
	}
	t, _ := total.Float64()
	return t, nil
}

func truncateFl(truncate int, num float64) (decimal.Decimal, error) {
	numAsStr := fmt.Sprintf("%.*f", truncate, num)
	f, err := strconv.ParseFloat(numAsStr, 64)
	if err != nil {
		return decimal.Decimal{}, err
	}
	dec := decimal.NewFromFloat(f)
	return dec, nil
}

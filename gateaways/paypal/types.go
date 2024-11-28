package paypal

import (
	"errors"
	"time"
)

type status string

const (
	COMPLETED          status = "COMPLETED"
	PENDING            status = "PENDING"
	PARTIALLY_REFUNDED status = "PARTIALLY_REFUNDED"
	DECLINED           status = "DECLINED"
	REFUNDED           status = "REFUNDED"
	FAILED             status = "FAILED"
)

func (s status) Valid() error {
	switch s {
	case COMPLETED, PENDING, PARTIALLY_REFUNDED, DECLINED, REFUNDED, FAILED:
		return nil
	default:
		return errors.New("not supported status")
	}
}

type Key string
type Param struct {
	Key Key
	Val string
}

type paypal struct {
	accessToken *accessToken
}

type accessToken struct {
	Scope        string        `json:"scope"`
	Token        string        `json:"access_token"`
	TokenType    string        `json:"token_type"`
	AppID        string        `json:"app_id"`
	Nonce        string        `json:"nonce"`
	ExpiresInInt time.Duration `json:"expires_in"`
	ExpiresIn    time.Time
}

type CreateOrderRequest struct {
	PurchaseUnits []PurchaseUnit `json:"purchase_units"`
	Intent        string         `json:"intent"`
	PaymentSource PaymentSource  `json:"payment_source"`
}

type PurchaseUnit struct {
	Amount      Amount `json:"amount"`
	ReferenceID string `json:"reference_id"`
}

type Amount struct {
	CurrencyCode string `json:"currency_code"`
	Value        string `json:"value"`
}

type PaymentSource struct {
	Paypal PayPalExp `json:"paypal"`
}

type PayPalExp struct {
	ExperienceContext ExperienceContext `json:"experience_context"`
}

type ExperienceContext struct {
	PaymentMethodPreference string `json:"payment_method_preference"`
	PaymentMethodSelected   string `json:"payment_method_selected"`
	BrandName               string `json:"brand_name"`
	Locale                  string `json:"locale"`
	LandingPage             string `json:"landing_page"`
	ShippingPreference      string `json:"shipping_preference"`
	UserAction              string `json:"user_action"`
	ReturnURL               string `json:"return_url"`
	CancelURL               string `json:"cancel_url"`
}

type PayPalDetails struct {
	EmailAddress  string `json:"email_address"`
	AccountID     string `json:"account_id"`
	AccountStatus string `json:"account_status"`
	Name          struct {
		GivenName string `json:"given_name"`
		Surname   string `json:"surname"`
	} `json:"name"`
	Address struct {
		CountryCode string `json:"country_code"`
	} `json:"address"`
}

type Capture struct {
	ID     string `json:"id"`
	Status status `json:"status"`
	Amount struct {
		CurrencyCode string `json:"currency_code"`
		Value        string `json:"value"`
	} `json:"amount"`
	FinalCapture     bool `json:"final_capture"`
	SellerProtection struct {
		Status            string   `json:"status"`
		DisputeCategories []string `json:"dispute_categories"`
	} `json:"seller_protection"`
	SellerReceivableBreakdown struct {
		GrossAmount struct {
			CurrencyCode string `json:"currency_code"`
			Value        string `json:"value"`
		} `json:"gross_amount"`
		PayPalFee struct {
			CurrencyCode string `json:"currency_code"`
			Value        string `json:"value"`
		} `json:"paypal_fee"`
		NetAmount struct {
			CurrencyCode string `json:"currency_code"`
			Value        string `json:"value"`
		} `json:"net_amount"`
	} `json:"seller_receivable_breakdown"`
	Links []struct {
		Href   string `json:"href"`
		Rel    string `json:"rel"`
		Method string `json:"method"`
	} `json:"links"`
	CreateTime string `json:"create_time"`
	UpdateTime string `json:"update_time"`
}

type CapturePurchaseUnit struct {
	ReferenceID string `json:"reference_id"`
	Shipping    struct {
		Name struct {
			FullName string `json:"full_name"`
		} `json:"name"`
		Address struct {
			AddressLine1 string `json:"address_line_1"`
			AdminArea2   string `json:"admin_area_2"`
			AdminArea1   string `json:"admin_area_1"`
			PostalCode   string `json:"postal_code"`
			CountryCode  string `json:"country_code"`
		} `json:"address"`
	} `json:"shipping"`
	Payments struct {
		Captures []Capture `json:"captures"`
	} `json:"payments"`
}

type Payer struct {
	Name struct {
		GivenName string `json:"given_name"`
		Surname   string `json:"surname"`
	} `json:"name"`
	EmailAddress string `json:"email_address"`
	PayerID      string `json:"payer_id"`
	Address      struct {
		CountryCode string `json:"country_code"`
	} `json:"address"`
}

type Link struct {
	Href   string `json:"href"`
	Rel    string `json:"rel"`
	Method string `json:"method"`
}

type Transaction struct {
	ID            string                `json:"id"`
	Status        status                `json:"status"`
	PaymentSource PaymentSource         `json:"payment_source"`
	PurchaseUnits []CapturePurchaseUnit `json:"purchase_units"`
	Payer         Payer                 `json:"payer"`
	Links         []Link                `json:"links"`
}

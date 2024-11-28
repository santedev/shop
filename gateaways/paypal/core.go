package paypal

import (
	"fmt"
	"shop/config"
)

func Init() error {
	getPaypal = func() *paypal {
		if pub == nil {
			pub = &paypal{}
		}
		return pub
	}
	return nil
}

const (
	v1 = "v1"
	v2 = "v2"
)

func origin() string {
	return fmt.Sprintf("%s/%s", getPaypalUrl(), v1)
}

func paypalOrderURL() string {
	return fmt.Sprintf("%s/%s/%s", getPaypalUrl(), v2, "checkout/orders")
}

func accessTokenURL() string {
	return fmt.Sprintf("%s/%s", origin(), "oauth2/token")
}

func captureOrderURL(orderId string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", getPaypalUrl(), v2, "checkout/orders", orderId, "capture")
}

var pub *paypal
var getPaypal func() *paypal

func getPaypalUrl() string {
	paypalUrl := "https://api-m.sandbox.paypal.com"
	prodUrl := "https://api-m.paypal.com"
	if config.Envs.Production {
		return prodUrl
	}
	return paypalUrl
}

func getPaypalUrlSdk() string {
	paypalSdk := "https://sandbox.paypal.com/sdk/js"
	prodSdk := "https://www.paypal.com/sdk/js"
	if config.Envs.Production {
		return prodSdk
	}
	return paypalSdk
}

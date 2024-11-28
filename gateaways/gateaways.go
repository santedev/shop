package gateaways

import "errors"

type Gateaway interface {
	Pay()
	Refund()
	RejectPay()
}

type PaymentProvider string

const (
	Paypal PaymentProvider = "paypal"
)

func (p PaymentProvider) Valid() error {
	switch p {
	case Paypal:
		return nil
	default:
		return errors.New("payment provider not supported")
	}
}

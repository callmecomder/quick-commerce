package payment

import "errors"

var ErrPaymentFailed = errors.New("payment service: charge failed")

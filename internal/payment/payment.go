package payment

import "context"

type Result struct {
	Success   bool
	Reference string
}

type Payment interface {
	Charge(ctx context.Context, amount int64, userID, requestID string) (Result, error)
}

type MockPayment struct {
	ShouldFail bool
}

func NewMockPayment() *MockPayment {
	return &MockPayment{}
}

func (m *MockPayment) Charge(_ context.Context, _ int64, _ string, requestID string) (Result, error) {
	if m.ShouldFail {
		return Result{Success: false}, ErrPaymentFailed
	}
	return Result{Success: true, Reference: "pay_" + requestID}, nil
}

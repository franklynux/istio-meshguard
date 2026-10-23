package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc"

	inventorypb "github.com/franklynux/istio-meshguard/services/orders/pb/inventory"
	notificationspb "github.com/franklynux/istio-meshguard/services/orders/pb/notifications"
	paymentspb "github.com/franklynux/istio-meshguard/services/orders/pb/payments"
)

// ---- hand-rolled fakes for the three downstream services --------------------

// callLog records the order of downstream calls, shared by all the fakes.
type callLog struct{ events []string }

func (l *callLog) record(event string) { l.events = append(l.events, event) }

type fakeInventory struct {
	log         *callLog
	onCheck     func(*inventorypb.CheckStockRequest) (*inventorypb.CheckStockResponse, error)
	onDecrement func(*inventorypb.DecrementRequest) (*inventorypb.DecrementResponse, error)
	checkReqs   []*inventorypb.CheckStockRequest
}

func (f *fakeInventory) CheckStock(_ context.Context, in *inventorypb.CheckStockRequest, _ ...grpc.CallOption) (*inventorypb.CheckStockResponse, error) {
	f.log.record("check")
	f.checkReqs = append(f.checkReqs, in)
	if f.onCheck != nil {
		return f.onCheck(in)
	}
	return &inventorypb.CheckStockResponse{Available: true, Found: true}, nil
}

func (f *fakeInventory) Decrement(_ context.Context, in *inventorypb.DecrementRequest, _ ...grpc.CallOption) (*inventorypb.DecrementResponse, error) {
	f.log.record("decrement")
	if f.onDecrement != nil {
		return f.onDecrement(in)
	}
	return &inventorypb.DecrementResponse{Success: true}, nil
}

type fakePayments struct {
	log      *callLog
	onCharge func(*paymentspb.ChargeRequest) (*paymentspb.ChargeResponse, error)
}

func (f *fakePayments) Charge(_ context.Context, in *paymentspb.ChargeRequest, _ ...grpc.CallOption) (*paymentspb.ChargeResponse, error) {
	f.log.record("charge")
	if f.onCharge != nil {
		return f.onCharge(in)
	}
	return &paymentspb.ChargeResponse{Success: true, TransactionId: "txn_test"}, nil
}

type fakeNotifications struct {
	log    *callLog
	onSend func(*notificationspb.SendConfirmationRequest) (*notificationspb.SendConfirmationResponse, error)
}

func (f *fakeNotifications) SendConfirmation(_ context.Context, in *notificationspb.SendConfirmationRequest, _ ...grpc.CallOption) (*notificationspb.SendConfirmationResponse, error) {
	f.log.record("notify")
	if f.onSend != nil {
		return f.onSend(in)
	}
	return &notificationspb.SendConfirmationResponse{Sent: true}, nil
}

// Compile-time proof that the fakes satisfy the generated client interfaces.
var (
	_ inventorypb.InventoryClient         = (*fakeInventory)(nil)
	_ paymentspb.PaymentsClient           = (*fakePayments)(nil)
	_ notificationspb.NotificationsClient = (*fakeNotifications)(nil)
)

// harness wires an Orchestrator to fakes that succeed by default. Each test
// overrides only the behaviour it cares about.
type harness struct {
	orch  *Orchestrator
	inv   *fakeInventory
	pay   *fakePayments
	notif *fakeNotifications
	log   *callLog
}

func newHarness() *harness {
	log := &callLog{}
	h := &harness{
		log:   log,
		inv:   &fakeInventory{log: log},
		pay:   &fakePayments{log: log},
		notif: &fakeNotifications{log: log},
	}
	h.orch = NewOrchestrator(h.inv, h.pay, h.notif)
	return h
}

// calls returns the downstream calls made so far, e.g. "check,charge".
func (h *harness) calls() string { return strings.Join(h.log.events, ",") }

const testEmail = "frank@example.com"

var oneItem = []OrderItem{{SKU: "SKU-001", Quantity: 2}}

// ---- tests ------------------------------------------------------------------

func TestCreateOrder_Success(t *testing.T) {
	h := newHarness()

	res, err := h.orch.CreateOrder(context.Background(), oneItem, testEmail, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusConfirmed {
		t.Fatalf("status = %q, want %q (reason: %s)", res.Status, StatusConfirmed, res.Reason)
	}
	if !strings.HasPrefix(res.OrderID, "ord_") {
		t.Errorf("order id = %q, want an ord_ prefix", res.OrderID)
	}
	// Payment is taken BEFORE stock is decremented.
	if got, want := h.calls(), "check,charge,decrement,notify"; got != want {
		t.Errorf("call order = %s, want %s", got, want)
	}
}

func TestCreateOrder_InsufficientStock(t *testing.T) {
	h := newHarness()
	h.inv.onCheck = func(*inventorypb.CheckStockRequest) (*inventorypb.CheckStockResponse, error) {
		return &inventorypb.CheckStockResponse{Available: false, Found: true}, nil
	}

	res, err := h.orch.CreateOrder(context.Background(), oneItem, testEmail, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusRejected || !strings.Contains(res.Reason, "insufficient stock") {
		t.Errorf("got status=%q reason=%q, want rejected / insufficient stock", res.Status, res.Reason)
	}
	if got := h.calls(); got != "check" {
		t.Errorf("calls = %q, want only the stock check", got)
	}
}

func TestCreateOrder_UnknownSKU(t *testing.T) {
	h := newHarness()
	h.inv.onCheck = func(*inventorypb.CheckStockRequest) (*inventorypb.CheckStockResponse, error) {
		return &inventorypb.CheckStockResponse{Found: false}, nil
	}

	res, err := h.orch.CreateOrder(context.Background(), oneItem, testEmail, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusRejected || !strings.Contains(res.Reason, "unknown SKU") {
		t.Errorf("got status=%q reason=%q, want rejected / unknown SKU", res.Status, res.Reason)
	}
	if got := h.calls(); got != "check" {
		t.Errorf("calls = %q, want only the stock check", got)
	}
}

func TestCreateOrder_PaymentDeclined(t *testing.T) {
	h := newHarness()
	h.pay.onCharge = func(*paymentspb.ChargeRequest) (*paymentspb.ChargeResponse, error) {
		return &paymentspb.ChargeResponse{Success: false, Error: "exceeds card limit"}, nil
	}

	res, err := h.orch.CreateOrder(context.Background(), oneItem, testEmail, 1500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusPaymentFailed || res.Reason != "exceeds card limit" {
		t.Errorf("got status=%q reason=%q, want payment_failed / the gateway's reason", res.Status, res.Reason)
	}
	// No Decrement and no notification: a declined payment must not touch stock.
	if got := h.calls(); got != "check,charge" {
		t.Errorf("calls = %q, want check,charge only", got)
	}
}

func TestCreateOrder_StockChangedAfterPayment(t *testing.T) {
	h := newHarness()
	h.inv.onDecrement = func(*inventorypb.DecrementRequest) (*inventorypb.DecrementResponse, error) {
		return &inventorypb.DecrementResponse{Success: false, Error: "insufficient stock"}, nil
	}

	res, err := h.orch.CreateOrder(context.Background(), oneItem, testEmail, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusRejected || !strings.Contains(res.Reason, "refund") {
		t.Errorf("got status=%q reason=%q, want rejected with a refund note", res.Status, res.Reason)
	}
	if got := h.calls(); got != "check,charge,decrement" {
		t.Errorf("calls = %q, want no confirmation after a failed decrement", got)
	}
}

func TestCreateOrder_InventoryUnavailable(t *testing.T) {
	errBoom := errors.New("connection refused")
	h := newHarness()
	h.inv.onCheck = func(*inventorypb.CheckStockRequest) (*inventorypb.CheckStockResponse, error) {
		return nil, errBoom
	}

	_, err := h.orch.CreateOrder(context.Background(), oneItem, testEmail, 100)
	if !errors.Is(err, errBoom) {
		t.Fatalf("err = %v, want it to wrap the downstream error", err)
	}
	// Fail closed: if we can't check stock, we must not charge.
	if got := h.calls(); got != "check" {
		t.Errorf("calls = %q, want nothing charged", got)
	}
}

func TestCreateOrder_NotificationFailureStillConfirms(t *testing.T) {
	h := newHarness()
	h.notif.onSend = func(*notificationspb.SendConfirmationRequest) (*notificationspb.SendConfirmationResponse, error) {
		return nil, errors.New("smtp down")
	}

	res, err := h.orch.CreateOrder(context.Background(), oneItem, testEmail, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusConfirmed {
		t.Errorf("status = %q, want confirmed: a paid, fulfilled order must not fail over an email", res.Status)
	}
}

func TestCreateOrder_InvalidRequests(t *testing.T) {
	cases := []struct {
		name  string
		items []OrderItem
	}{
		{"no items", nil},
		{"missing sku", []OrderItem{{SKU: "", Quantity: 1}}},
		{"zero quantity", []OrderItem{{SKU: "SKU-001", Quantity: 0}}},
		{"negative quantity", []OrderItem{{SKU: "SKU-001", Quantity: -3}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness()
			res, err := h.orch.CreateOrder(context.Background(), tc.items, testEmail, 100)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Status != StatusRejected {
				t.Errorf("status = %q, want rejected", res.Status)
			}
			if got := h.calls(); got != "" {
				t.Errorf("calls = %q, want no downstream calls for an invalid order", got)
			}
		})
	}
}

func TestCreateOrder_DuplicateSKUsAreAggregated(t *testing.T) {
	h := newHarness()
	items := []OrderItem{{SKU: "SKU-002", Quantity: 6}, {SKU: "SKU-002", Quantity: 6}}

	if _, err := h.orch.CreateOrder(context.Background(), items, testEmail, 100); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Two lines of 6 must be checked as one line of 12, not as two lines that
	// each look fine against a stock of 10.
	if len(h.inv.checkReqs) != 1 || h.inv.checkReqs[0].Quantity != 12 {
		t.Errorf("check requests = %v, want a single check for quantity 12", h.inv.checkReqs)
	}
}
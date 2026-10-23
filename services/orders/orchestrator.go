package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math"
	"strings"

	inventorypb "github.com/franklynux/istio-meshguard/services/orders/pb/inventory"
	notificationspb "github.com/franklynux/istio-meshguard/services/orders/pb/notifications"
	paymentspb "github.com/franklynux/istio-meshguard/services/orders/pb/payments"
)

// Order statuses returned to callers.
const (
	StatusConfirmed     = "confirmed"
	StatusRejected      = "rejected"
	StatusPaymentFailed = "payment_failed"
)

// OrderItem is one line of an order.
type OrderItem struct {
	SKU      string
	Quantity int32
}

// Result is the outcome of an order that reached a decision. Reason is set
// whenever Status is not StatusConfirmed. System failures (a downstream service
// being unreachable) are reported separately, as a Go error.
type Result struct {
	OrderID string
	Status  string
	Reason  string
}

// Orchestrator coordinates the inventory, payments and notifications services.
type Orchestrator struct {
	inventory     inventorypb.InventoryClient
	payments      paymentspb.PaymentsClient
	notifications notificationspb.NotificationsClient
}

func NewOrchestrator(
	inv inventorypb.InventoryClient,
	pay paymentspb.PaymentsClient,
	notif notificationspb.NotificationsClient,
) *Orchestrator {
	return &Orchestrator{inventory: inv, payments: pay, notifications: notif}
}

// CreateOrder runs the order flow: check stock, charge, decrement stock, notify.
func (o *Orchestrator) CreateOrder(ctx context.Context, items []OrderItem, email string, amount float64) (Result, error) {
	orderID := newOrderID()

	items, reason := normalize(items)
	if reason != "" {
		return rejected(orderID, reason), nil
	}

	// 1. Check stock for every line. Nothing is charged or changed yet, so a
	//    "no" here costs nothing.
	for _, item := range items {
		resp, err := o.inventory.CheckStock(ctx, &inventorypb.CheckStockRequest{Sku: item.SKU, Quantity: item.Quantity})
		if err != nil {
			// Fail closed: if we cannot check stock, we do not charge.
			return Result{}, fmt.Errorf("check stock for %s: %w", item.SKU, err)
		}
		if !resp.GetFound() {
			return rejected(orderID, "unknown SKU: "+item.SKU), nil
		}
		if !resp.GetAvailable() {
			return rejected(orderID, "insufficient stock for "+item.SKU), nil
		}
	}

	// 2. Take payment BEFORE touching stock (see the note below the code).
	charge, err := o.payments.Charge(ctx, &paymentspb.ChargeRequest{OrderId: orderID, Amount: amount})
	if err != nil {
		return Result{}, fmt.Errorf("charge order %s: %w", orderID, err)
	}
	if !charge.GetSuccess() {
		reason := charge.GetError()
		if reason == "" {
			reason = "payment declined"
		}
		return Result{OrderID: orderID, Status: StatusPaymentFailed, Reason: reason}, nil
	}

	// 3. Payment is taken, so now reserve the stock.
	for _, item := range items {
		dec, err := o.inventory.Decrement(ctx, &inventorypb.DecrementRequest{Sku: item.SKU, Quantity: item.Quantity})
		if err != nil {
			slog.Error("decrement failed after successful charge",
				"order_id", orderID, "transaction_id", charge.GetTransactionId(), "sku", item.SKU, "error", err)
			return Result{OrderID: orderID}, fmt.Errorf("decrement %s for paid order %s: %w", item.SKU, orderID, err)
		}
		if !dec.GetSuccess() {
			slog.Error("stock changed after payment, charge needs refunding",
				"order_id", orderID, "transaction_id", charge.GetTransactionId(), "sku", item.SKU, "reason", dec.GetError())
			return rejected(orderID, fmt.Sprintf(
				"stock for %s changed during checkout after payment was taken; the charge (%s) must be refunded",
				item.SKU, charge.GetTransactionId())), nil
		}
	}

	// 4. Best-effort confirmation. The order is already paid and fulfilled, so a
	//    failed email must not turn it into a failed order.
	resp, err := o.notifications.SendConfirmation(ctx, &notificationspb.SendConfirmationRequest{OrderId: orderID, Email: email})
	if err != nil || !resp.GetSent() {
		slog.Warn("order confirmed but confirmation not sent", "order_id", orderID, "error", err)
	}

	return Result{OrderID: orderID, Status: StatusConfirmed}, nil
}

// normalize validates the items and merges lines that repeat a SKU, keeping
// first-seen order. A non-empty reason means the order is invalid.
func normalize(items []OrderItem) ([]OrderItem, string) {
	if len(items) == 0 {
		return nil, "order has no items"
	}

	totals := make(map[string]int64, len(items))
	var skus []string // first-seen order, since map iteration order is random
	for _, item := range items {
		if item.SKU == "" {
			return nil, "item is missing a SKU"
		}
		if item.Quantity <= 0 {
			return nil, fmt.Sprintf("quantity for %s must be positive", item.SKU)
		}
		if _, seen := totals[item.SKU]; !seen {
			skus = append(skus, item.SKU)
		}
		totals[item.SKU] += int64(item.Quantity) // int64 so the sum cannot overflow
	}

	merged := make([]OrderItem, 0, len(skus))
	for _, sku := range skus {
		if totals[sku] > math.MaxInt32 {
			return nil, fmt.Sprintf("total quantity for %s is too large", sku)
		}
		merged = append(merged, OrderItem{SKU: sku, Quantity: int32(totals[sku])})
	}
	return merged, ""
}

func rejected(orderID, reason string) Result {
	return Result{OrderID: orderID, Status: StatusRejected, Reason: reason}
}

func newOrderID() string {
	return "ord_" + strings.ToLower(rand.Text()[:12])
}
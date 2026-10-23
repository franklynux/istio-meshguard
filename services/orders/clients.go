package main

import (
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	inventorypb "github.com/franklynux/istio-meshguard/services/orders/pb/inventory"
	notificationspb "github.com/franklynux/istio-meshguard/services/orders/pb/notifications"
	paymentspb "github.com/franklynux/istio-meshguard/services/orders/pb/payments"
)

// Clients holds the three downstream clients and their connections.
type Clients struct {
	Inventory     inventorypb.InventoryClient
	Payments      paymentspb.PaymentsClient
	Notifications notificationspb.NotificationsClient
	conns         []*grpc.ClientConn
}

// NewClients builds the clients from INVENTORY_ADDR, PAYMENTS_ADDR and
// NOTIFICATIONS_ADDR (each host:port).
func NewClients() (*Clients, error) {
	c := &Clients{}

	inv, err := c.dial("INVENTORY_ADDR")
	if err != nil {
		return nil, err
	}
	pay, err := c.dial("PAYMENTS_ADDR")
	if err != nil {
		c.Close()
		return nil, err
	}
	notif, err := c.dial("NOTIFICATIONS_ADDR")
	if err != nil {
		c.Close()
		return nil, err
	}

	c.Inventory = inventorypb.NewInventoryClient(inv)
	c.Payments = paymentspb.NewPaymentsClient(pay)
	c.Notifications = notificationspb.NewNotificationsClient(notif)
	return c, nil
}

func (c *Clients) dial(envVar string) (*grpc.ClientConn, error) {
	addr := os.Getenv(envVar)
	if addr == "" {
		return nil, fmt.Errorf("%s is not set", envVar)
	}
	// Plaintext on purpose: inside the mesh, the Istio sidecars add mTLS on the
	// wire, so the application itself does no TLS.
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create client for %s (%s): %w", envVar, addr, err)
	}
	c.conns = append(c.conns, conn)
	return conn, nil
}

// Close releases all connections.
func (c *Clients) Close() {
	for _, conn := range c.conns {
		_ = conn.Close()
	}
}
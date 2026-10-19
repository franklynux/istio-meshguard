package main

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrNotFound          = errors.New("sku not found")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrInvalidQuantity   = errors.New("quantity must be positive")
)

// Store holds in-memory stock levels, keyed by SKU. It is safe for
// concurrent use: gRPC serves each request on its own goroutine.
type Store struct {
	mu    sync.Mutex
	stock map[string]int32
}

// NewStore returns a Store seeded with the fixed demo inventory.
func NewStore() *Store {
	return &Store{
		stock: map[string]int32{
			"SKU-001": 50,
			"SKU-002": 10,
			"SKU-003": 0,
		},
	}
}

// CheckStock reports whether quantity units of sku are available.
// found is false when the SKU does not exist at all.
func (s *Store) CheckStock(sku string, quantity int32) (available, found bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	have, ok := s.stock[sku]
	if !ok {
		return false, false
	}
	return have >= quantity, true
}

// Decrement removes quantity units of sku. It returns success=false and a
// descriptive error if the SKU is unknown, the quantity is not positive, or
// there is not enough stock. A rejected call leaves stock unchanged.
func (s *Store) Decrement(sku string, quantity int32) (success bool, err error) {
	if quantity <= 0 {
		return false, ErrInvalidQuantity
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	have, ok := s.stock[sku]
	if !ok {
		return false, fmt.Errorf("%w: %s", ErrNotFound, sku)
	}
	if have < quantity {
		return false, fmt.Errorf("%w: %s has %d, requested %d", ErrInsufficientStock, sku, have, quantity)
	}
	s.stock[sku] = have - quantity
	return true, nil
}

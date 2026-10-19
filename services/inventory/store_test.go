package main

import (
	"errors"
	"sync"
	"testing"
)

func TestCheckStock_Available(t *testing.T) {
	s := NewStore()
	available, found := s.CheckStock("SKU-001", 10)
	if !available || !found {
		t.Fatalf("expected available=true found=true, got available=%v found=%v", available, found)
	}
}

func TestCheckStock_InsufficientQuantity(t *testing.T) {
	s := NewStore()
	available, found := s.CheckStock("SKU-002", 999)
	if available || !found {
		t.Fatalf("expected available=false found=true, got available=%v found=%v", available, found)
	}
}

func TestCheckStock_UnknownSKU(t *testing.T) {
	s := NewStore()
	_, found := s.CheckStock("SKU-DOES-NOT-EXIST", 1)
	if found {
		t.Fatal("expected found=false for unknown SKU")
	}
}

func TestDecrement_Success(t *testing.T) {
	s := NewStore()
	ok, err := s.Decrement("SKU-001", 10)
	if !ok || err != nil {
		t.Fatalf("expected success, got ok=%v err=%v", ok, err)
	}
	// 50 - 10 = 40 left: 40 is available, 41 is not.
	if available, _ := s.CheckStock("SKU-001", 40); !available {
		t.Fatal("expected 40 units still available")
	}
	if available, _ := s.CheckStock("SKU-001", 41); available {
		t.Fatal("expected 41 units to be unavailable")
	}
}

func TestDecrement_InsufficientStock(t *testing.T) {
	s := NewStore()
	ok, err := s.Decrement("SKU-002", 999)
	if ok || !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("expected ok=false err=ErrInsufficientStock, got ok=%v err=%v", ok, err)
	}
	// A rejected decrement must leave stock untouched.
	if available, _ := s.CheckStock("SKU-002", 10); !available {
		t.Fatal("rejected decrement changed the stock")
	}
}

func TestDecrement_UnknownSKU(t *testing.T) {
	s := NewStore()
	ok, err := s.Decrement("SKU-DOES-NOT-EXIST", 1)
	if ok || !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ok=false err=ErrNotFound, got ok=%v err=%v", ok, err)
	}
}

func TestDecrement_InvalidQuantity(t *testing.T) {
	cases := []struct {
		name     string
		quantity int32
	}{
		{"zero", 0},
		{"negative", -5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewStore()
			ok, err := s.Decrement("SKU-001", tc.quantity)
			if ok || !errors.Is(err, ErrInvalidQuantity) {
				t.Fatalf("expected ok=false err=ErrInvalidQuantity, got ok=%v err=%v", ok, err)
			}
		})
	}
}

func TestDecrement_Concurrent(t *testing.T) {
	s := NewStore() // SKU-001 starts at 50
	const attempts = 100

	var wg sync.WaitGroup
	results := make(chan bool, attempts)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, _ := s.Decrement("SKU-001", 1)
			results <- ok
		}()
	}
	wg.Wait()
	close(results)

	successes := 0
	for ok := range results {
		if ok {
			successes++
		}
	}
	if successes != 50 {
		t.Fatalf("expected exactly 50 successful decrements, got %d", successes)
	}
}

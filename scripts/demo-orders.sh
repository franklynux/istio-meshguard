#!/usr/bin/env bash
# Places a set of demo orders against the orders service.
# Usage: scripts/demo-orders.sh [host:port]   (default: localhost:50050)
set -euo pipefail

ADDR="${1:-localhost:50050}"

order() {
  local label="$1" payload="$2"
  echo "--- ${label}"
  grpcurl -plaintext -d "${payload}" "${ADDR}" orders.Orders/CreateOrder || true
  echo
}

order "Confirmed: 2 x SKU-001, amount 100" \
  '{"items":[{"sku":"SKU-001","quantity":2}],"email":"frank@example.com","amount":100}'
order "Payment declined: amount 1500 is over the 1000 card limit" \
  '{"items":[{"sku":"SKU-001","quantity":2}],"email":"frank@example.com","amount":1500}'
order "Rejected: insufficient stock (999 x SKU-002)" \
  '{"items":[{"sku":"SKU-002","quantity":999}],"email":"frank@example.com","amount":10}'
order "Rejected: unknown SKU" \
  '{"items":[{"sku":"NOPE","quantity":1}],"email":"frank@example.com","amount":10}'
order "Rejected: out-of-stock SKU-003" \
  '{"items":[{"sku":"SKU-003","quantity":1}],"email":"frank@example.com","amount":10}'

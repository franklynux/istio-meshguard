# proto/

Shared gRPC contracts for istio-meshguard. Each `.proto` file here is the single
source of truth for one service's API. Every consumer generates its client or
server code from the same file, so the Go, Python, and Node services cannot drift
out of sync with each other.

| File | Served by | Called by |
|---|---|---|
| `orders.proto` | orders (Go) | external clients via the ingress gateway |
| `inventory.proto` | inventory (Go) | orders |
| `payments.proto` | payments (Python) | orders |
| `notifications.proto` | notifications (Node.js) | orders |

Generated code is committed. After changing a `.proto` file, re-run the commands
for every service that uses it and commit the result.

All commands below run from the **repository root**.

## Prerequisites

```sh
sudo apt install -y protobuf-compiler          # protoc
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Make sure `$(go env GOPATH)/bin` is on your `PATH` so `protoc` can find the Go plugins.

## Go: inventory

```sh
mkdir -p services/inventory/pb
protoc -I proto \
  --go_out=services/inventory/pb --go_opt=paths=source_relative \
  --go-grpc_out=services/inventory/pb --go-grpc_opt=paths=source_relative \
  inventory.proto
```

`-I proto` sets the directory `protoc` resolves `.proto` files against, and
`paths=source_relative` writes the output directly into the target directory.
Without `-I proto`, the `proto/` prefix would be copied into the output path.

## Go: orders

`orders` serves its own API and is a client of the other three services, so it
needs code for all four contracts. Each lives in its own Go package under
`services/orders/pb/`.

```sh
mkdir -p services/orders/pb/inventory services/orders/pb/payments services/orders/pb/notifications

# orders: the API this service serves
protoc -I proto \
  --go_out=services/orders/pb --go_opt=paths=source_relative \
  --go-grpc_out=services/orders/pb --go-grpc_opt=paths=source_relative \
  orders.proto

# payments and notifications: clients. Their go_package already points at orders/pb/<name>.
protoc -I proto \
  --go_out=services/orders/pb/payments --go_opt=paths=source_relative \
  --go-grpc_out=services/orders/pb/payments --go-grpc_opt=paths=source_relative \
  payments.proto

protoc -I proto \
  --go_out=services/orders/pb/notifications --go_opt=paths=source_relative \
  --go-grpc_out=services/orders/pb/notifications --go-grpc_opt=paths=source_relative \
  notifications.proto

# inventory: client. Its go_package points into the inventory service's own module,
# so override the Go package for this invocation with an M option.
INV=github.com/franklynux/istio-meshguard/services/orders/pb/inventory
protoc -I proto \
  --go_out=services/orders/pb/inventory --go_opt=paths=source_relative,Minventory.proto=$INV \
  --go-grpc_out=services/orders/pb/inventory --go-grpc_opt=paths=source_relative,Minventory.proto=$INV \
  inventory.proto
```

The three client packages cannot share one output directory: they declare
different Go packages, and Go allows one package per directory.

## Python: payments

```sh
pip install -r services/payments/requirements-dev.txt

python -m grpc_tools.protoc -I proto \
  --python_out=services/payments \
  --pyi_out=services/payments \
  --grpc_python_out=services/payments \
  proto/payments.proto
```

The generated files go next to `server.py`, not in a subfolder. The generated
`payments_pb2_grpc.py` contains `import payments_pb2`, a top-level import, so both
files must be importable as top-level modules.

## Node.js: notifications

No codegen step. `@grpc/proto-loader` parses `proto/notifications.proto` when the
process starts. The service finds it through the `PROTO_PATH` environment
variable, which defaults to the repo's `proto/` directory.

Because the proto lives outside `services/notifications/`, build the image with
the **repository root** as the build context:

```sh
docker build -f services/notifications/Dockerfile -t notifications:local .
```

The Dockerfile copies the single real `proto/notifications.proto` into the image,
so there is no second copy that could go stale.
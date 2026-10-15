# proto/

Shared gRPC contracts for istio-meshguard. Each `.proto` file here is the single
source of truth for one service's API — every consumer generates its client
code from the same file, so the Go, Python, and Node services can never drift
out of sync with each other.

## Go (inventory, orders)

Install the codegen plugins once per machine:

```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Generate code (run from the repo root):

```sh
protoc --go_out=services/inventory/pb --go_opt=paths=source_relative \
       --go-grpc_out=services/inventory/pb --go-grpc_opt=paths=source_relative \
       proto/inventory.proto

protoc --go_out=services/orders/pb --go_opt=paths=source_relative \
       --go-grpc_out=services/orders/pb --go-grpc_opt=paths=source_relative \
       proto/orders.proto proto/payments.proto proto/notifications.proto
```

`orders` needs all four `.proto` files generated into its own `pb/` tree
because it acts as a client to `inventory`, `payments`, and `notifications`
in addition to serving its own `Orders` API. `inventory` only needs its own.

## Python (payments)

Install once:

```sh
pip install grpcio grpcio-tools --break-system-packages
```

Generate code:

```sh
python -m grpc_tools.protoc -I proto \
    --python_out=services/payments/pb \
    --grpc_python_out=services/payments/pb \
    proto/payments.proto
```

## Node.js (notifications)

No codegen step. `@grpc/proto-loader` reads `proto/notifications.proto`
directly at process startup, so the service just needs the `.proto` file
available at a known path (copied into the service's Docker build context).

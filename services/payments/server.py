"""gRPC server for the Payments service."""

import logging
import signal
from concurrent import futures

import grpc

import payments_pb2
import payments_pb2_grpc
from charge import charge

PORT = 50052
logger = logging.getLogger("payments")


class PaymentsServicer(payments_pb2_grpc.PaymentsServicer):
    def Charge(self, request, context):
        result = charge(order_id=request.order_id, amount=request.amount)
        logger.info(
            "charge order=%s amount=%.2f success=%s",
            request.order_id,
            request.amount,
            result.success,
        )
        # Business failures (over the card limit) go in the response, not as a
        # gRPC error, so `orders` can read the reason from the message.
        return payments_pb2.ChargeResponse(
            success=result.success,
            transaction_id=result.transaction_id or "",
            error=result.error or "",
        )


def serve() -> None:
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    payments_pb2_grpc.add_PaymentsServicer_to_server(PaymentsServicer(), server)
    server.add_insecure_port(f"[::]:{PORT}")
    server.start()
    logger.info("payments listening on :%d", PORT)

    # Kubernetes sends SIGTERM when stopping a pod: stop accepting new calls
    # and give in-flight ones up to 5 seconds to finish.
    def shutdown(signum, frame):
        logger.info("shutting down")
        server.stop(grace=5)

    signal.signal(signal.SIGTERM, shutdown)
    signal.signal(signal.SIGINT, shutdown)
    server.wait_for_termination()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    serve()
import math
import uuid
from dataclasses import dataclass

CARD_LIMIT = 1000.0


@dataclass(frozen=True)
class ChargeResult:
    success: bool
    transaction_id: str | None = None
    error: str | None = None


def charge(order_id: str, amount: float) -> ChargeResult:
    """Attempt to charge `amount` for `order_id`.

    Failures are returned as a ChargeResult, never raised, so the gRPC layer
    can always send a well-formed response.
    """
    # NaN compares False with everything, so `amount <= 0` alone would let it
    # through. Check for finite values explicitly.
    if not math.isfinite(amount) or amount <= 0:
        return ChargeResult(success=False, error="amount must be a positive number")

    if amount > CARD_LIMIT:
        return ChargeResult(
            success=False,
            error=f"amount {amount:.2f} exceeds card limit {CARD_LIMIT:.2f}",
        )

    return ChargeResult(success=True, transaction_id=f"txn_{uuid.uuid4().hex[:12]}")
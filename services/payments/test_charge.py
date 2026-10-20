import math

import pytest

from charge import charge


def test_charge_succeeds_under_limit():
    result = charge(order_id="o1", amount=500.0)
    assert result.success is True
    assert result.transaction_id is not None
    assert result.error is None


def test_charge_succeeds_exactly_at_limit():
    # The rule is "fails when amount > 1000", so 1000 itself is fine.
    result = charge(order_id="o1", amount=1000.0)
    assert result.success is True


def test_charge_fails_over_limit():
    result = charge(order_id="o2", amount=1500.0)
    assert result.success is False
    assert result.transaction_id is None
    assert "limit" in result.error


@pytest.mark.parametrize("amount", [0, -5.0, math.nan, math.inf])
def test_charge_rejects_invalid_amount(amount):
    result = charge(order_id="o3", amount=amount)
    assert result.success is False
    assert result.error is not None


def test_transaction_ids_are_unique():
    first = charge(order_id="o4", amount=10.0)
    second = charge(order_id="o5", amount=10.0)
    assert first.transaction_id != second.transaction_id
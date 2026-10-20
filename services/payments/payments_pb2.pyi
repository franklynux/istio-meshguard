from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ChargeRequest(_message.Message):
    __slots__ = ("order_id", "amount")
    ORDER_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    order_id: str
    amount: float
    def __init__(self, order_id: _Optional[str] = ..., amount: _Optional[float] = ...) -> None: ...

class ChargeResponse(_message.Message):
    __slots__ = ("success", "transaction_id", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_ID_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    transaction_id: str
    error: str
    def __init__(self, success: _Optional[bool] = ..., transaction_id: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

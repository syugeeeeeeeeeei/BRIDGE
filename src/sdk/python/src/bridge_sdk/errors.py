class BridgeError(RuntimeError):
    def __init__(self, message: str, *, exit_code=None, stdout="", stderr="", code="", category="", retryable=False, request_id=""):
        super().__init__(message)
        self.exit_code = exit_code
        self.stdout = stdout
        self.stderr = stderr
        self.code = code
        self.category = category
        self.retryable = bool(retryable)
        self.request_id = request_id

class BridgeBinaryNotFoundError(BridgeError): pass
class BridgeBinaryPermissionError(BridgeError): pass
class BridgeValidationError(BridgeError): pass
class BridgeIOError(BridgeError): pass
class BridgeTimeoutError(BridgeError): pass
class BridgeCancelledError(BridgeError): pass
class BridgeAcceptanceError(BridgeError): pass
class BridgeInternalError(BridgeError): pass
class BridgeProtocolError(BridgeError): pass
class BridgeVersionError(BridgeError): pass

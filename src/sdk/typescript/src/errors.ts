export class BridgeError extends Error {
  constructor(message: string, public readonly exitCode?: number, public readonly stdout = "", public readonly stderr = "") { super(message); }
}
export class BridgeBinaryNotFoundError extends BridgeError {}
export class BridgeBinaryPermissionError extends BridgeError {}
export class BridgeValidationError extends BridgeError {}
export class BridgeIOError extends BridgeError {}
export class BridgeTimeoutError extends BridgeError {}
export class BridgeCancelledError extends BridgeError {}
export class BridgeAcceptanceError extends BridgeError {}
export class BridgeInternalError extends BridgeError {}
export class BridgeProtocolError extends BridgeError {}
export class BridgeVersionError extends BridgeError {}

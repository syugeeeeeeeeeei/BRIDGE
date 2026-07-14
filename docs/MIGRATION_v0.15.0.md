# BRIDGE v0.15.0 migration guide

v0.15.0 is a deliberate breaking release. Deprecated one-shot internals, detached DG6 code, legacy boolean termination semantics, and compatibility fallbacks are not retained.

## Required changes

- Treat `TerminationStatus` as the authoritative, exclusive outcome.
- Use ANCHOR `Session` for interruption, epoch stepping, snapshot, and resume.
- Use `HandoffRequest` and `HandoffResult` for ANCHOR–BOLTS cooperation.
- Register only validated `Evidence`; empirical evidence is not a proof.
- Interpret all work fields under `WorkModelVersion = 2.0`.
- Use ULTRASOUND `AnytimeCurve` and `ComputeReuse` for bound and reuse analysis.

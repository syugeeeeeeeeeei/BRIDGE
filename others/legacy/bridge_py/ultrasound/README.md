# ULTRASOUND

ULTRASOUND is the development and validation observer for BEARING events.

- Normative field and event meanings: [`TRACE_SEMANTICS.md`](TRACE_SEMANTICS.md)
- Executable registry and validator: `semantics.py`
- In-memory and JSONL adapter: `observer.py`

A trace should be treated as valid data only after `observer.validate().valid` is true. `write_jsonl()` and `read_jsonl()` validate by default.

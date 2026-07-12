# BENCHMARK DATASET SPEC v1

## Scope

`bridge.dataset.v1` is a TRAFFIC-only development and research contract. Production routing through GATE does not read graph files.

## Required fields

- `schema_version`: `bridge.dataset.v1`
- `id`: stable dataset identifier
- `source`: acquisition source or provenance statement
- `license`: SPDX identifier or exact license name
- `directed`: graph directionality
- `nodes`: contiguous node count; IDs are `0..nodes-1`
- `edges`: `from`, `to`, and finite non-negative `weight`

Optional fields are `positions`, `default_query`, and ordered `preprocessing` records. Dataset bytes are SHA-256 hashed and the hash, source, license, path, and preprocessing records are copied into each raw run.

## Reproducibility

Node IDs and adjacency lists are canonicalized. Dataset-specific conversion must occur before production of this document and every conversion step must be listed in `preprocessing`. A changed byte sequence is a different dataset artifact even if the graph is semantically equivalent.

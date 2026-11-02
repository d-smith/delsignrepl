## Delsign backend API

The API contract is in `docs/delsign-api/`, copied from the delsignbackend repo (see `VERSION`).
- Read `docs/delsign-api/client-guide.md` before writing or changing code that calls the API.
  Use `openapi.yaml` for exact shapes and status codes.
- Wei amounts are decimal strings on the wire and `bigint` in code, never `number`.
- Signing code must pass a test against `test-vectors.json` (the guide has one). Sends sign
  the request's exact strings, so don't change them after signing.
- Don't edit `docs/delsign-api/`; ask for a fresh export from the backend instead.

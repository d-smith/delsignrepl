# Plan 01: catch up with the new delsign backend API

Status: **reviewed; questions answered 2026-11-02.** No code has been changed yet.
Contract: `docs/delsign-api/` (backend commit `e0c51ee`, branch `feature/refresh`, exported 2026-11-02).
Branch: `feature/backend-update`.
Constraint: this app talks to the backend **only through its HTTP API**. It doesn't import,
vendor, copy or generate code from backend modules (see §1, "Boundary with the backend").
Dev chain: the backend uses **Anvil** (Foundry), not Ganache.

## Questions and answers

There are no open questions left. The answers below are folded into the decisions and
tasks.

1. **Where may the RSA private key live?** A plain key file with owner-only permissions
   is fine for development. It must be gitignored, and the readme must explain how to
   create one. See D1 and Phases 0, 3 and 5.
2. **Accounts locked out by earlier in-memory keys.** Assume the backend owner resets them.
   The app does nothing special about them.
3. **Tokens and the cheat code.** Keep the cheat code as a dev feature and document how to
   use it. See D3 and Phase 5. This is the one accepted exception to the API-only
   boundary, because it needs the backend's JWT secret.
4. **Production.** There's no production environment for now. Production requirements
   are out of scope; there's a note in §6.
5. **Amount entry.** Keep wei. See D7.
6. **Do we need backend API changes before proceeding?** **No.** None of the gaps blocks
   this plan:

   | Gap | Why it doesn't block | Follow-up |
   |---|---|---|
   | 6.1 No way to find out what happened to an earlier send (after a 409 or no response) | The UI says "the transfer may have happened, check the balance", as the guide recommends | Optional feature request to the backend owner: look up a send by `requestId` |
   | 6.2 A retry after `expiry`: 409 or 400? | The app never retries after its own `expiry` (§3, "Errors and retries"), so it never asks the server this | None |
   | 6.3 No transaction status | The app says "broadcast, not yet mined" and points at the balance | Optional feature request |
   | 6.4 No server time for `expiry` | In dev the app, backend and Anvil share one machine's clock, so skew is negligible | Revisit if a remote environment appears |
   | 6.5 No key recovery | Dev keys are files (D1); lost dev accounts are reset by the backend owner (answer 2) | None |

   Recommendation: proceed now. Send 6.1 and 6.3 to the backend owner as non-blocking
   requests (Phase 5).

---

## 1. Scope

**In scope.** Bring every backend call in this repo up to the contract in
`docs/delsign-api/`. Specifically:

- every change in the guide's "Changes from the earlier API" section;
- persisting the user's RSA key in a gitignored dev key file, and supporting key rotation;
- one configurable base URL instead of seven hardcoded copies;
- status-code-based error handling and the guide's send retry rules;
- tests: the signing test against `test-vectors.json` first, then tests for each flow
  against an `httptest` fake server;
- readme updates:
  - key files;
  - the cheat code;
  - configuration;
  - the Anvil dev chain, replacing the Ganache notes.

**Not a browser app.** This repo is a Go terminal UI built with `tview` (`main.go`). The guide
is written for browser clients (TypeScript, WebCrypto, IndexedDB, CORS), so this plan
translates it:

| Guide (browser) | This app (Go) |
|---|---|
| `crypto.subtle` RSASSA-PKCS1-v1_5 / SHA-256 | `rsa.SignPKCS1v15` + `sha256` (already in `keys/keys.go:34`, and already matches the vectors; see §2.3) |
| `exportKey("spki")` hex | `x509.MarshalPKIXPublicKey` hex (`keys/keys.go:20`, already matches) |
| Non-extractable key in IndexedDB | A `0600` PKCS #8 PEM file, gitignored (D1) |
| `crypto.randomUUID()` | A UUIDv4 from `crypto/rand` (D5) |
| `bigint` | `*big.Int`. Never `int64` or `float64`. CLAUDE.md's "bigint in code" means `*big.Int` here |
| Same origin / dev-server proxy | Not needed. See §3, "Connecting" |

**Boundary with the backend.** The app interacts with the backend only through the HTTP API
described in `docs/delsign-api/`. Concretely:

- **No backend code.** `go.mod` gets no `require` or `replace` for the delsignbackend
  module. Nothing is vendored, copied or adapted from backend source. That includes its
  request types, validation and signature-checking code. As of 2026-11-02 the app has no
  such dependency (checked `go.mod`, `go.sum` and every import).
- **Types come from the contract.** The request and response types in `api/` are written
  by hand from `openapi.yaml`, not from the backend's Go structs. If the API grows, the
  answer is a fresh export of the docs, not a look at backend source.
- **Contract data is allowed.** `test-vectors.json` and `openapi.yaml` are part of the
  exported contract. Tests may read them. They are data, not modules.
- **No side doors.** The app doesn't read the backend's database, config or logs, and
  doesn't call the Anvil node directly. Anything the app needs that the API doesn't offer
  becomes a request to the backend owner, not a workaround.
- **Tests stay on our side of the wire.** Fake servers in tests are `httptest` handlers
  written from `openapi.yaml`. They don't import backend handlers. Manual testing runs the
  backend and Anvil as separate processes, and the app reaches the backend only over HTTP.

**Accepted exception:** the dev cheat code uses the backend's JWT secret, supplied through an
env var (answer 3, D3). It's dev-only, documented, and never committed.

**Out of scope:**

- production deployment (answer 4; see §6);
- moving HTTP calls off the UI goroutine (D6);
- splitting UI from logic beyond what testing needs;
- entering amounts in ETH (answer 5);
- any change to `docs/delsign-api/`.

---

## 2. Current-state survey

`go build ./...`, `go vet ./...` and `go test ./...` all pass on the branch today (one test,
`keys/keys_test.go`). `go.mod` declares `go 1.20` and marks every dependency
`// indirect`; the installed toolchain is 1.27.3. There's no `.gitignore`. `readme.md:25-36`
has demo notes for Ganache (an address, plus `web3` console commands), which are now
out of date.

### 2.1 Every backend call

All seven use `http.DefaultClient`, which has **no timeout**, so a hung server freezes the TUI
for good. All seven run on the tview UI goroutine. None reads the `text/plain` error body,
none closes `resp.Body`, and each one overwrites the `http.NewRequest` error before checking it.

| # | Call | Where | Expects | Problems against the new contract |
|---|---|---|---|---|
| 1 | `POST /api/v1/keyreg` | `keys/keypages.go:101-122` (URL `:108`) | 200 | No `sig4rotate` (see §3.3). Any non-200 becomes `"error registering key"` (`:118`) with no status or text |
| 2 | `POST /api/v1/wallets` | `wallets/wallets.go:41-62` (URL `:42`) | 201 | Sets `Content-Type: application/json` with no body (`:43`). Harmless per the spec, but the guide only sets it with a body |
| 3 | `GET /api/v1/wallets` | `wallets/address.go:70-90` (URL `:72`) | **201** (`:80`) | **The spec says 200.** This always "fails", and `DoAddressGeneration` drops the error (`:12`) and shows "No wallets found". **Address generation is broken against the new backend.** |
| 4 | `POST /api/v1/wallets/{id}/addresses` | `wallets/address.go:92-116` (URL `:93`) | 201 | `Content-Type` with no body (`:94`). Ownership failure is now 403; shown only as a generic message |
| 5 | `GET /api/v1/walletctx` | `wallets/wallets.go:69-89` (URL `:71`) | 200 | OK |
| 6 | `GET /api/v1/wallets/balance/{a}` | `wallets/balances.go:43-69` (URL `:46`) | 200 | **`amount` is decoded into `*big.Int` (`:60`), which rejects a JSON string.** I checked: `math/big: cannot unmarshal "\"1000\"" into a *big.Int`. **Balance is broken.** |
| 7 | `POST /api/v1/wallets/send` | `send/send.go:205-240` (URL `:216`) | 200 | **Old message, no `requestId`/`expiry`, `amount` sent as a JSON number.** I checked: `{"amount":5}`. **Sends are broken.** |

Never called today: `GET /api/v1/authping` and `GET /api/v1/users/{email}`.

### 2.2 Tokens

- The only token source is the startup page `token/input.go:53-99`, shown once
  (`main.go:60`). Nothing returns to it, so an expired token means restarting the app.
- The "cheat code" (`token/input.go:56-85`) mints an HS256 token with a hardcoded secret
  `"secret"` (`:79`), `exp` = now + 8 h (`:40`), plus `sub`, `email`, `iat`, `nbf`.
- `token.ExtractEmailFromToken` (`token/token.go:9-27`) *verifies* the token with
  `"secret"` just to read `email`; it's called at `keys/keypages.go:67`. A token signed
  with any other secret fails here, before the request is ever sent.
- The token input caps length at 256 (`token/input.go:54`). A pasted token longer than that
  is silently cut short. The cheat code's `SetText` gets around the cap.

### 2.3 Keys and signing

- **Generation:** `keys.Generate` (`keys/keys.go:12-16`) makes an RSA-2048 key and discards
  the error. `DoKeyGeneration` (`keys/keypages.go:16-35`) **overwrites** `state.PrivateKey`
  (`:21-22`) every time.
- **Storage:** process memory only: `state.PrivateKey`, `state.PublicKeyDER`
  (`state/state.go:11-12`). Never written to disk. `keys.Encode` / `Decode`
  (`keys/keys.go:18-32`) use PKCS #1 for the private key and are used only by the test.
  `Decode` ignores every error.
- **Signing:** `keys.Sign` (`keys/keys.go:34-43`): SHA-256 then `SignPKCS1v15`, hex. It
  panics on error.
- **Already matches the vectors.** In a throwaway test (since deleted), I loaded both
  vector keys from their PKCS #8 hex. `keys.Sign` reproduced **all six** vector
  signatures, and `keys.Encode`'s public hex equalled each `publicKeySpkiHex`. So the RSA
  primitive and key encoding don't need to change. What's wrong is the **messages and
  bodies** built around them.
- **Registration:** `DoKeyRegistration` (`keys/keypages.go:65-87`) signs the email and
  posts `api.KeyReg` (`api/types.go:3-7`: `email`, `pubkey`, `sig4reg`). It panics if no
  key was generated first (`state.PrivateKey` is nil at `:72`).

### 2.4 Amounts

- **Input:** `send/send.go:117-120` uses `tview.InputFieldInteger`. I checked: it **accepts
  `-5`, `007` and `+3`** and **rejects anything above int64** (~9.22 ETH in wei).
- **Parsing:** `send/send.go:187-188`, where `SetString`'s ok flag is ignored. Empty or
  invalid input gives an undefined amount.
- **Signing vs. body:** the message is formatted with `%d` from the `*big.Int` (`:190`).
  The body holds the same `*big.Int` (`:182`), which marshals as a number. Under the new
  rules, the amount string has to be built once and reused for both.
- **Display:** `DoGetBalance` (`wallets/balances.go:22-23`) prints wei and
  `weiToEther` (`:71-73`), which divides as integers. 0.9 ETH shows as "0 ether".

### 2.5 Lists

`walletctx` and `wallets` decode into nil slices, and every caller checks `len(...) == 0`
(`state/state.go:33`, `wallets/address.go:13`, `send/send.go:83`). `[]` and `null` both
work. No change is needed (§3.8).

---

## 3. The API changes and where this app is affected

### 3.1 Tokens need `exp`

**Affected, but only slightly.** Cheat-code tokens already have `exp` (`token/input.go:40`).
Changes:

- A pasted token with no `exp` now gets 403 on every call. Check this locally when the
  token is saved, and say so.
- The 8 h expiry needs a way back to the token page without restarting (`main.go:60` is
  the only entry point).
- A 403 means "check `/authping`" (§3.6), and the token page is the "log in again"
  destination.

The cheat code stays as a documented dev feature (answer 3, D3).

### 3.2 Sends use the new signed format with `requestId` and `expiry`

**Affected.** Changes in `send/send.go`:

- **`:179-184`:** `SendPayload` becomes `api.SendRequest` with `source`, `dest`, `amount`
  (string), `requestId` (string), `expiry` (int64) and `sig`, and no other fields.
- **`:190`:** the message `"%s%s%d"` becomes
  `api.SendMessage(r) = r.Source + r.Dest + r.Amount + r.RequestID + strconv.FormatInt(r.Expiry, 10)`,
  built from the same struct that is then marshalled.
- **`:186-203`:** becomes `BuildSend(priv, source, dest, wei, now, newID)`:
  - validates both addresses and that the amount is positive;
  - sets `expiry = now + 120`;
  - signs;
  - marshals **once**. The exact bytes are kept so that a retry resends them unchanged.
- **`:136-137`:** the request is built **on confirmation**, not when the form is filled in.
  A confirmation step is added (D8).
- `source` comes from `state.Address` (checksum case from `walletctx`) and is never
  reformatted.
- `dest` is typed by the user:
  - trim it and validate it as `0x` + 40 hex digits **before** signing;
  - after signing, never change it.

### 3.3 Key rotation needs `sig4rotate`

**Affected, and it's the most serious change.** The current design generates a fresh key
in memory each session and registers it. That only worked because the old backend
accepted re-registration. Now:

- **The second time the app runs, registration gets 403 for good.** The old key is gone,
  so `sig4rotate` can't be produced. Key persistence (D1) is a prerequisite.
- `DoKeyGeneration` (`keys/keypages.go:21-22`) overwrites the current key in place. This
  destroys the very key a rotation needs. Generation must make a *pending* key and keep
  the current one until the backend confirms the rotation.
- `api.KeyReg` (`api/types.go:3-7`) gains `Sig4Rotate string \`json:"sig4rotate,omitempty"\``.
  `omitempty` matters: an empty `"sig4rotate": ""` fails the hex pattern and would turn a
  first registration into a 400/403.
- `DoKeyRegistration` (`keys/keypages.go:65-87`) is rebuilt around `GET /users/{email}` (D2).

Accounts already locked out by an earlier in-memory key are reset by the backend owner
(answer 2).

### 3.4 `GET /users/{email}` needs a token

**Not affected today:** the app never calls it. The plan adds calls (D2: registration
status and "is my local key the registered one?"). They go through the shared client, so
they always carry the bearer token, and they use only the token's own email,
escaped with `url.PathEscape`. A 403 here means a token problem or the wrong email. The
second can't happen, because the email comes from the token.

### 3.5 Request bodies with unknown fields get a 400

**Affected only through the new code.** Today's bodies are exact:

- `KeyReg` has `email`, `pubkey`, `sig4reg`;
- `SendPayload` has `source`, `dest`, `amount`, `sig`.

Bodyless POSTs (`wallets/wallets.go:42`, `wallets/address.go:93`) send none. The risks are
in what this plan adds:

- `sig4rotate` must be `omitempty`;
- `SendRequest` must have exactly six fields;
- no request struct embeds anything.

The vector test checks this: it decodes each vector body with `DisallowUnknownFields` and
re-marshals it to an identical object (§3, "Signing"). Bodyless POSTs stop sending
`Content-Type`, as the guide's wrapper does.

### 3.6 Ownership failures are 403, node failures are 502

**Affected in effect, though no code switches on these codes today.** Every caller turns
any non-2xx into a fixed string and drops the body, so nothing *breaks*, but the UI can't
follow the guide. The plan adds one error type, `api.Error{Status, Text}`, and the handling
in §3, "Errors and retries":

- **403 on**
  - `POST /wallets/{id}/addresses` (`wallets/address.go:102`);
  - `POST /wallets/send` (`send/send.go:225`);
  - `GET /users/{email}`;

  For all three, call `/authping`. If that also gets 403, go to the token page. Otherwise
  report "not yours".
- **502 on**
  - `GET /wallets/balance/{a}` (`wallets/balances.go:54`): show the node's text;
  - `POST /wallets/send`: see retries.

### 3.7 Wei amounts are decimal strings in sends and balances

**Affected, and both directions are broken today** (§2.4, rows 6-7 of §2.1). Amounts stay
in wei (answer 5). Changes:

- **Balance (`wallets/balances.go:58-61`):** decode `amount` as a `string`, then
  `api.ParseWei`:
  - matches `^(0|[1-9][0-9]*)$`;
  - parses with `SetString(s, 10)`;
  - checks the value is at most 2^256-1.
- **Send:** the amount input stays in wei. It accepts digits only, with no length cap
  below 78 (D7), and is parsed with `api.ParseWei`. Zero is refused. The body carries
  `wei.String()`, the only form the server accepts, which is also the string that gets
  signed.
- **Display:** replace `weiToEther` (`wallets/balances.go:71-73`) with `api.FormatEther`,
  an exact `big.Int` quotient/remainder with the fraction padded to 18 digits and trailing
  zeros trimmed. Show wei and ETH on both the balance and the send confirmation.
- No `int64`, `float64` or `strconv.ParseInt` anywhere on an amount path.

### 3.8 Empty lists are `[]`, not `null`

**Not affected.** Go decodes `[]` into an empty slice and `null` into a nil slice. Every
consumer only checks `len(...) == 0` (§2.5). One test in Phase 2 pins this down for both
list endpoints.

---

## 3 (cont). Cross-cutting topics

### Connecting

**No proxy needed.** CORS and preflight checks are enforced by browsers. Go's `net/http`
client sends no preflight and ignores CORS headers, so this app talks to
`http://localhost:3010` directly. The guide's Vite proxy applies only if a browser front end
is ever added.

What the app does need:

- **One base URL.** It comes from `DELSIGN_API_URL`, defaulting to
  `http://localhost:3010`. It replaces the seven hardcoded copies:
  - `keys/keypages.go:108`;
  - `send/send.go:216`;
  - `wallets/address.go:72`, `:93`;
  - `wallets/balances.go:46`;
  - `wallets/wallets.go:42`, `:71`.
- **One `api.Client`:**
  - a `*http.Client` with a timeout (D6);
  - sets `Authorization: Bearer` on every call;
  - sets `Content-Type` only when there's a body;
  - reads and closes every body;
  - turns non-2xx into `api.Error`.

**Production** is out of scope (answer 4). What a future environment would need is noted in
§6 so it isn't lost.

### Signing: test against the vectors first

The new file is `keys/vectors_test.go` (package `keys_test`). It reads
`../docs/delsign-api/test-vectors.json`; `go test` runs in the package directory. For every
vector and every signature it checks:

1. `keys.Sign(s.message, key)` equals `s.signature`, with the key parsed by
   `x509.ParsePKCS8PrivateKey` from `privateKeyPkcs8Hex`. **This already passes** (§2.3).
2. The public hex of each key equals `publicKeySpkiHex`.
3. `body[s.field] == s.signature`.
4. **`sig`:**
   - the body decodes into `api.SendRequest` under `DisallowUnknownFields`;
   - `api.SendMessage(req) == s.message`;
   - `json.Marshal(req)` decodes (with `UseNumber`) to the same object as the vector body,
     which proves `amount` is a string and `expiry` a number.
5. **`sig4rotate`:**
   - `email + pubkey == s.message`;
   - `keys.BuildKeyReg(email, next=key2, current=key1)` equals the `rotate` body exactly.
6. **`register`:** `keys.BuildKeyReg(email, key1, nil)` equals the `register` body, with
   no `sig4rotate` key present.
7. **`BuildSend`:** with `now = 1800000060 - 120` and an injected id generator returning
   `6f1c2a7e-…`, it reproduces all three send bodies byte-for-byte as objects. This covers
   the lowercase-address and large-amount (> 2^53) cases. PKCS #1 v1.5 is deterministic,
   so exact equality is a valid check.

The test is written **before** the send and keyreg code changes:

- Phase 1 adds it, together with *stub* declarations (`api.SendRequest`,
  `api.SendMessage`, `keys.BuildKeyReg`, `send.BuildSend`) so it compiles.
- Checks 1-3 pass.
- Checks 4-7 fail on the stubs. That's the expected red.
- Phases 3-4 make them pass.

The test keys are used only in tests. The test uses only the Go standard library and this
repo's packages. The vectors file is its only input from the backend side, and that file is
part of the exported contract.

### Errors and retries

All handling switches on `api.Error.Status` and shows the server's text. None of it parses
that text. Every outcome below is a tview modal.

| Status | Any call | Send specifically |
|---|---|---|
| no response (transport error or client timeout) | "Can't reach the server" + error | See the send flow below |
| 400 | "Request rejected: \<text\>". Logged as a client bug | If it's an expiry problem (the user waited): back to the form; a new confirmation builds a new request |
| 401 | n/a | "Signature rejected. Your key may not be the registered one." Run the key check (D2) and show the result. **No retry** |
| 403 | Call `/authping`. 403 → "Session expired or token invalid", then the token page. 200 → "Not permitted: \<text\>" (not your wallet or address) | Same |
| 404 | `GET /users`: "No key registered", which starts registration | n/a |
| 409 | keyreg: "Your key changed meanwhile". Reload with `GET /users` and re-run the register flow **once** | See the send flow below |
| 413 / 415 | "Client error \<status\>". These are bugs | |
| 500 | "Server error". The text is generic | |
| 502 | Balance: "Ethereum node error: \<text\>" | See the send flow below |

**The send flow.** The request is built once, on confirmation; the exact JSON bytes and
the `expiry` are kept.

- **200:** "Broadcast. Transaction `<txnid>`. The balance changes once it's mined."
- **No response:** modal "No response. The transfer may or may not have happened."
  - **[Retry same request]** resends the *identical bytes*. It's offered only while
    `now < expiry`. Once expiry passes, the button is replaced by the text "This request
    has expired; check the balance before sending again." The app never resends an expired
    body, which is why gap 6.2 doesn't matter.
  - **[Check balance]**
  - **[Menu]**

  There's no automatic retry, so the user always sees that the first attempt was
  ambiguous.
- **409** (only reachable through a retry, since every confirmation gets a fresh UUID):
  "The server already processed this request. The transfer may have happened. Check the
  balance before sending again." Buttons: **[Check balance] [Menu]**. **Never** retries
  with a new id.
- **502:** "Ethereum node error: \<text\>. The transfer was most likely not sent. That
  request can't be reused."
  - **[Edit and send again]** goes back to the send form with source, destination and
    amount filled in. Pressing Send opens the confirmation again, and confirming builds a
    **new** `requestId`, `expiry` and signature. That counts as the new user confirmation
    the guide asks for.
  - **[Menu]**

Calls stay synchronous on the UI goroutine (D6). That blocks a double submit for free:
the Send button can't fire twice while a request is in flight.

### Where this app departs from the guide, and what the API doesn't offer

- **Non-extractable key storage.** Go has no WebCrypto equivalent. The key is a `0600`
  dev file instead (answer 1, D1).
- **"The app never makes tokens."** The cheat code does make them. It's kept as a
  documented dev feature (answer 3, D3).
- **Telling the user that a lost key means locked funds.** Shown when a key is generated,
  and documented in the readme.
- **Gaps in the API** (question 6): none blocks this plan. See the table at the top.

---

## 4. Decisions

Each records the options and the chosen one.

**D1. Key persistence: resolved (answer 1)**

**Chosen:** a PKCS #8 PEM file, mode `0600`, written atomically (temp file plus rename).

- **Location:** `$DELSIGN_KEY_DIR/<email>/current.pem` (and `pending.pem` during a
  rotation). `DELSIGN_KEY_DIR` defaults to `.delsign/keys` under the working directory, so
  a dev run from the repo keeps its keys next to it.
- **Gitignore:** `.gitignore` gets `/.delsign/` and `*.pem`, the second as a backstop if
  someone points `DELSIGN_KEY_DIR` elsewhere inside the repo.
- **Loading:** the app refuses a key file that group or others can read, and says
  `chmod 600 <file>`, as ssh does.
- **Two ways to get a key** (documented in the readme):
  - **(a)** "Generate key" in the app;
  - **(b)** create one with
    `openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out .delsign/keys/<email>/current.pem && chmod 600 …`.
    `genpkey` writes PKCS #8 PEM, which the app loads.

  There's no way to download a key from the backend; it only ever holds the public half.
- Generating a key shows a warning: "This file is your only way to authorise sends. If
  it's lost, funds in your addresses are locked unless the backend owner resets your
  account."

Not chosen:

- passphrase encryption: not needed for dev;
- the OS keychain: awkward on WSL;
- in-memory only: breaks after one session (§3.3).

**D2. Registration and rotation flow**

- a) One "Register key" action that reconciles against `GET /users/{email}`.
- b) Separate "Register" and "Rotate" menu items.

**Chosen: a.** It's robust to lost responses, and a 409 reload falls out of the same code
path. "Generate key" makes `current` if none exists. Otherwise it makes `pending`, after
asking "Replace your key?". "Register key" then does:

- **404** → register `current` (no `sig4rotate`).
- **200, registered == current, pending exists** → rotate:
  - `sig4reg = pending.sign(email)`;
  - `sig4rotate = current.sign(email + pendingPubHex)`;
  - on 200, promote `pending` to `current` (rename).
- **200, registered == pending** → an earlier rotation succeeded but its response was
  lost: promote `pending`.
- **200, registered == current, no pending** → "Already registered".
- **200, matches neither** → "This machine's key isn't the registered one. Sends will be
  rejected. Ask the backend owner to reset the account." No action.

The pending key goes to disk **before** keyreg is sent, so no outcome loses a key. The
`pubkey` comparison is case-insensitive (`strings.EqualFold`). The same check runs
read-only after login and after a send gets 401.

**D3. Tokens and the cheat code: resolved (answer 3)**

**Chosen:**

- **Reading the token:** `email` and `exp` are read with `jwt.Parser.ParseUnverified` (it
  exists in the pinned v3.2.2; checked). The backend is the verifier. Delete the local
  verification with `"secret"` (`keys/keypages.go:67`).
- **The cheat code stays as a dev feature.** Its secret comes from `DELSIGN_DEV_JWT_SECRET`,
  which must equal the backend's `DELSIGN_JWT_SECRET`. The checkbox appears only when the
  variable is set. The hardcoded `"secret"` (`token/input.go:79`) goes. The secret is
  never committed.
- **On save:**
  - require `email` and `exp`;
  - reject an already-expired token;
  - call `/authping`. A 403 stays on the token page and says to check that the secret
    matches the backend's.
- Raise the 256-character input cap to 4096.
- Add a **"Log in again"** menu item (`t`). The 403 handling also routes to it.
- The readme documents the cheat code: set the variable, enter an email, tick the box,
  then Save. The token lasts 8 h.

**D4. Package layout**

**Chosen:**

- `api/` has no app imports. It holds:
  - `Client` and `Error`;
  - every request and response type;
  - `SendMessage`;
  - `ParseWei` and `FormatEther`;
  - `ValidAddress`.
- `keys/` gains `store.go` (D1) and `BuildKeyReg`.
- `send/` gains `BuildSend`, with injected `now` and id source for tests.

This avoids an import cycle, because `keys` already imports `api`. Types are written by
hand from `openapi.yaml`. Generating them from the contract would be allowed but is more
tooling than seven endpoints need. Generating or copying them from backend Go source isn't
an option.

**D5. UUIDs**

- a) `github.com/google/uuid` (`uuid.NewRandom`, which uses crypto/rand).
- b) A hand-rolled v4: 16 bytes from `crypto/rand`, then set the version and variant bits.

**Chosen: a.** It's well known and saves owning bit-twiddling code. The test checks the
format against the spec's pattern either way.

**D6. Concurrency and timeouts**

- a) Keep calls synchronous on the UI goroutine, with a 20 s client timeout.
- b) Move calls to goroutines with `app.QueueUpdateDraw` and a "Working…" page.

**Chosen: a.** It's the smallest change. It makes double submits impossible. The worst
case is a 20 s freeze, which is acceptable against a local backend and Anvil.

**D7. Amount input: resolved (answer 5)**

Wei, digits only, with no int64 cap. ETH is shown on the confirmation and the balance
only.

**D8. Send confirmation**

The send form gets a confirmation modal: source, destination, and amount in wei and ETH,
with **[Confirm] [Cancel]**. The request is built and signed on Confirm, and sent at once,
as the guide asks.

**D9. Expiry window**

**Chosen: 120 s**, as in the guide. It leaves room for a manual "retry same request" and
stays well under the server's 300 s limit.

---

## 5. Phased tasks

Each phase ends with `go build ./...`, `go vet ./...` and `go test ./...` passing (apart
from the planned red in Phase 1), plus the manual check listed. All phases land on
`feature/backend-update`. Nothing merges to `main` until Phase 5 is done.

Manual checks run against a local backend and Anvil, each as its own process. Anvil is
used only through its own tools (for example `cast`) to fund addresses and check
balances. The app reaches only the backend API.

### Phase 0: housekeeping

- [x] Add `.gitignore` with `/.delsign/` and `*.pem` (D1).
- [x] `go mod tidy`, so direct dependencies stop being marked `// indirect`. Kept the
      `go 1.20` directive.
- Adding `github.com/google/uuid` moved to Phase 4, where `BuildSend` first imports it.
  Until then `go mod tidy` would only remove it again.

**Done when:** it builds, the existing test passes, and `git check-ignore .delsign/keys/x/current.pem`
confirms the file is ignored.

### Phase 1: signing test first

- [x] `keys/vectors_test.go` as in §3, "Signing", checks 1-7.
- [x] Stubs, so it compiles:
  - `api.SendRequest` with the six fields and their JSON tags;
  - `api.SendMessage` returning `""`;
  - `keys.BuildKeyReg` and `send.BuildSend` returning zero values.
- [x] Turn `keys.Sign`'s `panic` into a returned error (and refuse a nil key). Its two
      callers (`keys/keypages.go`, `send/send.go`) now show or return the error; both are
      rewritten in Phases 3-4.

**Done when:** checks 1-3 pass for all six signatures, and checks 4-7 fail on the stubs
and nowhere else.

Result (2026-11-02): `TestVectorSignatures` passes (6/6 signatures and both public keys).
`TestVectorSendMessage`, `TestVectorKeyReg` and `TestVectorBuildSend` fail only on the stubs.
A throwaway implementation of `SendMessage` (reverted) turned `TestVectorSendMessage` green,
so that test isn't vacuous. Until Phases 3-4 land, `go test ./...` is expected to fail in
`delsignrepl/keys`; don't merge in between.

### Phase 2: API client foundation

- [x] `api.Client`:
  - base URL from `DELSIGN_API_URL`;
  - a `*http.Client` with a 20 s timeout;
  - bearer token on every call;
  - `Content-Type` only with a body;
  - every body read and closed;
  - `api.Error{Status, Text}`;
  - `DoRaw` for resending exact bytes.
- [x] Move the seven calls onto it, removing every hardcoded URL (§3, "Connecting").
- [x] **Fix `GET /wallets` to expect 200** (`wallets/address.go:80`), and stop dropping its
      error (`:12`).
- [x] Add `AuthPing()` and `GetUser(email)`.
- [x] A shared modal helper that maps `api.Error` to the messages in §3, "Errors and
      retries", including the 403 → authping check.
- [x] Tests with `httptest.Server`. The handlers are written from `openapi.yaml`'s status
      codes and examples, with no backend imports:
  - headers are correct;
  - bodyless POSTs send no `Content-Type`;
  - a non-2xx response becomes `api.Error` with the trimmed text;
  - a timeout is a transport error, not an `api.Error`;
  - `[]` and `null` both give `len == 0` for `/wallets` and `/walletctx`.

**Done when:** tests pass. Manually: create a wallet, generate an address (works again), and
the walletctx list works.

Result (2026-11-02):
- `api` and `ui` tests pass. `keys` is still red only on the Phase 1 stubs.
- **Manual check not yet run:** no backend was listening on `localhost:3010`.
- Files and choices:
  - the client lives in `api/client.go` and `api/endpoints.go`;
  - errors are typed: `*api.Error` for non-2xx, `*api.NetworkError` for no response;
  - error messages go through a new `ui` package (`ui.Describe`, `ui.ShowError`);
  - `wallets.WalletAddressPair` is now an alias of `api.WalletAddressPair`.
- **Side effect:** balance works again. The contract-typed `api.Balance` has a string
  `amount`, parsed with `big.Int.SetString(s, 10)` for now. Phase 4 swaps in `ParseWei`.
- **Still broken until Phase 4:** sends still post the old payload.
- **Still to come in Phase 3:** a 403 that `/authping` confirms as a token problem says
  "Log in again", but routing to the token page arrives with the "Log in again" menu item.

### Phase 3: tokens and keys

- [ ] Tokens (D3):
  - `ParseUnverified`;
  - require `email` and `exp`;
  - `authping` on save;
  - raise the input cap to 4096;
  - the "Log in again" menu item;
  - cheat code driven by `DELSIGN_DEV_JWT_SECRET`, with the hardcoded `"secret"` removed
    from `token/input.go:79` and `keys/keypages.go:67`.
- [ ] Key store (D1):
  - PKCS #8 PEM, `0600`, atomic write, in `DELSIGN_KEY_DIR` (default `.delsign/keys`);
  - refuse files that group or others can read;
  - load at login for the token's email;
  - `Decode`/`Encode` return errors instead of discarding them.
- [ ] Key generation makes `current` or `pending` and never overwrites `current`. Show the
      lost-key warning.
- [ ] `api.KeyReg.Sig4Rotate` with `omitempty`. `keys.BuildKeyReg`.
- [ ] Register flow as in D2, including 409 → reload and re-run once, and 403 → authping
      check.
- [ ] A read-only key check after login. A menu line or modal says whether the local key
      matches the registered one.
- [ ] Tests:
  - vector checks 5 and 6 pass;
  - store round trip and file mode;
  - a group-readable key file is refused;
  - a key made by `openssl genpkey` (a fixture made in the test with `x509.MarshalPKCS8PrivateKey` + PEM, the same format) loads;
  - the D2 flow against a fake server covering 404 → register, rotate, a lost-response
    promote, a mismatch, and 409 → reload;
  - a token with no `exp` is refused;
  - with `DELSIGN_DEV_JWT_SECRET` unset, the cheat checkbox is hidden.

**Done when:** tests pass. Manually, with a fresh (backend-reset) email and the cheat code:

- register;
- quit and restart: the key loads from `.delsign/keys` and the check says "matches";
- rotate: the check still says "matches", now for the new key;
- `git status` shows no key files.

### Phase 4: amounts and sends

- [ ] `api.ParseWei`, `api.FormatEther` and `api.ValidAddress`. Balance decodes `amount`
      as a string and shows wei and ETH. Delete `weiToEther`.
- [ ] Send form:
  - amount input in wei, digits only (D7);
  - destination is trimmed and validated before signing;
  - zero and empty amounts are refused.
- [ ] Confirmation modal (D8).
- [ ] `go get github.com/google/uuid` (D5).
- [ ] `api.SendMessage` and `send.BuildSend` (D9, expiry 120 s), marshalled once, bytes
      kept.
- [ ] The send outcome flow from §3, "Errors and retries": 200, no response with retry of
      the same bytes until expiry, 409, 502 with "Edit and send again", 401 with the key
      check, 400 and 403.
- [ ] Tests:
  - vector checks 4 and 7 pass;
  - `ParseWei` rejects `-1`, `007`, `+3`, `1e3`, `""` and 2^256, and accepts 2^256-1;
  - `FormatEther` gives exact results for 1 wei, 0.9 ETH and a large amount;
  - `BuildSend` gives a UUID matching the spec pattern and `expiry == now+120`, and the
    signature verifies with the public key;
  - the send flow against a fake server: a retry sends identical bytes; a 409 never leads
    to a new id; a 502 leads to a new id only after another confirmation.

**Done when:** tests pass, with no red left from Phase 1. Manually:

- fund one of your addresses from an Anvil dev account using `cast send`;
- the balance shows wei and ETH;
- send 0.001 ETH (`1000000000000000` wei) to another of your addresses and get a `txnid`;
- the balance changes after mining (Anvil mines straight away by default);
- a send with insufficient funds shows the 502 text;
- stopping the backend mid-send shows the "no response" flow.

### Phase 5: wrap-up

- [ ] Update `readme.md`:
  - **Configuration:** `DELSIGN_API_URL`, `DELSIGN_KEY_DIR`, `DELSIGN_DEV_JWT_SECRET`.
  - **Dev tokens (cheat code):**
    - set `DELSIGN_DEV_JWT_SECRET` to the backend's `DELSIGN_JWT_SECRET`;
    - enter an email and tick "Apply Cheat Code";
    - Save;
    - "Log in again" after 8 h.
  - **Key files:**
    - where they live and that they're gitignored;
    - how to create one, in the app or with `openssl genpkey` + `chmod 600`;
    - there's no way to download one;
    - a lost key needs a backend-owner reset;
    - rotation through "Generate key" then "Register key".
  - **Dev chain:** replace the Ganache notes (`readme.md:25-36`) with Anvil: start it, fund
    an app address with `cast send … --value …`, check a balance with `cast balance`. Check
    the exact commands against the installed Foundry version when writing them.
- [ ] A last pass for leftovers: `grep -rn 'localhost:3010\|\*big.Int.*json\|ParseInt\|float64\|"secret"'`.
- [ ] A boundary check:
  - `go.mod` and `go.sum` have no `require` or `replace` for the delsignbackend module;
  - `go list -deps ./...` shows only this module, the standard library and the agreed
    third-party libraries;
  - every outbound call goes through `api.Client`, and nothing else uses `net/http`.
- [ ] Send gaps 6.1 (look up a send by `requestId`) and 6.3 (transaction status) to the
      backend owner as non-blocking requests.

**Done when:** the full manual run from Phases 2-4 passes in one session, after a restart,
and after a rotation. A reader can set the app up from the readme alone.

---

## 6. Notes and decision log

- **2026-11-02.** Surveyed the repo against the export of backend `e0c51ee`. Findings:
  - **Already broken:** address generation (`GET /wallets` expects 201), balance (string
    `amount`) and send (old message, numeric `amount`).
  - **Breaks on second use:** registration, because there's no key persistence and no
    `sig4rotate`.
- **2026-11-02.** I checked that the existing `keys.Sign` and `keys.Encode` already
  reproduce every vector signature and public key. The RSA code stays; message and body
  construction change.
- **2026-11-02.** CORS and the dev proxy don't apply, because this is a Go client, not a
  browser.
- **2026-11-02.** CLAUDE.md's "`bigint` in code" means `*big.Int` in this Go codebase.
- **2026-11-02.** Constraint from review: the app interacts with the backend only through
  its API and may not import or reuse backend modules. The app had no such dependency.
  "Backend" is read as including its database and its Anvil node. Added the boundary
  section and the Phase 5 check.
- **2026-11-02.** Review answers:
  - the dev chain is **Anvil**, not Ganache;
  - a `0600` key file is fine for dev; gitignore it and document it in the readme (D1);
  - assume the backend owner resets locked-out accounts;
  - keep the cheat code as a documented dev feature (D3);
  - there's no production environment;
  - keep wei (D7).

  On question 6, my conclusion is that no backend change is needed before starting (see
  the table at the top). Defaults chosen while folding these in:
  - the key directory is `.delsign/keys` under the working directory, overridable with
    `DELSIGN_KEY_DIR`;
  - `*.pem` is ignored as a backstop;
  - group- or world-readable key files are refused;
  - the cheat-code secret comes from `DELSIGN_DEV_JWT_SECRET`, replacing the hardcoded
    `"secret"`;
  - the `DELSIGN_ALLOW_INSECURE` and HTTPS-only rule are dropped, since there's no
    production environment.
- **If a production environment appears later** (out of scope now), it will need:
  - HTTPS, through a TLS-terminating reverse proxy in front of the plain-HTTP backend;
  - the proxy to pass `Authorization` and bodies through unchanged and keep `/api/v1`;
  - the proxy **never to retry POSTs** upstream, since a retried send turns into a 409;
  - an upstream timeout at least as long as the client's 20 s;
  - a client rule refusing plain `http` to non-loopback hosts;
  - the cheat code to be unavailable there;
  - a fresh look at server time for `expiry` (gap 6.4) and at key storage (D1).

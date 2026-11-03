# Requests for the delsign backend owner

From the delsignrepl client, after catching up with the API exported from backend
`e0c51ee` (`feature/refresh`). Neither request blocks the client, which works today by
telling the user to check the balance. Both would replace that guesswork with an answer.

## 1. Look up a send by `requestId` (client gap 6.1)

**Problem.** When `POST /api/v1/wallets/send` gets no response (a timeout or a dropped
connection), or a retry of the same body gets a 409, the client can't tell whether the
transfer was broadcast. All it can do is say "the transfer may have happened; check the
balance before sending again". On a busy address, the balance doesn't settle it.

**Request.** An endpoint such as `GET /api/v1/wallets/send/{requestId}`, scoped to the
calling user, that returns:

- 404 if the backend never recorded that `requestId`, meaning nothing was sent and the
  user can safely send again;
- otherwise the outcome it recorded: the `txnid` if it broadcast, or the node error if
  broadcasting failed.

The client would call it after a no-response or a 409, instead of pointing at the balance.

## 2. Transaction status (client gap 6.3)

**Problem.** A 200 from send returns a `txnid` once the transaction is broadcast. The
client can only say "broadcast; the balance changes once it's mined". It can't say whether
the transaction was mined, or whether it reverted.

**Request.** An endpoint such as `GET /api/v1/wallets/tx/{txnid}`, again scoped to the
user's own transactions, returning `pending`, `mined` (with block number) or `failed`.

The client talks to the backend only through its API, so it can't ask the node itself.

## Not requested

- **A retry after `expiry` (gap 6.2):** the client never resends after its own expiry, so
  it doesn't matter whether the server answers 409 or 400.
- **Server time (gap 6.4):** in dev, the client, backend and Anvil share one clock. Worth
  revisiting if a remote environment appears, since `expiry` is computed on the client.

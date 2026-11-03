# delsignrepl - Demo repl for delegated signing

This project illustrates a delegated signing model. In this model, public key
encryption, in conjunction with authentication of a user via some IDP, is
used to establish the authenticity of API requests, some of which provide
access to private keys used to sign transactions for accounts/addresses owned 
by the user.

In this example, the demo code generates an RSA key pair, registering the public
key with our example backend. Wallets and addresses can be created for the user,
who must digitally sign the request to spend from an account with their private key. On the server side, the public key registered by the user is used to verify the signature, and if it checks out the parameters passed by the user are used to construct a transaction, which is then signed by the server and broadcast to the network.

The key concept is the server never has access to the user's private key, and
can build controls and policies around the use of the private key, Of course the use will have to trust the operator of the service, but any rug pull activity is
easily detectable on the public blockchain.

Note for this model something like webauthn/passkeys could be used in a key
generation and signup flow securely generate and store keys on the end user 
device, and then register the public key with the backend.

See [this page](https://webauthn.guide/) for more details on the flow.



## Running it

You need Go, a running delsign backend, and the Anvil node the backend uses. The app
talks to the backend only through its HTTP API (contract in `docs/delsign-api/`); it
never talks to Anvil. You use Anvil directly only to fund addresses and check balances.

```sh
source .delsign/env   # see Configuration
go run .
```

It's a full-screen terminal app: run it in your own terminal, not through a pipe.

### Configuration

Everything is set through environment variables:

| Variable | Default | Purpose |
|---|---|---|
| `DELSIGN_API_URL` | `http://localhost:3010` | Backend base URL, without `/api/v1` |
| `DELSIGN_KEY_DIR` | `.delsign/keys` | Where signing keys are stored (see Key files) |
| `DELSIGN_DEV_JWT_SECRET` | unset | Enables the cheat code (see Dev tokens). Dev only |

A convenient place for them is `.delsign/env`, which is gitignored:

```sh
export DELSIGN_API_URL=http://localhost:3010
export DELSIGN_DEV_JWT_SECRET='<the backend DELSIGN_JWT_SECRET>'
export ETH_RPC_URL=http://127.0.0.1:8545   # for cast, not the app
```

Never commit the JWT secret. To check the backend is reachable:
`curl -i "$DELSIGN_API_URL/api/v1/authping"` should answer 403 (no token sent).

### Dev tokens (cheat code)

The app logs in with a JWT. It reads the email from the token and checks the token with
the backend's `GET /authping` before accepting it. With a token from the backend owner,
paste it into **Token value** and press **Save**.

For development, the app can make its own token instead:

1. Set `DELSIGN_DEV_JWT_SECRET` to the backend's `DELSIGN_JWT_SECRET`. The **Apply Cheat
   Code** checkbox only appears when it's set.
2. Type your email into **Token value**.
3. Tick **Apply Cheat Code**. The email is replaced by a token signed with the secret,
   valid for 8 hours.
4. Press **Save**.

When the token expires, choose **Log in again** (`t`) from the menu. A login that fails
with a 403 usually means the secret doesn't match the backend's.

### Key files

Sends are authorised with an RSA key that lives only on your machine; the backend holds
just the public half. Keys are stored as PKCS #8 PEM files, readable only by you:

```
$DELSIGN_KEY_DIR/<email>/current.pem   the registered key
$DELSIGN_KEY_DIR/<email>/pending.pem   a replacement waiting to be registered
```

The default `.delsign/keys` is gitignored, as is any `*.pem`. The app refuses a key
file that group or others can read, and tells you to `chmod 600` it.

There are two ways to get a key:

- **In the app:** **Generate key** (`k`), then **Register key** (`r`).
- **With openssl**, then **Register key** in the app:

  ```sh
  mkdir -p .delsign/keys/you@example.com
  openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 \
    -out .delsign/keys/you@example.com/current.pem
  chmod 600 .delsign/keys/you@example.com/current.pem
  ```

There's no way to download a key from the backend. **If you lose the key file, the funds
in your addresses are locked** until the backend owner resets your account.

**Key status** (`i`) checks that this machine's key is the registered one. To replace
your key, choose **Generate key** again: it saves the new key as `pending` and never
overwrites the current one. **Register key** then switches the backend over, signing the
change with the current key, and makes the new key current. If that response is lost,
running **Register key** again finishes the switch.

### Using the app

1. Log in (see Dev tokens).
2. **Generate key** (`k`) and **Register key** (`r`), once per email.
3. **Create wallet** (`w`), then **Generate address** (`a`) and pick the wallet ID.
4. **Set wallet/address context** (`c`). Balance and Send use this pair.
5. Fund the address from Anvil (below), then **Get balance** (`b`).
6. **Send ETH** (`s`): the amount is in wei (`1000000000000000` is 0.001 ETH). A
   confirmation shows the amount in wei and ETH before anything is signed.

If a send gets no response, the transfer may or may not have happened. **Retry same
request** resends the identical request, and is offered only until the request expires
(2 minutes). Otherwise, check the balance before sending again.

To copy text such as an address, hold **Shift** while selecting with the mouse.

### Dev chain (Anvil)

The backend uses [Anvil](https://book.getfoundry.sh/anvil/) as its dev chain. Anvil
prints ten funded accounts and their private keys when it starts; by default it listens
on `http://127.0.0.1:8545` and mines each transaction straight away.

With `ETH_RPC_URL` set (see Configuration):

```sh
# fund an app address from an Anvil account
cast send 0xYourAddress --value 0.01ether --private-key <Anvil account private key>

# check a balance, in wei or ETH
cast balance 0xYourAddress
cast balance --ether 0xYourAddress

# look up a send, using the transaction ID the app shows
cast receipt <txnid>
```

The Anvil account keys are public test keys: use them only on the dev chain.

### Tests

```sh
go test ./...
```

Signing is tested against the backend's `docs/delsign-api/test-vectors.json`. Don't edit
`docs/delsign-api/`; ask for a fresh export from the backend instead.

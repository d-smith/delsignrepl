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

The app starts on the login page. It logs in with a JWT: it reads the email from the
token and checks the token with the backend's `GET /authping` before accepting it. With
a token from the backend owner, paste it into **Token** and press **Save**.

For development, the app can make its own token instead:

1. Set `DELSIGN_DEV_JWT_SECRET` to the backend's `DELSIGN_JWT_SECRET`. The **Email**
   field and the **Apply Cheat Code** button only appear when it's set.
2. Type your email into **Email**.
3. Press **Apply Cheat Code**. It puts a token for that email, signed with the secret and
   valid for 8 hours, into **Token**. Editing the email afterwards clears it.
4. Press **Save**.

After logging in you land on Home. When the token expires, choose **Log in again** (`l`)
from the menu. A login that fails with a 403 usually means the secret doesn't match the
backend's.

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

- **In the app:** open **Keys** (`k`), press `g` to generate a key, then `r` to
  register it.
- **With openssl**, then `r` on the Keys screen:

  ```sh
  mkdir -p .delsign/keys/you@example.com
  openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 \
    -out .delsign/keys/you@example.com/current.pem
  chmod 600 .delsign/keys/you@example.com/current.pem
  ```

There's no way to download a key from the backend. **If you lose the key file, the funds
in your addresses are locked** until the backend owner resets your account.

The Keys screen shows whether this machine's key is the registered one, when that was
last checked (`R` checks again), and the paths of both key files. To replace your key,
press `g` again: it saves the new key as `pending` and never overwrites the current one.
`r` then switches the backend over, signing the change with the current key, and makes
the new key current. If that response is lost, pressing `r` again finishes the switch.

### Using the app

The screen has four parts:

- **Header:** who you're logged in as, the key status, and the wallet, address and
  balance that Send uses. It's always up to date.
- **Menu** on the left, with a shortcut key for each item.
- **Content:** the open screen. Home shows the session in full and the next step to take.
- **Footer:** the last message, then the key hints.

Menu shortcuts work on Home. Elsewhere, **Esc** goes back (closing a dialog first), and
**Esc** from any screen eventually gets you Home. **Tab** moves between fields and
buttons, **Enter** selects, and **?** shows all the keys.

A first session:

1. Log in (see Dev tokens).
2. **Keys** (`k`): `g` generates a key, `r` registers it. Once per email.
3. **Wallets** (`w`): `n` creates a wallet, `a` adds an address to the selected wallet,
   and **Enter** makes the selected address the one Balance and Send use (marked `*`).
4. Fund the address from Anvil (below), then **Refresh** (`R`) on Home, or
   **Get balance** (`b`).
5. **Send ETH** (`s`): the amount is in wei, with its value in ETH shown as you type
   (`1000000000000000` is 0.001 ETH). **Pick from my addresses** fills in the
   destination from your other addresses. A confirmation shows the amount in wei and ETH
   before anything is signed.
6. **Activity** (`a`): this session's wallets, addresses, key changes and every send
   attempt, newest first. **Enter** shows an entry in full: from, to, amount, request
   id, outcome and txnid. The log is kept in memory only.

If a send gets no response, the transfer may or may not have happened. **Retry same
request** resends the identical request, and is offered only until the request expires
(2 minutes). Otherwise, check the balance before sending again.

### Copy and paste

- **`c`** copies the selected address on Wallets, or the selected entry's txnid (or
  address, wallet ID, key path) on Activity. After a send, **Copy txnid** does the same.
  The app copies through the terminal (OSC 52, which Windows Terminal, iTerm2, kitty
  and others support) and with `clip.exe`, `pbcopy`, `wl-copy`, `xclip` or `xsel` if one
  is installed. The footer says what it copied.
- **F2** turns the mouse off, so you can drag to select any text on screen and copy it
  with your terminal (Ctrl+Shift+C in most). F2 again turns it back on. With the mouse
  on, hold **Shift** while dragging instead.
- **Paste** into a field with your terminal's paste (Ctrl+Shift+V in most).
  Spaces and newlines around pasted text are dropped, so a copied line pastes cleanly.

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

# look up a send: c on its Activity entry copies the txnid
cast receipt <txnid>
```

The Anvil account keys are public test keys: use them only on the dev chain.

### Tests

```sh
go test ./...
```

Signing is tested against the backend's `docs/delsign-api/test-vectors.json`. Don't edit
`docs/delsign-api/`; ask for a fresh export from the backend instead.

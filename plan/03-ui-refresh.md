# Plan: refresh the terminal interface (tview polish)

## Context

The backend catch-up (PR #1, merged as `5974eb3`) left the app working but rough:

- **Flat menu.** One list of 10 single-letter actions (`main.go:17-53`), each opening its own page or modal.
- **No visible state.** Who's logged in, the key status, the wallet context and the balance are only seen by opening a modal.
- **Dead ends.** Some pages need a magic key to leave (the wallet page waits for `m`, `wallets/wallets.go:31`). Others return to hard-coded page names (`"Menu"`, `"SendEth"`).
- **Pickers.** Wallets and addresses are chosen from dropdowns labelled "Select an option (hit Enter)".
- **Unclear labels.** "Token value" doubles as the email box for the cheat code.
- **Clipboard.** Copy/paste works only with Shift+drag, and there's no record of what was sent.

The user chose to **keep tview/tcell** and restructure. Their priorities are:

1. always-visible context;
2. consistent navigation;
3. copy/paste and a session activity log.

Calls stay synchronous (plan D6 stands, no spinner work). The non-UI layers don't change: `api`, `keys` (store, `Register`, `CheckKey`), `send.Flow` and `BuildSend`, and `token.Login`. Only about 700 lines of page code are reworked.

On approval, this plan is copied to `plan/03-ui-refresh.md` in the repo, the same pattern as `plan/01`, and run phase by phase with the user committing.

## Target layout

```
+ delsign -- you@example.com -- key: matches -------------------+
| Wallet 2 | 0xAb12...9F3c | 0.049 ETH                          |   header (always on)
+------------+--------------------------------------------------+
| > Home     |                                                  |
|   Keys     |   content: the current screen                    |   sidebar + content pages
|   Wallets  |                                                  |
|   Send     |                                                  |
|   Activity |                                                  |
|   Log in   |                                                  |
+------------+--------------------------------------------------+
| Tab move  Enter select  Esc back  F2 mouse:on  ? help  q quit |   footer: key hints + last status
+---------------------------------------------------------------+
```

## Design

**1. App shell (`ui/shell.go`, new).** A `Shell` struct owns these:
- the `tview.Application`;
- the root `Flex`: header `TextView`, a body `Flex` (sidebar `List` plus content `Pages`), and footer `TextView`.

Its API:
- `Show(name, p)` displays a screen and focuses it.
- `Back()` returns to the previous screen, or Home.
- `Modal(text, buttons, done)` overlays the current screen and returns to it when closed, so no hard-coded return pages are needed.
- `Status(msg)` writes the footer's last-message slot.
- `Refresh()` redraws the header.

Global keys go in `app.SetInputCapture`:
- Esc goes back. Forms let Esc through when a dropdown is open.
- `?` opens help.
- F2 toggles the mouse.
- `q` quits, from the sidebar only, so typing in fields is safe.

**2. One modal and error path.** These are folded into `Shell.Modal` and a `Shell.Error(prefix, err)`:
- `ui.ShowError` (`ui/errors.go:83`);
- `keys.showMessage` (`keys/keypages.go:15`);
- `send.showModal` (`send/send.go`);
- the ad hoc modals in `wallets/`, `state/` and `token/`.

`Shell.Error` keeps `describeErr` and the 403 → `/authping` → login routing. The message tables and `ui/errors_test.go` are unchanged.

**3. Session and header (`state/`).**
- `appToken` (now a global in `main.go`) moves into `state` next to `Email`, `PrivateKey`, `WalletId` and `Address`.
- Add `KeyStatus keys.Status` and `Balance *big.Int`, each with the time it was fetched.
- The header shows the email, the key status, the wallet, a shortened address (`0xAb12...9F3c`) and the balance in ETH, or "-" for anything unknown.
- Values are refreshed by calling the existing functions synchronously at natural points:
  - login and register: `keys.CheckKey`;
  - setting the context, after a send, and `R` (refresh) on Home: `api.Balance` + `api.ParseWei`.
- The cycle `state` → `keys` is fine: `keys` imports `state` today for `PrivateKey`. If it becomes awkward, move the UI-only fields into a `session` struct on the `Shell`, with no package cycle.

**4. Screens.** One file each, replacing the 10 menu actions:
- **Home:** a summary of the header's details, plus what to do next (for example "No key yet: go to Keys"). `R` refreshes.
- **Keys** (`keys/keypages.go`): the status line, and the Generate and Register buttons, on one screen. It reuses `DoKeyGeneration` and `DoKeyRegistration` logic, the lost-key warning and the pending-key refusal as they are.
- **Wallets** (`wallets/`): a `tview.Table` of wallet/address pairs from `WalletContext()`.
  - Selecting a row and pressing Enter sets it as the context and refreshes the balance.
  - `n` makes a new wallet; `a` makes a new address in the selected row's wallet.
  - This replaces the Create wallet, Generate address and Set context pages and both dropdowns. Wallets with no addresses yet (from `ListWallets()`) appear as rows marked "(no address)".
- **Send** (`send/send.go`): the same form, validation, confirmation and outcome modals, using `send.Flow`, unchanged.
  - A live "= 0.001 ETH" preview under the amount field, via `api.FormatEther` on change.
  - "Pick from my addresses" opens the Wallets table in picker mode instead of the dropdown.
  - "Edit and send again" stays.
- **Activity** (`ui/activity.go`, new): an in-memory session log of wallet and address creation, key register and rotate, and every send attempt. A send entry records the time, from, to, amount, `requestId`, outcome and `txnid`. It's shown newest first in a `Table`.
- **Log in** (`token/input.go`): split into an **Email** field (shown only with the cheat code) and a **Token** field. Save stays; `token.Login` is unchanged.

**5. Copy/paste.**
- F2 toggles `EnableMouse`, so plain drag-select works with the mouse off. The footer shows `mouse:on/off`. This needs no dependency change.
- A `c` key on the Wallets and Activity tables copies the selected address or `txnid`. It tries, in order:
  - an OSC 52 escape written to the terminal (works in Windows Terminal, iTerm2 and kitty);
  - falling back to `clip.exe`, `pbcopy`, `wl-copy` or `xclip`, if found on the PATH.

  The footer confirms "copied".
- **First check in Phase 0:** whether upgrading tview/tcell to current versions gives native clipboard support (OSC 52) and bracketed paste. If it does, use them instead of the hand-written escape, and paste stops turning a trailing newline into Enter. If the upgrade breaks things, stay on the current versions and use the fallback above.

## Phases

0. **Dependencies and shell skeleton.**
   - Try the tview/tcell upgrade and record what it gives.
   - Add `ui/shell.go` with the header, sidebar, content and footer.
   - Home is a placeholder; the existing pages are mounted in the content area as they are.

   Result (2026-11-03):
   - **Upgrade:** tview `v0.42.0` and tcell `v2.13.10`. It builds and every existing test
     passes unchanged. `go mod tidy` raised the `go` line to `1.24.0` (the new versions
     need it; the installed Go is 1.27).
   - **What the upgrade gives:**
     - `tcell.Screen.SetClipboard` writes OSC 52 on any xterm-like terminal
       (`TERM=xterm-256color` here), so Phase 5 uses it instead of a hand-written escape,
       keeping `clip.exe`/`pbcopy`/`wl-copy` as a fallback;
     - `Application.EnablePaste` gives bracketed paste into input fields, now on. A paste
       goes through the field's acceptance function as a whole, so pasting `1000\n` into
       the wei field is refused outright; Phase 3 trims pasted text.
   - **Shell:** `ui/shell.go` has the header, sidebar, content and footer. The focus
     follows the content pages: Home gives it to the sidebar, any other screen takes it,
     so typing in a field never fires a sidebar shortcut. `ui.HomePage` is named `"Menu"`
     until Phase 1 replaces the named returns.
   - **main.go:** the old menu's ten actions sit in the sidebar, and the existing screens
     run unchanged in the content area. The app starts on the login page.
   - **Tests:** `ui/uitest` runs an app on tcell's simulation screen (inject keys, settle,
     read the screen). `ui/shell_test.go` covers the layout and focus; `main_test.go`
     covers starting on login, a sidebar action opening a modal in the content area, and
     OK returning to Home. They're stable over 10 runs and under `-race`.
1. **Navigation and modals.**
   - Global keys: Esc, `?`, F2, `q`.
   - Migrate every modal and error onto `Shell.Modal` and `Shell.Error`.
   - Remove the named-page returns and the `m` dead end.

   Result (2026-11-03):
   - **Shell API** (`ui/shell.go`):
     - `Show(name, p)` opens a screen (or an existing page if `p` is nil) and
       remembers the one it leaves. The history is cut back when a screen is revisited,
       so going round in circles doesn't grow it.
     - `Back`, `Home` and `Current`.
     - `Modal(text, buttons, done)` overlays the current screen. Closing it, or Esc
       (label `""`), returns there, then calls `done`.
     - `Message(text)`, and `Error(prefix, err)`, which keeps `describeErr` and the
       403 -> `/authping` -> login routing. It replaces `ui.ShowError`, which is gone.
     - `Token func() string` is set by `main.go`. `SetMouse` and `Mouse`.
   - **Global keys:**
     - **Esc** goes back everywhere, including from a text field. A modal or an open
       dropdown gets Esc first, so one Esc closes it and the next leaves the screen.
     - **`?`** opens help except while typing.
     - **F2** toggles the mouse; the footer shows `mouse:on/off`.
     - **`q`** quits from the sidebar only.
     - There's also a **Help** item in the sidebar.
   - **Migrated:** every screen now takes a `*ui.Shell` (none take `*tview.Pages` or a
     token any more):
     - keys, wallets, wallet context, send (form, picker, confirmation and outcomes),
       and login;
     - every `SwitchToPage("Menu")`/`("SendEth")` and hand-built modal is gone. Cancel is
       `Back`; finished actions go `Home`;
     - "Create wallet" is a modal, removing the `m` dead end;
     - the dropdowns are labelled "Wallet ID:" and "Wallet | address:" instead of
       "Select an option (hit Enter)".
   - **Found on the way:**
     - `Pages.RemovePage` re-focuses the content itself, so closing a modal on Home left
       the focus in Home's text instead of the sidebar. `Modal` re-applies the focus
       rule after removing itself.
     - An open dropdown moves the focus to its internal `*tview.List`. The shell treats a
       focused list other than the sidebar as an open dropdown. Screens shouldn't use
       plain `List`s; Phase 3 uses `Table`.
   - **Tests:**
     - `ui/shell_test.go`: Esc back through the history, no growth in circles, a modal
       returning to its screen (and to the sidebar on Home), Esc closing a modal, help,
       F2, `q`/`?` typed in a field, and a token error routing to login while others
       stay put;
     - `main_test.go` (fake backend): start on login, Esc from every sidebar screen, Esc
       closing an open dropdown before leaving, and the send picker returning to the
       form with the address filled in.

     All are stable over 10 runs and under `-race`.
2. **Context.** Session fields, the header and Home, refreshed at the points listed under Design 3.

   Result (2026-11-03):
   - **`state` is now a pure session package.** It imports only `api`. It holds `Email`,
     `Token` (replacing `main.go`'s `appToken`), `PrivateKey`, `WalletId`, `Address`,
     `KeyStatus`/`KeyNext`/`KeyCheckedAt`, and `Balance`/`BalanceAt`/`BalanceErr`.
     - `Login(email, token)` clears the previous user's context and key status when the
       email changes.
     - `SetContext` forgets the old balance.
     - `HeaderText`, `HomeText`, `NextStep` and `ShortAddress` render it.
   - **Import cycle avoided:** the wallet context picker moved from `state` to
     `wallets.DoSetContext`, so `state` no longer imports `wallets` or `ui`. The key status
     is stored as text (`Status.Label()` and `Status.Next()`, in `keys/keypages.go`), so
     `state` doesn't need the `keys.Status` type. `keys/keyreg.go` is unchanged.
   - **Header and Home are recomputed before every draw** (`Shell.SetHeaderFunc` and
     `SetHomeFunc` via `app.SetBeforeDrawFunc`), so they can't go stale whichever screen
     changed the session. Home's text is only reset when it changes, to keep its scroll.
   - **Refresh points:**
     - key status: after login, Key status, generating a key, registering, and Refresh;
     - balance: after setting the context, Get balance, a successful send, and Refresh.

     **Refresh** is a sidebar action on `R`, since the sidebar has the focus on Home. It
     reports "Refreshed at ..." in the footer.
   - **Home's "Next" step**, in order:
     1. log in;
     2. the key's step (generate, register, finish a rotation, or the mismatch warning);
     3. create and choose a wallet and address;
     4. fund it from Anvil if the balance is 0;
     5. send.
   - **Test harness fix:** tview reads keys and queued updates from separate channels,
     so `Settle` alone could finish while typed keys were still waiting, which broke a
     test typing a ~250-character JWT. `uitest` now counts key events through the app's
     input capture, and `Key` and `Type` wait until every injected key has been handled.
   - **Tests:**
     - `state/state_test.go`: header, Home and next step across the session's stages, a
       failed balance fetch, a login as someone else clearing the session, and
       `SetContext` dropping the balance;
     - `main_test.go`, `TestHeaderAndHomeShowTheSession`: type a JWT into the real login
       form, see "key: none on this machine" and its next step, save a key outside the
       app and press `R` to see "not registered", then set the context and see the
       wallet, short address and 0.049 ETH in the header, with the full address and wei
       on Home.

     All are stable over 10 runs and under `-race`.
3. **Wallets and Send.** The table with picker mode; the Send screen with the ETH preview.

   Result (2026-11-03):
   - **Wallets screen** (`wallets/wallets.go`, sidebar `w`):
     - a table of every address with its wallet (`GET /walletctx`), then every wallet
       with no address yet (`GET /wallets`), with the context marked `*`;
     - **Enter** makes the row the context and fetches its balance, staying on the table;
       on a wallet with no address it says to press `a`;
     - **n** creates a wallet and selects it; **a** creates an address in the selected
       row's wallet and selects it;
     - an empty account shows "No wallets yet: press n".

     It replaces the Create wallet, Generate address and Set wallet context items, their
     pages and both dropdowns (`wallets/address.go` and `wallets/context.go` are gone).
     The app has no dropdowns left.
   - **Picker mode** (`wallets.PickAddress`): the same table listing the user's addresses
     except the source. Enter picks and goes back; Esc goes back without picking.
   - **Send form:**
     - From is a read-only line;
     - To and Amount have placeholders;
     - a live preview under the amount shows `= 0.001 ETH` as you type, or what's wrong
       ("no leading zeros", "more than the largest amount", "must be more than 0");
     - a **Pick from my addresses** button replaces the "Use Wallet Address" checkbox;
     - the confirmation and outcome flow is unchanged.
   - **Paste:** `ui.TrimPaste` wraps an input field so pasted text is trimmed. A copied
     line with a trailing newline used to be refused whole by the digits-only amount
     check. To and Amount use it. The wrapper has an `Unwrap` method, and the shell looks
     through it, so `?` is still typed rather than opening help.
   - **Found on the way:**
     - a text view in a form is only skipped by Tab when it isn't scrollable, which text
       views are by default. The From line and the preview set `SetScrollable(false)`;
     - footer status messages are cut off at the terminal width, so they use short
       addresses;
     - `Shell.Token` now defaults to returning `""` instead of nil, which panicked;
     - running every package under `-race` for the first time found test-only races in
       two older fakes (`send/flow_test.go` and `keys/register_test.go`). They record
       requests on the server's goroutine, and when they drop a connection nothing orders
       that against the test's reads. Both now take a mutex. No app code was affected.
   - **Tests:**
     - `ui/uitest/backend.go`: a stateful fake of the wallet endpoints, shared by the
       `main` and `wallets` tests;
     - `wallets/wallets_test.go`: the table's rows and order, Enter setting the context and
       balance (not on an empty wallet), `n` then `a` then Enter, the empty account, and
       the picker excluding the source, picking and Esc;
     - `main_test.go`: the live preview, and the picker returning to the form with the
       destination filled in and the amount kept;
     - `ui/shell_test.go`: Esc closing an open dropdown (moved from `main_test.go`), and a
       real bracketed paste via `uitest.Paste`. Without the wrapper the same paste is
       refused, so the test isn't vacuous. `?` is typed in a wrapped field;
     - `send/send_test.go`: `amountPreview` cases.

     All are stable over 10 runs, and every package is clean under `-race`.
4. **Keys and Log in.** The combined Keys screen; separate Email and Token fields.

   Result (2026-11-04):
   - **Keys screen** (`keys.ShowKeys`, sidebar `k`): the email, status label and full
     description, when it was checked, the current and replacement key file paths (or
     "none"), and the next step. Buttons and keys: **g** generate (a replacement if a key
     exists, as before), **r** register or rotate, **R** check again. Opening it checks
     the key. The text is a `ui.LiveText`, recomputed when drawn, so it follows each
     action's result under its dialog. It replaces the Generate key, Register key and Key
     status items. The generate and register logic, the lost-key warning and the
     pending-key refusal are unchanged.
   - **Login** no longer opens a key status dialog: it goes Home, where the header and
     Next show the result, and the footer says "Logged in as ...".
   - **Hints:** `Status.Next()` (shown on Home) names the screen and the key, e.g. "open
     Keys (k), generate a signing key (g), then register it (r)". `Status.String()` and
     `ErrNoKey` lost their `(k)`/`(r)`, which were wrong outside the old menu. The Send
     screen's no-key message points to Keys (k). Help lists the Keys and Wallets keys.
   - **Login form** (`token/input.go`): a **Token** field (trimmed paste, placeholder)
     and, with the dev secret, an **Email** field before it and an **Apply Cheat Code**
     button before Save. The button replaces the checkbox, which was broken on tview
     0.42: its change handler runs before the value is stored, so un-ticking it after an
     invalid email didn't stick. Editing the email takes back a token minted for the old
     one. Saving an empty token with the dev secret says what to do.
   - `ui.LiveText` also backs Home, replacing the shell's own Home text cache.
   - **Tests:**
     - `ui/uitest/backend.go` records key registrations (`POST /keyreg`, `GET /users`),
       without checking signatures (`keys/register_test.go` does);
     - `main_test.go`: the Keys screen through generate, register, replace and rotate,
       with the status and paths after each, and `R`; the cheat code login, including
       the empty-email message and editing the email clearing the token (fails with the
       clearing removed); login landing on Home; Keys in the Esc-from-every-screen test.

     All are stable over 10 runs, and every package is clean under `-race`.
   - **For Phase 6:** the readme still describes the checkbox and the Generate key (`k`)
     / Register key (`r`) / Key status (`i`) items.
5. **Activity and copy.** The activity log, the copy keys and the mouse toggle hint.

   Result (2026-11-04):
   - **Copy** (`ui/clipboard.go`, `Shell.Copy`): writes through the terminal with tcell's
     `SetClipboard` (OSC 52; the screen is captured in the before-draw hook) *and* runs
     the first clipboard command on the PATH (`clip.exe`, `pbcopy`, `wl-copy`, `xclip`,
     `xsel`), since neither can tell whether the other worked. `clip.exe` is on the PATH
     here. The footer says what was copied and how, or why the command failed.
     `ui.SystemCopy` is a variable so tests never touch the real clipboard.
   - **Activity** (`ui/activity.go`, sidebar `a`): an in-memory log on the shell
     (`Shell.Log`, `Entry`), shown newest first in a table: time, what, one-line summary.
     **Enter** opens the entry on its own screen, one value per line, unwrapped (a dialog
     wrapped 66-character txnids); **c** copies its value there or in the table.
     Logged: wallet and address creation (copy the id or address), key generated,
     replacement generated, registered, replaced, finished replacement (copy the file
     path), and every send attempt and retry: from, to, amount in ETH and wei, request
     id, outcome, txnid, error (copy the txnid, or the request id if none).
   - **Wallets:** **c** copies the selected address.
   - **Send:** the "Broadcast" dialog has a **Copy txnid** button and says Activity keeps
     the txnid.
   - **Mouse hint:** F2 says in the footer what the new mode means ("Mouse off: drag to
     select, copy with your terminal."). The footer now puts the status before the key
     hints, so a long message cuts the fixed hints rather than itself.
   - Help lists `c` and the per-screen keys.
   - **Tests:** the fake backend answers `POST /wallets/send`. `ui/activity_test.go`: Copy
     through the terminal (the simulation screen records OSC 52) and the command, with
     the none-found and failure messages; Activity order, Enter, c, Esc. `main_test.go`:
     a send through the form, Copy txnid, then its Activity entry and details; the
     key steps logged in order. `wallets_test.go`: c on an address and on an empty
     wallet; creation logged. All are stable over 10 runs, and every package is clean
     under `-race`.
6. **Readme.** Update "Using the app" and the key hints. Note that mouse off means plain selection.

   Result (2026-11-04):
   - **Dev tokens:** the Token field, and the Email field and Apply Cheat Code button
     (replacing the checkbox and "Token value"); login lands on Home.
   - **Key files:** the Keys screen's `g`, `r` and `R` replace Generate key, Register key
     and Key status.
   - **Using the app** rewritten: the header, menu, content and footer; Esc, Tab, Enter
     and `?`; a first session through Keys, Wallets (`n`, `a`, Enter), funding and
     `R`, Send (live ETH preview, Pick from my addresses) and Activity.
   - **Copy and paste** (new): `c` and Copy txnid, OSC 52 plus the clipboard command,
     F2 for plain drag-selection, and trimmed paste.
   - The Anvil section's `cast receipt` points to copying the txnid from Activity.
   - Checked that no old menu names or keys are left, and the readme is ASCII only.


## Critical files

- **New:** `ui/shell.go`, `ui/activity.go`, `ui/clipboard.go`, and a help text.
- **Reworked:** `main.go` (much smaller), `keys/keypages.go`, `wallets/*.go`, `state/state.go`, `send/send.go` (UI parts only), `token/input.go`.
- **Unchanged:**
  - `api/*`;
  - `keys/{keys,store,keyreg}.go`;
  - `send/{build,flow}.go`;
  - `token/token.go`;
  - `ui/errors.go`'s `describeErr`;
  - every existing test.

## Verification

- `go build ./...`, `go vet ./...`, `go test ./...` after each phase. The existing api, keys, send, token and ui tests must stay green.
- New tests with `tcell.NewSimulationScreen` driving the app via `InjectKey`:
  - Esc from every screen returns to Home;
  - `q` in a text field types `q` rather than quitting;
  - the header text after login and after setting the context, against `httptest` fakes in the same style as `keys/register_test.go` and `send/flow_test.go`;
  - a send entry appears in Activity with its `txnid`;
  - the shortened address and ETH preview are formatted correctly;
  - the clipboard writer emits the right OSC 52 sequence (tested on an `io.Writer`).
- **Manual, against the live backend and Anvil**, as before:
  1. log in with the cheat code;
  2. Keys → generate and register (the header shows "matches");
  3. Wallets → new wallet, new address, Enter to set the context (the header shows the address and balance);
  4. fund it with `cast send`, then `R` (the balance updates);
  5. Send with the live ETH preview, then confirm (the Activity entry has the `txnid`);
  6. `c` on Activity copies the `txnid`, and `cast receipt <pasted>` works;
  7. F2 off, then drag-select an address;
  8. Esc backs out of every screen.

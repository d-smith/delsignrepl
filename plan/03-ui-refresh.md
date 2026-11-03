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
3. **Wallets and Send.** The table with picker mode; the Send screen with the ETH preview.
4. **Keys and Log in.** The combined Keys screen; separate Email and Token fields.
5. **Activity and copy.** The activity log, the copy keys and the mouse toggle hint.
6. **Readme.** Update "Using the app" and the key hints. Note that mouse off means plain selection.

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

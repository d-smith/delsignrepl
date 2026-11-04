// Package wallets is the Wallets screen (a table of the user's wallets and addresses) and
// the balance of the context address.
package wallets

import (
	"delsignrepl/api"
	"delsignrepl/state"
	"delsignrepl/ui"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// WalletsPage and PickPage are the screens' page names.
const (
	WalletsPage = "Wallets"
	PickPage    = "Pick address"
)

// Row is one line of the Wallets table: an address in a wallet, or a wallet with no
// address yet (Address "").
type Row struct {
	Wallet  int
	Address string
}

// LoadRows returns every address with its wallet (GET /walletctx), then every wallet
// that has no address yet (GET /wallets), in the order the backend gives them.
func LoadRows(c *api.Client) ([]Row, error) {
	pairs, err := c.WalletContext()
	if err != nil {
		return nil, err
	}
	ids, err := c.ListWallets()
	if err != nil {
		return nil, err
	}
	var rows []Row
	hasAddress := map[int]bool{}
	for _, p := range pairs {
		rows = append(rows, Row{p.WalletId, p.Address})
		hasAddress[p.WalletId] = true
	}
	for _, id := range ids {
		if !hasAddress[id] {
			rows = append(rows, Row{Wallet: id})
		}
	}
	return rows, nil
}

// table is a Wallets table: a header row, then one row per Row.
type table struct {
	*tview.Table
	rows []Row
}

func newTable(title string) *table {
	t := &table{Table: tview.NewTable().SetFixed(1, 0).SetSelectable(true, false)}
	t.SetBorder(true).SetTitle(title).SetTitleAlign(tview.AlignLeft)
	return t
}

// fill redraws the table from rows, marking the context address, and selects the first
// row matching sel (or the first row).
func (t *table) fill(rows []Row, emptyText string, sel func(Row) bool) {
	t.Clear()
	t.rows = rows
	head := func(col int, text string) {
		t.SetCell(0, col, tview.NewTableCell(text).SetSelectable(false).
			SetTextColor(tcell.ColorYellow).SetAttributes(tcell.AttrBold))
	}
	head(0, " ")
	head(1, "Wallet")
	head(2, "Address")
	if len(rows) == 0 {
		t.SetCell(1, 2, tview.NewTableCell(emptyText).SetSelectable(false))
		return
	}
	selected := 1
	for i, r := range rows {
		mark, addr := " ", r.Address
		if r.Address == "" {
			addr = "(no address yet)"
		} else if r.Address == state.Address {
			mark = "*"
		}
		t.SetCell(i+1, 0, tview.NewTableCell(mark))
		t.SetCell(i+1, 1, tview.NewTableCell(fmt.Sprintf("%6d", r.Wallet)).SetAlign(tview.AlignRight))
		t.SetCell(i+1, 2, tview.NewTableCell(addr).SetExpansion(1))
		if sel != nil && sel(r) && selected == 1 {
			selected = i + 1
		}
	}
	t.Select(selected, 0)
}

// current returns the selected row, if any.
func (t *table) current() (Row, bool) {
	row, _ := t.GetSelection()
	if row < 1 || row > len(t.rows) {
		return Row{}, false
	}
	return t.rows[row-1], true
}

const walletsHelp = "Enter use for balance and send   n new wallet   a new address in this wallet   c copy   Esc back"

// ShowWallets opens the Wallets screen: all wallets and addresses, with keys to create
// them and to choose the context (marked *).
func ShowWallets(sh *ui.Shell) {
	c := api.New(sh.Token())
	rows, err := LoadRows(c)
	if err != nil {
		sh.Error("Error listing wallets", err)
		return
	}
	t := newTable(" Wallets ")
	const empty = "No wallets yet: press n to create one."
	t.fill(rows, empty, func(r Row) bool { return r.Address != "" && r.Address == state.Address })

	reload := func(sel func(Row) bool) bool {
		rows, err := LoadRows(c)
		if err != nil {
			sh.Error("Error listing wallets", err)
			return false
		}
		t.fill(rows, empty, sel)
		return true
	}

	t.SetSelectedFunc(func(int, int) {
		r, ok := t.current()
		if !ok {
			return
		}
		if r.Address == "" {
			sh.Status(fmt.Sprintf("Wallet %d has no address yet: press a to create one.", r.Wallet))
			return
		}
		state.SetContext(r.Wallet, r.Address)
		t.fill(t.rows, empty, func(x Row) bool { return x == r })
		if err := RefreshBalance(sh.Token()); err != nil {
			sh.Error("Using this address, but fetching its balance failed", err)
			return
		}
		sh.Status(fmt.Sprintf("Using wallet %d %s for balance and send.", r.Wallet, state.ShortAddress(r.Address)))
	})

	t.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch ev.Rune() {
		case 'n':
			id, err := c.CreateWallet()
			if err != nil {
				sh.Error("Error creating wallet", err)
				return nil
			}
			sh.Log(ui.Entry{What: "Created wallet", Summary: fmt.Sprintf("wallet %d", id),
				Details: []string{fmt.Sprintf("Wallet %d", id)}, Copy: fmt.Sprint(id), CopyIs: "wallet id"})
			if reload(func(r Row) bool { return r.Wallet == id }) {
				sh.Status(fmt.Sprintf("Created wallet %d. Press a to add an address to it.", id))
			}
			return nil
		case 'a':
			r, ok := t.current()
			if !ok {
				sh.Message("Create a wallet first (n).")
				return nil
			}
			addr, err := c.CreateAddress(r.Wallet)
			if err != nil {
				sh.Error("Error creating address", err)
				return nil
			}
			sh.Log(ui.Entry{What: "Created address", Summary: fmt.Sprintf("%s in wallet %d", addr, r.Wallet),
				Details: []string{"Address " + addr, fmt.Sprintf("Wallet  %d", r.Wallet)},
				Copy:    addr, CopyIs: "address " + state.ShortAddress(addr)})
			if reload(func(x Row) bool { return x.Address == addr }) {
				sh.Status(fmt.Sprintf("Created address %s in wallet %d. Press Enter to use it.",
					state.ShortAddress(addr), r.Wallet))
			}
			return nil
		case 'c':
			if r, ok := t.current(); ok && r.Address != "" {
				sh.Copy(r.Address, "address "+state.ShortAddress(r.Address))
			} else {
				sh.Status("This wallet has no address to copy yet.")
			}
			return nil
		}
		return ev
	})

	sh.Show(WalletsPage, withHint(t, walletsHelp))
}

// PickAddress opens the table in picker mode: the user's addresses except exclude.
// Enter calls onPick with the address and goes back; Esc goes back without picking.
func PickAddress(sh *ui.Shell, exclude string, onPick func(address string)) {
	rows, err := LoadRows(api.New(sh.Token()))
	if err != nil {
		sh.Error("Error listing your addresses", err)
		return
	}
	var pickable []Row
	for _, r := range rows {
		if r.Address != "" && r.Address != exclude {
			pickable = append(pickable, r)
		}
	}
	if len(pickable) == 0 {
		sh.Message("You have no other addresses to send to. Create one in Wallets (w).")
		return
	}
	t := newTable(" Send to one of my addresses ")
	t.fill(pickable, "", nil)
	t.SetSelectedFunc(func(int, int) {
		if r, ok := t.current(); ok {
			sh.Back()
			onPick(r.Address)
		}
	})
	sh.Show(PickPage, withHint(t, "Enter pick this address   Esc back"))
}

// withHint puts a one-line key hint under p.
func withHint(p tview.Primitive, hint string) tview.Primitive {
	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p, 0, 1, true).
		AddItem(tview.NewTextView().SetText(" "+hint).SetTextColor(tcell.ColorGray), 1, 0, false)
}

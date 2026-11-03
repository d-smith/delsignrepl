package wallets

import (
	"delsignrepl/api"
	"delsignrepl/ui"
	"fmt"
)

// DoWalletGeneration creates a wallet and shows its ID.
func DoWalletGeneration(sh *ui.Shell) {
	id, err := api.New(sh.Token()).CreateWallet()
	if err != nil {
		sh.Error("Error creating wallet", err)
		return
	}
	sh.Message(fmt.Sprintf("Created wallet %d.\n\nAdd an address to it with Generate address (a).", id))
}

// WalletAddressPair is one (wallet, address) entry from GET /api/v1/walletctx.
type WalletAddressPair = api.WalletAddressPair

// GetWalletsAndAddresses returns all the user's wallet/address pairs (empty if none).
func GetWalletsAndAddresses(appToken string) ([]WalletAddressPair, error) {
	return api.New(appToken).WalletContext()
}

func PairsToStrings(pairs []WalletAddressPair) []string {
	strings := make([]string, len(pairs))
	for i, v := range pairs {
		strings[i] = fmt.Sprintf("Wallet %6d | %s", v.WalletId, v.Address)
	}
	return strings
}

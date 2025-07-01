package solana

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/gagliardetto/solana-go"
	solanatoken "github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/perun-network/perun-solana-backend/channel"
	pchannel "perun.network/go-perun/channel"

	soladjudicator "github.com/perun-network/perun-solana-backend/channel/adjudicator"
	solfunder "github.com/perun-network/perun-solana-backend/channel/funder"
	solclient "github.com/perun-network/perun-solana-backend/client"
	solwallet "github.com/perun-network/perun-solana-backend/wallet"
)

const (
	AlicePrivateKeyPath = "solana/scripts/accounts/alice.json"
	BobPrivateKeyPath   = "solana/scripts/accounts/bob.json"
	PerunAddressPath    = "solana/scripts/addresses/perun_address.txt"
	AddressesPath       = "solana/scripts/addresses/"
)

var (
	TokenProgramID = solanatoken.ProgramID
)

type Setup struct {
	Accs    []*solwallet.Account
	Wallets []*solwallet.EphemeralWallet
	Cbs     []*solclient.ContractBackend
	Funders []*solfunder.Funder
	Adjs    []*soladjudicator.Adjudicator
	Asset   pchannel.Asset
}

func NewExampleSetup(sks []string, ccaddrs [][20]byte) (*Setup, error) {
	// Create a new RPC client:
	client := rpc.New(rpc.LocalNet_RPC)

	// Parse the keypair of Alice and Bob:
	alicePrivateKey, err := solana.PrivateKeyFromSolanaKeygenFile(AlicePrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Alice's private key: %w", err)
	}
	bobPrivateKey, err := solana.PrivateKeyFromSolanaKeygenFile(BobPrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Bob's private key: %w", err)
	}

	perunAddress, err := readProgramIDFromFile(PerunAddressPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Perun address from file: %w", err)
	}

	// Print the public keys of Alice and Bob:
	fmt.Printf("Alice Public Key: %s\n", alicePrivateKey.PublicKey())
	fmt.Printf("Bob Public Key: %s\n", bobPrivateKey.PublicKey())
	fmt.Printf("Perun Address: %s\n", perunAddress)

	// Fetch balances
	aliceBalanceResp, err := client.GetBalance(
		context.TODO(),
		alicePrivateKey.PublicKey(),
		rpc.CommitmentFinalized, // Use finalized commitment to ensure the balance is up-to-date
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get Alice's balance: %w", err)
	}
	fmt.Printf("Alice SOL Balance: %d lamports\n", aliceBalanceResp.Value)

	bobBalanceResp, err := client.GetBalance(
		context.TODO(),
		bobPrivateKey.PublicKey(),
		rpc.CommitmentFinalized, // Use finalized commitment to ensure the balance is up-to
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get Bob's balance: %w", err)
	}
	fmt.Printf("Bob SOL Balance: %d lamports\n", bobBalanceResp.Value)

	// Create SOLAsset
	solAsset := channel.NewSOLSolanaCrossAsset()

	// Create wallets for Alice and Bob
	aliceWallet := solwallet.NewEphemeralWallet()
	aliceAcc, err := solwallet.NewAccount(sks[0], alicePrivateKey.PublicKey(), ccaddrs[0])
	if err != nil {
		return nil, fmt.Errorf("failed to create Alice's account: %w", err)
	}
	err = aliceWallet.AddAccount(aliceAcc)
	if err != nil {
		return nil, fmt.Errorf("failed to add Alice's account to wallet: %w", err)
	}

	bobWallet := solwallet.NewEphemeralWallet()
	bobAcc, err := solwallet.NewAccount(sks[1], bobPrivateKey.PublicKey(), ccaddrs[1])
	if err != nil {
		return nil, fmt.Errorf("failed to create Bob's account: %w", err)
	}
	err = bobWallet.AddAccount(bobAcc)
	if err != nil {
		return nil, fmt.Errorf("failed to add Bob's account to wallet: %w", err)
	}

	// Create contract backends for Alice and Bob
	aliceScfg := solclient.NewSignerConfig(
		&alicePrivateKey,
		aliceAcc.Participant(),
		aliceAcc,
		solclient.NewTxSender(rpc.New(rpc.LocalNet_RPC)),
		rpc.LocalNet_RPC, // Use the localnet RPC URL
	)
	aliceCB := solclient.NewContractBackend(*aliceScfg, 6)

	bobScfg := solclient.NewSignerConfig(
		&bobPrivateKey,
		bobAcc.Participant(),
		bobAcc,
		solclient.NewTxSender(rpc.New(rpc.LocalNet_RPC)),
		rpc.LocalNet_RPC, // Use the localnet RPC URL
	)
	bobCB := solclient.NewContractBackend(*bobScfg, 6)

	// Create funders for Alice and Bob
	solAddr := solana.PublicKey{}
	aliceFunder := solfunder.NewFunder(aliceCB, perunAddress, []solana.PublicKey{solAddr})
	bobFunder := solfunder.NewFunder(bobCB, perunAddress, []solana.PublicKey{solAddr})

	// Create adjudicators for Alice and Bob
	aliceAdj := soladjudicator.NewAdjudicator()
	bobAdj := soladjudicator.NewAdjudicator()

	return &Setup{
		Accs:    []*solwallet.Account{aliceAcc, bobAcc},
		Wallets: []*solwallet.EphemeralWallet{aliceWallet, bobWallet},
		Cbs:     []*solclient.ContractBackend{aliceCB, bobCB},
		Funders: []*solfunder.Funder{aliceFunder, bobFunder},
		Adjs:    []*soladjudicator.Adjudicator{aliceAdj, bobAdj},

		Asset: solAsset,
	}, nil
}

func readProgramIDFromFile(path string) (solana.PublicKey, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("reading program ID from file: %w", err)
	}
	addressStr := strings.TrimSpace(string(content))
	pubkey, err := solana.PublicKeyFromBase58(addressStr)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("invalid program ID in file: %w", err)
	}
	return pubkey, nil
}

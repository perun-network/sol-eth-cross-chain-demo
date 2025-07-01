package client

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	ethchannel "github.com/perun-network/perun-eth-backend/channel"
	ethwallet "github.com/perun-network/perun-eth-backend/wallet"
	simplewallet "github.com/perun-network/perun-eth-backend/wallet/simple"

	ethwire "github.com/perun-network/perun-eth-backend/wire"
	swire "perun.network/go-perun/wire/net/simple"

	"github.com/pkg/errors"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/channel/multi"
	"perun.network/go-perun/client"
	"perun.network/go-perun/wallet"
	"perun.network/go-perun/watcher/local"
	"perun.network/go-perun/wire"

	solchannel "github.com/perun-network/perun-solana-backend/channel"
	soladjudicator "github.com/perun-network/perun-solana-backend/channel/adjudicator"
	solfunder "github.com/perun-network/perun-solana-backend/channel/funder"
	solwallet "github.com/perun-network/perun-solana-backend/wallet"
)

// PaymentClient is a payment channel client.
type PaymentClient struct {
	perunClient *client.Client                      // The core Perun client.
	account     map[wallet.BackendID]wallet.Address // The account we use for on-chain and off-chain transactions.
	waddress    map[wallet.BackendID]wire.Address
	currency    []channel.Asset      // The currency we expect to get paid in.
	channels    chan *PaymentChannel // Accepted payment channels.
}

// SetupPaymentClient creates a new payment client.
func SetupPaymentClient(
	bus wire.Bus, // bus is used of off-chain communication.
	ethWallet *simplewallet.Wallet, // w is the wallet used for signing ethereum transactions.
	acc common.Address, // acc is the address of the account to be used for signing transactions.
	ethAddress *ethwallet.Address, // ethAddress is the address of the Ethereum account to be used for signing transactions.
	nodeURL string, // nodeURL is the URL of the blockchain node.
	chainID uint64, // chainID is the identifier of the blockchain.
	adjudicator common.Address, // adjudicator is the address of the adjudicator.
	assetAddr ethwallet.Address, // asset is the address of the asset holder for our payment channels.

	solWallet *solwallet.EphemeralWallet,
	solAccount *solwallet.Account,
	solAsset channel.Asset,
	solFunder *solfunder.Funder,
	solAdj *soladjudicator.Adjudicator,
) (*PaymentClient, error) {
	multiAdjudicator := multi.NewAdjudicator()
	watcher, err := local.NewWatcher(multiAdjudicator)
	multiFunder := multi.NewFunder()
	ccWallet := map[wallet.BackendID]wallet.Wallet{1: ethWallet, 6: solWallet}

	solPart, ok := solAccount.Address().(*solwallet.Participant)
	if !ok {
		return nil, errors.New("solana account address is not a participant")
	}

	// Create Ethereum client and contract backend.
	cb, err := CreateContractBackend(nodeURL, chainID, ethWallet)
	if err != nil {
		return nil, fmt.Errorf("creating contract backend: %w", err)
	}

	// Validate contracts.
	err = ethchannel.ValidateAdjudicator(context.TODO(), cb, adjudicator)
	if err != nil {
		return nil, fmt.Errorf("validating adjudicator: %w", err)
	}
	err = ethchannel.ValidateAssetHolderETH(context.TODO(), cb, common.Address(assetAddr), adjudicator)
	if err != nil {
		return nil, fmt.Errorf("validating adjudicator: %w", err)
	}

	// Setup funder.
	ethFunder := ethchannel.NewFunder(cb)
	ethAssetID := ethchannel.MakeLedgerBackendID(big.NewInt(int64(chainID)))
	solAssetID := solchannel.MakeCCID(solchannel.MakeContractID("6"))
	multiFunder.RegisterFunder(ethAssetID, ethFunder)
	multiFunder.RegisterFunder(solAssetID, solFunder)

	dep := ethchannel.NewETHDepositor(50000)
	ethAcc := accounts.Account{Address: acc}
	ethAsset := ethchannel.NewAsset(big.NewInt(int64(chainID)), common.Address(assetAddr))
	ethFunder.RegisterAsset(*ethAsset, dep, ethAcc)

	// Setup adjudicator.
	ethAdj := ethchannel.NewAdjudicator(cb, adjudicator, acc, ethAcc, 1000000)
	multiAdjudicator.RegisterAdjudicator(ethAssetID, ethAdj)
	multiAdjudicator.RegisterAdjudicator(solAssetID, solAdj)

	// Setup Perun client.
	solWireAddr := swire.NewAddress(solPart.String())
	ethWireAddr := &ethwire.Address{Address: ethAddress}
	addresses := map[wallet.BackendID]wire.Address{1: ethWireAddr, 6: solWireAddr}
	perunClient, err := client.New(addresses, bus, multiFunder, multiAdjudicator, ccWallet, watcher)
	if err != nil {
		return nil, errors.WithMessage(err, "creating client")
	}

	// Setup Accounts
	account := map[wallet.BackendID]wallet.Address{1: ethAddress, 6: solAccount.Address()}

	// Create client and start request handler.
	c := &PaymentClient{
		perunClient: perunClient,
		account:     account,
		waddress:    addresses,
		currency:    []channel.Asset{ethAsset, solAsset},
		channels:    make(chan *PaymentChannel, 1),
	}
	go perunClient.Handle(c, c)

	return c, nil
}

// OpenChannel opens a new channel with the specified peer and funding.
func (c *PaymentClient) OpenChannel(peer map[wallet.BackendID]wire.Address, ethAmount float64, solAmount uint64) *PaymentChannel {
	// We define the channel participants. The proposer has always index 0. Here
	// we use the on-chain addresses as off-chain addresses, but we could also
	// use different ones.
	participants := []map[wallet.BackendID]wire.Address{c.waddress, peer}

	// We create an initial allocation which defines the starting balances.
	initAlloc := channel.NewAllocation(2, []wallet.BackendID{1, 6}, c.currency[0], c.currency[1])
	log.Println("ETH amount: ", ethAmount, c.currency[0])

	log.Println("SOL amount: ", solAmount, c.currency[1])
	initAlloc.SetAssetBalances(c.currency[0], []channel.Bal{
		EthToWei(big.NewFloat(ethAmount)), // Our initial balance.
		big.NewInt(0),                     // Peer's initial balance.
	})
	initAlloc.SetAssetBalances(c.currency[1], []channel.Bal{
		big.NewInt(0),                // Our initial balance.
		big.NewInt(int64(solAmount)), // Peer's initial balance.
	})

	// Prepare the channel proposal by defining the channel parameters.
	challengeDuration := uint64(1000) // On-chain challenge duration in seconds.
	log.Println("Creating channel proposal")
	proposal, err := client.NewLedgerChannelProposal(
		challengeDuration,
		c.account,
		initAlloc,
		participants,
	)
	if err != nil {
		panic(err)
	}

	// Send the proposal.
	log.Println("Sending channel proposal", proposal)
	// ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	// defer cancel()
	ch, err := c.perunClient.ProposeChannel(context.TODO(), proposal)
	if err != nil {
		panic(err)
	}

	// Start the on-chain event watcher. It automatically handles disputes.
	log.Println("Starting dispute watcher", ch.ID())
	c.startWatching(ch)

	return newPaymentChannel(ch, c.currency)
}

// startWatching starts the dispute watcher for the specified channel.
func (c *PaymentClient) startWatching(ch *client.Channel) {
	go func() {
		err := ch.Watch(c)
		if err != nil {
			fmt.Printf("Watcher returned with error: %v", err)
		}
	}()
}

// AcceptedChannel returns the next accepted channel.
func (c *PaymentClient) AcceptedChannel() *PaymentChannel {
	log.Println("Waiting for accepted channel", c.channels)
	return <-c.channels
}

// Shutdown gracefully shuts down the client.
func (c *PaymentClient) Shutdown() {
	c.perunClient.Close()
}

func (c *PaymentClient) Addresses() map[wallet.BackendID]wallet.Address {
	return c.account
}

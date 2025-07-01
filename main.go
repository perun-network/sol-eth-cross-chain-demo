// Copyright 2025 - See NOTICE file for copyright holders.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main provides the entry point for the application.
package main

import (
	"log"

	"github.com/ethereum/go-ethereum/crypto"
	ethwallet "github.com/perun-network/perun-eth-backend/wallet"
	"perun.network/go-perun/wire"
	"perun.network/sol-eth-cross-chain-demo/eth"
	"perun.network/sol-eth-cross-chain-demo/solana"
)

const (
	chainURL = "ws://127.0.0.1:8545"

	// Private keys.
	keyDeployer = "79ea8f62d97bc0591a4224c1725fca6b00de5b2cea286fe2e0bb35c5e76be46e"
	keyAlice    = "1af2e950272dd403de7a5760d41c6e44d92b6d02797e51810795ff03cc2cda4f"
	keyBob      = "f63d7d8e930bccd74e93cf5662fde2c28fd8be95edb70c73f1bdd863d07f412e"
)

func main() {
	// Configure log flags: date/time and file/line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Deploy contracts.
	log.Println("Deploying contracts.")

	adjudicator, assetHolder := eth.DeployContracts(chainURL, 1337, keyDeployer)
	log.Println("Adjudicator:", adjudicator.Hex())
	log.Println("Asset holder:", assetHolder.Hex())

	asset := *ethwallet.AsWalletAddr(assetHolder)

	// Setup clients.
	kAlice, err := crypto.HexToECDSA(keyAlice)
	if err != nil {
		panic(err)
	}
	kBob, err := crypto.HexToECDSA(keyBob)
	if err != nil {
		panic(err)
	}

	setup, err := solana.NewExampleSetup([]string{keyAlice, keyBob}, [][20]byte{crypto.PubkeyToAddress(kAlice.PublicKey), crypto.PubkeyToAddress(kBob.PublicKey)})
	if err != nil {
		log.Fatalf("Failed to create Solana setup: %v", err)
	}

	bus := wire.NewLocalBus() // Message bus used for off-chain communication.

	alice := eth.SetupPaymentClient(bus, chainURL, adjudicator, asset, kAlice,
		setup.Wallets[0], setup.Accs[0], setup.Asset, setup.Funders[0], setup.Adjs[0])

	bob := eth.SetupPaymentClient(bus, chainURL, adjudicator, asset, kBob,
		setup.Wallets[1], setup.Accs[1], setup.Asset, setup.Funders[1], setup.Adjs[1])

	// Open channel, transact, close.
	log.Println("Opening channel and depositing funds.")
	alice.OpenChannel(bob.WireAddress(), 1, 50)
	bob.AcceptedChannel()

	log.Println("Perun channel opened and funded successfully.")
	// Cleanup.
	alice.Shutdown()
	bob.Shutdown()
}

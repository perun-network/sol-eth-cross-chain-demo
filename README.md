<h1 align="center"><br>
    <a href="https://perun.network/"><img src=".assets/go-perun.png" alt="Perun" width="196"></a>
<br></h1>


# sol-eth-cross-chain-demo
Perun Channel Cross Chain Demo for Ethereum and Solana Assets.

## Dependencies
### Solana
 Install the [Solana SDK](https://solana.com/docs/intro/installation)

 Deployment scripts: `make`, `tmux` and `tmuxp`

 ```bash
cd solana/scripts

chmod +x *.sh
```

### Ethereum
Install [ganache-cli](https://github.com/trufflesuite/ganache-cli)


## Run Demo
1. Start the Solana Local Validators

```bash
make dev
```

This will start a local node validator for Solana with 2 funded accounts: Alice and Bob. A SPL-Token is also minted and its associated accounts for Alice and Bob will also be funded.

2. On another terminal.
```sh
KEY_DEPLOYER=0x79ea8f62d97bc0591a4224c1725fca6b00de5b2cea286fe2e0bb35c5e76be46e
KEY_ALICE=0x1af2e950272dd403de7a5760d41c6e44d92b6d02797e51810795ff03cc2cda4f
KEY_BOB=0xf63d7d8e930bccd74e93cf5662fde2c28fd8be95edb70c73f1bdd863d07f412e
BALANCE=100000000000000000000

ganache -h 127.0.0.1 --port 8545 --wallet.accounts $KEY_DEPLOYER,$BALANCE $KEY_ALICE,$BALANCE $KEY_BOB,$BALANCE -b 5 
```
This starts a local ganache node with three prefunded accounts. The first account is used to deploy the contract, and the other two are used as Alice and Bob in the example.

3. On the third terminal, run the demo.
```sh
go run .
```

The demo initializes 2 Perunc Clients for Alice and Bob utilizing both [perun-eth-backend](https://github.com/perun-network/perun-eth-backend) and [perun-solana-backend](https://github.com/perun-network/perun-solana-backend). 

It creates and funds a Perun payment channel on both chains showcasing the cross-chain capabilitiy of [go-perun](https://github.com/perun-network/go-perun)

4. View the On-chain transactions regarding the Perun Program: https://explorer.solana.com/address/GQtQCW4dREybk2FR1gabaSb89CFxGrNS74JX5fZ97Qmh/domains?cluster=custom
   



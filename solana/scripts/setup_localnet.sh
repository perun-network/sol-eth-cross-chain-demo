#!/bin/bash
set -e

SCRIPTS_DIR="scripts"
ACCOUNTS_DIR="accounts"
ADDRESS_DIR="addresses"

if [ -d $ACCOUNTS_DIR ]; then
  rm -rf $ACCOUNTS_DIR/*
fi
mkdir -p $ACCOUNTS_DIR

if [ -d $ADDRESS_DIR ]; then
  rm -rf $ADDRESS_DIR/*
fi
mkdir -p $ADDRESS_DIR

# Ensure CLI is localnet
solana config set -ul

echo "[keygen] Generating fee payer..."
solana-keygen new --no-bip39-passphrase --force --outfile $ACCOUNTS_DIR/fee_payer.json

echo "[setup] Generating keypairs"
solana-keygen new --no-bip39-passphrase --force --outfile $ACCOUNTS_DIR/alice.json
solana-keygen new --no-bip39-passphrase --force --outfile $ACCOUNTS_DIR/bob.json

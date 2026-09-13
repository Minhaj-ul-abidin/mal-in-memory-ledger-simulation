package main

import (
	"os"

	"mal-in-memory-ledger-simulation/ledger"
)

func main() {
	ledger.Replay().Print(os.Stdout)
}

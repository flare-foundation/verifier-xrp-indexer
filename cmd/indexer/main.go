package main

import (
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/verifier-indexer-framework/pkg/framework"
	"github.com/flare-foundation/verifier-xrp-indexer/internal/xrp"
)

func main() {
	input := framework.Input[xrp.Block, xrp.Config, xrp.Transaction]{
		NewBlockchainClient: xrp.New,
	}

	if err := framework.Run(input); err != nil {
		logger.Fatal(err)
	}

}

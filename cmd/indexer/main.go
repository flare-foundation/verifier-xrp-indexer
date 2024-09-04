package main

import (
	"gitlab.com/flarenetwork/libs/go-flare-common/pkg/logger"
	"gitlab.com/flarenetwork/fdc/verifier-indexer-framework/pkg/framework"
	"gitlab.com/flarenetwork/fdc/verifier-xrp-indexer/internal/xrp"
)

var log = logger.GetLogger()

func main() {
	input := framework.Input[xrp.Block, xrp.Config, xrp.Transaction]{
		NewBlockchain: xrp.New,
	}

	if err := framework.Run(input); err != nil {
		log.Fatal(err)
	}
}

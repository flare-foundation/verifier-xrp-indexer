package main

import (
	"gitlab.com/ryancollingham/flare-common/pkg/logger"
	"gitlab.com/ryancollingham/flare-indexer-framework/pkg/database"
	"gitlab.com/ryancollingham/flare-indexer-framework/pkg/framework"
	"gitlab.com/ryancollingham/flare-xrp-indexer/internal/xrp"
)

var log = logger.GetLogger()

func main() {
	input := framework.Input[*xrp.Config]{
		DefaultConfig: new(xrp.Config),
		Entities: database.ExternalEntities{
			Block:       new(xrp.Block),
			Transaction: new(xrp.Transaction),
		},
		NewBlockchain: xrp.New,
	}

	if err := framework.Run(input); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"gitlab.com/ryancollingham/flare-common/pkg/logger"
	"gitlab.com/ryancollingham/flare-indexer-framework/pkg/framework"
	"gitlab.com/ryancollingham/flare-xrp-indexer/internal/xrp"
)

var log = logger.GetLogger()

func main() {
	if err := framework.Run(xrp.New, new(xrp.Config)); err != nil {
		log.Fatal(err)
	}
}

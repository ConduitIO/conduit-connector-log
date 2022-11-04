package main

import (
	sdk "github.com/conduitio/conduit-connector-sdk"

	log "conduitio/conduit-connector-log"
)

func main() {
	sdk.Serve(log.Connector)
}

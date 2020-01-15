package scanner_legacy

import "github.com/google/wire"

var Set = wire.NewSet(
	NewImporter,
	NewItunesScanner,
	wire.Bind(new(Scanner), new(*ItunesScanner)),
)

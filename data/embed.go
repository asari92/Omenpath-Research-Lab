package data

import _ "embed"

// MTGPlanesExpandedSeed is the canonical plane catalog bundled with the
// executable so bootstrap never depends on the process working directory.
//
//go:embed mtg_planes_expanded_seed.json
var MTGPlanesExpandedSeed []byte

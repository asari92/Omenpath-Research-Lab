// Command server will run the Omenpath Research Lab API (REST + WebSocket
// + simulation loop). Stage 1 skeleton only — transport layers are
// explicitly out of scope (03_STAGE_00_01_TDD_FOUNDATION.md, "Do NOT
// implement" list); they arrive in Stages 12–13.
package main

import (
	"fmt"

	"omenpath-lab/internal/config"
)

func main() {
	fmt.Printf("omenpath-lab server skeleton (balance preset: %d slots, %d observers)\n",
		config.Default().MaxActivePortals, config.Default().ObserverCount)
}

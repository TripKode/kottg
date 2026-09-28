module github.com/kodechain/kottg

go 1.24.0

require tc_node_validator_v2 v0.0.0

require (
	github.com/cloudflare/circl v1.6.1 // indirect
	golang.org/x/sys v0.36.0 // indirect
)

replace tc_node_validator_v2 => ../kodechain-engine

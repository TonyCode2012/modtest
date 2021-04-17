module github.com/TonyCode2012/modtest

require (
	github.com/ipfs/go-block-format v0.0.2
	github.com/ipfs/go-cid v0.0.5
	github.com/ipfs/go-ipld-format v0.2.0
)

replace (
	github.com/ipfs/go-block-format => /home/yaoz/errands/IPFS/go-block-format
	github.com/ipfs/go-cid => /home/yaoz/errands/IPFS/go-cid@v0.0.7
)

go 1.15

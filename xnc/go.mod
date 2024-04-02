module github.com/comp529/xnc

go 1.20

require (
	github.com/itzmeanjan/kodr v0.2.2
	github.com/lucas-clemente/quic-go v0.25.0
)

require (
	github.com/bifurcation/mint v0.0.0-20200214151656-93c820e81448 // indirect
	github.com/cloud9-tools/go-galoisfield v0.0.0-20160311182916-a8cf2bffadf0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/lucas-clemente/aes12 v0.0.0-20171027163421-cd47fb39b79f // indirect
	github.com/lucas-clemente/fnv128a v0.0.0-20160504152609-393af48d3916 // indirect
	github.com/lucas-clemente/quic-go-certificates v0.0.0-20160823095156-d2f86524cced // indirect
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9 // indirect
)

replace github.com/lucas-clemente/quic-go => ../mp-quic

replace github.com/itzmeanjan/kodr => ../kodr

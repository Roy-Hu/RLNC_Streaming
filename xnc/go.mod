module github.com/comp529/xnc

go 1.20

require (
	github.com/itzmeanjan/kodr v0.2.2
	github.com/quic-go/quic-go v0.42.0
)

require (
	github.com/cloud9-tools/go-galoisfield v0.0.0-20160311182916-a8cf2bffadf0 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/google/pprof v0.0.0-20210407192527-94a9f03dee38 // indirect
	github.com/onsi/ginkgo/v2 v2.9.5 // indirect
	github.com/quic-go/qtls-go1-20 v0.3.0 // indirect
	golang.org/x/crypto v0.4.0 // indirect
	golang.org/x/exp v0.0.0-20221205204356-47842c84f3db // indirect
	golang.org/x/mod v0.11.0 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
	golang.org/x/tools v0.9.1 // indirect
)

// replace github.com/lucas-clemente/quic-go => ../mp-quic

replace github.com/itzmeanjan/kodr => ../kodr

replace github.com/quic-go/quic-go => ../quic-go

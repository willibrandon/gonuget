module github.com/willibrandon/gonuget/cmd/nuget-interop-test

go 1.25.2

require (
	github.com/willibrandon/gonuget v0.0.0
	software.sslmate.com/src/go-pkcs12 v0.6.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/crypto v0.43.0 // indirect
)

replace github.com/willibrandon/gonuget => ../..

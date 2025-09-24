[![Go Reference](https://pkg.go.dev/badge/github.com/SUNET/go-trust.svg)](https://pkg.go.dev/github.com/SUNET/go-trust)
[![Go Report Card](https://goreportcard.com/badge/github.com/SUNET/go-trust)](https://goreportcard.com/report/github.com/SUNET/go-trust)
![coverage](https://raw.githubusercontent.com/SUNET/go-trust/badges/.badges/main/coverage.svg)
[![License](https://img.shields.io/badge/License-BSD_2--Clause-orange.svg)](https://opensource.org/licenses/BSD-2-Clause)

# go-trust - a local trust engine

go-trust (gt) is a service that allows a client to abstract trust decisions as an AuthZEN policy decision point (PDP). The go-trust provides an authzen service that evaluates trust in subjects identified by X509 certificate in terms of a set of ETSI TS 119612 trust status lists.


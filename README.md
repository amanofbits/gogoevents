# Go-go-events ![Go-pkg doc](https://img.shields.io/badge/Go--pkg-doc-blue?link=https%3A%2F%2Fpkg.go.dev%2Fgithub.com%2Famanofbits%2Fgogoevents) [![Go Report Card](https://goreportcard.com/badge/github.com/amanofbits/gogoevents)](https://goreportcard.com/report/github.com/amanofbits/gogoevents)

Simple event bus for Golang.

- no dependencies
- simple
- concurrency support
- generic - supports custom-defined event object per bus
- supports wildcard subscribers
- fast (I hope, not yet tested thoroughly. Some benchmarks included)
- non-blocking events - every subscriber receives the event in its own goroutine
- has a dedicated sink for unhandled events (the ones that are not subscribed to). Useful for e.g. debugging and logging lost events.

## Attributions

- Uses modified code from [IGLOU-EU/go-wildcard v2.0.2](https://github.com/IGLOU-EU/go-wildcard/blob/2f93770ccbe7d1f3e102221d88ade4c0ecca52be/wildcard.go)
- Inspired by [jackhopner/go-events](https://github.com/jackhopner/go-events) and [dtomasi/go-event-bus](https://github.com/dtomasi/go-event-bus)

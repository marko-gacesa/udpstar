// Copyright (c) 2023 by Marko Gaćeša

package message

// MaxMessageSize is maximum safe UDP message size.
// https://stackoverflow.com/questions/1098897/what-is-the-largest-safe-udp-packet-size-on-the-internet
// https://www.linode.com/docs/guides/developing-udp-and-tcp-clients-and-servers-in-go/
const MaxMessageSize = 508

const (
	LenStoryConfirm      = 61
	LenLatencyReport     = 8
	LenLatencyReportName = 52
)

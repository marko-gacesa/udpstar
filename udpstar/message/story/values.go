// Copyright (c) 2023,2024 by Marko Gaćeša

package story

import "github.com/marko-gacesa/udpstar/udpstar/message"

const (
	CategoryStory message.Category = 3

	MaxMessageSize = message.MaxMessageSize - 1

	LenStoryConfirm      = 61
	LenLatencyReport     = 8
	LenLatencyReportName = 52
)

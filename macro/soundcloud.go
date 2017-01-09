package macro

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2017 ESSENTIAL KAOS                         //
//        Essential Kaos Open Source License <https://essentialkaos.com/ekol>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"strings"

	"github.com/essentialkaos/postmark"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// SoundcloudConfig contains properties for Soundcloud macro
type SoundcloudConfig struct {
	ID           string
	Width        int
	AutoPlay     bool
	HideRelated  bool
	HideComments bool
	HideUser     bool
	ShowReposts  bool
}

type soundcloudHandler func(config SoundcloudConfig) string
type soundcloudErrorHandler func(err error) string

type soundcloudStore struct {
	Handler      soundcloudHandler
	ErrorHandler soundcloudErrorHandler
	HTML         bool
}

// ////////////////////////////////////////////////////////////////////////////////// //

// YouTube is proxy for YouTube macro
//
// Supported properties:
// - width (300/450/600)
// - autoPlay (boolean)
// - hideRelated (boolean)
// - hideComments (boolean)
// - hideUser (boolean)
// - showReposts (boolean)
//
// Example:
// {soundcloud:268954121|autoPlay}
func Soundcloud(handler soundcloudHandler, errorHandler soundcloudErrorHandler) *postmark.Macro {
	return &postmark.Macro{
		Name:         "soundcloud",
		Multiline:    false,
		ProxyStore:   &soundcloudStore{handler, errorHandler, false},
		ProxyHandler: soundcloudMacroHandler,
		Properties: []string{
			"width",
			"autoPlay",
			"hideRelated",
			"hideComments",
			"hideUser",
			"showReposts",
		},
	}
}

// SoundcloudHTML is macro with HTML encoder
func SoundcloudHTML(errorHandler soundcloudErrorHandler) *postmark.Macro {
	return &postmark.Macro{
		Name:         "soundcloud",
		Multiline:    false,
		ProxyStore:   &soundcloudStore{nil, errorHandler, true},
		ProxyHandler: soundcloudMacroHandler,
		Properties: []string{
			"width",
			"autoPlay",
			"hideRelated",
			"hideComments",
			"hideUser",
			"showReposts",
		},
	}
}

// ////////////////////////////////////////////////////////////////////////////////// //

func soundcloudMacroHandler(store interface{}, data string, props map[string]string) string {
	var macroStore *soundcloudStore

	switch store.(type) {
	case *soundcloudStore:
		macroStore = store.(*soundcloudStore)
	default:
		return ""
	}

	if macroStore.HTML {
		return soundcloudMacroHTMLRender(soundcloudPropsToConfig(props))
	} else {
		if macroStore.Handler != nil {
			return macroStore.Handler(soundcloudPropsToConfig(props))
		}
	}

	return ""
}

func soundcloudMacroHTMLRender(config SoundcloudConfig) string {
	var arguments []string
	var argumentsStr string

	arguments = append(arguments, config.ID)
	arguments = append(arguments, fmt.Sprintf("auto_play=%t", config.AutoPlay))
	arguments = append(arguments, fmt.Sprintf("hide_related=%t", config.HideRelated))
	arguments = append(arguments, fmt.Sprintf("show_comments=%t", !config.HideComments))
	arguments = append(arguments, fmt.Sprintf("show_user=%t", !config.HideUser))
	arguments = append(arguments, "visual=true")

	argumentsStr = strings.Join(arguments, "&amp;")

	return fmt.Sprintf(
		"<iframe width=\"100%%\" height=\"%d\" scrolling=\"no\" frameborder=\"no\" src=\"https://w.soundcloud.com/player/?url=https%%3A//api.soundcloud.com/tracks/%s\"></iframe>",
		config.Width, argumentsStr,
	)
}

func soundcloudPropsToConfig(props map[string]string) SoundcloudConfig {
	return SoundcloudConfig{
		ID:           props[""],
		Width:        parseInt(props["width"], 450),
		AutoPlay:     parseBoolean(props["autoPlay"]),
		HideRelated:  parseBoolean(props["hideRelated"]),
		HideComments: parseBoolean(props["hideComments"]),
		HideUser:     parseBoolean(props["hideUser"]),
		ShowReposts:  parseBoolean(props["showReposts"]),
	}
}

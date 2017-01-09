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

// YouTubeConfig contains properties for YouTube macro
type YouTubeConfig struct {
	ID              string
	Width           int
	Height          int
	HideRelated     bool
	HideControls    bool
	HideInfo        bool
	EnhancedPrivacy bool
}

type youtubeHandler func(config YouTubeConfig) string
type youtubeErrorHandler func(err error) string

type youtubeStore struct {
	Handler      youtubeHandler
	ErrorHandler youtubeErrorHandler
	HTML         bool
}

// ////////////////////////////////////////////////////////////////////////////////// //

// YouTube is proxy for YouTube macro
//
// Supported properties:
// - size (1280x720)
// - hideRelated (boolean)
// - hideControls (boolean)
// - hideInfo (boolean)
// - enhancedPrivacy (boolean)
//
// Example:
// {youtube:yMn863_910w|size=560x315|hideRelated|hideControls|hideInfo|enhancedPrivacy}
func YouTube(handler youtubeHandler, errorHandler youtubeErrorHandler) *postmark.Macro {
	return &postmark.Macro{
		Name:         "youtube",
		Multiline:    false,
		ProxyStore:   &youtubeStore{handler, errorHandler, false},
		ProxyHandler: youTubeMacroHandler,
		Properties: []string{
			"size",
			"hideRelated",
			"hideControls",
			"hideInfo",
			"enhancedPrivacy",
		},
	}
}

// YouTubeHTML is macro with HTML encoder
func YouTubeHTML(errorHandler youtubeErrorHandler) *postmark.Macro {
	return &postmark.Macro{
		Name:         "youtube",
		Multiline:    false,
		ProxyStore:   &youtubeStore{nil, errorHandler, true},
		ProxyHandler: youTubeMacroHandler,
		Properties: []string{
			"size",
			"hideRelated",
			"hideControls",
			"hideInfo",
			"enhancedPrivacy",
		},
	}
}

// ////////////////////////////////////////////////////////////////////////////////// //

// youTubeMacroHandler is YouTube handler
func youTubeMacroHandler(store interface{}, data string, props map[string]string) string {
	var macroStore *youtubeStore

	switch store.(type) {
	case *youtubeStore:
		macroStore = store.(*youtubeStore)
	default:
		return ""
	}

	if macroStore.HTML {
		return youTubeMacroHTMLRender(youTubePropsToConfig(props))
	} else {
		if macroStore.Handler != nil {
			return macroStore.Handler(youTubePropsToConfig(props))
		}
	}

	return ""
}

func youTubeMacroHTMLRender(config YouTubeConfig) string {
	var arguments []string
	var argumentsStr string

	domain := "www.youtube.com"

	if config.HideControls {
		arguments = append(arguments, "controls=0")
	}

	if config.HideInfo {
		arguments = append(arguments, "showinfo=0")
	}

	if config.HideRelated {
		arguments = append(arguments, "rel=0")
	}

	if config.EnhancedPrivacy {
		domain = "www.youtube-nocookie.com"
	}

	if len(arguments) != 0 {
		argumentsStr = "?" + strings.Join(arguments, "&amp;")
	}

	return fmt.Sprintf(
		"<iframe width=\"%d\" height=\"%d\" src=\"https://%s/embed/%s%s\" frameborder=\"0\" allowfullscreen></iframe>",
		config.Width, config.Height, domain, config.ID, argumentsStr,
	)
}

// youtubePropsToConfig convert props to config struct
func youTubePropsToConfig(props map[string]string) YouTubeConfig {
	var width, height = parseSize(props["size"], 600, 340)

	return YouTubeConfig{
		ID:              props[""],
		Width:           width,
		Height:          height,
		HideRelated:     parseBoolean(props["hideRelated"]),
		HideControls:    parseBoolean(props["hideControls"]),
		HideInfo:        parseBoolean(props["hideInfo"]),
		EnhancedPrivacy: parseBoolean(props["enhancedPrivacy"]),
	}
}

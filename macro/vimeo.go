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

// VimeoConfig contains properties for Vimeo macro
type VimeoConfig struct {
	ID           string
	Width        int
	Height       int
	Color        string
	HidePortrait bool
	HideTitle    bool
	HideByline   bool
	Loop         bool
	Autoplay     bool
}

type vimeoHandler func(config VimeoConfig) string
type vimeoErrorHandler func(err error) string

type vimeoStore struct {
	Handler      vimeoHandler
	ErrorHandler vimeoErrorHandler
	HTML         bool
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Vimeo is proxy for Vimeo macro
//
// Supported properties:
// - size (1280x720)
// - color (string)
// - hidePortrait (boolean)
// - hideTitle (boolean)
// - hideByline (boolean)
// - loop (boolean)
// - autoplay (boolean)
//
// Example:
// {vimeo:126553902|size=560x315|hidePortrait|loop}
func Vimeo(handler vimeoHandler, errorHandler vimeoErrorHandler) *postmark.Macro {
	return &postmark.Macro{
		Name:         "vimeo",
		Multiline:    false,
		ProxyStore:   &vimeoStore{handler, errorHandler, false},
		ProxyHandler: vimeoMacroHandler,
		Properties: []string{
			"size",
			"color",
			"hidePortrait",
			"hideTitle",
			"hideByline",
			"loop",
			"autoplay",
		},
	}
}

// YouTubeHTML is macro with html encoder
func VimeoHTML(errorHandler vimeoErrorHandler) *postmark.Macro {
	return &postmark.Macro{
		Name:         "vimeo",
		Multiline:    false,
		ProxyStore:   &vimeoStore{nil, errorHandler, true},
		ProxyHandler: vimeoMacroHandler,
		Properties: []string{
			"size",
			"color",
			"hidePortrait",
			"hideTitle",
			"hideByline",
			"loop",
			"autoplay",
		},
	}
}

// ////////////////////////////////////////////////////////////////////////////////// //

// youTubeMacroHandler is YouTube handler
func vimeoMacroHandler(store interface{}, data string, props map[string]string) string {
	var macroStore *vimeoStore

	switch store.(type) {
	case *vimeoStore:
		macroStore = store.(*vimeoStore)
	default:
		return ""
	}

	if macroStore.HTML {
		return vimeoMacroHTMLRender(vimeoPropsToConfig(props))
	} else {
		if macroStore.Handler != nil {
			return macroStore.Handler(vimeoPropsToConfig(props))
		}
	}

	return ""
}

func vimeoMacroHTMLRender(config VimeoConfig) string {
	var arguments []string
	var argumentsStr string

	if config.Color != "" {
		arguments = append(arguments, "color="+config.Color)
	}

	if config.HidePortrait {
		arguments = append(arguments, "portrait=0")
	}

	if config.HideTitle {
		arguments = append(arguments, "title=0")
	}

	if config.HideByline {
		arguments = append(arguments, "byline=0")
	}

	if config.Loop {
		arguments = append(arguments, "loop=1")
	}

	if config.Autoplay {
		arguments = append(arguments, "autoplay=1")
	}

	if len(arguments) != 0 {
		argumentsStr = "?" + strings.Join(arguments, "&amp;")
	}

	return fmt.Sprintf(
		"<iframe src=\"https://player.vimeo.com/video/%s%s\" width=\"%d\" height=\"%d\" frameborder=\"0\" webkitallowfullscreen mozallowfullscreen allowfullscreen></iframe>",
		config.ID, argumentsStr, config.Width, config.Height,
	)
}

func vimeoPropsToConfig(props map[string]string) VimeoConfig {
	var width, height = parseSize(props["size"], 640, 360)

	return VimeoConfig{
		ID:           props[""],
		Width:        width,
		Height:       height,
		Color:        parseColor(props["color"]),
		HidePortrait: parseBoolean(props["hidePortrait"]),
		HideTitle:    parseBoolean(props["hideTitle"]),
		HideByline:   parseBoolean(props["hideByline"]),
		Loop:         parseBoolean(props["loop"]),
		Autoplay:     parseBoolean(props["autoplay"]),
	}
}

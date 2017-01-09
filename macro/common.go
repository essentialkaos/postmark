package macro

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2017 ESSENTIAL KAOS                         //
//        Essential Kaos Open Source License <https://essentialkaos.com/ekol>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"strconv"
	"strings"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// parseSize parse size property and return width and height
func parseSize(v string, width, height int) (int, int) {
	if v == "" {
		return width, height
	}

	var w, h int
	var err error

	sizeSlice := strings.Split(v, "x")

	if len(sizeSlice) == 2 {
		w, err = strconv.Atoi(sizeSlice[0])

		if err != nil {
			return width, height
		}

		h, err = strconv.Atoi(sizeSlice[1])

		if err != nil {
			return width, height
		}

		width = w
		height = h
	}

	return width, height
}

// parseBoolean parse boolean
func parseBoolean(v string) bool {
	if v == "" || v == "false" {
		return false
	}

	return true
}

// parseInt parse numebers
func parseInt(v string, def int) int {
	vi, err := strconv.Atoi(v)

	if err != nil {
		return def
	}

	return vi
}

// parseColor parse colors
func parseColor(v string) string {
	if strings.Contains(v, "#") {
		return strings.TrimLeft(v, "#")
	}

	return v
}

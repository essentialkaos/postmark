package postmark

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2016 Essential Kaos                         //
//      Essential Kaos Open Source License <http://essentialkaos.com/ekol?en>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	TYPE_POST  uint8 = 0
	TYPE_PHOTO       = 1
	TYPE_QUOTE       = 2
	TYPE_LINK        = 3
)

// ////////////////////////////////////////////////////////////////////////////////// //

type PostMeta struct {
	Title     string    // Post title
	Name      string    // Post name
	Author    string    // Post author
	Date      time.Time // Post date
	Tags      []string  // Tag list
	Type      uint8     // Post type
	Protected bool      // Protected post flag
}

type Post struct {
	Meta *PostMeta // Post meta
	Data string    // Post rendered data
}

type Render struct {
	Header    func(text string, level int) string
	Paragraph func(text string) string
	Bold      func(text string) string
	Italic    func(text string) string
	Del       func(text string) string
	Sup       func(text string) string
	Sub       func(text string) string
	Hr        func() string
	Link      func(url, text string) string
	Image     func(url, alt, caption string) string
	Code      func(text string) string

	UnsupportedMacro func(text string) string

	// You can extend formating by custom macroses
	Macroses []*Macro
}

type Macro struct {
	Name      string                                            // Macro name
	Multiline bool                                              // Mutliline flag
	Handler   func(data string, props map[string]string) string // Handler function
}

// ////////////////////////////////////////////////////////////////////////////////// //

var postTypes = map[string]uint8{
	"":      TYPE_POST,
	"post":  TYPE_POST,
	"photo": TYPE_PHOTO,
	"quote": TYPE_QUOTE,
	"link":  TYPE_LINK,
}

var (
	rxMeta      = regexp.MustCompile(`^[+]{4}`)
	rxFmtHeader = regexp.MustCompile(`h([1-6])\. (.*)`)
	rxFmtItalic = regexp.MustCompile(`_([\p{L}\d\S]{1}.*?)_`)
	rxFmtBold   = regexp.MustCompile(`\*([\p{L}\d\S]{1}.*?)\*`)
	rxFmtDel    = regexp.MustCompile(`\-([\p{L}\d\S]{1}.*?)\-`)
	rxFmtSup    = regexp.MustCompile(`\^([\p{L}\d\S]{1}.*?)\^`)
	rxFmtSub    = regexp.MustCompile(`\~([\p{L}\d\S]{1}.*?)\~`)
	rxFmtCode   = regexp.MustCompile("\\`" + `([\p{L}\d\S]{1}.*)` + "\\`")
	rxFmtHr     = regexp.MustCompile(`^[-]{4,}`)
	rxFmtImage  = regexp.MustCompile(`\![\w\S]{1,}\.(jpg|jpeg|gif|png)(?:(!|\|)).*`)
	rxFmtLink   = regexp.MustCompile(`\[(.*?)?(?:\|)?((?:http|https|ftp|mailto)[\S]{3,})\]`)
	rxMacro     = regexp.MustCompile(`^\{([a-z0-9-]{2,})(?:\:)?(.*)\}`)
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Process parse and render post file
func Process(file string, render *Render) (*Post, error) {
	return processFile(file, render)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// IsValid validate post data and meta
func (p *Post) IsValid() bool {
	if p.Meta == nil {
		return false
	}

	if p.Meta.Author == "" {
		return false
	}

	if len(p.Data) == 0 {
		return false
	}

	return true
}

// ////////////////////////////////////////////////////////////////////////////////// //

// processFile parse given file and render data with give render
func processFile(file string, render *Render) (*Post, error) {
	fd, err := os.OpenFile(file, os.O_RDONLY, 0)

	if err != nil {
		return nil, err
	}

	defer fd.Close()

	reader := bufio.NewReader(fd)
	scanner := bufio.NewScanner(reader)

	post := &Post{Meta: &PostMeta{}}

	var (
		isMeta     bool
		isMacro    bool
		macroName  string
		macroProps map[string]string
		macro      *Macro
		buffer     string
	)

	var hasMacroses = len(render.Macroses) != 0

	for scanner.Scan() {
		line := scanner.Text()

		if rxMeta.MatchString(line) {
			isMeta = !isMeta
			continue
		}

		if hasMacroses && rxMacro.MatchString(line) {
			if isMacro {
				processMutlilineMacro(macro, macroProps, buffer, render, post)
				isMacro, buffer = false, ""
				continue
			}

			macroName, macro, macroProps = parseMacro(line, render)

			if macro == nil {
				processUnsuportedMacro(macroName, render, post)
				continue
			}

			if macro.Multiline {
				isMacro = true
				continue
			}

			processSimpleMacro(macro, macroProps, render, post)
			continue
		}

		switch {
		case isMeta:
			processMetaData(line, post.Meta)

		case isMacro:
			buffer += line + "\n"

		default:
			if strings.Trim(line, " ") == "" {
				continue
			}

			processContentData(line, render, post)
		}
	}

	return post, nil
}

// processMetaLine process line with some metadata info
func processMetaData(data string, meta *PostMeta) error {
	var err error

	metaSlice := strings.Split(data, ":")

	if len(metaSlice) < 2 {
		return fmt.Errorf("Misformatted meta")
	}

	property := strings.TrimLeft(metaSlice[0], " ")
	value := strings.TrimLeft(strings.Join(metaSlice[1:], ":"), " ")

	switch strings.ToLower(property) {
	case "title":
		meta.Title = value

	case "name":
		meta.Name = value

	case "date":
		meta.Date, err = time.Parse("2006/01/02 15:04", value)

		if err != nil {
			return err
		}

	case "author":
		meta.Author = value

	case "tags":
		meta.Tags = strings.Fields(value)

	case "type":
		meta.Type = postTypes[strings.ToLower(value)]

	case "protected":
		meta.Protected = strings.ToLower(value) == "true"

	default:
		return fmt.Errorf("Unsupported property \"%s\"", property)
	}

	return nil
}

// processContentData process and render post content
func processContentData(data string, render *Render, post *Post) error {
	var err error

	switch {
	case rxFmtHeader.MatchString(data):
		err = processHeaderData(data, render, post)
	case rxFmtImage.MatchString(data):
		err = processImageData(data, render, post)
	default:
		err = processParagraphData(data, render, post)
	}

	return err
}

// processHeaderData process and render header data
func processHeaderData(data string, render *Render, post *Post) error {
	if render.Header == nil {
		appendPostData(post, data)
	}

	text, level := parseHeader(data)

	if level == -1 {
		return fmt.Errorf("Can't parse header line \"%s\"", data)
	}

	appendPostData(post, render.Header(text, level))

	return nil
}

// processImageData process and render image data
func processImageData(data string, render *Render, post *Post) error {
	if render.Image == nil {
		appendPostData(post, data)
	}

	url, alt, caption := parseImage(data)

	if caption != "" {
		caption = parseParagraph(caption, render)
	}

	appendPostData(post, render.Image(url, alt, caption))

	return nil
}

// processParagraphData process and render paragraph data
func processParagraphData(data string, render *Render, post *Post) error {
	data = parseParagraph(data, render)

	if render.Paragraph == nil {
		appendPostData(post, data)
	}

	appendPostData(post, render.Paragraph(data))

	return nil
}

func processSimpleMacro(macro *Macro, macroProps map[string]string, render *Render, post *Post) error {
	if macro.Handler == nil {
		return fmt.Errorf("Handler is nil for \"%s\" macro", macro.Name)
	}

	appendPostData(post, macro.Handler("", macroProps))

	return nil
}

func processMutlilineMacro(macro *Macro, macroProps map[string]string, data string, render *Render, post *Post) error {
	if macro.Handler == nil {
		return fmt.Errorf("Handler is nil for \"%s\" macro", macro.Name)
	}

	appendPostData(post, macro.Handler(data, macroProps))

	return nil
}

// processUnsuportedMacro process unsupported macro
func processUnsuportedMacro(macroName string, render *Render, post *Post) {
	if render.UnsupportedMacro != nil {
		appendPostData(post, render.UnsupportedMacro(macroName))
	}
}

// parseHeader parse header tag
func parseHeader(data string) (string, int) {
	headerInfo := rxFmtHeader.FindStringSubmatch(data)
	level, err := strconv.Atoi(headerInfo[1])

	if err != nil {
		return "", -1
	}

	return headerInfo[2], level
}

// parseImage parse image tag
func parseImage(data string) (string, string, string) {
	var url, alt, caption string

	imageTag := rxFmtImage.FindAllString(data, -1)
	imageTagSlice := strings.Split(imageTag[0], "!")

	url, caption = imageTagSlice[1], strings.TrimLeft(imageTagSlice[2], " ")

	if strings.Contains(url, "|") {
		urlSlice := strings.Split(url, "|")
		url, alt = urlSlice[0], urlSlice[1]
	}

	return url, alt, caption
}

// parseParagraph parse paragraph data
func parseParagraph(data string, render *Render) string {
	var (
		tags [][]string
		tag  []string
	)

	if render.Bold != nil && rxFmtBold.MatchString(data) {
		tags = rxFmtBold.FindAllStringSubmatch(data, -1)

		for _, tag = range tags {
			data = strings.Replace(data, tag[0], render.Bold(tag[1]), -1)
		}
	}

	if render.Italic != nil && rxFmtItalic.MatchString(data) {
		tags = rxFmtItalic.FindAllStringSubmatch(data, -1)

		for _, tag = range tags {
			data = strings.Replace(data, tag[0], render.Italic(tag[1]), -1)
		}
	}

	if render.Del != nil && rxFmtDel.MatchString(data) {
		tags = rxFmtDel.FindAllStringSubmatch(data, -1)

		for _, tag = range tags {
			data = strings.Replace(data, tag[0], render.Del(tag[1]), -1)
		}
	}

	if render.Sup != nil && rxFmtSup.MatchString(data) {
		tags = rxFmtSup.FindAllStringSubmatch(data, -1)

		for _, tag = range tags {
			data = strings.Replace(data, tag[0], render.Sup(tag[1]), -1)
		}
	}

	if render.Sub != nil && rxFmtSub.MatchString(data) {
		tags = rxFmtSub.FindAllStringSubmatch(data, -1)

		for _, tag = range tags {
			data = strings.Replace(data, tag[0], render.Sub(tag[1]), -1)
		}
	}

	if render.Hr != nil && rxFmtHr.MatchString(data) {
		tags = rxFmtHr.FindAllStringSubmatch(data, -1)

		for _, tag = range tags {
			data = strings.Replace(data, tag[0], render.Hr(), -1)
		}
	}

	if render.Code != nil && rxFmtCode.MatchString(data) {
		tags := rxFmtCode.FindAllStringSubmatch(data, -1)

		for _, tag = range tags {
			data = strings.Replace(data, tag[0], render.Code(tag[1]), -1)
		}
	}

	if render.Link != nil && rxFmtLink.MatchString(data) {
		tags := rxFmtLink.FindAllStringSubmatch(data, -1)

		for _, tag := range tags {
			data = strings.Replace(data, tag[0], render.Link(tag[2], tag[1]), -1)
		}
	}

	return data
}

// parseMacro extract macro name from given data and return macro struct
func parseMacro(data string, render *Render) (string, *Macro, map[string]string) {
	var (
		macro *Macro
		name  string
		props map[string]string
	)

	macroTag := rxMacro.FindStringSubmatch(data)
	name = macroTag[1]

	for _, m := range render.Macroses {
		if name == m.Name {
			macro = m
			break
		}
	}

	if macroTag[2] != "" {
		props = parseMacroProps(macroTag[2])
	}

	return name, macro, props
}

// parseMacroProps parse macro properties and return it as prop->value map
func parseMacroProps(data string) map[string]string {
	if data == "" {
		return nil
	}

	result := make(map[string]string)

	for _, prop := range strings.Split(data, "|") {
		if !strings.Contains(prop, "=") {
			result[""] = prop
		} else {
			propSlice := strings.Split(prop, "=")
			result[propSlice[0]] = propSlice[1]
		}
	}

	return result
}

// appendPostData append rendered data to post
func appendPostData(post *Post, data string) {
	post.Data += data + "\n"
}

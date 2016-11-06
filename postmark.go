package postmark

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2016 Essential Kaos                         //
//      Essential Kaos Open Source License <http://essentialkaos.com/ekol?en>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	TYPE_POST  = "post"
	TYPE_PHOTO = "photo"
	TYPE_QUOTE = "quote"
	TYPE_LINK  = "link"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type PostMeta struct {
	Title      string    // Post title
	Name       string    // Post name
	Author     string    // Post author
	AuthorLink string    // URL to author account
	Date       time.Time // Post date
	Tags       []string  // Tag list
	Type       string    // Post type
	Protected  bool      // Protected post flag
}

type Post struct {
	Meta    *PostMeta // Post meta
	Content string    // Post rendered data
}

type Render struct {
	Header    func(text string, level int) string
	Paragraph func(text string) string
	Bold      func(text string) string
	Italic    func(text string) string
	Del       func(text string) string
	Sup       func(text string) string
	Sub       func(text string) string
	Code      func(text string) string
	Hr        func() string
	Link      func(url, text string) string
	Image     func(url, alt, caption string) string

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

var (
	rxMeta        = regexp.MustCompile(`^[+]{4}`)
	rxFmtHeader   = regexp.MustCompile(`h([1-6])\. (.*)`)
	rxFmtItalic   = regexp.MustCompile(`_([\p{L}\d\S]{1}.*?)_`)
	rxFmtBold     = regexp.MustCompile(`\*([\p{L}\d\S]{1}.*?)\*`)
	rxFmtDel      = regexp.MustCompile(`\-([\p{L}\d\S]{1}.*?)\-`)
	rxFmtSup      = regexp.MustCompile(`\^([\p{L}\d\S]{1}.*?)\^`)
	rxFmtSub      = regexp.MustCompile(`\~([\p{L}\d\S]{1}.*?)\~`)
	rxFmtCode     = regexp.MustCompile("\\`" + `([\p{L}\d\S]{1}.*)` + "\\`")
	rxFmtHr       = regexp.MustCompile(`^[-]{4,}`)
	rxFmtImage    = regexp.MustCompile(`\![\w\S]{1,}\.(jpg|jpeg|gif|png)(?:(!|\|)).*`)
	rxFmtLink     = regexp.MustCompile(`\[(.*?)?(?:\|)?((?:http|https|ftp|mailto)[\S]{3,})\]`)
	rxMacro       = regexp.MustCompile(`^\{([a-z0-9-]{2,})(?:\:)?(.*)\}`)
	rxInlineMacro = regexp.MustCompile(`\{([a-z0-9-]{2,})(?:\:)?(.*)\}`)
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Process parse and render post file
func Process(file string, render *Render) (*Post, error) {
	data, err := ioutil.ReadFile(file)

	if err != nil {
		return nil, err
	}

	meta, content, err := extractMeta(string(data))

	if err != nil {
		return nil, err
	}

	post := &Post{Meta: meta}

	post.Content, err = render.Apply(content)

	if err != nil {
		return nil, err
	}

	return post, nil
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

	if len(p.Content) == 0 {
		return false
	}

	return true
}

// Apply render given data
func (r *Render) Apply(text string) (string, error) {
	return parseContent(text, r)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// extractMeta extract meta from post content and return meta struct and post content
// without metadata
func extractMeta(content string) (*PostMeta, string, error) {
	var err error
	var isMeta bool

	var meta = &PostMeta{Type: TYPE_POST}

	var i int
	var r rune

	var buffer = bytes.NewBuffer(nil)

	for i, r = range content {
		if r != '\n' {
			buffer.WriteRune(r)
			continue
		}

		if buffer.String() == "++++" {
			buffer.Reset()

			if isMeta {
				break
			}

			isMeta = true
			continue
		}

		err = parseMetadataRecord(buffer.String(), meta)

		if err != nil {
			return nil, "", err
		}

		buffer.Reset()
	}

	return meta, content[i:], nil
}

// parseMetadataRecord process line with some metadata info
func parseMetadataRecord(data string, meta *PostMeta) error {
	var (
		err      error
		property string
		value    string
	)

	delimiter := strings.Index(data, ":")

	if delimiter == -1 || len(data) < delimiter+3 {
		return fmt.Errorf("Misformatted meta")
	}

	property, value = data[:delimiter], data[delimiter+2:]

	// Remove spaces
	property = strings.TrimLeft(property, " ")
	value = strings.TrimLeft(value, " ")

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

	case "authorlink":
		meta.AuthorLink = value

	case "tags":
		meta.Tags = strings.Fields(value)

	case "type":
		meta.Type = strings.ToLower(value)

	case "protected":
		meta.Protected = strings.ToLower(value) == "true"

	default:
		return fmt.Errorf("Unsupported property \"%s\"", property)
	}

	return nil
}

func parseContent(data string, render *Render) (string, error) {
	var err error

	var dataSize = len(data)
	var cursor = -1

	var buffer = bytes.NewBuffer(nil)
	var macroBuffer = bytes.NewBuffer(nil)

	var content string
	var result string

	var (
		isMacro    bool
		macroName  string
		macroProps map[string]string
		macro      *Macro
	)

	var hasMacroses = len(render.Macroses) != 0

	for {
		cursor++

		if cursor == dataSize {
			break
		}

		if data[cursor] != '\n' {
			buffer.WriteByte(data[cursor])
			continue
		}

		// Macro processing
		if hasMacroses && rxMacro.Match(buffer.Bytes()) {
			if isMacro {
				result, err = processMutlilineMacro(macro, macroProps, macroBuffer, render)

				if err != nil {
					return "", err
				}

				content += result + "\n"
				isMacro, macro, macroName, macroProps = false, nil, "", nil

				// Clean both buffers
				macroBuffer.Reset()
				buffer.Reset()

				continue
			}

			macroName, macro, macroProps = parseMacro(buffer, render)

			if macro == nil {
				result, err = processUnsuportedMacro(macroName, render)

				if err != nil {
					return "", err
				}

				content += result + "\n"
				buffer.Reset()
				continue
			}

			if macro.Multiline {
				isMacro = true
				buffer.Reset()
				continue
			}

			result, err = processSimpleMacro(macro, macroProps, render)

			if err != nil {
				return "", err
			}

			content += result + "\n"
			buffer.Reset()
			continue
		}

		if isMacro {
			buffer.WriteTo(macroBuffer)
			macroBuffer.WriteByte('\n')
			buffer.Reset()
			continue
		}

		if len(bytes.Trim(buffer.Bytes(), " ")) == 0 {
			buffer.Reset()
			continue
		}

		result, err = processContentData(buffer, render)

		if err != nil {
			return "", err
		}

		content += result + "\n"

		buffer.Reset()
	}

	return content, nil
}

// processContentData process and render post content
func processContentData(data *bytes.Buffer, render *Render) (string, error) {
	switch {
	case rxFmtHeader.Match(data.Bytes()):
		return processHeaderData(data, render)
	case rxFmtImage.Match(data.Bytes()):
		return processImageData(data, render)
	}

	return processParagraphData(data, render)
}

// processHeaderData process and render header data
func processHeaderData(data *bytes.Buffer, render *Render) (string, error) {
	if render.Header == nil {
		return data.String(), nil
	}

	text, level := parseHeader(data)

	if level == -1 {
		return "", fmt.Errorf("Can't parse header line \"%s\"", data)
	}

	return render.Header(text, level), nil
}

// processImageData process and render image data
func processImageData(data *bytes.Buffer, render *Render) (string, error) {
	var err error

	if render.Image == nil {
		return data.String(), nil
	}

	url, alt, caption := parseImage(data)

	if caption != "" {
		caption, err = parseParagraph(bytes.NewBufferString(caption), render)

		if err != nil {
			return "", err
		}
	}

	return render.Image(url, alt, caption), nil
}

// processParagraphData process and render paragraph data
func processParagraphData(data *bytes.Buffer, render *Render) (string, error) {
	result, err := parseParagraph(data, render)

	if err != nil {
		return "", err
	}

	if render.Paragraph == nil {
		return result, nil
	}

	return render.Paragraph(result), nil
}

// parseHeader parse header tag
func parseHeader(data *bytes.Buffer) (string, int) {
	headerInfo := rxFmtHeader.FindStringSubmatch(data.String())
	level, err := strconv.Atoi(headerInfo[1])

	if err != nil {
		return "", -1
	}

	return headerInfo[2], level
}

// parseImage parse image tag
func parseImage(data *bytes.Buffer) (string, string, string) {
	var url, alt, caption string

	imageTag := rxFmtImage.FindAllString(data.String(), -1)
	imageTagSlice := strings.Split(imageTag[0], "!")

	url, caption = imageTagSlice[1], strings.TrimLeft(imageTagSlice[2], " ")

	if strings.Contains(url, "|") {
		urlSlice := strings.Split(url, "|")
		url, alt = urlSlice[0], urlSlice[1]
	}

	return url, alt, caption
}

// parseParagraph parse paragraph data
func parseParagraph(data *bytes.Buffer, render *Render) (string, error) {
	var (
		tags [][][]byte
		tag  [][]byte
	)

	var dataBytes = data.Bytes()
	var hasMacro = len(render.Macroses) != 0

	if render.Bold != nil && rxFmtBold.Match(dataBytes) {
		tags = rxFmtBold.FindAllSubmatch(dataBytes, -1)

		for _, tag = range tags {
			dataBytes = bytes.Replace(dataBytes, tag[0], []byte(render.Bold(string(tag[1]))), -1)
		}
	}

	if render.Italic != nil && rxFmtItalic.Match(dataBytes) {
		tags = rxFmtItalic.FindAllSubmatch(dataBytes, -1)

		for _, tag = range tags {
			dataBytes = bytes.Replace(dataBytes, tag[0], []byte(render.Italic(string(tag[1]))), -1)
		}
	}

	if render.Del != nil && rxFmtDel.Match(dataBytes) {
		tags = rxFmtDel.FindAllSubmatch(dataBytes, -1)

		for _, tag = range tags {
			dataBytes = bytes.Replace(dataBytes, tag[0], []byte(render.Del(string(tag[1]))), -1)
		}
	}

	if render.Sup != nil && rxFmtSup.Match(dataBytes) {
		tags = rxFmtSup.FindAllSubmatch(dataBytes, -1)

		for _, tag = range tags {
			dataBytes = bytes.Replace(dataBytes, tag[0], []byte(render.Sup(string(tag[1]))), -1)
		}
	}

	if render.Sub != nil && rxFmtSub.Match(dataBytes) {
		tags = rxFmtSub.FindAllSubmatch(dataBytes, -1)

		for _, tag = range tags {
			dataBytes = bytes.Replace(dataBytes, tag[0], []byte(render.Sub(string(tag[1]))), -1)
		}
	}

	if render.Hr != nil && rxFmtHr.Match(dataBytes) {
		tags = rxFmtHr.FindAllSubmatch(dataBytes, -1)

		for _, tag = range tags {
			dataBytes = bytes.Replace(dataBytes, tag[0], []byte(render.Hr()), -1)
		}
	}

	if render.Code != nil && rxFmtCode.Match(dataBytes) {
		tags = rxFmtCode.FindAllSubmatch(dataBytes, -1)

		for _, tag = range tags {
			dataBytes = bytes.Replace(dataBytes, tag[0], []byte(render.Code(string(tag[1]))), -1)
		}
	}

	if render.Link != nil && rxFmtLink.Match(dataBytes) {
		tags = rxFmtLink.FindAllSubmatch(dataBytes, -1)

		for _, tag = range tags {
			dataBytes = bytes.Replace(dataBytes, tag[0], []byte(render.Link(string(tag[2]), string(tag[1]))), -1)
		}
	}

	if hasMacro && rxInlineMacro.Match(dataBytes) {
		tags = rxInlineMacro.FindAllSubmatch(dataBytes, -1)

		for _, tag = range tags {
			macroName, macro, macroProps := parseMacro(bytes.NewBuffer(tag[0]), render)

			var result string
			var err error

			if macro == nil {
				result, err = processUnsuportedMacro(macroName, render)
			} else {
				if macro.Multiline {
					continue
				}

				result, err = processSimpleMacro(macro, macroProps, render)
			}

			if err != nil {
				return "", err
			}

			dataBytes = bytes.Replace(dataBytes, tag[0], []byte(result), -1)
		}
	}

	return string(dataBytes), nil
}

// parseMacro extract macro name from given data and return macro struct
func parseMacro(data *bytes.Buffer, render *Render) (string, *Macro, map[string]string) {
	var (
		macro *Macro
		name  string
		props map[string]string
	)

	macroTag := rxInlineMacro.FindSubmatch(data.Bytes())
	name = string(macroTag[1])

	for _, m := range render.Macroses {
		if name == m.Name {
			macro = m
			break
		}
	}

	if len(macroTag[2]) != 0 {
		props = parseMacroProps(string(macroTag[2]))
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

func processSimpleMacro(macro *Macro, macroProps map[string]string, render *Render) (string, error) {
	if macro.Handler == nil {
		return "", fmt.Errorf("Handler is nil for \"%s\" macro", macro.Name)
	}

	return macro.Handler("", macroProps), nil
}

func processMutlilineMacro(macro *Macro, macroProps map[string]string, data *bytes.Buffer, render *Render) (string, error) {
	fmt.Println(data.String())

	if macro.Handler == nil {
		return "", fmt.Errorf("Handler is nil for \"%s\" macro", macro.Name)
	}

	return macro.Handler(data.String(), macroProps), nil
}

// processUnsuportedMacro process unsupported macro
func processUnsuportedMacro(macroName string, render *Render) (string, error) {
	if render.UnsupportedMacro != nil {
		return render.UnsupportedMacro(macroName), nil
	}

	return "", nil
}

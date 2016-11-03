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

	"pkg.re/essentialkaos/ek.v5/fsutil"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	TYPE_POST  uint8 = 0
	TYPE_PHOTO       = 1
	TYPE_QUOTE       = 2
)

// ////////////////////////////////////////////////////////////////////////////////// //

type PostMeta struct {
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
	Link      func(url, text string) string
	Image     func(url, alt, caption string) string
	Code      func(text string) string
	CodeBlock func(text, lang string) string

	// You can extend formating by custom macroses
	Macroses map[string]func(args []string) string
}

// ////////////////////////////////////////////////////////////////////////////////// //

var postTypes = map[string]uint8{
	"":      TYPE_POST,
	"post":  TYPE_POST,
	"photo": TYPE_PHOTO,
	"quote": TYPE_QUOTE,
}

var (
	rxMeta         = regexp.MustCompile(`^[+]{4}`)
	rxFmtHeader    = regexp.MustCompile(`h([1-6])\. (.*)`)
	rxFmtItalic    = regexp.MustCompile(`_([\p{L}\d\S]{1}.*?)_`)
	rxFmtBold      = regexp.MustCompile(`\*([\p{L}\d\S]{1}.*?)\*`)
	rxFmtDel       = regexp.MustCompile(`~([\p{L}\d\S]{1}.*?)~`)
	rxFmtQuote     = regexp.MustCompile(`^>.*`)
	rxFmtCode      = regexp.MustCompile("\\`" + `([\p{L}\d\S]{1}.*)` + "\\`")
	rxFmtImage     = regexp.MustCompile(`\![\w\S]{1,}\.(jpg|jpeg|gif|png)(?:(!|\|)).*`)
	rxFmtLink      = regexp.MustCompile(`\[(.*?)?(?:\|)?((?:http|https|ftp|mailto)[\S]{3,})\]`)
	rxFmtCodeBlock = regexp.MustCompile("^[\\`]{3,4}([\\w\\d]{1,})?")
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Process parse and render post file
func Process(file string, render *Render) (*Post, error) {
	err := checkFile(file)

	if err != nil {
		return nil, err
	}

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

// checkFile check file for some errors
func checkFile(file string) error {
	if !fsutil.IsExist(file) {
		return fmt.Errorf("File %s is not exist", file)
	}

	if !fsutil.IsRegular(file) {
		return fmt.Errorf("File %s is not a file", file)
	}

	if !fsutil.IsReadable(file) {
		return fmt.Errorf("File %s is not readable", file)
	}

	if !fsutil.IsNonEmpty(file) {
		return fmt.Errorf("File %s is empty", file)
	}

	return nil
}

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

	var isMeta bool
	var isQuote bool
	var isCode bool

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Trim(line, " ") == "" {
			continue
		}

		switch {
		case rxMeta.MatchString(line):
			isMeta = !isMeta
			continue
		case rxFmtQuote.MatchString(line):
			isQuote = !isQuote
		case rxFmtCodeBlock.MatchString(line):
			isCode = !isCode
		}

		if isMeta {
			processMetaData(line, post.Meta)
		} else {
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

func processParagraphData(data string, render *Render, post *Post) error {
	data = parseParagraph(data, render)

	if render.Paragraph == nil {
		appendPostData(post, data)
	}

	appendPostData(post, render.Paragraph(data))

	return nil
}

func parseHeader(data string) (string, int) {
	headerInfo := rxFmtHeader.FindStringSubmatch(data)
	level, err := strconv.Atoi(headerInfo[1])

	if err != nil {
		return "", -1
	}

	return headerInfo[2], level
}

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

func parseParagraph(data string, render *Render) string {
	if render.Bold != nil && rxFmtBold.MatchString(data) {
		tags := rxFmtBold.FindAllStringSubmatch(data, -1)

		for _, tag := range tags {
			data = strings.Replace(data, tag[0], render.Bold(tag[1]), -1)
		}
	}

	if render.Italic != nil && rxFmtItalic.MatchString(data) {
		tags := rxFmtItalic.FindAllStringSubmatch(data, -1)

		for _, tag := range tags {
			data = strings.Replace(data, tag[0], render.Italic(tag[1]), -1)
		}
	}

	if render.Del != nil && rxFmtDel.MatchString(data) {
		tags := rxFmtDel.FindAllStringSubmatch(data, -1)

		for _, tag := range tags {
			data = strings.Replace(data, tag[0], render.Del(tag[1]), -1)
		}
	}

	if render.Code != nil && rxFmtCode.MatchString(data) {
		tags := rxFmtCode.FindAllStringSubmatch(data, -1)

		for _, tag := range tags {
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

func appendPostData(post *Post, data string) {
	post.Data += data + "\n"
}

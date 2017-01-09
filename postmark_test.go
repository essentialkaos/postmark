package postmark

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2017 ESSENTIAL KAOS                         //
//        Essential Kaos Open Source License <https://essentialkaos.com/ekol>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	. "pkg.re/check.v1"
)

// ////////////////////////////////////////////////////////////////////////////////// //

func Test(t *testing.T) { TestingT(t) }

// ////////////////////////////////////////////////////////////////////////////////// //

type PostmarkSuite struct{}

// ////////////////////////////////////////////////////////////////////////////////// //

var _ = Suite(&PostmarkSuite{})

// ////////////////////////////////////////////////////////////////////////////////// //

var emptyRender = &Render{}

var debugRender = &Render{
	Header:      func(text string, level int) string { return fmt.Sprintf("Level: %d Text: %s", level, text) },
	Paragraph:   func(text string) string { return fmt.Sprintf("  %s", text) },
	Bold:        func(text string) string { return fmt.Sprintf("(Bold: %s)", text) },
	Italic:      func(text string) string { return fmt.Sprintf("(Italic: %s)", text) },
	Underline:   func(text string) string { return fmt.Sprintf("(Underline: %s)", text) },
	Del:         func(text string) string { return fmt.Sprintf("(Del: %s)", text) },
	Sup:         func(text string) string { return fmt.Sprintf("(Sup: %s)", text) },
	Sub:         func(text string) string { return fmt.Sprintf("(Sub: %s)", text) },
	Code:        func(text string) string { return fmt.Sprintf("(Code: %s)", text) },
	Hr:          func() string { return "(HR)" },
	Link:        func(url, text string) string { return fmt.Sprintf("(URL: %s Text: \"%s\")", url, text) },
	InlineImage: func(url, alt string) string { return fmt.Sprintf("(URL: %s Alt: \"%s\")", url, alt) },
	Image: func(url, alt, caption string) string {
		return fmt.Sprintf("(URL: %s Alt: \"%s\" Caption: \"%s\"", url, alt, caption)
	},

	UnsupportedMacro: func(macroName string) string { return fmt.Sprintf("(Unsupported macro \"%s\")", macroName) },

	Macroses: []*Macro{
		{
			Name:      "macro1",
			Multiline: false,
			Handler: func(data string, props map[string]string) string {
				return fmt.Sprintf("(macro1 Data: \"%s\" Props: [%s])", data, props[""])
			}},
		{
			Name:      "macro2",
			Multiline: false,
			Handler: func(data string, props map[string]string) string {
				return fmt.Sprintf("(macro2 Data: \"%s\" Props: [%s %s %s %s])",
					data, props[""], props["prop1"], props["prop2"], props["prop3"])
			}},
		{
			Name:      "macro3",
			Multiline: true,
			Handler: func(data string, props map[string]string) string {
				return fmt.Sprintf("(macro3 Data: \"%s\" Props: [%s])",
					strings.Replace(data, "\n", "+", -1), props[""])
			}},
	},
}

// ////////////////////////////////////////////////////////////////////////////////// //

func (s *PostmarkSuite) TestParsingErrors(c *C) {
	post, err := Process("testdata/empty.post", nil)

	c.Assert(post, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Render is nil")

	post, err = Process("testdata/empty.post", emptyRender)

	c.Assert(post, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "File is empty")

	post, err = Process("testdata/without-meta.post", emptyRender)

	c.Assert(post, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Metadata section is missed")

	post, err = Process("testdata/not-exist.post", emptyRender)

	c.Assert(post, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "open testdata/not-exist.post: no such file or directory")
}

func (s *PostmarkSuite) TestMetaParsing(c *C) {
	post, err := Process("testdata/all.post", emptyRender)

	c.Assert(post, NotNil)
	c.Assert(err, IsNil)

	c.Assert(post.Meta.Title, Equals, "Post title")
	c.Assert(post.Meta.Name, Equals, "my_unique_post")
	c.Assert(post.Meta.Author, Equals, "John Doe")
	c.Assert(post.Meta.AuthorLink, Equals, "https://www.domain.com")
	c.Assert(post.Meta.Date.Unix(), Equals, int64(1443133080))
	c.Assert(post.Meta.Tags, DeepEquals, []string{"tag1", "tag2", "tag3"})
	c.Assert(post.Meta.Type, Equals, "my-super-type")
	c.Assert(post.Meta.Protected, Equals, true)
}

func (s *PostmarkSuite) TestMetaParsingErrors(c *C) {
	meta, err := extractMeta(bytes.NewBufferString("++++\nTest\n++++\n"))

	c.Assert(meta, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Metadata section is missformated")

	meta, err = extractMeta(bytes.NewBufferString("++++\nDate: ABCD\n++++\n"))

	c.Assert(meta, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "parsing time \"ABCD\" as \"2006/01/02 15:04\": cannot parse \"ABCD\" as \"2006\"")

	meta, err = extractMeta(bytes.NewBufferString("++++\nUnknown: 1234\n++++\n"))

	c.Assert(meta, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Unsupported property \"Unknown\"")
}

func (s *PostmarkSuite) TestEmptyRender(c *C) {
	post, err := Process("testdata/all.post", emptyRender)

	c.Assert(post, NotNil)
	c.Assert(err, IsNil)

	data, err := ioutil.ReadFile("testdata/all-empty.result")

	c.Assert(data, NotNil)
	c.Assert(err, IsNil)

	c.Assert(post.Content, Equals, string(data))
}

func (s *PostmarkSuite) TestDebugRender(c *C) {
	post, err := Process("testdata/all.post", debugRender)

	c.Assert(post, NotNil)
	c.Assert(err, IsNil)

	data, err := ioutil.ReadFile("testdata/all-debug.result")

	c.Assert(data, NotNil)
	c.Assert(err, IsNil)

	c.Assert(post.Content, Equals, string(data))
}

func (s *PostmarkSuite) TestRenderApply(c *C) {
	data1 := "This is example _italic_ text."
	data2 := "This is example _italic_ text.\nThis is example *bold* text.\n"

	rendered, err := emptyRender.Apply("")

	c.Assert(rendered, Equals, "")
	c.Assert(err, IsNil)

	rendered, err = emptyRender.Apply(data1)

	c.Assert(rendered, Equals, data1)
	c.Assert(err, IsNil)

	rendered, err = emptyRender.Apply(data2)

	c.Assert(rendered, Equals, data2)
	c.Assert(err, IsNil)

	rendered, err = debugRender.Apply(data1)

	c.Assert(rendered, Equals, "This is example (Italic: italic) text.")
	c.Assert(err, IsNil)

	rendered, err = debugRender.Apply(data2)

	c.Assert(rendered, Equals, "  This is example (Italic: italic) text.\n  This is example (Bold: bold) text.\n")
	c.Assert(err, IsNil)
}

func (s *PostmarkSuite) TestPostValidation(c *C) {
	var post0 *Post

	post1 := &Post{}
	post2 := &Post{Meta: &PostMeta{}}
	post3 := &Post{Meta: &PostMeta{Title: "ABCD", Author: "John"}}
	post4 := &Post{Meta: &PostMeta{Title: "ABCD", Author: "John"}, Content: "DATA"}

	c.Assert(post0.IsValid(), Equals, false)
	c.Assert(post1.IsValid(), Equals, false)
	c.Assert(post2.IsValid(), Equals, false)
	c.Assert(post3.IsValid(), Equals, false)
	c.Assert(post4.IsValid(), Equals, true)
}

func (s *PostmarkSuite) TestMacroHandlers(c *C) {
	macro := &Macro{Name: "test"}

	rendered, err := processSimpleMacro(macro, nil, emptyRender)

	c.Assert(rendered, Equals, "")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Handler is nil for \"test\" macro")

	rendered, err = processMutlilineMacro(macro, nil, nil, emptyRender)

	c.Assert(rendered, Equals, "")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Handler is nil for \"test\" macro")

	rendered, err = processUnsuportedMacro("test", emptyRender)

	c.Assert(rendered, Equals, "")
	c.Assert(err, IsNil)
}

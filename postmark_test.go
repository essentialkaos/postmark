package postmark

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2016 Essential Kaos                         //
//      Essential Kaos Open Source License <http://essentialkaos.com/ekol?en>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
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
		{"macro1", false, func(data string, props map[string]string) string {
			return fmt.Sprintf("(Data: \"%s\" Props: %v)", data, props)
		}},
		{"macro2", false, func(data string, props map[string]string) string {
			return fmt.Sprintf("(Data: \"%s\" Props: %v)", data, props)
		}},
		{"macro3", true, func(data string, props map[string]string) string {
			return fmt.Sprintf("(Data: \"%s\" Props: %v)", strings.Replace(data, "\n", "+", -1), props)
		}},
	},
}

// ////////////////////////////////////////////////////////////////////////////////// //

func (s *PostmarkSuite) TestMetaParsingErrors(c *C) {
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

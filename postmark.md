# Postmark Syntax

## Metadata
Metadata must be placed at top of post. Metadata must be preceded and followed by `++++` tag.

Example:

```
++++
Title: Post title
Date: 2015/09/24 22:18
Author: John Doe
++++

This is post content.
```

List of supported metadata fields:
* `Title` - post title
* `Name` - custom post name (_can be used for url generation_)
* `Date` - publication date
* `Author` - author name
* `Tags` - space separated tags
* `Type` - post type (`post / image / quote / link`)
* `Protected` - protection flag (_for secret posts of something_)

## Content
### Basic
#### Headers

* `h1.` Level 1 header
* `h2.` Level 2 header
* `h3.` Level 3 header
* `h4.` Level 4 header
* `h5.` Level 5 header
* `h6.` Level 6 header

Example:

```
h1. My suppa-duppa post
```

#### Text modificators

* `_content_` - Italic
* `*content*` - Bold
* `-content-` - Deleted
* `+content+` - Underline
* `^content^` - Superscript
* `~content~` - Subscript
* ``content`` - Monospace

Example:
```
This is _italic_ and *bold* text.
```

package main

import (
	"testing"
)

func assert(t *testing.T, expected interface{}, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Error("Expected:", expected, "but got", actual, "---")
	}
}

// Tests padding of a capitalised command line like
func TestPadding_Command(t *testing.T) {
	dockerfile := `
FROM golang:alpine as builder
RUN blah
`
	expectedFmt := `
FROM  golang:alpine as builder
RUN   blah
`

	df := &file{
		currentLine:  1,
		originalFile: []byte(dockerfile),
	}
	df.calculateLongestLineLength()

	actual, err := doFmt(dockerfile)
	if err != nil {
		t.Error(err)
	}
	assert(t, expectedFmt, actual)
}

func TestPadding_Comment(t *testing.T) {
	dockerfile := `
##
# Test comment
## Two hashes

#Slightly different comment
##Two hashes
`
	expectedFmt := `
##
# Test comment
## Two hashes

# Slightly different comment
## Two hashes
`

	df := &file{
		currentLine:  1,
		originalFile: []byte(dockerfile),
	}
	df.calculateLongestLineLength()

	actual, err := doFmt(dockerfile)
	if err != nil {
		t.Error(err)
	}
	assert(t, expectedFmt, actual)
}

func TestPadding_CommentInline(t *testing.T) {

}

func TestPadding_CommentTabbed(t *testing.T) {
	dockerfile := `
# Test comment
#

	# Foo
RUN echo "bloop"
	#Slightly different comment
`
	expectedFmt := `
# Test comment
#

# Foo
RUN   echo "bloop"
# Slightly different comment
`

	df := &file{
		currentLine:  1,
		originalFile: []byte(dockerfile),
	}
	df.calculateLongestLineLength()

	actual, err := doFmt(dockerfile)
	if err != nil {
		t.Error(err)
	}
	assert(t, expectedFmt, actual)
}

func TestPadding_DoubleComment(t *testing.T) {
	dockerfile := `
##
## Test comment
##

	# Foo
RUN echo "bloop"
	#Slightly different comment
`
	expectedFmt := `
##
## Test comment
##

# Foo
RUN   echo "bloop"
# Slightly different comment
`

	df := &file{
		currentLine:  1,
		originalFile: []byte(dockerfile),
	}
	df.calculateLongestLineLength()

	actual, err := doFmt(dockerfile)
	if err != nil {
		t.Error(err)
	}
	assert(t, expectedFmt, actual)
}

func TestPadding_CapitalisedNonCommand(t *testing.T) {
	dockerfile := `
ENV	VAR=foo
	BAR=baz
RUN echo "honk" && \
	SOME_VAR=bat echo "$SOME_VAR"
`
	expectedFmt := `
ENV   VAR=foo
      BAR=baz
RUN   echo "honk" \
      && SOME_VAR=bat echo "$SOME_VAR"
`

	df := &file{
		currentLine:  1,
		originalFile: []byte(dockerfile),
	}
	df.calculateLongestLineLength()

	actual, err := doFmt(dockerfile)
	if err != nil {
		t.Error(err)
	}
	assert(t, expectedFmt, actual)
}

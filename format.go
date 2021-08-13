package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

// const formatHelp = `Format the Dockerfile(s).`

// func (cmd *formatCommand) Name() string      { return "fmt" }
// func (cmd *formatCommand) Args() string      { return "[OPTIONS] DOCKERFILE [DOCKERFILE...]" }
// func (cmd *formatCommand) ShortHelp() string { return formatHelp }
// func (cmd *formatCommand) LongHelp() string  { return formatHelp }
// func (cmd *formatCommand) Hidden() bool      { return false }

// func (cmd *formatCommand) Register(fs *flag.FlagSet) {
// 	fs.BoolVar(&cmd.diff, "diff", false, "display diffs instead of rewriting files")
// 	fs.BoolVar(&cmd.diff, "D", false, "display diffs instead of rewriting files")

// 	fs.BoolVar(&cmd.list, "list", false, "list files whose formatting differs from dockfmt's")
// 	fs.BoolVar(&cmd.list, "l", false, "list files whose formatting differs from dockfmt's")

// 	fs.BoolVar(&cmd.write, "write", false, "write result to (source) file instead of stdout")
// 	fs.BoolVar(&cmd.write, "w", false, "write result to (source) file instead of stdout")
// }

// type formatCommand struct {
// 	diff  bool
// 	list  bool
// 	write bool
// }

type file struct {
	currentLine       int
	name              string
	originalFile      []byte
	longestLineLength int
}

var (
	// Char/chars to use as tab
	tabChars = "    "
	// How far to indent 2nd column, commands longer than this will be single spaced
	initialTab = 6
)

// func (cmd *formatCommand) Run(ctx context.Context, args []string) error {
func Run(ctx context.Context, args []string) error {

	err := forFile(args, func(f string, nodes []*parser.Node) error {

		og, err := ioutil.ReadFile(f)
		if err != nil {
			return err
		}

		df := &file{
			currentLine:       1,
			name:              f,
			originalFile:      og,
			longestLineLength: 0,
		}

		var result string
		s, _ := df.getOriginalAsString()
		r, _ := doFmt(s)

		result += r

		// make a temporary backup before overwriting original
		bakname, err := backupFile(f+".", og, 0644)
		if err != nil {
			return err
		}

		if err := ioutil.WriteFile(f, []byte(result), 0644); err != nil {
			os.Rename(bakname, f)
			return err
		}

		if err := os.Remove(bakname); err != nil {
			return fmt.Errorf("could not remove backup file %s: %v", bakname, err)
		}

		return nil
	})

	return err
}

func doFmt(s string) (result string, err error) {

	lines := strings.Split(s, "\n")

	nextLookback := 1
	for i, line := range lines {

		// don't do anything within quotes
		rQuotes, _ := regexp.Compile(`(?:[^\\]((\\.)*))('(?:\\.|[^\\"'])*'|"(?:\\.|[^\\"'])*")`)
		m := rQuotes.FindAllString(line, -1)

		var reconstituted string
		if m != nil {
			// for everything not quoted in the line go to work on it
			for i, q := range rQuotes.Split(line, -1) {

				// It's a comment
				rComments, _ := regexp.Compile(`#(\S)`)
				if rComments.MatchString(q) {

					q = rComments.ReplaceAllString(q, "# $1")

				}

				// Minimise whitespace, we'll add in desired padding later
				line = removeWhitespace(line)

				//reconstruct
				if i < len(m) {
					reconstituted += q + m[i]
					continue
				}
				reconstituted += q
			}

			if reconstituted != "" {
				line = reconstituted
			}

		} else {
			// No quoted string
			// Minimise whitespace, we'll add in desired padding later
			line = removeWhitespace(line)
		}

		line = strings.TrimSpace(line)

		// Pad square brackets
		squareBraces := regexp.MustCompile(`\[(.+)\]`)
		line = squareBraces.ReplaceAllString(line, "[ $1 ]")

		// Check the previous line to see if it ends with && \
		// If it does, remove it and add it here at the start
		if strings.HasPrefix(line, "#") {
			// Skip processing "&& \" for this line but remember to look back past it next check
			nextLookback += 1
		} else {

			aa := regexp.MustCompile(`(?mU)^(.*)&& (\\)$`)
			if i > nextLookback {
				if aa.MatchString(lines[i-nextLookback]) {

					lines[i-nextLookback] = aa.ReplaceAllString(lines[i-nextLookback], "$1$2")

					line = "&& " + line
				}
				// Now that we've caught up, reset nextLookback
				nextLookback = 1
			}
		}

		// Update lines with what we've got before we pass to padLine (we've been working with a copy)
		lines[i] = line

		// Perform whitespace padding
		lines[i] = padLine(i, lines)

	}

	result = strings.Join(lines, "\n")
	return
}

func removeWhitespace(l string) string {

	mws := regexp.MustCompile(`\s\s+`)

	l = mws.ReplaceAllString(l, " ")
	l = strings.TrimSpace(l)

	return l

}

var currentCmd string

func padLine(index int, lines []string) string {

	line := lines[index]
	prevLine := ""
	if index > 0 {
		prevLine = lines[index-1]
	}

	// TODO: Maybe indent comments when they are directly below an indented block
	// Line has a comment without a space directly following it
	rCmtSpace := regexp.MustCompile(`(#+)([^\r\n\t\f\v #]+)`)
	if rCmtSpace.MatchString(line) {
		// Make sure theres a space after however many initial hashes they used
		line = rCmtSpace.ReplaceAllString(line, "$1 $2")

	}

	f := strings.Fields(line)

	// Starts with a capitalised command
	// Do this after other transforms due to tabing
	if regexp.MustCompile(`^[A-Z]+\s+`).MatchString(line) {

		currentCmd = f[0]
		tab := initialTab
		if len(f[0]) > initialTab {
			tab = len(f[0]) + 1
		}
		line = fmt.Sprintf("%-*s%s", tab, f[0], strings.Join(f[1:], " "))

	} else {

		if len(f) > 0 {
			// if a comment
			if regexp.MustCompile(`\s*# ?`).MatchString(line) {
				if strings.HasSuffix(prevLine, "\\") {

					line = indent(line, 4)

				}
			} else {
				// not a comment
				// Line up && lines at left indent border
				if strings.HasPrefix(line, "&& ") {
					line = indent(line, 0)
				} else {
					// Line up non && lines 4 spaces further in
					switch currentCmd {
					case "ENV":
						line = indent(line, 0) // ENV lines should line up at the regular indent
					default:
						line = indent(line, 4)
					}

				}
			}
		}
	}

	if len(f) > 0 {

		if regexp.MustCompile(`^\s+$`).MatchString(line) {
			line = strings.TrimSpace(line)
		}

	}

	return line
}

func indent(s string, level int) string {
	return fmt.Sprintf("%*s%s", initialTab+level, " ", s)
}

func (df *file) calculateLongestLineLength() {
	scanner := bufio.NewScanner(bytes.NewBuffer(df.originalFile))
	scanner.Split(bufio.ScanLines)
	var (
		i = 1
		t string
		l int
		r int
	)
	r = df.longestLineLength

	for scanner.Scan() {
		// scanner parses a little bit so may not be great for getting the raw length of each line...
		t = scanner.Text()
		l = len(t)
		if l > r {
			r = l
		}
		i++
	}

	df.longestLineLength = r

}

func (df *file) getOriginalAsString() (string, error) {
	scanner := bufio.NewScanner(bytes.NewBuffer(df.originalFile))
	scanner.Split(bufio.ScanLines)
	var (
		i = 1
		l string
	)
	for scanner.Scan() {
		l += scanner.Text() + "\n"
		i++
	}

	return l, nil
}

func getCmd(n *parser.Node, cmd []string) []string {
	if n == nil {
		return cmd
	}
	cmd = append(cmd, n.Value)
	if len(n.Flags) > 0 {
		cmd = append(cmd, n.Flags...)
	}
	if n.Next != nil {
		for node := n.Next; node != nil; node = node.Next {
			cmd = append(cmd, node.Value)
			if len(node.Flags) > 0 {
				cmd = append(cmd, node.Flags...)
			}
		}
	}
	return cmd
}

func (df *file) padToRight(s string) int {
	length := df.longestLineLength - len(s) + initialTab + 4
	if length < 4 {
		length = 4
	}
	return length
}

func trimAll(a []string) []string {
	for i, v := range a {
		a[i] = strings.TrimSpace(v)
	}
	return a
}

const chmodSupported = runtime.GOOS != "windows"

// backupFile writes data to a new file named filename<number> with permissions perm,
// with <number randomly chosen such that the file name is unique. backupFile returns
// the chosen file name.
func backupFile(filename string, data []byte, perm os.FileMode) (string, error) {
	// create backup file
	f, err := ioutil.TempFile(filepath.Dir(filename), filepath.Base(filename))
	if err != nil {
		return "", err
	}

	bakname := f.Name()
	if chmodSupported {
		err = f.Chmod(perm)
		if err != nil {
			f.Close()
			os.Remove(bakname)
			return bakname, err
		}
	}

	// write data to backup file
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}

	if err1 := f.Close(); err == nil {
		err = err1
	}

	return bakname, err
}

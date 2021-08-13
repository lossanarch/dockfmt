package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/genuinetools/pkg/cli"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/sirupsen/logrus"

	"github.com/lossanarch/dockfmt/version"
)

var (
	flagDebug   bool
	flagVersion bool
)

func main() {
	// Create a new cli program.
	p := cli.NewProgram()
	p.Name = "dockfmt"
	p.Description = "Dockerfile format."

	// Set the GitCommit and Version.
	p.GitCommit = version.GITCOMMIT
	p.Version = version.VERSION

	// Setup the global flags.
	p.FlagSet = flag.NewFlagSet("global", flag.ExitOnError)
	p.FlagSet.BoolVar(&flagDebug, "debug", false, "enable debug logging")
	p.FlagSet.BoolVar(&flagDebug, "d", false, "enable debug logging")
	p.FlagSet.BoolVar(&flagVersion, "v", false, "print version information")

	// p.Commands = []cli.Command{
	// 	// &baseCommand{},
	// 	// &dumpCommand{},
	// 	&formatCommand{},
	// 	// &maintainerCommand{},
	// }

	// cmd := &formatCommand{}

	p.Action = Run

	// Set the before function.
	p.Before = func(ctx context.Context) error {
		// Set the log level.
		if flagDebug {
			logrus.SetLevel(logrus.DebugLevel)
		}

		if flagVersion {
			fmt.Fprintf(os.Stderr, "Version: %s\nCommit: %s\n", version.VERSION, version.GITCOMMIT)
		}

		if p.FlagSet.NArg() < 1 {
			// return errors.New("pass in Dockerfile(s)")
			// just grab Dockerfile in pwd
		}

		return nil
	}

	// Run our program.
	p.Run()
}

type pair struct {
	Key   string
	Value int
}

type pairList []pair

func (p pairList) Len() int           { return len(p) }
func (p pairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p pairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func rank(images map[string]int) pairList {
	pl := make(pairList, len(images))
	i := 0
	for k, v := range images {
		pl[i] = pair{k, v}
		i++
	}
	sort.Sort(sort.Reverse(pl))
	return pl
}

func labelSearch(search string, n *parser.Node, a map[string]int) map[string]int {
	if n.Value == "label" {
		if n.Next != nil && strings.EqualFold(n.Next.Value, search) {
			i := strings.Trim(n.Next.Next.Value, "\"")
			if v, ok := a[i]; ok {
				a[i] = v + 1
			} else {
				a[i] = 1

			}
		}
	}
	return a
}

func nodeSearch(search string, n *parser.Node, a map[string]int) map[string]int {
	if n.Value == search {
		i := strings.Trim(n.Next.Value, "\"")
		if v, ok := a[i]; ok {
			a[i] = v + 1
		} else {
			a[i] = 1

		}
	}
	return a
}

func forFile(args []string, fnc func(string, []*parser.Node) error) error {
	for _, fn := range args {
		logrus.Debugf("parsing file: %s", fn)

		f, err := os.Open(fn)
		if err != nil {
			return err
		}
		defer f.Close()

		result, err := parser.Parse(f)
		if err != nil {
			return err
		}
		ast := result.AST
		nodes := []*parser.Node{}
		if ast.Children != nil {
			nodes = append(nodes, ast.Children...)
		}
		if err := fnc(fn, nodes); err != nil {
			return err
		}
	}
	return nil
}

package main

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/rosszurowski/tandem/ansi"
	"github.com/rosszurowski/tandem/tandem"
	"github.com/urfave/cli/v2"
)

var (
	name    = "tandem"
	version = "dev"
)

func main() {
	cwd, cwdErr := os.Getwd()
	app := &cli.App{
		Name:    name,
		Version: version,
		Usage:   "Run multiple commands in tandem",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "directory",
				Aliases:     []string{"d"},
				Usage:       "`path` to run commands from",
				Value:       cwd,
				DefaultText: "cwd",
				Action: func(ctx *cli.Context, v string) error {
					if v == "" && cwdErr != nil {
						return fmt.Errorf("could not get current working directory: %v", cwdErr)
					}
					return nil
				},
			},
			&cli.IntFlag{
				Name:  "timeout",
				Value: 5,
				Usage: "timeout (in `seconds`) for commands to exit gracefully before being killed",
				Action: func(ctx *cli.Context, v int) error {
					if v < 0 {
						return fmt.Errorf("--timeout/-t value must be above 0, got %v", v)
					}
					if v >= 65536 {
						return fmt.Errorf("--timeout/-t value must be below 65535, got %v", v)
					}
					return nil
				},
			},
			&cli.BoolFlag{
				Name:  "silent",
				Usage: "silence non-command output",
				Value: false,
			},
		},
		Action: func(c *cli.Context) error {
			args := c.Args()
			if args.Len() < 1 {
				return ErrNoCommands
			}
			pm, err := tandem.New(tandem.Config{
				Cmds:    args.Slice(),
				Root:    c.String("directory"),
				Timeout: c.Int("timeout"),
				Silent:  c.Bool("silent"),
			})
			if err != nil {
				return err
			}
			pm.Run()
			return nil
		},
		HideHelpCommand:       true,
		CustomAppHelpTemplate: usage,
	}

	sort.Sort(cli.FlagsByName(app.Flags))

	if err := app.Run(os.Args); err != nil {
		if errors.Is(err, ErrNoCommands) {
			fmt.Fprintf(os.Stderr, "%s %v\n", ansi.Red("Error:"), err)
		} else {
			fmt.Fprintf(os.Stderr, "%s %v\n", ansi.Red("Error:"), err)
		}
		os.Exit(1)
	}
}

var (
	ErrNoCommands = fmt.Errorf("no commands given")

	usage = fmt.Sprintf(`
  %s {{if .VisibleFlags}}[options]{{end}}{{if .ArgsUsage}}{{.ArgsUsage}}{{else}} <arguments...>{{end}}
{{if .Commands}}
  %s

{{range .Commands}}{{if not .HideHelp}}    {{join .Names ", "}}{{ "\t"}}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
  %s

  {{range .VisibleFlags}}  {{.}}
  {{end}}{{end}}
  %s

    %s

    $ {{.Name}} 'command-a "arg1"' 'command-b "arg1" "arg2'

    %s

    $ {{.Name}} 'npm:build:*'

`,
		ansi.Bold(name),
		ansi.Dim("Commands:"),
		ansi.Dim("Options:"),
		ansi.Dim("Examples:"),
		ansi.Dim("Run two commands in tandem:"),
		ansi.Dim("Run npm scripts starting with 'build:' in tandem"),
	)
)

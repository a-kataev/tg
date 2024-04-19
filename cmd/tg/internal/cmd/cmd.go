package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"
)

type FlagFunc func(*flag.FlagSet)

func EmptyFlagFunc() func(*flag.FlagSet) {
	return func(_ *flag.FlagSet) {}
}

type RunFunc func() error

type command struct {
	name string
	desc string
	flag *flag.FlagSet
	run  RunFunc
}

type Commander struct {
	output io.Writer
	cmds   map[string]*command
}

const rootCmd = "_"

func New() *Commander {
	cm := new(Commander)
	cm.output = os.Stderr
	cm.cmds = make(map[string]*command)

	cm.cmds[rootCmd] = new(command)
	cm.cmds[rootCmd].name = path.Base(os.Args[0])
	cm.cmds[rootCmd].flag = flag.NewFlagSet("", flag.ContinueOnError)
	cm.cmds[rootCmd].flag.SetOutput(io.Discard)

	return cm
}

func (c *Commander) SetOutput(out io.Writer) {
	c.output = out
}

func defaultValue(value flag.Value, defValue string) string {
	flagType := reflect.TypeOf(value)

	var newValue reflect.Value

	if flagType.Kind() == reflect.Pointer {
		newValue = reflect.New(flagType.Elem())
	} else {
		newValue = reflect.Zero(flagType)
	}

	newFlag, ok := newValue.Interface().(flag.Value)
	if !ok || defValue == newFlag.String() || defValue == "" {
		return ""
	}

	if newValue.String() == "<*flag.stringValue Value>" {
		return fmt.Sprintf(" (default %q)", defValue)
	}

	return fmt.Sprintf(" (default %v)", defValue)
}

func (c *Commander) usage(fset *flag.FlagSet) {
	maxLen := 0

	fset.VisitAll(func(ff *flag.Flag) {
		name, _ := flag.UnquoteUsage(ff)

		strLen := len(name) + len(ff.Name)

		if strLen > maxLen {
			maxLen = strLen
		}
	})

	maxLen += 4

	fset.VisitAll(func(ff *flag.Flag) {
		name, _ := flag.UnquoteUsage(ff)

		spaces := strings.Repeat(" ", maxLen-(len(name)+len(ff.Name)))
		def := defaultValue(ff.Value, ff.DefValue)

		fmt.Fprintf(c.output, "  --%s %s%s%s%s\n", ff.Name, name, spaces, ff.Usage, def)
	})
}

func (c *Commander) Root(name string, fn FlagFunc) {
	c.cmds[rootCmd].name = name

	fn(c.cmds[rootCmd].flag)
}

func (c *Commander) rootHelp() {
	fmt.Fprintf(c.output, "Usage:\n  %s [flags]", c.cmds[rootCmd].name)

	if len(c.cmds) > 1 {
		fmt.Fprintf(c.output, " [command]\n\nAvailable Commands:\n")

		cmds := make([]string, 0, len(c.cmds))

		maxCmdLen := 0

		for cmd := range c.cmds {
			if cmd == "_" {
				continue
			}

			cmds = append(cmds, cmd)

			if len(cmd) > maxCmdLen {
				maxCmdLen = len(cmd)
			}
		}

		sort.Strings(cmds)

		maxCmdLen += 4

		for _, cmd := range cmds {
			spaces := strings.Repeat(" ", maxCmdLen-len(cmd))

			fmt.Fprintf(c.output, "  %s%s%s\n", cmd, spaces, c.cmds[cmd].desc)
		}
	} else {
		fmt.Fprint(c.output, "\n")
	}

	fmt.Fprint(c.output, "\nFlags:\n")

	c.usage(c.cmds[rootCmd].flag)
}

func (c *Commander) rootError(err error) {
	fmt.Fprintf(c.output, "Error: %s\n\nRun '%s --help' for usage.\n", err, c.cmds[rootCmd].name)
}

func (c *Commander) Command(name, desc string, fn FlagFunc, runFn RunFunc) {
	fset := flag.NewFlagSet(name, flag.ContinueOnError)
	fset.SetOutput(io.Discard)

	fn(fset)

	c.cmds[name] = &command{
		name: name,
		desc: desc,
		flag: fset,
		run:  runFn,
	}
}

func (c *Commander) commandHelp(name string) {
	if cmd, ok := c.cmds[name]; ok {
		if cmd.desc != "" {
			fmt.Fprintf(c.output, "Description: %s\n\n", cmd.desc)
		}
	}

	flags := false

	c.cmds[name].flag.VisitAll(
		func(*flag.Flag) {
			if !flags {
				flags = true
			}
		},
	)

	fmt.Fprintf(c.output, "Usage:\n  %s [global flags] %s", c.cmds[rootCmd].name, name)

	if flags {
		fmt.Fprint(c.output, " [flags]\n\nFlags:\n")

		if cmd, ok := c.cmds[name]; ok {
			c.usage(cmd.flag)
		}
	} else {
		fmt.Fprint(c.output, "\n")
	}

	fmt.Fprint(c.output, "\nGlobal Flags:\n")

	c.usage(c.cmds[rootCmd].flag)
}

func (c *Commander) commandError(name string, err error) {
	fmt.Fprintf(c.output, "Error: %s\n\nRun '%s %s--help' for usage.\n", err, c.cmds[rootCmd].name, name)
}

func (c *Commander) Run() { //nolint:cyclop
	if err := c.cmds[rootCmd].flag.Parse(os.Args[1:]); err != nil {
		if errors.Is(flag.ErrHelp, err) {
			c.rootHelp()
		} else {
			c.rootError(err)
		}

		os.Exit(1)
	}

	if len(c.cmds[rootCmd].flag.Args()) == 0 {
		c.rootHelp()

		os.Exit(1)
	}

	name := c.cmds[rootCmd].flag.Args()[0]
	args := c.cmds[rootCmd].flag.Args()[1:]

	cmd, ok := c.cmds[name]
	if !ok || name == rootCmd {
		err := fmt.Sprintf("unknown command %q", name)

		c.rootError(errors.New(err)) //nolint:goerr113

		os.Exit(1)
	}

	if len(args) == 0 {
		flags := false

		c.cmds[name].flag.VisitAll(func(_ *flag.Flag) {
			if !flags {
				flags = true
			}
		})

		if flags {
			c.commandHelp(name)

			os.Exit(1)
		}
	}

	if err := cmd.flag.Parse(args); err != nil {
		if errors.Is(flag.ErrHelp, err) {
			c.commandHelp(name)
		} else {
			c.commandError(name, err)
		}

		os.Exit(1)
	}

	if err := cmd.run(); err != nil {
		fmt.Fprintln(c.output, "Error: ", err.Error())

		os.Exit(1)
	}

	os.Exit(0)
}

package main

import (
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

type readLiner interface {
	ReadLine() (string, error)
}

type commandContext struct {
	args           []string
	stdin          readLiner
	stdout, stderr io.Writer
	pty            bool
	user           string
}

type command interface {
	execute(context commandContext) (uint32, error)
}

var commands = map[string]command{
	"sh":     cmdShell{},
	"true":   cmdTrue{},
	"false":  cmdFalse{},
	"echo":   cmdEcho{},
	"cat":    cmdCat{},
	"su":     cmdSu{},
	"whoami": cmdWhoami{},
	"ls":     cmdLs{},
	"ps":     cmdPs{},
}

var shellProgram = []string{"sh"}

func executeProgram(context commandContext) (uint32, error) {
	if len(context.args) == 0 {
		return 0, nil
	}
	command := commands[context.args[0]]
	if command == nil {
		_, err := fmt.Fprintf(context.stderr, "%v: command not found\n", context.args[0])
		return 127, err
	}
	return command.execute(context)
}

type cmdShell struct{}

func (cmdShell) execute(context commandContext) (uint32, error) {
	var prompt string
	if context.pty {
		switch context.user {
		case "root":
			prompt = "# "
		default:
			prompt = "$ "
		}
	}
	var lastStatus uint32
	var line string
	var err error
	for {
		_, err = fmt.Fprint(context.stdout, prompt)
		if err != nil {
			return lastStatus, err
		}
		line, err = context.stdin.ReadLine()
		if err != nil {
			return lastStatus, err
		}
		args := strings.Fields(line)
		if len(args) == 0 {
			continue
		}
		if args[0] == "exit" {
			var err error
			var status uint64 = uint64(lastStatus)
			if len(args) > 1 {
				status, err = strconv.ParseUint(args[1], 10, 32)
				if err != nil {
					status = 255
				}
			}
			return uint32(status), nil
		}
		newContext := context
		newContext.args = args
		if lastStatus, err = executeProgram(newContext); err != nil {
			return lastStatus, err
		}
	}
}

type cmdTrue struct{}

func (cmdTrue) execute(context commandContext) (uint32, error) {
	return 0, nil
}

type cmdFalse struct{}

func (cmdFalse) execute(context commandContext) (uint32, error) {
	return 1, nil
}

type cmdWhoami struct{}

func (cmdWhoami) execute(context commandContext) (uint32, error) {
	cmd := exec.Command("whoami")
	out, err := cmd.Output()

	if err != nil {
		fmt.Fprintln(context.stdout, err)
	}

	fmt.Fprint(context.stdout, string(out))
	return 0, err
}

type cmdLs struct{}

func (cmdLs) execute(context commandContext) (uint32, error) {
	cmd := exec.Command("ls")
	out, err := cmd.Output()

	if err != nil {
		fmt.Fprintln(context.stderr, err)
	}

	fmt.Fprint(context.stdout, string(out))
	return 0, err
}

type cmdPs struct{}

func (cmdPs) execute(context commandContext) (uint32, error) {
	cmd := exec.Command("ps")
	out, err := cmd.Output()

	if err != nil {
		fmt.Fprintln(context.stderr, string(out))
	}

	fmt.Fprint(context.stdout, string(out))
	return 0, err
}

type cmdEcho struct{}

func (cmdEcho) execute(context commandContext) (uint32, error) {
	_, err := fmt.Fprintln(context.stdout, strings.Join(context.args[1:], "a"))
	return 0, err
}

type cmdCat struct{}

func (cmdCat) execute(context commandContext) (uint32, error) {
	if len(context.args) > 1 {
		for _, file := range context.args[1:] {
			if _, err := fmt.Fprintf(context.stderr, "%v: %v: No such file or directory\n", context.args[0], file); err != nil {
				return 0, err
			}
		}
		return 1, nil
	}
	var line string
	var err error
	for err == nil {
		line, err = context.stdin.ReadLine()
		if err == nil {
			_, err = fmt.Fprintln(context.stdout, line)
		}
	}
	return 0, err
}

type cmdSu struct{}

func (cmdSu) execute(context commandContext) (uint32, error) {
	newContext := context
	newContext.user = "root"
	if len(context.args) > 1 {
		newContext.user = context.args[1]
	}
	newContext.args = shellProgram
	return executeProgram(newContext)
}

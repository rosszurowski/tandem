package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"

	"github.com/pkg/term/termios"
)

type ptyPipe struct {
	pty, tty *os.File
}

type multiOutput struct {
	maxNameLength int
	mutex         sync.Mutex
	pipes         map[*process]*ptyPipe
	printProcName bool
}

func (m *multiOutput) openPipe(proc *process) (pipe *ptyPipe) {
	var err error

	pipe = m.pipes[proc]

	pipe.pty, pipe.tty, err = termios.Pty()
	fatalOnErr(err)

	proc.Stdout = pipe.tty
	proc.Stderr = pipe.tty
	proc.Stdin = pipe.tty
	proc.SysProcAttr = &syscall.SysProcAttr{Setctty: true, Setsid: true}

	return
}

func (m *multiOutput) Connect(proc *process) {
	if len(proc.Name) > m.maxNameLength {
		m.maxNameLength = len(proc.Name)
	}

	if m.pipes == nil {
		m.pipes = make(map[*process]*ptyPipe)
	}

	m.pipes[proc] = &ptyPipe{}
}

func (m *multiOutput) PipeOutput(proc *process) {
	pipe := m.openPipe(proc)

	go func(proc *process, pipe *ptyPipe) {
		scanLines(pipe.pty, func(b []byte) bool {
			m.WriteLine(proc, b)
			return true
		})
	}(proc, pipe)
}

func (m *multiOutput) ClosePipe(proc *process) {
	if pipe := m.pipes[proc]; pipe != nil {
		pipe.pty.Close()
		pipe.tty.Close()
	}
}

func (m *multiOutput) WriteLine(proc *process, p []byte) {
	var buf bytes.Buffer

	if m.printProcName {
		color := fmt.Sprintf("\033[0;38;5;%vm", proc.Color)
		buf.WriteString(color)
		if m.printProcName {
			buf.WriteString(proc.Name)
			for i := len(proc.Name); i <= m.maxNameLength; i++ {
				buf.WriteByte(' ')
			}
		}
		buf.WriteString("\033[0m ")
	}

	// We trim the "/bin/sh: " prefix from the output of the command
	// since the fact that we're running things in the /bin/sh shell isn't
	// super relevant.
	buf.Write(bytes.TrimPrefix(p, []byte("/bin/sh: ")))
	buf.WriteByte('\n')

	m.mutex.Lock()
	defer m.mutex.Unlock()
	buf.WriteTo(os.Stdout)
}

func (m *multiOutput) WriteErr(proc *process, err error) {
	m.WriteLine(proc, []byte(red(err.Error())))
}

func scanLines(r io.Reader, callback func([]byte) bool) error {
	var (
		err      error
		line     []byte
		isPrefix bool
	)

	reader := bufio.NewReader(r)
	buf := new(bytes.Buffer)

	for {
		line, isPrefix, err = reader.ReadLine()
		if err != nil {
			break
		}

		buf.Write(line)

		if !isPrefix {
			if !callback(buf.Bytes()) {
				return nil
			}
			buf.Reset()
		}
	}
	if err != io.EOF && err != io.ErrClosedPipe {
		return err
	}
	return nil
}

func fatalOnErr(err error) {
	if err != nil {
		fatal(err)
	}
}

func fatal(i ...interface{}) {
	fmt.Fprint(os.Stderr, name+": ")
	fmt.Fprintln(os.Stderr, i...)
	os.Exit(1)
}

func red(s string) string {
	return "\033[0;31m" + s + "\033[0m"
}

func gray(s string) string {
	return "\033[0;38;5;8m" + s + "\033[0m"
}

func dim(s string) string {
	return "\033[0;2m" + s + "\033[0m"
}

func bold(s string) string {
	return "\033[1m" + s + "\033[0m"
}

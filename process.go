package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

var colors = []int{2, 3, 4, 5, 6, 42, 130, 103, 129, 108}

type command struct {
	name string
	cmd  string
}

type processManager struct {
	output      *multiOutput
	procs       []*process
	procWg      sync.WaitGroup
	done        chan bool
	interrupted chan os.Signal
	timeout     time.Duration
	silent      bool
}

func newProcessManager(root string, timeout int, cmds []string, silent bool) (*processManager, error) {
	pm := &processManager{
		output:  &multiOutput{printProcName: true},
		procs:   make([]*process, 0),
		timeout: time.Duration(timeout) * time.Second,
		silent:  silent,
	}

	env := os.Environ()
	nodeBin := filepath.Join(root, "node_modules/.bin")
	if fi, err := os.Stat(nodeBin); err == nil && fi.IsDir() {
		injectPathVal(env, nodeBin)
	}

	var namedCmds []command
	var npmCommandIdx []int // indexes of commands with 'npm:' prefix
	for i, cmd := range cmds {
		name := filterCmdName(cmd)
		if name == "" {
			name = "cmd"
		}
		if strings.HasPrefix(name, "npm:") {
			npmCommandIdx = append(npmCommandIdx, i)
		}
		namedCmds = append(namedCmds, command{
			name: name,
			cmd:  cmd,
		})
	}

	// For commands prefixed with 'npm:', read the command contents from
	// the package.json file. Error on any missing commands.
	if len(npmCommandIdx) > 0 {
		scripts, err := parseNpmScripts(root)
		if err != nil {
			return nil, err
		}
		var missingCommands []string
		for _, idx := range npmCommandIdx {
			npmCmd := namedCmds[idx]
			npmCmd.name = strings.TrimPrefix(npmCmd.name, "npm:")
			s, ok := scripts[npmCmd.name]
			if !ok {
				missingCommands = append(missingCommands, npmCmd.name)
				continue
			}
			npmCmd.cmd = s
			namedCmds[idx] = npmCmd
		}
		if len(missingCommands) > 0 {
			return nil, fmt.Errorf("npm scripts %q missing from package.json", strings.Join(missingCommands, ","))
		}
	}

	// If there are multiple processes with the same name, append a number to each
	// one, so we can distinguish them.
	namesMap := map[string][]int{} // name -> indexes of procs with name
	for i, cmd := range namedCmds {
		name := cmd.name
		namesMap[name] = append(namesMap[name], i)
	}
	for name, idxs := range namesMap {
		if len(idxs) > 1 {
			for i, idx := range idxs {
				cmd := namedCmds[idx]
				cmd.name = fmt.Sprintf("%s.%d", name, i+1)
				namedCmds[idx] = cmd
			}
		}
	}

	for i, cmd := range namedCmds {
		pm.procs = append(pm.procs, newProcess(&processConfig{
			Name:   cmd.name,
			Cmd:    cmd.cmd,
			Color:  colors[i%len(colors)],
			Dir:    root,
			Env:    env,
			Output: pm.output,
			Silent: pm.silent,
		}))
	}
	return pm, nil
}

func (pm *processManager) Run() {
	pm.done = make(chan bool, len(pm.procs))
	pm.interrupted = make(chan os.Signal)
	signal.Notify(pm.interrupted, syscall.SIGINT, syscall.SIGTERM)
	for _, proc := range pm.procs {
		pm.runProcess(proc)
	}
	go pm.waitForExit()
	pm.procWg.Wait()
}

func (pm *processManager) runProcess(proc *process) {
	pm.procWg.Add(1)
	go func() {
		defer pm.procWg.Done()
		defer func() { pm.done <- true }()
		proc.Run()
	}()
}

func (pm *processManager) waitForDoneOrInterrupt() {
	select {
	case <-pm.done:
	case <-pm.interrupted:
	}
}

func (pm *processManager) waitForTimeoutOrInterrupt() {
	select {
	case <-time.After(pm.timeout):
	case <-pm.interrupted:
	}
}

func (pm *processManager) waitForExit() {
	pm.waitForDoneOrInterrupt()
	for _, proc := range pm.procs {
		go proc.Interrupt()
	}
	pm.waitForTimeoutOrInterrupt()
	for _, proc := range pm.procs {
		go proc.Kill()
	}
}

type process struct {
	*exec.Cmd
	Name   string
	Color  int
	output *multiOutput
	silent bool
}

type processConfig struct {
	Name   string
	Cmd    string
	Dir    string
	Env    []string
	Color  int
	Output *multiOutput
	Silent bool
}

func newProcess(cfg *processConfig) *process {
	p := &process{
		Cmd:    exec.Command("/bin/sh", "-c", cfg.Cmd),
		Name:   cfg.Name,
		Color:  cfg.Color,
		output: cfg.Output,
		silent: cfg.Silent,
	}
	p.Cmd.Dir = cfg.Dir
	p.Cmd.Env = cfg.Env
	p.output.Connect(p)
	return p
}

func (p *process) Running() bool {
	return p.Process != nil && p.ProcessState == nil
}

func (p *process) signal(sig os.Signal) {
	group, err := os.FindProcess(-p.Process.Pid)
	if err != nil {
		p.writeErr(err)
		return
	}
	if err = group.Signal(sig); err != nil {
		p.writeErr(err)
	}
}

func (p *process) writeLine(b []byte) {
	p.output.WriteLine(p, b)
}

func (p *process) writeErr(err error) {
	p.output.WriteErr(p, err)
}

func (p *process) Run() {
	p.output.PipeOutput(p)
	defer p.output.ClosePipe(p)
	if !p.silent {
		p.writeLine([]byte(dim("Starting...")))
	}
	if err := p.Cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == 1 {
				p.writeErr(err)
			} else {
				p.writeLine([]byte(dim(fmt.Sprintf("exit status %d", exitErr.ExitCode()))))
			}
			return
		}
		p.writeErr(err)
		return
	}
	if !p.silent {
		p.writeLine([]byte(dim("Process exited")))
	}
}

func (p *process) Interrupt() {
	if p.Running() {
		if !p.silent {
			p.writeLine([]byte(dim("Interrupting...")))
		}
		p.signal(syscall.SIGINT)
	}
}

func (p *process) Kill() {
	if p.Running() {
		if !p.silent {
			p.writeLine([]byte(dim("Killing...")))
		}
		p.signal(syscall.SIGKILL)
	}
}

type packageJSON struct {
	Scripts map[string]string `json:"scripts"`
}

func parseNpmScripts(root string) (map[string]string, error) {
	path := filepath.Join(root, "package.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading package.json: %v", err)
	}
	var p packageJSON
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("parsing package.json: %v", err)
	}
	return p.Scripts, nil
}

// injectPathVal injects a value into the start of a PATH environment variable.
// It expects a string slice of env variables in "KEY=VALUE" format, like those
// provided from os.Environ().
func injectPathVal(env []string, val string) []string {
	for i, v := range env {
		if strings.HasPrefix(v, "PATH=") {
			env[i] = fmt.Sprintf(
				"PATH=%s:%s",
				val,
				strings.TrimPrefix(v, "PATH="))
		}
	}
	return env
}

// filterCmdName returns the name of the command to be run, filtering out any
// path information.
func filterCmdName(cmd string) string {
	name, _, _ := strings.Cut(strings.TrimSpace(cmd), " ")
	name = filepath.Base(name)
	if name == "." || name == string(filepath.Separator) {
		return ""
	}
	return name
}

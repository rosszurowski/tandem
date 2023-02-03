// Package tandem provides a process manager utility for running multiple
// processes and combining their output.
package tandem

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

	"github.com/rosszurowski/tandem/ansi"
)

var colors = []int{2, 3, 4, 5, 6, 42, 130, 103, 129, 108}

// ProcessManager manages a set of processes, combining their output and exiting
// all of them gracefully when one of them exits.
type ProcessManager struct {
	output      *multiOutput
	procs       []*process
	procWg      sync.WaitGroup
	done        chan bool
	interrupted chan os.Signal
	timeout     time.Duration
	silent      bool
}

// Config is the configuration for a process manager.
type Config struct {
	Cmds    []string // Shell commands to run
	Root    string   // Root directory for commands to run from
	Timeout int      // Timeout in seconds for commands to exit gracefully before being killed. Defaults to 0.
	Silent  bool     // Whether to silence process management messages like "Starting..."
}

// New creates a new process manager with the given configuration.
func New(cfg Config) (*ProcessManager, error) {
	root, err := filepath.Abs(cfg.Root)
	if err != nil {
		return nil, fmt.Errorf("could not get absolute path for directory: %v", err)
	}

	pm := &ProcessManager{
		output:  &multiOutput{printProcName: true},
		procs:   make([]*process, 0),
		timeout: time.Duration(cfg.Timeout) * time.Second,
		silent:  cfg.Silent,
	}

	env := os.Environ()
	nodeBin := filepath.Join(cfg.Root, "node_modules/.bin")
	if fi, err := os.Stat(nodeBin); err == nil && fi.IsDir() {
		injectPathVal(env, nodeBin)
	}

	namedCmds, err := parseCommands(root, cfg.Cmds)
	if err != nil {
		return nil, err
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

// Run starts all processes and waits for them to exit or be interrupted.
func (pm *ProcessManager) Run() {
	pm.done = make(chan bool, len(pm.procs))
	pm.interrupted = make(chan os.Signal)
	signal.Notify(pm.interrupted, syscall.SIGINT, syscall.SIGTERM)
	for _, proc := range pm.procs {
		pm.runProcess(proc)
	}
	go pm.waitForExit()
	pm.procWg.Wait()
}

func (pm *ProcessManager) runProcess(proc *process) {
	pm.procWg.Add(1)
	go func() {
		defer pm.procWg.Done()
		defer func() { pm.done <- true }()
		proc.Run()
	}()
}

func (pm *ProcessManager) waitForDoneOrInterrupt() {
	select {
	case <-pm.done:
	case <-pm.interrupted:
	}
}

func (pm *ProcessManager) waitForTimeoutOrInterrupt() {
	select {
	case <-time.After(pm.timeout):
	case <-pm.interrupted:
	}
}

func (pm *ProcessManager) waitForExit() {
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

func (p *process) writeDebug(s string) {
	p.writeLine([]byte(ansi.Dim(s)))
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
		p.writeDebug("Starting...")
	}
	if err := p.Cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == 1 {
				p.writeErr(err)
			} else {
				p.writeLine([]byte(ansi.Dim(fmt.Sprintf("exit status %d", exitErr.ExitCode()))))
			}
			return
		}
		p.writeErr(err)
		return
	}
	if !p.silent {
		p.writeDebug("Process exited")
	}
}

func (p *process) Interrupt() {
	if p.Running() {
		if !p.silent {
			p.writeDebug("Interrupting...")
		}
		p.signal(syscall.SIGINT)
	}
}

func (p *process) Kill() {
	if p.Running() {
		if !p.silent {
			p.writeDebug("Killing...")
		}
		p.signal(syscall.SIGKILL)
	}
}

type command struct {
	name string
	cmd  string
}

func parseCommands(root string, cmds []string) ([]command, error) {
	var result []command
	var npmCommands []string
	for _, cmd := range cmds {
		name := filterCmdName(cmd)
		if name == "" {
			name = "cmd"
		}
		if strings.HasPrefix(name, "npm:") {
			npmCommands = append(npmCommands, cmd)
			continue
		}
		result = append(result, command{
			name: name,
			cmd:  cmd,
		})
	}

	// For commands prefixed with 'npm:', read the command contents from
	// the package.json file. Error on any missing commands.
	if len(npmCommands) > 0 {
		b, err := os.ReadFile(filepath.Join(root, "package.json"))
		if err != nil {
			return nil, fmt.Errorf("reading package.json: %v", err)
		}
		scripts, err := parseNpmScripts(b, npmCommands)
		if err != nil {
			return nil, err
		}
		result = append(result, scripts...)
	}

	// If there are multiple processes with the same name, append a number to each
	// one, so we can distinguish them.
	namesMap := map[string][]int{} // name -> indexes of procs with name
	for i, cmd := range result {
		name := cmd.name
		namesMap[name] = append(namesMap[name], i)
	}
	for name, idxs := range namesMap {
		if len(idxs) > 1 {
			for i, idx := range idxs {
				cmd := result[idx]
				cmd.name = fmt.Sprintf("%s.%d", name, i+1)
				result[idx] = cmd
			}
		}
	}
	return result, nil
}

type packageJSON struct {
	Scripts map[string]string `json:"scripts"`
}

// parseNpmScripts parses a package.json file and set of command strings, and
// returns a set of named commands, including the paths to run for each command.
func parseNpmScripts(b []byte, cmds []string) ([]command, error) {
	var pkg packageJSON
	if err := json.Unmarshal(b, &pkg); err != nil {
		return nil, fmt.Errorf("parsing package.json: %v", err)
	}

	var result []command
	var missingCommands []string
	for _, cmd := range cmds {
		scriptName := strings.TrimPrefix(cmd, "npm:")
		if s, ok := pkg.Scripts[scriptName]; ok {
			// Exact match? Add it to the list.
			result = append(result, command{
				name: scriptName,
				cmd:  s,
			})
			continue
		}
		if !strings.Contains(scriptName, "*") {
			missingCommands = append(missingCommands, scriptName)
			continue
		}
		// Check if scriptName wildcard matches scriptName
		hasMatch := false
		for name, pcmd := range pkg.Scripts {
			if !wildcardMatch(scriptName, name) {
				continue
			}
			result = append(result, command{
				name: name,
				cmd:  pcmd,
			})
			hasMatch = true
		}
		if !hasMatch {
			return nil, fmt.Errorf("no npm scripts matching %q found in package.json", scriptName)
		}
	}
	if len(missingCommands) > 0 {
		noun := "script"
		if len(missingCommands) != 1 {
			noun = "scripts"
		}
		return nil, fmt.Errorf("no npm %s named %q found in package.json", noun, strings.Join(missingCommands, ","))
	}
	return result, nil
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

// wildcardMatch takes a pattern that optionally includes a * character, and
// returns whether or not string s matches that wildcard. The matching currently
// only supports one wildcard and prefix/suffix matching.
func wildcardMatch(pattern, s string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return strings.EqualFold(pattern, s)
	}
	if len(parts) > 2 {
		return false
	}
	if parts[0] == "" {
		return strings.HasSuffix(s, parts[1])
	}
	if parts[1] == "" {
		return strings.HasPrefix(s, parts[0])
	}
	return strings.HasPrefix(s, parts[0]) && strings.HasSuffix(s, parts[1])
}

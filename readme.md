<div align="center">
  <img src=".github/cover.svg" width="400" alt="tandem — a fast and small parallel task runner (logo by Hannah Lee)" />
</div>

<br />

<div align="center">
  <b>tandem</b> is a task runner for parallel dev servers.<br />Small, fast, and pairs great with Makefiles!<br>
</div>

## Features

- Small, fast, static binary.
- Shuts down each command if one fails. No more processes clinging to ports.
- Supports running npm scripts and binaries.
- Labels output for each command.

## Demo

![A screencast demo of using the tandem CLI.](.github/demo.gif)

## Installation

On macOS, you can install with:

```shell
brew install rosszurowski/tap/tandem
```

If you have Go installed, you can install from the source with:

```shell
go install github.com/rosszurowski/tandem@latest
```

If you're using tandem from a Makefile, [this snippet](#using-in-makefiles) shows how to download a locally cached copy.

## Usage

Use tandem by passing a set of commands to run in parallel. Wrap each command in quotes, like so:

```shell
tandem 'command1 "arg"' 'command2 "arg"' 'command3 "arg"'
```

### Running a front-end and a backend at once

Working on a Next.js app, you might want to run your front-end dev server alongside a backend API server with live updating changes through [nodemon](https://nodemon.io/):

```shell
$ tandem 'next dev' 'nodemon --quiet ./server.js'
next     ready - started server on 0.0.0.0:3000, url: http://localhost:3000
next     event - compiled client and server successfully in 15 ms (25 modules)
nodemon  starting server...
nodemon  listening on http://localhost:3001
```

### Running npm scripts

If your scripts are defined in `package.json`, you can reference them by using `npm:` as a prefix:

```json
{
  "scripts": {
    "dev:php": "...",
    "dev:js": "...",
    "dev:css": "..."
  }
}
```

```shell
$ tandem 'npm:dev:php' 'npm:dev:js' 'npm:dev:css'
```

Support for npm wildcards like `tandem 'npm:dev:*'` is tracked in https://github.com/rosszurowski/tandem/issues/2

### Using in Makefiles

In a Makefile, use this snippet to fetch a local copy for your project. Change the `.cache` path as needed, and add it to your `.gitignore`.

```makefile
dev: node_modules .cache/tandem
	@.cache/tandem 'command1' 'command2'
.PHONY: dev

.cache/tandem:
	@mkdir -p $$(dirname $@)
	@curl -fsSL https://raw.githubusercontent.com/rosszurowski/tandem/main/install.sh | bash -s -- --dest="$$(dirname $@)"
```

Running `make dev` will download tandem once, and then use it for every run from there on out.

## Motivation

I regularly use Makefiles to automate project commands and tools. Makefiles are mostly great! But their biggest failing (also a failing of shells generally) is that it's shockingly hard to coordinate multiple commands as one group:

- `make -jN <a> <b> <c>` doesn't end all tasks when another one fails. For running local dev servers, this means you can lose your CSS or JS watcher and not realize.
- `command1 & command2 & wait` often leaves commands hanging around in the background, which is annoying when it eats up a port you want to use.
- Tools like GNU parallel have a confusing syntax, and I've never been able to figure out how to stop all tasks when one fails.

tandem makes running concurrent servers easy. It takes inspiration from [concurrently](https://www.npmjs.com/package/concurrently) or [npm-run-all](https://www.npmjs.com/package/npm-run-all), but improves performance and works as a static binary.

## Acknowledgements

tandem owes a big thanks to [hivemind](https://github.com/DarthSim/hivemind), from which much of the source is drawn. `tandem` can be thought of as a fork of hivemind, but rather than defining commands in a Procfile, defining them from a list of arguments.

tandem's illustration was drawn by [Hannah Lee](https://hannahlee.ca/).

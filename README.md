
<!-- Title -->
<div align="center">
  <a href="https://github.com/jeffreytse/zsh-vi-mode">
    <img alt="vi-mode →~ zsh" src="https://user-images.githubusercontent.com/9413601/103399068-46bfcb80-4b7a-11eb-8741-86cff3d85a69.png" width="600">
  </a>
  <p>  Modern shell readline library  </p>

  <br> <h1>  Readline </h1>

</div>

<!-- Badges -->
<p align="center">
![Github Actions (workflows)](https://github.com/reeflective/readline/workflows/:workflow-name/badge.svg?branch=:branch-name)
[![Go module version](https://img.shields.io/github/go-mod/go-version/reeflective/readline.svg)](https://github.com/reeflective/readline)
[![GoDoc reference](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/reeflective/go/readline)
[![Go Report Card](https://goreportcard.com/badge/github.com/reeflective/readline)](https://goreportcard.com/report/github.com/reeflective/readline)
[![codecov](https://codecov.io/gh/reeflective/readline/branch/master/graph/badge.svg)](https://codecov.io/gh/reeflective/readline)
[![License: BSD-3](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)
</p>

This library offers a modern, pure Go readline implementation, enhanced with editing and user 
interface features commonly found in modern shells, all in little more than 10K lines of code.
It is used, between others, to power the [console](https://github.com/reeflective/console) library.

## Features

### Editing
- Near-native Emacs and Vim modes.
- Configurable bind keymaps, with live reload and sane defaults.
- Extended list of line edition/movement/control widgets (emacs and Vim).
- Vim Insert and Replace (once/many).
- Many Vim text objects.
- Support for Vim Visual/Operator pending mode & cursor styles indications.
- All Vim registers, with completion support.
- Extended surround select/change/add fonctionality, with highlighting.
- Keywords switching (operators, booleans, hex/binary/digit) with iterations.
- Undo/redo history.
- Command-line edition in $EDITOR.
- Support for an arbitrary number of history sources.

### Interface
- Support for most of `oh-my-posh` prompts (PS1/PS2/RPROMPT/transient/tooltip).
- Extended completion system, keymap-based and configurable, easy to populate & use.
- Multiple completion display styles, with color support.
- Completion & History incremental search system & highlighting (fuzzy-search).
- Automatic & context-aware suffix removal for efficient flags/path/list completion.
- Optional asynchronous autocomplete.
- Usage/hint message display.
- Support for syntax highlighting

## Showcases
- Emacs edition
- Vim edition
- Vim selection & movements
- Vim surround
- Keyword swithing
- Vim registers & completion
- Undo/redo line history
- History movements & completion
- Completion classic
- Completion isearch
- Suffix autoremoval
- Prompts
- Logging

## Credits

- While most of the code has been rewritten from scratch, the original library used is [lmorg/readline](https://github.com/lmorg/readline).
  I would have never ventured myself doing this if he had not ventured writing a Vim mode core in the first place. 


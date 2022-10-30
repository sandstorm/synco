# synco

## Usage
> an Database and File Dump Downloader for synchronizing production, staging, and local development

synco

## Description

```
This is a template CLI application, which can be used as a boilerplate for awesome CLI tools written in Go.
This template prints the date or time to the terminal.
```
## Examples

```bash
synco  date
cli-template date --format 20060102
cli-template time
cli-template time --live
```

## Flags
|Flag|Usage|
|----|-----|
|`--debug`|enable debug messages|
|`--disable-update-checks`|disables update checks|
|`--raw`|print unstyled raw output (set it if output is written to a file)|

## Commands
|Command|Usage|
|-------|-----|
|`synco completion`|Generate the autocompletion script for the specified shell|
|`synco help`|Help about any command|
|`synco receive`|Wizard to be executed in target|
|`synco serve`|Wizard to be executed in source|
# ... completion
`synco completion`

## Usage
> Generate the autocompletion script for the specified shell

synco completion

## Description

```
Generate the autocompletion script for synco for the specified shell.
See each sub-command's help for details on how to use the generated script.

```

## Commands
|Command|Usage|
|-------|-----|
|`synco completion bash`|Generate the autocompletion script for bash|
|`synco completion fish`|Generate the autocompletion script for fish|
|`synco completion powershell`|Generate the autocompletion script for powershell|
|`synco completion zsh`|Generate the autocompletion script for zsh|
# ... completion bash
`synco completion bash`

## Usage
> Generate the autocompletion script for bash

synco completion bash

## Description

```
Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(synco completion bash)

To load completions for every new session, execute once:

#### Linux:

	synco completion bash > /etc/bash_completion.d/synco

#### macOS:

	synco completion bash > /usr/local/etc/bash_completion.d/synco

You will need to start a new shell for this setup to take effect.

```

## Flags
|Flag|Usage|
|----|-----|
|`--no-descriptions`|disable completion descriptions|
# ... completion fish
`synco completion fish`

## Usage
> Generate the autocompletion script for fish

synco completion fish

## Description

```
Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	synco completion fish | source

To load completions for every new session, execute once:

	synco completion fish > ~/.config/fish/completions/synco.fish

You will need to start a new shell for this setup to take effect.

```

## Flags
|Flag|Usage|
|----|-----|
|`--no-descriptions`|disable completion descriptions|
# ... completion powershell
`synco completion powershell`

## Usage
> Generate the autocompletion script for powershell

synco completion powershell

## Description

```
Generate the autocompletion script for powershell.

To load completions in your current shell session:

	synco completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.

```

## Flags
|Flag|Usage|
|----|-----|
|`--no-descriptions`|disable completion descriptions|
# ... completion zsh
`synco completion zsh`

## Usage
> Generate the autocompletion script for zsh

synco completion zsh

## Description

```
Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions for every new session, execute once:

#### Linux:

	synco completion zsh > "${fpath[1]}/_synco"

#### macOS:

	synco completion zsh > /usr/local/share/zsh/site-functions/_synco

You will need to start a new shell for this setup to take effect.

```

## Flags
|Flag|Usage|
|----|-----|
|`--no-descriptions`|disable completion descriptions|
# ... help
`synco help`

## Usage
> Help about any command

synco help [command]

## Description

```
Help provides help for any command in the application.
Simply type synco help [path to command] for full details.
```
# ... receive
`synco receive`

## Usage
> Wizard to be executed in target

synco receive

## Description

```
...
```
## Examples

```bash
synco receive [url] [password]
```

## Flags
|Flag|Usage|
|----|-----|
|`--interactive`|identifier for the decryption (default true)|
# ... serve
`synco serve`

## Usage
> Wizard to be executed in source

synco serve

## Description

```
...
```
## Examples

```bash
synco serve 
```

## Flags
|Flag|Usage|
|----|-----|
|`--id string`|identifier for the decryption|
|`--listen string`|port to create a HTTP server on, if any|
|`--password string`|password to encrypt the files for|


---
> **Documentation automatically generated with [PTerm](https://github.com/pterm/cli-template) on 30 October 2022**

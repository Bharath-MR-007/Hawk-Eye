<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

## hawkeye completion zsh

Generate the autocompletion script for zsh

### Synopsis

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(hawkeye completion zsh)

To load completions for every new session, execute once:

#### Linux:

	hawkeye completion zsh > "${fpath[1]}/_hawkeye"

#### macOS:

	hawkeye completion zsh > $(brew --prefix)/share/zsh/site-functions/_hawkeye

You will need to start a new shell for this setup to take effect.


```
hawkeye completion zsh [flags]
```

### Options

```
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.hawkeye.yaml)
```

### SEE ALSO

* [hawkeye completion](hawkeye_completion.md)	 - Generate the autocompletion script for the specified shell


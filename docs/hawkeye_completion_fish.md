<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

## hawkeye completion fish

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	hawkeye completion fish | source

To load completions for every new session, execute once:

	hawkeye completion fish > ~/.config/fish/completions/hawkeye.fish

You will need to start a new shell for this setup to take effect.


```
hawkeye completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.hawkeye.yaml)
```

### SEE ALSO

* [hawkeye completion](hawkeye_completion.md)	 - Generate the autocompletion script for the specified shell


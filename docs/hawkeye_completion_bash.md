<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

## hawkeye completion bash

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(hawkeye completion bash)

To load completions for every new session, execute once:

#### Linux:

	hawkeye completion bash > /etc/bash_completion.d/hawkeye

#### macOS:

	hawkeye completion bash > $(brew --prefix)/etc/bash_completion.d/hawkeye

You will need to start a new shell for this setup to take effect.


```
hawkeye completion bash
```

### Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.hawkeye.yaml)
```

### SEE ALSO

* [hawkeye completion](hawkeye_completion.md)	 - Generate the autocompletion script for the specified shell


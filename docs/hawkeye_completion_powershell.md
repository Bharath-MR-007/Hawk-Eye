<!--
SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH

SPDX-License-Identifier: CC-BY-4.0
-->

## hawkeye completion powershell

Generate the autocompletion script for powershell

### Synopsis

Generate the autocompletion script for powershell.

To load completions in your current shell session:

	hawkeye completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.


```
hawkeye completion powershell [flags]
```

### Options

```
  -h, --help              help for powershell
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.hawkeye.yaml)
```

### SEE ALSO

* [hawkeye completion](hawkeye_completion.md)	 - Generate the autocompletion script for the specified shell


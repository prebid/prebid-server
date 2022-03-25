
### vscode in-container development

The quickest way to get up and running with PBS-Go development in a reproducible environment isolated
from your host OS is by loading the repository in a [Docker](https://docs.docker.com/get-docker/)
container under [Visual Studio Code](https://code.visualstudio.com/).

This covers installing Go and necessary IDE plugins for Golang editing and debugging, and is
all automated via the VSCode [.devcontainer](.devcontainer/) configuration. See
[VSCode Remote Containers](https://code.visualstudio.com/docs/remote/containers) for more
details about how to customize.

#### Setup

Install:

- [Docker](https://docs.docker.com/get-docker/)
- [Visual Studio Code](https://code.visualstudio.com/)
- [VSCode Remote Development Extension Pack](https://aka.ms/vscode-remote/download/extension)

Then:

- start VSCode and open repository
- accept VSCode suggestion to _reopen in container_
- VSCode will build a new container and install the IDE support and extensions in it

Optionally, to use your github ssh key for accessing non-public GitHub repositories:
- Method 1: add your github ssh key to agent. This is needed after each OS restart.
```sh
ssh-add ~/.ssh/id_rsa # or your ssh key for github
```
- Method 2: map your ~/.ssh (or just the key) as a docker volume in .devcontainer.json

Feel free to customize .devcontainer.json if needed. You can add preset environment variables,
other vscode extensions to preload and additional volume mounts.

#### Starting PBS-Go

- `Shift`-`Cmd`-`D` or `Run` icon brings up the `Launch prebid-server` panel with interactive
debugger. Breakpoints can be set to stop execution.
- CTRL-`\`` opens the terminal to start prebid-server non-interactively
```sh
go run main.go --alsologtostderr
```
- Create a pbs.yaml file if neccessary, with configuration overrides.

#### Testing

- Open any `*_test.go` file in editor
- Individual test functions can be run directly by clicking the `run test`/`debug test` annotation
above each test function.
- At the top of the file you can see `run package tests | run file tests`
- TIP: use `run package tests` at the top of the test file to quickly check code coverage:
  the open editors for files in the tested package will have lines highlighted in green (covered)
  and red (not covered)
- CTRL-`\`` opens the terminal to run the test suite
```sh
./validate.sh
```

#### Editing
- Style, imports are automatically updated on save.
- Editor can suggest correct names and

- Remote container commands popup by clicking on _Dev Container: Go_ at bottom left
- `F1` -> type _rebuild container_ to restart with a fresh container
- `F1` -> `^`-`\`` to toggle terminal

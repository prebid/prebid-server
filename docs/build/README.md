## Overview

As of v2.31.0, Prebid Server contains a module that requires CGo which introduces both build and runtime dependencies. To build, you need a C compiler, preferably gcc. To run, you may require one or more runtime dependencies, most notably libatomic.

## Examples
For a containerized example, see the Dockerfile.
For manual build examples, including some cross-compilation use cases, see below.

### From darwin amd64

#### To darwin amd64
`GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build`

Running the built binary on mac amd64:
`./prebid-server --stderrthreshold=WARNING -v=2`

#### To darwin arm64
`GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build`

Running the built binary on mac arm64:
`./prebid-server --stderrthreshold=WARNING -v=2`

#### To windows amd64
<b>Build</b>
Install mingw-w64 which consists of a gcc compiler port you can use to generate windows binaries:
`brew install mingw-w64`

`GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC="x86_64-w64-mingw32-gcc" go build`

<b>Run</b>
Running the built binary on windows:
`.\prebid-server.exe --sderrthreshold=WARNING =v=2`

You may receive the following errors or something similar:
```
"The code execution cannot proceed because libatomic-1.dll was not found."
"The code execution cannot proceed because libwinpthread-1.dll was not found."
```

To resolve these errors, copy the following files from mingw-64 on your mac to `C:/windows/System32` and re-run:
`/usr/local/Cellar/mingw-w64/12.0.0_1/toolchain-x86_64/x86_64-w64-mingw32/lib/libatomic-1.dll`
`/usr/local/Cellar/mingw-w64/12.0.0_1/toolchain-x86_64/x86_64-w64-mingw32/bin/libwinpthread-1.dll`

### From windows amd64
#### To windows amd64
<b>Build</b>
`set CGO_ENABLED=1`
`set GOOS=windows`
`set GOARCH=amd64`
`go build . && .\prebid-server.exe --stderrthreshold=WARNING -v=2`

You may receive the following error or something similar:
```
# runtime/cgo
cgo: C compiler "gcc" not found: exec: "gcc": executable file not found in %PATH%
```

To resolve the error, install MSYS2:
1) Download the installer (https://www.msys2.org/)
2) Run the installer and follow the steps of the installation wizard
3) Run MSYS2 which will open an MSYS2 terminal for you
4) In the MSYS2 terminal, install windows/amd64 gcc toolchain: `pacman -S --needed base-devel mingw-w64-x86_64-gcc`
5) Enter `Y` when prompted whether to proceed with the installation
6) Add the path of your MinGW-w64 `bin` folder to the Windows `PATH` environment variable by using the following steps:
    - In the Windows search bar, type <b>Settings</b> to open your Windows Settings.
    - Search for <b>Edit environment variables for your account</b>.
    - In your <b>User variables</b>, select the `Path` variable and then select <b>Edit</b>.
    - Select </b>New and add the MinGW-w64 destination folder you recorded during the installation process to the list. If you used the default settings above, then this will be the path: `C:\msys64\ucrt64\bin`.
    - Select <b>OK</b>, and then select <b>OK</b> again in the <b>Environment Variables</b> window to update the `PATH` environment variable. You have to reopen any console windows for the updated `PATH` environment variable to be available.
7) Confirm gcc installed: `gcc --version`

<b>Run</b>
Running the built binary on windows:
`go build . && .\prebid-server.exe --stderrthreshold=WARNING -v=2`

You may receive the following errors or something similar:
```
"The code execution cannot proceed because libatomic-1.dll was not found."
"The code execution cannot proceed because libwinpthread-1.dll was not found."
```
To resolve these errors, copy the following files from MSYS2 installation to `C:/windows/System32` and re-run:
`C:\mysys64\mingw64\bin\libatomic-1.dll`
`C:\mysys64\mingw64\bin\libwinpthread-1.dll`

### From linux amd64
#### To linux amd64
<b>Note</b>
These instructions are for building and running on Debian-based distributions

<b>Build</b>
`GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build`

You may receive the following error or something similar:
```
# runtime/cgo
cgo: C compiler "gcc" not found: exec: "gcc": executable file not found in $PATH
```
To resolve the error, install gcc and re-build:
`sudo apt-get install -y gcc`

<b>Run</b>
Running the built binary on Linux:
`./prebid-server --stderrthreshold=WARNING -v=2`

You may receive the following error or something similar:
```
... error while loading shared libraries: libatomic.so.1: cannot open shared object file ...
```
To resolve the error, install libatomic1 and re-run:
`sudo apt-get install -y libatomic1`
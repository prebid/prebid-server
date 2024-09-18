## Overview

Prebid Server contains at least one module that requires CGo which introduces both build and runtime dependencies.
To build, you need a C compiler, preferably gcc.
To run, you may require one or more runtime dependencies, most notably libatomic.

## Examples (Build --> Target)
Here are some manual build examples, including some cross-compilation use cases, that have been tested:

### darwin amd64 --> darwin amd64
`GOOS=darwin GOARCH=amd64 go build`

Running the built binary on mac amd64:
`./prebid-server --stderrthreshold=WARNING -v=2`

### darwin amd64 --> darwin arm64
`GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build`

Running the built binary on mac arm64:
`./prebid-server --stderrthreshold=WARNING -v=2`

### darwin amd64 --> windows amd64
<b>Build (mac):</b>
Install mingw-w64 which consists of a gcc compiler port you can use to generate windows binaries:
`brew install mingw-w64`

From the root of the project:
`GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC="x86_64-w64-mingw32-gcc" go build`

<b>Run (windows)</b>
`.\prebid-server.exe --sderrthreshold=WARNING =v=2`

You may see the following errors:
```
"The code execution cannot proceed because libatomic-1.dll was not found."
"The code execution cannot proceed because libwinpthread-1.dll was not found."
```

To resolve these errors:
1) Copy the following files from mingw-64 on your mac to `C:/windows/System32`:
`/usr/local/Cellar/mingw-w64/12.0.0_1/toolchain-x86_64/x86_64-w64-mingw32/lib/libatomic-1.dll`
`/usr/local/Cellar/mingw-w64/12.0.0_1/toolchain-x86_64/x86_64-w64-mingw32/bin/libwinpthread-1.dll`
2) Register the DLLs on your windows machine using the regsvr32 command:
`regsvr32 "C:\Windows\System32\libatomic-1.dll"`
`regsvr32 "C:\Windows\System32\libwinpthread-1.dll"`

`.\prebid-server.exe --sderrthreshold=WARNING =v=2`

### windows amd64 --> windows amd64
<b>Build</b>
`set CGO_ENABLED=1`
`set GOOS=windows`
`set GOARCH=amd64`
`go build . && .\prebid-server.exe --stderrthreshold=WARNING -v=2`

If during the build you get an error similar to:
```
# runtime/cgo
cgo: C compiler "gcc" not found: exec: "gcc": executable file not found in %PATH%
```

install MSYS2:
1) download the installer (<b>TODO</b>: link)
2) run the installer
3) run MSYS2
4) install windows/amd64 gcc toolchain: `pacman -S --needed base-devel mingw-w64-x86_64-gcc`
5) enter `Y` when prompted whether to proceed with the installation
6) Add the path of your MinGW-w64 bin folder to the Windows PATH environment variable by using the following steps:
- In the Windows search bar, type Settings to open your Windows Settings.
- Search for Edit environment variables for your account.
- In your User variables, select the Path variable and then select Edit.
- Select New and add the MinGW-w64 destination folder you recorded during the installation process to the list. If you used the default settings above, then this will be the path: C:\msys64\ucrt64\bin.
- Select OK, and then select OK again in the Environment Variables window to update the PATH environment variable. You have to reopen any console windows for the updated PATH environment variable to be available.
7) confirm gcc installed: `gcc --version`

<b>Run</b>
`go build . && .\prebid-server.exe --stderrthreshold=WARNING -v=2`

### linux amd64 --> linux amd64
<b>Tests</b>
Debian or Ubuntu Linux targeting a Debian-based Linux distribution

<b>Build</b>
`GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build`

If during the build you get an error similar to:
```
# runtime/cgo
cgo: C compiler "gcc" not found: exec: "gcc": executable file not found in $PATH
```
install gcc, by running  `sudo apt-get install -y gcc`

<b>Run</b>
Running the built binary on Linux:
`./prebid-server --stderrthreshold=WARNING -v=2`
If you get an error:
```
... error while loading shared libraries: libatomic.so.1: cannot open shared object file ...
```
install libatomic1, by running `sudo apt-get install -y libatomic1`
# Sample

The Sample describes several demos of quickly spinning up different Prebid Server instances with various preset configurations. These samples are intended for audiences with little knowledge about Prebid Server and plan to play around with it locally and see how it works.

# Installation

In the Sample, we use `docker` and `docker-compose` to instantiate examples; with docker providing a unified setup and interface,  you can spin up a demo server instance locally with only one command without knowing all the complexities.
The docker image used in `docker-compose.yml` is the `Dockerfile` residing in the root level of the repository. 

## Option 1 - Standard Docker Engine
Install `docker` and `docker-compose` via the [official docker page](https://docs.docker.com/compose/install/#scenario-one-install-docker-desktop). If you cannot use the official docker engine due to restrictions of its license, see the option below about using Podman instead of Docker. 

## Option 2 - Podman
From MacOS, you can use [podman](https://podman.io/) with these additional steps:

```sh
$ brew install podman docker-compose
$ podman machine init
$ podman machine set --rootful
$ podman machine start
$ cd sample
$ docker-compose up <number>_<name>
```

# Examples

## Common File & Structures
All required files for each example are stored in a folder that follows the name pattern <number>_<name>. The `<number>` suggests its order and `<name`>` describes its title.

The following files will be present for every example and are exclusively catered to that example.
1. `app.yaml` - the prebid server app config.
2. `pbjs.html` - the HTML file with `Prebid JS` integration and communicates with the Prebid Server. It also provides a detailed explanation of the example.
3. `*.json` - additional files required to support the example. e.g. stored request and stored response.

## Common steps 

### Steps
1. Bring up an instance by running `docker-compose up <number>_<name>` in the `sample` folder.

2. Wait patiently until you see ` Admin server starting on: :6060` and `Main server starting on: :8000` in the command line output. This marks the Prebid Server instance finishing its initialization and is ready to serve the auction traffic.

3. you can copy the URL `http://localhost:8000/status` and paste it into your browser. You should see `ok` in the response which is another way to tell the Prebid Server that the main auction server is up and running.

4. Open a new tab in your browser and turn on the console UI. If you are using Chrome, you can right-click on the page and click `inspect`. Once the console UI is on, click on the `Network` tab to inspect the traffic later.

5. Copy the URL `http://localhost:8000/static/pbjs.html?pbjs_debug=true` into your browser. It starts the example immediately with debugging information from `Prebid JS`, and you can inspect the request and response between `Prebid JS` and `Prebid Server`.

6. After playing with the example, type `docker-compose down`. This is to shut down the existing Sample so you can start the next one you want to select.

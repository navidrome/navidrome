# CloudSonic Server

[![Build Status](https://github.com/cloudsonic/sonic-server/workflows/CI/badge.svg)](https://github.com/cloudsonic/sonic-server/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/cloudsonic/sonic-server)](https://goreportcard.com/report/github.com/cloudsonic/sonic-server)

CloudSonic is a music collection server and streamer, allowing you to listen to your music collection from anywhere. 
It relies on the huge selection of available mobile and web apps compatible with [Subsonic](http://www.subsonic.org), 
[Airsonic](https://airsonic.github.io/) and [Madsonic](https://www.madsonic.org/)

It is already functional (see [Installation](#installation) below), but still in its early stages.

Version 1.0 main goals are:
- Be fully compatible with available [Subsonic clients](http://www.subsonic.org/pages/apps.jsp)
  (actively being tested with
  [DSub](http://www.subsonic.org/pages/apps.jsp#dsub),
  [Music Stash](https://play.google.com/store/apps/details?id=com.ghenry22.mymusicstash) and
  [Jamstash](http://www.subsonic.org/pages/apps.jsp#jamstash))
- Implement smart/dynamic playlists (similar to iTunes)
- Optimized ro run on cheap hardware (Raspberry Pi) and VPS

### Supported Subsonic API version

I'm currently trying to implement all functionality from API v1.8.0, with some exceptions.

Check the (almost) up to date [compatibility chart](https://github.com/cloudsonic/sonic-server/wiki/Compatibility) 
for what is working.

### Installation

As this is a work in progress, there are no installers yet. To have the server running in your computer, follow 
the steps in the [Development Environment](#development-environment) section below, then run it with:

```
$ export SONIC_MUSICFOLDER="/path/to/your/music/folder"
$ make
```

The server should start listening for requests. The default configuration is:

- Port: `4533`
- User: `anyone`
- Password: `wordpass`

To override this or any other configuration, create a file named `sonic.toml` in the project folder.
 For all options see the [configuration.go](conf/configuration.go) file

### Development Environment

You will need to install [Go 1.13](https://golang.org/dl/)

Then install dependencies:

```
$ make setup
```

Some useful commands:

```bash
# Start local server (with hot reload)
$ make

# Run all tests
$ make test
```

### Copying

CloudSonic - Copyright (C) 2017-2020 Deluan Cotts Quintao

The source code is licensed under GNU Affero GPL v3. License is available [here](/LICENSE)

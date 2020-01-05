# CloudSonic Server

[![Build Status](https://travis-ci.org/cloudsonic/sonic-server.svg?branch=master)](https://travis-ci.org/cloudsonic/sonic-server)
[![Go Report Card](https://goreportcard.com/badge/github.com/cloudsonic/sonic-server)](https://goreportcard.com/report/github.com/cloudsonic/sonic-server)

**This is still a work in progress, and has no releases available**

CloudSonic is a music collection server and streamer, optmized to run on cheap VPS servers. It implements the
[Subsonic](http://www.subsonic.org) API

The project's main goals are:

- Be fully compatible with available [Subsonic clients](http://www.subsonic.org/pages/apps.jsp)
  (actively being tested with
  [DSub](http://www.subsonic.org/pages/apps.jsp#dsub),
  [SubFire](http://www.subsonic.org/pages/apps.jsp#subfire) and
  [Jamstash](http://www.subsonic.org/pages/apps.jsp#jamstash))
- Import and use all metadata from iTunes, so that you can optionally keep using iTunes to manage your music
- Implement Smart Playlists, as iTunes
- Help me learn Go ;) [![Gopher](https://blog.golang.org/favicon.ico)](https://golang.org)

### Supported Subsonic API version

I'm currently trying to implement all functionality from API v1.8.0, with some exceptions.

Check the (almost) up to date [compatibility chart](https://github.com/cloudsonic/sonic-server/wiki/Compatibility) for what is working.

### Installation

As this is a work in progress, there are no installers yet. To have the server running in your computer, follow the steps in the
Development Environment section below, then run it with:

```
$ export SONIC_MUSICFOLDER="/path/to/your/iTunes Library.xml"
$ make run
```

The server should start listening on port 4533.

### Development Environment

You will need to install [Go 1.13](https://golang.org/dl/)

Then install dependencies:

```
$ make setup
```

Some useful commands:

```bash
# Start local server (with hot reload)
$ make run

# Run all tests
$ make test
```

### Copying

CloudSonic - Copyright (C) 2017-2020 Deluan Cotts Quintao

The source code is licensed under GNU Affero GPL v3. License is available [here](/LICENSE)

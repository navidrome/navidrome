CloudSonic Server
=======

[![Build Status](https://travis-ci.org/cloudsonic/sonic-server.svg?branch=master)](https://travis-ci.org/cloudsonic/sonic-server) 
[![Go Report Card](https://goreportcard.com/badge/github.com/cloudsonic/sonic-server)](https://goreportcard.com/report/github.com/cloudsonic/sonic-server)

__This is still a work in progress, and has no releases available__

CloudSonic is an application that implements the [Subsonic API](http://www.subsonic.org/pages/api.jsp), but instead of
having its own music library like the original [Subsonic application](http://www.subsonic.org), it interacts directly
with your iTunes library.

The project's main goals are:

* Be fully compatible with available [Subsonic clients](http://www.subsonic.org/pages/apps.jsp)
  (actively being tested with
    [DSub](http://www.subsonic.org/pages/apps.jsp#dsub),
    [SubFire](http://www.subsonic.org/pages/apps.jsp#subfire) and
    [Jamstash](http://www.subsonic.org/pages/apps.jsp#jamstash))
* Use all metadata from iTunes, so that you can keep using iTunes to manage your music
* Keep iTunes stats (play counts, last played dates, ratings, etc..) updated, at least on Mac OS X.
  This allows smart playlists to be used in Subsonic Clients
* Help me learn Go ;) [![Gopher](https://blog.golang.org/favicon.ico)](https://golang.org)


###  Supported Subsonic API version

I'm currently trying to implement all functionality from API v1.8.0, with some exceptions.

Check the (almost) up to date [compatibility chart](https://github.com/cloudsonic/sonic-server/wiki/Compatibility) for what is working.

### Installation

As this is a work in progress, there are no installers yet. To have the server running in your computer, follow the steps in the
Development Environment section below, then run it with:
```
$ export SONIC_MUSICFOLDER="/path/to/your/iTunes Library.xml"
$ bee run
```
The server should start listening on port 4533.

### Development Environment

You will need to install [Go 1.7](https://golang.org/dl/)

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

CloudSonic - Copyright (C) 2017  Deluan Cotts Quintao

The source code is licensed under GNU Affero GPL v3. License is available [here](/LICENSE)

# Navidrome Music Streamer

[![Build Status](https://github.com/deluan/navidrome/workflows/Build/badge.svg)](https://github.com/deluan/navidrome/actions)

Navidrome is an open source web-based music collection server and streamer. It gives you freedom to listen to your 
music collection from any browser or mobile device.

This is _alpha quality_ software. Expect some changes in the feature set and the way it works. 

__Any feedback is welcome!__ If you need/want a new feature, find a bug or think of any way to improve Navidrome, 
please fill a [GitHub issue](https://github.com/deluan/navidrome/issues) 

## Features

- Handles very large music collections
- Streams virtually any audio format available
- Reads and uses all your beautifully curated metadata (id3 tags)
- Multi-user, each user has their own play counts, playlists, favourites, etc..
- Very low resource usage: Ex: with a library of 300GB (~29000 songs), it uses less than 50MB of RAM
- Multi-platform, runs on macOS, Linux and Windows. Docker images are also provided
- Automatically monitors your library for changes, importing new files and reloading new metadata 
- Modern and responsive Web interface based on Material UI, to manage users and browse your library
- Compatible with the huge selection of clients for [Subsonic](http://www.subsonic.org), 
   [Airsonic](https://airsonic.github.io/) and [Madsonic](https://www.madsonic.org/). 
   See the [complete list of available mobile and web apps](https://airsonic.github.io/docs/apps/)

## Road map

This project is being actively worked on. Expect a more polished experience and new features/releases 
on a frequent basis. Some upcoming features planned: 

- Transcoding/Downsampling on-the-fly
- Last.FM integration
- Integrated music player
- Pre-build binaries for all platforms, including Raspberry Pi
- Smart/dynamic playlists (similar to iTunes)
- Jukebox mode
- Sharing links to albums/songs/playlists
- Podcasts

## Installation

Currently there are no downloadable binaries (WIP). The available options are:

### Run it with Docker

```yaml
# This is just an example. Customize it to your needs.

version: "3"
services:
  navidrome:
    image: deluan/navidrome:latest
    ports:
      - "4533:4533"
    environment:
      # All options with their default values:
      ND_MUSICFOLDER: /music
      ND_DBPATH: /data/navidrome.db
      ND_SCANINTERVAL: 1m
      ND_LOGLEVEL: info  
      ND_PORT: 4533
    volumes:
      - "./data:/data"
      - "/Users/deluan/Music/iTunes/iTunes Media/Music:/music"
```

### Build it yourself / Development Environment

You will need to install [Go 1.13](https://golang.org/dl/) and [Node 13.7](http://nodejs.org).
You'll also need [ffmpeg](ffmpeg.org) installed in your system

After the prerequisites above are installed, build the application with:

```
$ make setup
$ make buildall
```

This will generate the `navidrome` binary in the project's root folder. Start the server with:
```shell script
./navidrome
```
The server should start listening for requests on the default port __4533__

### Running for the first time

After starting Navidrome for the first time, go to http://localhost:4533. It will ask you to create your first admin 
user.

## Screenshots

<p align="center">
<p float="left">
    <img width="270" src="https://raw.githubusercontent.com/deluan/navidrome/master/.github/screenshots/screenshot-login-mobile.png">
    <img width="270" src="https://raw.githubusercontent.com/deluan/navidrome/master/.github/screenshots/screenshot-mobile.png">
    <img width="270" src="https://raw.githubusercontent.com/deluan/navidrome/master/.github/screenshots/screenshot-users-mobile.png">
    <img width="900"src="https://raw.githubusercontent.com/deluan/navidrome/master/.github/screenshots/screenshot-desktop.png">
</p>
</p>



## Subsonic API Version Compatibility

Check the up to date [compatibility table](https://github.com/deluan/navidrome/blob/master/API_COMPATIBILITY.md) 
for the latest Subsonic features available.

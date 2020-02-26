# Navidrome Music Streamer

[![Build](https://img.shields.io/github/workflow/status/deluan/navidrome/Build?style=for-the-badge)](https://github.com/deluan/navidrome/actions)
[![Last Release](https://img.shields.io/github/v/release/deluan/navidrome?label=latest&style=for-the-badge)](https://github.com/deluan/navidrome/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/deluan/navidrome?style=for-the-badge)](https://hub.docker.com/r/deluan/navidrome)
[![Join the Chat](https://img.shields.io/discord/671335427726114836?style=for-the-badge)](https://discord.gg/xh7j7yF)

Navidrome is an open source web-based music collection server and streamer. It gives you freedom to listen to your 
music collection from any browser or mobile device. It's like your personal Spotify!

__Any feedback is welcome!__ If you need/want a new feature, find a bug or think of any way to improve Navidrome, 
please fill a [GitHub issue](https://github.com/deluan/navidrome/issues) or join the chat in our [Discord server](https://discord.gg/xh7j7yF)


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
- Transcoding/Downsampling on-the-fly (WIP. Experimental support is available)
- Integrated music player (WIP)

## Road map

This project is being actively worked on. Expect a more polished experience and new features/releases 
on a frequent basis. Some upcoming features planned: 

- Last.FM integration
- Pre-build binaries for Raspberry Pi
- Smart/dynamic playlists (similar to iTunes)
- Support for audiobooks (bookmarking)
- Jukebox mode
- Sharing links to albums/songs/playlists
- Podcasts


## Installation

Various options are available:

### Pre-built executables

Just head to the [releases page](https://github.com/deluan/navidrome/releases) and download the latest version for you 
platform. There are builds available for Linux, macOS and Windows (32 and 64 bits). 

Remember to install [ffmpeg](https://ffmpeg.org/download.html) in your system, a requirement for Navidrome to work properly.
You may find the latest static build for your platform here: https://johnvansickle.com/ffmpeg/ 

If you have any issues with these binaries, or need a binary for a different platform, please 
[open an issue](https://github.com/deluan/navidrome/issues) 

### Docker

[Docker images](https://hub.docker.com/r/deluan/navidrome) are available. They include everything needed to run Navidrome. Example of usage:

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
      ND_DATAFOLDER: /data
      ND_SCANINTERVAL: 1m
      ND_LOGLEVEL: info  
      ND_PORT: 4533
    volumes:
      - "./data:/data"
      - "/path/to/your/music/folder:/music"
```

To get the cutting-edge, latest version from master, use the image `deluan/navidrome:develop`

### Build from source

You will need to install [Go 1.14](https://golang.org/dl/) and [Node 13.7.0](http://nodejs.org).
You'll also need [ffmpeg](https://ffmpeg.org) installed in your system. The setup is very strict, and 
the steps bellow only work with these specific versions (enforced in the Makefile) 

After the prerequisites above are installed, clone this repository and build the application with:

```shell script
$ git clone https://github.com/deluan/navidrome
$ cd navidrome
$ make setup        # Install tools required for Navidrome's development 
$ make buildall     # Build UI and server, generates a single executable
```

This will generate the `navidrome` executable binary in the project's root folder. 

### Running for the first time

Start the server with:
```shell script
./navidrome
```
The server should start listening for requests on the default port __4533__

After starting Navidrome for the first time, go to http://localhost:4533. It will ask you to create your first admin 
user.

For more options, run `navidrome --help` 

## Screenshots

<p align="center">
<p float="left">
    <img width="270" src="https://raw.githubusercontent.com/deluan/navidrome/master/.github/screenshots/ss-mobile-login.png">
    <img width="270" src="https://raw.githubusercontent.com/deluan/navidrome/master/.github/screenshots/ss-mobile-player.png">
    <img width="270" src="https://raw.githubusercontent.com/deluan/navidrome/master/.github/screenshots/ss-mobile-album-view.png">
    <img width="900"src="https://raw.githubusercontent.com/deluan/navidrome/master/.github/screenshots/ss-desktop-player.png">
</p>
</p>


## Subsonic API Version Compatibility

Check the up to date [compatibility table](https://github.com/deluan/navidrome/blob/master/API_COMPATIBILITY.md) 
for the latest Subsonic features available.

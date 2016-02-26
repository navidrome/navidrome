GoSonic
=======

[![Build Status](https://travis-ci.org/deluan/gosonic.svg?branch=master)](https://travis-ci.org/deluan/gosonic)

### About

__This is still a work in progress, and has no releases available__

GoSonic is an application that implements the [Subsonic API](http://www.subsonic.org/pages/api.jsp), but instead of
having its own music library like the original [Subsonic application](http://www.subsonic.org), it interacts directly
with your iTunes library.

The project's main goals are:

* Full compatibility with the available [Subsonic clients](http://www.subsonic.org/pages/apps.jsp)
  (only being tested with
    [DSub](http://www.subsonic.org/pages/apps.jsp#dsub),
    [Jamstash](http://www.subsonic.org/pages/apps.jsp#jamstash))
* Use all metadata from iTunes
* You can keep using iTunes to manage your music
* Update play counts, last played dates, ratings, etc..  on iTunes (at least on Mac OS)
* Learning Go ;) [![Gopher](https://blog.golang.org/favicon.ico)](https://golang.org)

Currently implementing [API version](http://www.subsonic.org/pages/api.jsp#versions):

* _1.0.0 &larr; In progress_
* 1.2.0

### Development Environment

You will need to install [Go 1.6](https://golang.org/dl/)
    
Then install dependencies:
```
$ go get github.com/beego/bee   
$ go get github.com/gpmgo/gopm
$ gopm get -v -g
```  

From here it's a normal Go development cycle:

* Test with `go test ./.. -v`
* Start local server with `bee run`
* Start test runner with `goconvey` 


### Useful Links

#### Frameworks/Projects
* https://github.com/golang/go/wiki/Projects
* https://golanglibs.com/
* https://github.com/deluan/tuner

#### REST/Web
* http://beego.me/

#### DB
https://github.com/HouzuoGuo/tiedot

#### Search
https://github.com/blevesearch/bleve

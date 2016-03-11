GoSonic
=======

[![Build Status](https://travis-ci.org/deluan/gosonic.svg?branch=master)](https://travis-ci.org/deluan/gosonic)

__This is still a work in progress, and has no releases available__

GoSonic is an application that implements the [Subsonic API](http://www.subsonic.org/pages/api.jsp), but instead of
having its own music library like the original [Subsonic application](http://www.subsonic.org), it interacts directly
with your iTunes library.

The project's main goals are:

* Full compatibility with the available [Subsonic clients](http://www.subsonic.org/pages/apps.jsp)
  (only being tested with
    [DSub](http://www.subsonic.org/pages/apps.jsp#dsub) and
    [Jamstash](http://www.subsonic.org/pages/apps.jsp#jamstash))
* Use all metadata from iTunes, so that you can keep using iTunes to manage your music
* Keep iTunes stats (play counts, last played dates, ratings, etc..) updated, at least on Mac OS X
* Learning Go ;) [![Gopher](https://blog.golang.org/favicon.ico)](https://golang.org)


###  Supported Subsonic API version

I'm currently trying to implement all functionality from API v1.4.0, with some exceptions.

Check the (almost) up to date [compatibility chart](https://github.com/deluan/gosonic/wiki/Compatibility) for what is working.

### Development Environment

You will need to install [Go 1.6](https://golang.org/dl/)

Then install dependencies:
```
$ go get github.com/beego/bee   
$ go get github.com/gpmgo/gopm
$ gopm get -v -g
```  

From here it's a normal [BeeGo](http://beego.me) development cycle. Some useful commands:

```bash
# Start local server (with hot reload)
$ bee run

# Start test runner on the browser
$ NOLOG=1 goconvey --port 9090

# Run all tests
$ go test ./... -v
```


### Useful Links

#### Frameworks/Projects
* https://github.com/golang/go/wiki/Projects
* https://golanglibs.com/

#### REST/Web
* http://beego.me/

#### DB
* http://ledisdb.com

#### Search
* https://github.com/sunfmin/redisgosearch
* http://patshaughnessy.net/2011/11/29/two-ways-of-using-redis-to-build-a-nosql-autocomplete-search-index

#### Testing
* http://goconvey.co/

audiotags
=========

library and command to retrieve audio metadata tags (uses [TagLib](http://taglib.github.io/))

- uses libtag directly (doesn't use the C bindings)
- read only support
- returns the extended metadata (albumartist, composer, discnumber, etc...)
- builds with cgo

## Install command and library

    go get github.com/nicksellen/audiotags/audiotags
    
## Install library

    go get github.com/nicksellen/audiotags
    
    
## Command example

    # audiotags '/store/music/John Holloway/J.S. Bach_ The Sonatas and Partitas for violin solo/2-09 Allegro assai [Sonata No. 3 in C major BWV 1005].mp3'
    album J.S. Bach: The Sonatas and Partitas for violin solo
    date 2007
    filetype MPG/3
    title Allegro assai [Sonata No. 3 in C major BWV 1005]
    tracknumber 9
    albumartist John Holloway
    artist John Holloway
    composer Johann Sebastian Bach
    discnumber 2/2
    genre Classical
    length 320
    bitrate 222
    samplerate 44100
    channels 2
    
# Library example

    package main
    
    import (
    	"fmt"
    	"github.com/nicksellen/audiotags"
    	"log"
    	"os"
    )
    
    func main() {
    
    	if len(os.Args) != 2 {
    		fmt.Println("pass path to file")
    		return
    	}
    
    	props, audioProps, err := audiotags.Read(os.Args[1])
    	
    	if err != nil {
    		log.Fatal(err)
    	}
    	
    	for k, v := range props {
    		fmt.Printf("%s %s\n", k, v)
    	}
    
    	fmt.Printf("length %d\nbitrate %d\nsamplerate %d\nchannels %d\n",
    		audioProps.Length, audioProps.Bitrate, audioProps.Samplerate, audioProps.Channels)
    
    }


# Dependencies

On Debian/Ubuntu:

    apt-get install libtag1-vanilla
    
On OS X:

    brew install taglib
    
    

    

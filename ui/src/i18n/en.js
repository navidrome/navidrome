import deepmerge from 'deepmerge'
import englishMessages from 'ra-language-english'

export default deepmerge(englishMessages, {
  resources: {
    song: {
      fields: {
        albumArtist: 'Album Artist',
        duration: 'Time',
        trackNumber: 'Track #'
      },
      bulk: {
        addToQueue: 'Play Later'
      }
    },
    album: {
      fields: {
        albumArtist: 'Album Artist',
        duration: 'Time'
      }
    }
  },
  ra: {
    auth: {
      welcome1: 'Thanks for installing Navidrome!',
      welcome2: 'To start, create an admin user',
      confirmPassword: 'Confirm Password',
      buttonCreateAdmin: 'Create Admin'
    },
    validation: {
      invalidChars: 'Please only use letter and numbers',
      passwordDoesNotMatch: 'Password does not match'
    }
  },
  menu: {
    library: 'Library'
  },
  player: {
    panelTitle: 'Play Queue',
    playModeText: {
      order: 'In order',
      orderLoop: 'Repeat',
      singleLoop: 'Repeat One',
      shufflePlay: 'Shuffle'
    }
  }
})

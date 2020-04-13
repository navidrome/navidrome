import deepmerge from 'deepmerge'
import englishMessages from 'ra-language-english'

export default deepmerge(englishMessages, {
  languageName: 'English',
  resources: {
    song: {
      name: 'Song |||| Songs',
      fields: {
        albumArtist: 'Album Artist',
        duration: 'Time',
        trackNumber: 'Track #',
        playCount: 'Plays'
      },
      bulk: {
        addToQueue: 'Play Later'
      }
    },
    album: {
      fields: {
        albumArtist: 'Album Artist',
        duration: 'Time',
        songCount: 'Songs',
        playCount: 'Plays'
      },
      actions: {
        playAll: 'Play',
        playNext: 'Play Next',
        addToQueue: 'Play Later',
        shuffle: 'Shuffle'
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
    library: 'Library',
    settings: 'Settings',
    version: 'Version %{version}',
    theme: 'Theme',
    personal: {
      name: 'Personal',
      options: {
        theme: 'Theme'
      }
    }
  },
  player: {
    playListsText: 'Play Queue',
    openText: 'Open',
    closeText: 'Close',
    notContentText: 'No music',
    clickToPlayText: 'Click to play',
    clickToPauseText: 'Click to pause',
    nextTrackText: 'Next track',
    previousTrackText: 'Previous track',
    reloadText: 'Reload',
    volumeText: 'Volume',
    toggleLyricText: 'Toggle lyric',
    toggleMiniModeText: 'Minimize',
    destroyText: 'Destroy',
    downloadText: 'Download',
    removeAudioListsText: 'Delete audio lists',
    controllerTitle: '',
    clickToDeleteText: `Click to delete %{name}`,
    emptyLyricText: 'No lyric',
    playModeText: {
      order: 'In order',
      orderLoop: 'Repeat',
      singleLoop: 'Repeat One',
      shufflePlay: 'Shuffle'
    }
  }
})

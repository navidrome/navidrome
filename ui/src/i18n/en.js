import deepmerge from 'deepmerge'
import englishMessages from 'ra-language-english'

export default deepmerge(englishMessages, {
  languageName: 'English',
  resources: {
    song: {
      name: 'Song |||| Songs',
      fields: {
        title: 'Title',
        artist: 'Artist',
        album: 'Album',
        path: 'File Path',
        genre: 'Genre',
        compilation: 'Compilation',
        duration: 'Time',
        year: 'Year',
        trackNumber: '#',
        playCount: 'Plays',
        size: 'File Size',
        updatedAt: 'Updated At',
      },
      actions: {
        playNow: 'Play Now',
        addToQueue: 'Play Later',
      },
    },
    album: {
      name: 'Album |||| Albums',
      fields: {
        albumArtist: 'Album Artist',
        name: 'Name',
        artist: 'Artist',
        songCount: 'Songs',
        playCount: 'Plays',
        genre: 'Genre',
        compilation: 'Compilation',
        duration: 'Duration',
        year: 'Year',
      },
      actions: {
        playAll: 'Play Now',
        playNext: 'Play Next',
        addToQueue: 'Play Later',
        shuffle: 'Shuffle',
      },
    },
    artist: {
      name: 'Artist |||| Artists',
      fields: {
        name: 'Nome',
        albumCount: 'Album Count',
      },
    },
    user: {
      name: 'User |||| Users',
      fields: {
        userName: 'Username',
        name: 'Name',
        isAdmin: 'Is Admin?',
        lastLoginAt: 'Last Login',
        updatedAt: 'Updated At',
      },
    },
    player: {
      name: 'Player |||| Players',
      fields: {
        name: 'Name',
        transcodingId: 'Transcoding',
        maxBitRate: 'Max BitRate',
        client: 'Client',
        userName: 'Username',
        lastSeen: 'Last Seen',
      },
    },
    transcoding: {
      name: 'Transcoding |||| Transcodings',
      fields: {
        name: 'Name',
        targetFormat: 'Target Format',
        defaultBitRate: 'Default BitRate',
        command: 'Command',
      },
    },
  },
  ra: {
    auth: {
      welcome1: 'Thanks for installing Navidrome!',
      welcome2: 'To start, create an admin user',
      confirmPassword: 'Confirm Password',
      buttonCreateAdmin: 'Create Admin',
    },
    validation: {
      invalidChars: 'Please only use letter and numbers',
      passwordDoesNotMatch: 'Password does not match',
    },
  },
  message: {
    note: 'NOTE',
    transcodingDisabled:
      'Changing the transcoding configuration through the web interface is disabled for security ' +
      'reasons. If you would like to change (edit or add) transcoding options, restart the server with ' +
      'the %{config} configuration option.',
    transcodingEnabled:
      'Navidrome is currently running with %{config}, making it possible to run system ' +
      'commands from the transcoding settings using the web interface. We recommend to disable it for security reasons ' +
      'and only enable it when configuring Transcoding options.',
  },
  menu: {
    library: 'Library',
    settings: 'Settings',
    version: 'Version %{version}',
    theme: 'Theme',
    personal: {
      name: 'Personal',
      options: {
        theme: 'Theme',
        language: 'Language',
      },
    },
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
      shufflePlay: 'Shuffle',
    },
  },
})

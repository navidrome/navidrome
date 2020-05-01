import deepmerge from 'deepmerge'
import germanMessages from 'ra-language-german'

export default deepmerge(germanMessages, {
  languageName: 'German',
  resources: {
    song: {
      name: 'Song |||| Songs',
      fields: {
        title: 'Titel',
        artist: 'Interpret',
        album: 'Album',
        path: 'Dateipfad',
        genre: 'Genre',
        compilation: 'Zusammenstellung',
        duration: 'Dauer',
        year: 'Jahr',
        trackNumber: '#',
        playCount: 'Wiedergaben',
        size: 'Dateigröße',
        updatedAt: 'Aktualisiert am',
      },
      actions: {
        playNow: 'Jetzt abspielen',
        addToQueue: 'Zu Wiedergabeliste hinzufügen',
      },
    },
    album: {
      name: 'Album |||| Alben',
      fields: {
        albumArtist: 'Albuminterpret',
        name: 'Name',
        artist: 'Interpret',
        songCount: 'Songanzahl',
        playCount: 'Wiedergaben',
        genre: 'Genre',
        compilation: 'Zusammenstellung',
        duration: 'Dauer',
        year: 'Jahr',
      },
      actions: {
        playAll: 'Jetzt abspielen',
        playNext: 'Als Nächstes abspielen',
        addToQueue: 'Zu Wiedergabeliste hinzufügen',
        shuffle: 'Shuffle',
      },
    },
    artist: {
      name: 'Interpret |||| Interpreten',
      fields: {
        name: 'Name',
        albumCount: 'Albenanzahl',
      },
    },
    user: {
      name: 'Benutzer |||| Benutzer',
      fields: {
        userName: 'Benutzername',
        name: 'Name',
        isAdmin: 'Ist Administrator?',
        lastLoginAt: 'Letzter Login',
        updatedAt: 'Aktualisiert',
      },
    },
    player: {
      name: 'Gerät |||| Geräte',
      fields: {
        name: 'Name',
        transcodingId: 'Transkodierung',
        maxBitRate: 'Maximale Bitrate',
        client: 'Client',
        userName: 'Benutzername',
        lastSeen: 'Zuletzt gesehen',
      },
    },
    transcoding: {
      name: 'Transkodierung |||| Transkodierungen',
      fields: {
        name: 'Name',
        targetFormat: 'Zielformat',
        defaultBitRate: 'Standard-Bitrate',
        command: 'Befehl',
      },
    },
  },
  ra: {
    auth: {
      welcome1: 'Vielen Dank für das Installieren von Navidrome!',
      welcome2: 'Bitte erstelle zuerst einen Admin-Benutzer:',
      confirmPassword: 'Passwort bestätigen',
      buttonCreateAdmin: 'Admin-Benutzer erstellen',
    },
    validation: {
      invalidChars: 'Bitte benutze nur Buchstaben und Zahlen',
      passwordDoesNotMatch: 'Passwörter stimmen nicht überein',
    },
  },
  message: {
    note: 'NOTIZ',
    transcodingDisabled:
      'Das Ändern der Transkodierungeinstellungen über das Web-Interface ist aus Sicherheitsgründen ' +
      'deaktiviert. Um die Transkodierungsoptionen zu ändern (bearbeiten oder hinzufügen), kann der Server ' +
      'mit der Option %{config} neugestartet werden.',
    transcodingEnabled:
      'Navidrome läuft derzeit mit %{config}, wodurch es möglich ist, Systembefehle über die ' +
      'Transkodierungseinstellungen des Web-Interfaces auszuführen. Wir empfehlen, dies aus Sicherheitsgründen ' +
      'zu deaktivieren und nur beim Konfigurieren von Transkodierungsoptionen zu aktivieren.',
  },
  menu: {
    library: 'Bibliothek',
    settings: 'Einstellungen',
    version: 'Version %{version}',
    theme: 'Thema',
    personal: {
      name: 'Persönlich',
      options: {
        theme: 'Thema',
        language: 'Sprache',
      },
    },
  },
  player: {
    playListsText: 'Wiedergabeliste',
    openText: 'Öffnen',
    closeText: 'Schließen',
    notContentText: 'Keine Musik',
    clickToPlayText: 'Klicken zum Abspielen',
    clickToPauseText: 'Klicken zum Pausieren',
    nextTrackText: 'Nächster Song',
    previousTrackText: 'Letzter Song',
    reloadText: 'Neu laden',
    volumeText: 'Lautstärke',
    toggleLyricText: 'Songtexte umschalten',
    toggleMiniModeText: 'Minimieren',
    destroyText: 'Zerstören',
    downloadText: 'Download',
    removeAudioListsText: 'Lösche Wiedergabeliste',
    controllerTitle: '',
    clickToDeleteText: `Klicke um %{name} zu löschen`,
    emptyLyricText: 'Keine Songtexte',
    playModeText: {
      order: 'In Reihenfolge',
      orderLoop: 'Wiederholen',
      singleLoop: 'Titel wiederholen',
      shufflePlay: 'Shuffle',
    },
  },
})

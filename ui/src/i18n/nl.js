import deepmerge from 'deepmerge'
import englishMessages from 'ra-language-dutch'

export default deepmerge(englishMessages, {
  languageName: 'Nederlands',
  resources: {
    song: {
      name: 'Nummer |||| Nummers',
      fields: {
        albumArtist: 'Album Artiest',
        duration: 'Tijd',
        trackNumber: 'Nummer #',
        playCount: 'Aantal keren afgespeeld',
      },
      actions: {
        addToQueue: 'Toevoegen aan afspeellijst',
      },
    },
    album: {
      fields: {
        albumArtist: 'Album Artiest',
        artist: 'Artiest',
        duration: 'Tijd',
        songCount: 'Nummerss',
        playCount: 'Aantal keren afgespeeld',
      },
      actions: {
        playAll: 'Afspelen',
        playNext: 'Hierna afspelen',
        addToQueue: 'Toevoegen aan afspeellijst',
        shuffle: 'Shuffle',
      },
    },
  },
  ra: {
    auth: {
      welcome1: 'Bedankt voor het installeren van Navidrome!',
      welcome2: 'Maak om te beginnen een beheerdersaccount',
      confirmPassword: 'Bevestig wachtwoord',
      buttonCreateAdmin: 'Beheerder maken',
    },
    validation: {
      invalidChars: 'Gebruik alleen letters en cijfers',
      passwordDoesNotMatch: 'Wachtwoord komt niet overeen',
    },
  },
  menu: {
    library: 'Bibliotheek',
    settings: 'Instellingen',
    version: 'Versie %{version}',
    theme: 'Thema',
    personal: {
      name: 'Persoonlijk',
      options: {
        theme: 'Thema',
        language: 'Taal',
      },
    },
  },
  player: {
    playListsText: 'Afspeellijst afspelen',
    openText: 'Openen',
    closeText: 'Sluiten',
    notContentText: 'Geen muziek',
    clickToPlayText: 'Klik om af te spelen',
    clickToPauseText: 'Klik om te pauzeren',
    nextTrackText: 'Volgende',
    previousTrackText: 'Vorige',
    reloadText: 'Herladen',
    volumeText: 'Volume',
    toggleLyricText: 'Songtekst aan/uit',
    toggleMiniModeText: 'Minimaliseren',
    destroyText: 'Vernietigen',
    downloadText: 'Downloaden',
    removeAudioListsText: 'Audiolijsten verwijderen',
    controllerTitle: '',
    clickToDeleteText: `Klik om %{name} te verwijderen`,
    emptyLyricText: 'Geen songtekst',
    playModeText: {
      order: 'In volgorde',
      orderLoop: 'Herhalen',
      singleLoop: 'Herhaal Eenmalig',
      shufflePlay: 'Shuffle',
    },
  },
})

import deepmerge from 'deepmerge'
import italianMessages from 'ra-language-italian'

export default deepmerge(italianMessages, {
  languageName: 'Italiano',
  resources: {
    song: {
      name: 'Traccia |||| Tracce',
      fields: {
        title: 'Titolo',
        artist: 'Artista',
        album: 'Album',
        path: 'Percorso',
        genre: 'Genere',
        compilation: 'Compilation',
        duration: 'Durata',
        year: 'Anno',
        playCount: 'Riproduzioni',
        trackNumber: '#',
        size: 'Dimensioni',
        updatedAt: 'Ultimo aggiornamento',
      },
      bulk: {
        addToQueue: 'Aggiungi alla coda',
      },
    },
    album: {
      name: 'Album |||| Album',
      fields: {
        name: 'Nome',
        artist: 'Artista',
        songCount: 'Tracce',
        genre: 'Genere',
        playCount: 'Riproduzioni',
        compilation: 'Compilation',
        duration: 'Durata',
        year: 'Anno',
      },
      actions: {
        playAll: 'Riproduci',
        playNext: 'Riproduci come successivo',
        addToQueue: 'Aggiungi alla coda',
        shuffle: 'Riprodici casualmente',
      },
    },
    artist: {
      name: 'Artista |||| Artisti',
      fields: {
        name: 'Nome',
        albumCount: 'Album',
      },
    },
    user: {
      name: 'Utente |||| Utenti',
      fields: {
        userName: 'Utente',
        isAdmin: 'Amministratore',
        lastLoginAt: 'Ultimo accesso',
        updatedAt: 'Ultima modifica',
        name: 'Nome',
      },
    },
    player: {
      name: 'Client |||| Client',
      fields: {
        name: 'Nome',
        transcodingId: 'Transcodifica',
        maxBitRate: 'Bitrate massimo',
        client: 'Applicazione',
        userName: 'Utente',
        lastSeen: 'Ultimo acesso',
      },
    },
    transcoding: {
      name: 'Transcodifica |||| Transcodifiche',
      fields: {
        name: 'Nome',
        targetFormat: 'Formato',
        defaultBitRate: 'Bitrate predefinito',
        command: 'Comando',
      },
    },
  },
  ra: {
    auth: {
      welcome1: 'Grazie per aver installato Navidrome!',
      welcome2: 'Per iniziare, crea un amministratore',
      confirmPassword: 'Conferma la password',
      buttonCreateAdmin: 'Crea amministratore',
    },
    validation: {
      invalidChars: 'Per favore usa solo lettere e numeri',
      passwordDoesNotMatch: 'Le password non coincidono',
    },
  },
  menu: {
    library: 'Libreria',
    settings: 'Impostazioni',
    version: 'Versione %{version}',
    personal: {
      name: 'Personale',
      options: {
        theme: 'Tema',
        language: 'Lingua',
      },
    },
  },
  player: {
    playListsText: 'Coda',
    openText: 'Apri',
    closeText: 'Chiudi',
    clickToPlayText: 'Clicca per riprodurre',
    clickToPauseText: 'Clicca per mettere in pausa',
    nextTrackText: 'Traccia successiva',
    previousTrackText: 'Traccia precedente',
    volumeText: 'Volume',
    toggleMiniModeText: 'Minimizza',
    removeAudioListsText: 'Cancella coda',
    clickToDeleteText: `Clicca per rimuovere %{name}`,
    playModeText: {
      order: 'In ordine',
      orderLoop: 'Ripeti',
      singleLoop: 'Ripeti una volta',
      shufflePlay: 'Casuale',
    },
  },
})

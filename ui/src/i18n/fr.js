import deepmerge from 'deepmerge'
import frenchMessages from 'ra-language-french'

export default deepmerge(frenchMessages, {
  languageName: 'Français',
  resources: {
    song: {
      name: 'Piste |||| Pistes',
      fields: {
        title: 'Titre',
        artist: 'Artiste',
        album: 'Album',
        path: 'Chemin',
        genre: 'Genre',
        compilation: 'Compilation',
        duration: 'Durée',
        year: 'Année',
        playCount: "Nombre d'écoutes",
        trackNumber: '#',
        size: 'Taille',
        updatedAt: 'Mise à jour',
      },
      bulk: {
        addToQueue: 'Ajouter à la file',
      },
    },
    album: {
      name: 'Album |||| Albums',
      fields: {
        name: 'Nom',
        artist: 'Artiste',
        songCount: 'Numéro de piste',
        genre: 'Genre',
        playCount: "Numbre d'écoutes",
        compilation: 'Compilation',
        duration: 'Durée',
        year: 'Année',
      },
      actions: {
        playAll: 'Lire',
        playNext: 'Lire ensuite',
        addToQueue: 'Ajouter à la file',
        shuffle: 'Mélanger',
      },
    },
    artist: {
      name: 'Artiste |||| Artistes',
      fields: {
        name: 'Nom',
        albumCount: "Nombre d'albums",
      },
    },
    user: {
      name: 'Utilisateur |||| Utilisateurs',
      fields: {
        userName: "Nom d'utilisateur",
        isAdmin: 'Administrateur',
        lastLoginAt: 'Dernière connexion',
        updatedAt: 'Dernière mise à jour',
        name: 'Nom',
      },
    },
    player: {
      name: 'Lecteur |||| Lecteurs',
      fields: {
        name: 'Nom',
        transcodingId: 'Transcodage',
        maxBitRate: 'Bitrate maximum',
        client: 'Client',
        userName: "Nom d'utilisateur",
        lastSeen: 'Vu pour la dernière fois',
      },
    },
    transcoding: {
      name: 'Conversion |||| Conversions',
      fields: {
        name: 'Nom',
        targetFormat: 'Format',
        defaultBitRate: 'Bitrate  par défaut',
        command: 'Commande',
      },
    },
  },
  ra: {
    auth: {
      welcome1: "Merci d'avoir installé Navidrome !",
      welcome2: 'Pour commencer, créez un compte administrateur',
      confirmPassword: 'Confirmer votre mot de passe',
      buttonCreateAdmin: 'Créer un compte administrateur',
    },
    validation: {
      invalidChars: "Merci d'utiliser uniquement des chiffres et des lettres",
      passwordDoesNotMatch: 'Les mots de passes ne correspondent pas',
    },
  },
  menu: {
    library: 'Bibliothèque',
    settings: 'Paramètres',
    version: 'Version%{version}',
    personal: {
      name: 'Paramètres personel',
      options: {
        theme: 'Thème',
        language: 'Langue',
      },
    },
  },
  player: {
    playListsText: 'File de lecture',
    openText: 'Ouvrir',
    closeText: 'Fermer',
    clickToPlayText: 'Cliquer pour lire',
    clickToPauseText: 'Cliquer pour mettre en pause',
    nextTrackText: 'Morceau suivant',
    previousTrackText: 'Morceau précédent',
    volumeText: 'Volume',
    toggleMiniModeText: 'Minimiser',
    removeAudioListsText: 'Vider la liste de lecture',
    clickToDeleteText: `Cliquer pour supprimer %{name}`,
    playModeText: {
      order: 'Ordonner',
      orderLoop: 'Tout répéter',
      singleLoop: 'Repéter',
      shufflePlay: 'Aleatoire',
    },
  },
})

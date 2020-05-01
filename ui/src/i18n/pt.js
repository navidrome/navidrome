import deepmerge from 'deepmerge'
import portugueseMessages from 'ra-language-portuguese'

export default deepmerge(portugueseMessages, {
  languageName: 'Português',
  resources: {
    song: {
      name: 'Música |||| Músicas',
      fields: {
        title: 'Título',
        artist: 'Artista',
        album: 'Álbum',
        path: 'Arquivo',
        genre: 'Gênero',
        compilation: 'Coletânea',
        duration: 'Duração',
        year: 'Ano',
        playCount: 'Execuções',
        trackNumber: '#',
        size: 'Tamanho',
        updatedAt: 'Últ. Atualização',
      },
      actions: {
        playNow: 'Tocar agora',
        addToQueue: 'Tocar por último',
      },
    },
    album: {
      name: 'Álbum |||| Álbuns',
      fields: {
        name: 'Nome',
        artist: 'Artista',
        songCount: 'Músicas',
        genre: 'Gênero',
        playCount: 'Execuções',
        compilation: 'Coletânea',
        duration: 'Duração',
        year: 'Ano',
      },
      actions: {
        playAll: 'Tocar',
        playNext: 'Tocar em seguida',
        addToQueue: 'Tocar no fim',
        shuffle: 'Aleatório',
      },
    },
    artist: {
      name: 'Artista |||| Artistas',
      fields: {
        name: 'Nome',
        albumCount: 'Total de Álbuns',
      },
    },
    user: {
      name: 'Usuário |||| Usuários',
      fields: {
        userName: 'Usuário',
        isAdmin: 'Admin?',
        lastLoginAt: 'Últ. Login',
        updatedAt: 'Últ. Atualização',
        name: 'Nome',
      },
    },
    player: {
      name: 'Tocador |||| Tocadores',
      fields: {
        name: 'Nome',
        transcodingId: 'Conversão',
        maxBitRate: 'Bitrate máx',
        client: 'Cliente',
        userName: 'Usuário',
        lastSeen: 'Últ. acesso',
      },
    },
    transcoding: {
      name: 'Conversão |||| Conversões',
      fields: {
        name: 'Nome',
        targetFormat: 'Formato',
        defaultBitRate: 'Bitrate padrão',
        command: 'Comando',
      },
    },
  },
  ra: {
    auth: {
      welcome1: 'Obrigado por instalar Navidrome!',
      welcome2: 'Para iniciar, crie um usuário admin',
      confirmPassword: 'Confirme a senha',
      buttonCreateAdmin: 'Criar Admin',
    },
    validation: {
      invalidChars: 'Somente use letras e numeros',
      passwordDoesNotMatch: 'Senha não confere',
    },
    page: {
      create: 'Criar %{name}',
    },
  },
  menu: {
    library: 'Biblioteca',
    settings: 'Configurações',
    version: 'Versão %{version}',
    personal: {
      name: 'Pessoal',
      options: {
        theme: 'Tema',
        language: 'Língua',
      },
    },
  },
  player: {
    playListsText: 'Fila de Execução',
    openText: 'Abrir',
    closeText: 'Fechar',
    clickToPlayText: 'Clique para tocar',
    clickToPauseText: 'Clique para pausar',
    nextTrackText: 'Próxima faixa',
    previousTrackText: 'Faixa anterior',
    volumeText: 'Volume',
    toggleMiniModeText: 'Minimizar',
    removeAudioListsText: 'Limpar fila de execução',
    clickToDeleteText: `Clique para remover %{name}`,
    playModeText: {
      order: 'Em ordem',
      orderLoop: 'Repetir tudo',
      singleLoop: 'Repetir',
      shufflePlay: 'Aleatório',
    },
  },
})

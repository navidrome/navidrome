import deepmerge from 'deepmerge'
import en from './en'
import portugueseMessages from 'ra-language-portuguese'

export default deepmerge.all([
  en,
  portugueseMessages,
  {
    languageName: 'Português',
    resources: {
      song: {
        name: 'Música |||| Músicas',
        fields: {
          title: 'Título',
          artist: 'Artista',
          album: 'Álbum',
          path: 'Caminho',
          genre: 'Gênero',
          compilation: 'Coletânea',
          duration: 'Duração',
          year: 'Ano',
          trackNumber: '#'
        },
        bulk: {
          addToQueue: 'Play Later'
        }
      },
      album: {
        name: 'Álbum |||| Álbuns',
        fields: {
          name: 'Nome',
          artist: 'Artista',
          songCount: 'Songs',
          genre: 'Gênero',
          playCount: 'Plays',
          compilation: 'Coletânea',
          duration: 'Duração',
          year: 'Ano'
        },
        actions: {
          playAll: 'Play',
          playNext: 'Play Next',
          addToQueue: 'Play Later',
          shuffle: 'Shuffle'
        }
      },
      artist: {
        name: 'Artista |||| Artistas',
        fields: {
          name: 'Nome'
        }
      },
      user: {
        name: 'Usuário |||| Usuários',
        fields: {
          name: 'Nome'
        }
      },
      transcoding: {
        name: 'Conversão |||| Conversões',
        fields: {
          name: 'Nome'
        }
      },
      player: {
        name: 'Tocador |||| Tocadores',
        fields: {
          name: 'Nome'
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
      library: 'Biblioteca',
      settings: 'Configurações',
      version: 'Versão %{version}',
      personal: {
        name: 'Pessoal',
        options: {
          theme: 'Tema',
          language: 'Língua'
        }
      }
    },
    player: {
      playListsText: 'Fila de Execução',
      openText: 'Abrir',
      closeText: 'Fechar',
      clickToPlayText: 'Clique para tocar',
      clickToPauseText: 'Clique para pausar',
      nextTrackText: 'Próxima faixa',
      previousTrackText: 'Faixa anterior',
      clickToDeleteText: `Clique para remover %{name}`,
      playModeText: {
        order: 'Em ordem',
        orderLoop: 'Repetir tudo',
        singleLoop: 'Repetir',
        shufflePlay: 'Aleatório'
      }
    }
  }
])

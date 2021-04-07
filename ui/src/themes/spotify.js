const spotifyGreen = {
  300: '#62ec83',
  500: '#1db954',
  900: '#008827',
}

// For Album, Playlist
const musicListActions = {
  padding: 0,
  alignItems: 'center',
  '@global': {
    button: {
      margin: 5,
      border: '1px solid transparent',
      backgroundColor: 'inherit',
      color: '#b3b3b3',
      '&:hover': {
        border: '1px solid #b3b3b3',
        backgroundColor: 'inherit !important',
      },
    },
    'button:first-child': {
      '@media screen and (max-width: 720px)': {
        transform: 'scale(1.5)',
        margin: '1rem',
        '&:hover': {
          transform: 'scale(1.6) !important',
        },
      },
      transform: 'scale(2)',
      margin: '0 1.5rem',
      minWidth: 0,
      padding: 5,
      transition: 'transform .3s ease',
      background: spotifyGreen['500'],
      color: '#fff',
      borderRadius: 500,
      border: 0,
      '&:hover': {
        transform: 'scale(2.1)',
        backgroundColor: `${spotifyGreen['500']} !important`,
        border: 0,
      },
    },
    'button:first-child>span:first-child': {
      padding: 0,
    },
    'button:first-child>span:first-child>span': {
      display: 'none',
    },
    'button>span:first-child>span, button:not(:first-child)>span:first-child>svg': {
      color: '#b3b3b3',
    },
  },
}

export default {
  themeName: 'Spotify-ish',
  typography: {
    fontFamily: "system-ui, 'Helvetica Neue', Helvetica, Arial",
    h6: {
      fontSize: '1rem', // AppBar title
    },
  },
  palette: {
    primary: {
      light: spotifyGreen['300'],
      main: spotifyGreen['500'],
    },
    secondary: {
      main: '#fff',
      contrastText: '#fff',
    },
    background: {
      default: '#121212',
      paper: 'inherit',
    },
    type: 'dark',
  },
  overrides: {
    MuiPopover: {
      paper: {
        backgroundColor: '#121212',
      },
    },
    MuiDialog: {
      paper: {
        backgroundColor: '#121212',
      },
    },
    MuiFormGroup: {
      root: {
        color: spotifyGreen['500'],
      },
    },
    MuiMenuItem: {
      root: {
        fontSize: '0.875rem',
      },
    },
    MuiDivider: {
      root: {
        margin: '.75rem 0',
      },
    },
    MuiIconButton: {
      label: {
        // color: '#fff'
      },
    },
    MuiButton: {
      root: {
        background: spotifyGreen['500'],
        color: '#fff',
        // margin: '5px',
        border: '1px solid transparent',
        borderRadius: 500,
        '&:hover': {
          background: `${spotifyGreen['900']} !important`,
        },
      },
      textSecondary: {
        border: '1px solid #b3b3b3',
        background: '#000',
        '&:hover': {
          border: '1px solid #fff !important',
          background: '#000 !important',
        },
      },
      label: {
        color: '#fff',
        paddingRight: '1rem',
        paddingLeft: '0.7rem',
      },
    },
    MuiDrawer: {
      root: {
        background: '#000',
        paddingTop: '10px',
      },
    },
    MuiTableRow: {
      root: {
        padding: '10px 0',
        transition: 'background-color .3s ease',
        '&:hover': {
          backgroundColor: '#1d1d1d !important',
        },
        '@global': {
          'td:nth-child(4)': {
            color: '#fff !important',
          },
        },
      },
    },
    MuiTableCell: {
      root: {
        borderBottom: '1px solid #1d1d1d',
        padding: '10px !important',
        color: '#b3b3b3 !important',
      },
      head: {
        borderBottom: '1px solid #282828',
        fontSize: '0.75rem',
        textTransform: 'uppercase',
        letterSpacing: 1.2,
      },
    },
    MuiAppBar: {
      positionFixed: {
        backgroundColor: '#000 !important',
        boxShadow: 'none',
      },
    },
    NDAlbumGridView: {
      albumName: {
        marginTop: '0.5rem',
        fontWeight: 700,
        textTransform: 'none',
        color: '#fff',
      },
      albumSubtitle: {
        color: '#b3b3b3',
      },
      albumContainer: {
        backgroundColor: '#181818',
        borderRadius: '.5rem',
        padding: '.75rem',
        transition: 'background-color .3s ease',
        '&:hover': {
          backgroundColor: '#282828',
        },
      },
      albumPlayButton: {
        backgroundColor: spotifyGreen['500'],
        borderRadius: '50%',
        boxShadow: '0 8px 8px rgb(0 0 0 / 30%)',
        padding: '0.35rem',
        transition: 'padding .3s ease',
        '&:hover': {
          background: `${spotifyGreen['500']} !important`,
          padding: '0.45rem',
        },
      },
    },
    NDPlaylistDetails: {
      container: {
        background: 'linear-gradient(#1d1d1d, transparent)',
        borderRadius: 0,
        paddingTop: '2.5rem !important',
        boxShadow: 'none',
      },
      title: {
        fontSize: 'calc(1.5rem + 1.5vw);',
        fontWeight: 700,
        color: '#fff',
      },
      details: {
        fontSize: '.875rem',
        color: 'rgba(255,255,255, 0.8)',
      },
    },
    NDAlbumDetails: {
      root: {
        background: 'linear-gradient(#1d1d1d, transparent)',
        borderRadius: 0,
        boxShadow: 'none',
      },
      cardContents: {
        alignItems: 'center',
        paddingTop: '1.5rem',
      },
      recordName: {
        fontSize: 'calc(1rem + 1.5vw);',
        fontWeight: 700,
      },
      recordArtist: {
        fontSize: '.875rem',
        fontWeight: 700,
      },
      recordMeta: {
        fontSize: '.875rem',
        color: 'rgba(255,255,255, 0.8)',
      },
      commentBlock: {
        fontSize: '.875rem',
        color: 'rgba(255,255,255, 0.8)',
      },
    },
    NDAlbumShow: {
      albumActions: musicListActions,
    },
    NDPlaylistShow: {
      playlistActions: musicListActions,
    },
    NDAudioPlayer: {
      audioTitle: {
        color: '#fff',
        fontSize: '0.875rem',
        '&.songTitle': {
          fontWeight: 400,
        },
        '&.songInfo': {
          fontSize: '0.675rem',
          color: '#b3b3b3',
        },
      },
      player: {
        border: '10px solid blue',
      },
    },
    NDLogin: {
      main: {
        boxShadow: 'inset 0 0 0 2000px rgba(0, 0, 0, .8)',
      },
      systemNameLink: {
        color: '#fff',
        textDecoration: 'none',
      },
      systemName: {
        marginTop: '0.5em',
        // borderBottom: '1px solid #282828',
        marginBottom: '1em',
      },
      icon: {
        backgroundColor: 'inherit',
      },
      card: {
        // background: 'none',
        background: 'none',
        boxShadow: 'none',
        padding: '10px 0',
        minWidth: 360,
      },
      avatar: {
        marginBottom: 0,
      },
    },
    RaLayout: {
      content: {
        padding: '0 !important',
        background: 'linear-gradient(#171717, #121212)',
      },
    },
    RaListToolbar: {
      toolbar: {
        padding: '0 .55rem !important',
      },
    },
    RaSearchInput: {
      input: {
        paddingLeft: '.9rem',
        border: 0,
      },
    },
    RaFilterButton: {
      root: {
        marginRight: '1rem',
      },
    },
    RaAutocompleteSuggestionList: {
      suggestionsPaper: {
        backgroundColor: '#121212',
      },
    },
    RaPaginationActions: {
      button: {
        backgroundColor: 'inherit',
        minWidth: 48,
        margin: '0 4px',
        border: '1px solid #282828',
        '@global': {
          '> .MuiButton-label': {
            padding: 0,
          },
        },
      },
      actions: {
        '@global': {
          '.next-page': {
            marginLeft: 8,
            marginRight: 8,
          },
          '.previous-page': {
            marginRight: 8,
          },
        },
      },
    },
  },
  player: {
    theme: 'dark',
  },
}

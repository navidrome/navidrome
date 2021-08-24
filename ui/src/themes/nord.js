export default {
  themeName: 'Nord',
  palette: {
    primary: {
      main: '#D8DEE9',
    },
    secondary: {
      main: '#4C566A'
    },
    type: 'dark',
  },
  overrides: {
    MuiFormGroup: {
      root: {
        color: '#D8DEE9',
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
    MuiButton: {
      root: {
        backgroundColor: '#4C566A !important',
        border: '1px solid transparent',
        borderRadius: 500,
        '&:hover': {
          backgroundColor: `${'#5E81AC !important'}`,
        },
      },
      label: {
        color: '#D8DEE9',
        paddingRight: '1rem',
        paddingLeft: '0.7rem',
      },
    },
    MuiIconButton: {
      label:{
        color: '#D8DEE9',
      },
    },
    MuiDrawer: {
      root: {
        background: '#2E3440',
        paddingTop: '10px',
      },
    },

    MuiList:{
      root: {
        color: '#D8DEE9',
        background: 'none',
      },
    },
    MuiPaper:{
      root:{
        backgroundColor: '#3B4252',
      },
    },
    MuiTableRow: {
      root: {
        color: '#434C5E',
        transition: 'background-color .3s ease',
        '&:hover': {
          backgroundColor: '#434C5E !important',
        },
      },
      head: {
        color: '#4C566A',
      }
    },
    MuiToolbar:{
      root:{
        backgroundColor: '#3B4252 !important',
      },
    },
    MuiTableCell: {
      root: {
        borderBottom: '1px solid #4C566A !important',
        color: '#b3b3b3 !important',
        padding: '5px !important',
      },
      head: {
        borderBottom: '1px solid #2E3440',
        fontSize: '0.75rem',
        textTransform: 'uppercase',
        letterSpacing: 1.2,
        backgroundColor: '#4C566A !important',
        color: '#D8DEE9 !important',
      },
      body: {
        color: '#D8DEE9 !important',
      }
    },
    RaToolBar:{
      regular:{
        backgroundColor: 'none !important',
      },
    },
    MuiAppBar: {
      positionFixed: {
        backgroundColor: '#4C566A !important',
        boxShadow: '0 8px 8px rgb(0 0 0 / 30%)',
      },
    },
    NDAlbumGridView: {
      albumName: {
        marginTop: '0.5rem',
        fontWeight: 700,
        textTransform: 'none',
        color: '#E5E9F0',
      },
      albumSubtitle: {
        color: '#b3b3b3',
      },
      albumContainer: {
        backgroundColor: '#434C5E',
        borderRadius: '.5rem',
        padding: '.75rem',
        transition: 'background-color .3s ease',
        '&:hover': {
          backgroundColor: '#4C566A',
        },
      },
      albumPlayButton: {
        backgroundColor: '#3B4252',
        borderRadius: '50%',
        boxShadow: '0 8px 8px rgb(0 0 0 / 30%)',
        padding: '0.35rem',
        transition: 'padding .3s ease',
        '&:hover': {
          background: `${'#4C566A'} !important`,
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
        minWidth: '75vw',
        color: 'rgba(255,255,255, 0.8)',
      },
    },
    NDAlbumDetails: {
      root: {
        background: '#434C5E',
        borderRadius: 0,
        boxShadow: '0 8px 8px rgb(0 0 0 / 20%)',
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
    NDAudioPlayer: {
      audioTitle: {
        color: '#D8DEE9',
        fontSize: '0.875rem',
      },
      songTitle: {
        fontWeight: 400,
      },
      songInfo: {
        fontSize: '0.675rem',
        color: '#b3b3b3',
      },
      player: {
        border: '10px solid #4C566A',
        backgroundColor: '#4C566A !important',
      },
    },
    NDLogin: {
      main: {
        boxShadow: 'inset 0 0 0 2000px rgba(0, 0, 0, .75)',
      },
      systemNameLink: {
        color: '#fff',
      },
      card: {
        border: '1px solid #282828',
      },
      avatar: {
        marginBottom: 0,
      },
    },
    RaLayout: {
      content: {
        padding: '0 !important',
        background: '#2E3440',
      },
    },
    RaList: {
      content: {
        backgroundColor: '#3B4252',
      },
    },
    RaListToolbar: {
      toolbar: {
        backgroundColor: '#2E3440',
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
    RaPaginationActions: {
      currentPageButton: {
        border: '1px solid #b3b3b3',
      },
      button: {
        backgroundColor: '#4C566A',
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


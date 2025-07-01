import stylesheet from './nord.css.js'

// For Album, Playlist
const musicListActions = {
  alignItems: 'center',
  '@global': {
    'button:first-child:not(:only-child)': {
      '@media screen and (max-width: 720px)': {
        transform: 'scale(1.5)',
        margin: '1rem',
        '&:hover': {
          transform: 'scale(1.6) !important',
        },
      },
      transform: 'scale(2)',
      margin: '1.5rem',
      minWidth: 0,
      padding: 5,
      transition: 'transform .3s ease',
      backgroundColor: '#5E81AC !important',
      color: '#fff',
      borderRadius: 500,
      border: 0,
      '&:hover': {
        transform: 'scale(2.1)',
        backgroundColor: '#5E81AC !important',
        border: 0,
      },
    },
    'button:only-child': {
      margin: '1.5rem',
    },
    'button:first-child>span:first-child': {
      padding: 0,
    },
    'button:first-child>span:first-child>span': {
      display: 'none',
    },
    'button>span:first-child>span, button:not(:first-child)>span:first-child>svg':
      {
        color: 'rgba(255, 255, 255, 0.8)',
      },
  },
}

export default {
  themeName: 'Nord',
  palette: {
    primary: {
      main: '#D8DEE9',
    },
    secondary: {
      main: '#4C566A',
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
        paddingTop: '4px',
        paddingBottom: '4px',
        paddingLeft: '10px',
        margin: '5px',
        borderRadius: '8px',
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
      contained: {
        boxShadow: 'none',
        '&:hover': {
          boxShadow: 'none',
        },
      },
    },
    MuiIconButton: {
      label: {
        color: '#D8DEE9',
      },
    },
    MuiDrawer: {
      root: {
        background: '#2E3440',
        paddingTop: '10px',
      },
    },

    MuiList: {
      root: {
        color: '#D8DEE9',
        background: 'none',
      },
    },
    MuiListItem: {
      button: {
        transition: 'background-color .1s ease !important',
      },
    },
    MuiPaper: {
      root: {
        backgroundColor: '#3B4252',
      },
      rounded: {
        borderRadius: '8px',
      },
      elevation1: {
        boxShadow: 'none',
      },
    },
    MuiTableRow: {
      root: {
        color: '#434C5E',
        transition: 'background-color .3s ease',
        '&:hover': {
          backgroundColor: '#434C5E !important',
        },
        '&:last-child': {
          borderBottom: '1px solid #4C566A !important',
        },
      },
      head: {
        color: '#4C566A',
      },
    },
    MuiToolbar: {
      root: {
        backgroundColor: '#3B4252 !important',
      },
    },
    MuiTableCell: {
      root: {
        borderBottom: 'none',
        color: '#b3b3b3 !important',
        padding: '10px !important',
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
      },
    },
    MuiSwitch: {
      track: {
        width: '89%',
        transform: 'translateX(.1rem) scale(140%)',
        opacity: '0.7 !important',
        backgroundColor: 'rgba(255,255,255,0.25)',
      },
      thumb: {
        transform: 'scale(60%)',
        boxShadow: 'none',
      },
    },
    RaToolBar: {
      regular: {
        backgroundColor: 'none !important',
      },
    },
    MuiAppBar: {
      positionFixed: {
        backgroundColor: '#4C566A !important',
        boxShadow:
          'rgba(15, 17, 21, 0.25) 0px 4px 6px, rgba(15, 17, 21, 0.1) 0px 5px 7px',
      },
    },
    MuiOutlinedInput: {
      root: {
        borderRadius: '8px',
        '&:hover': {
          borderColor: '#D8DEE9',
        },
      },
      notchedOutline: {
        transition: 'border-color .1s',
      },
    },
    MuiSelect: {
      select: {
        '&:focus': {
          borderRadius: '8px',
        },
      },
    },
    MuiChip: {
      root: {
        backgroundColor: '#4C566A',
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
        borderRadius: '8px',
        padding: '.75rem',
        transition: 'background-color .3s ease',
        '&:hover': {
          backgroundColor: '#4C566A',
        },
      },
      albumPlayButton: {
        backgroundColor: '#5E81AC',
        borderRadius: '50%',
        boxShadow: '0 8px 8px rgb(0 0 0 / 30%)',
        padding: '0.35rem',
        transition: 'padding .3s ease',
        '&:hover': {
          background: `${'#5E81AC'} !important`,
          padding: '0.45rem',
        },
      },
    },
    NDPlaylistDetails: {
      container: {
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
    NDAlbumShow: {
      albumActions: musicListActions,
    },
    NDPlaylistShow: {
      playlistActions: musicListActions,
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
    },
    NDCollapsibleComment: {
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
    NDSubMenu: {
      sidebarIsClosed: {
        '& a': {
          paddingLeft: '10px',
        },
      },
    },
    RaLayout: {
      content: {
        padding: '0 !important',
        background: '#3B4252',
        backgroundColor: 'rgb(59, 66, 82)',
      },
      root: {
        backgroundColor: '#2E3440',
      },
    },
    RaList: {
      content: {
        backgroundColor: '#3B4252',
        borderRadius: '0px',
      },
    },
    RaListToolbar: {
      toolbar: {
        backgroundColor: '#2E3440',
        padding: '0 .55rem !important',
      },
    },
    RaSidebar: {
      fixed: {
        backgroundColor: '#2E3440',
      },
      drawerPaper: {
        backgroundColor: '#2E3440 !important',
      },
    },
    RaSearchInput: {
      input: {
        paddingLeft: '.9rem',
        marginTop: '36px',
        border: 0,
      },
    },
    RaDatagrid: {
      headerCell: {
        '&:first-child': {
          borderTopLeftRadius: '0px !important',
        },
        '&:last-child': {
          borderTopRightRadius: '0px !important',
        },
      },
    },
    RaButton: {
      button: {
        margin: '0 5px 0 5px',
      },
    },
    RaLink: {
      link: {
        color: '#81A1C1',
      },
    },
    RaPaginationActions: {
      currentPageButton: {
        border: '2px solid rgba(255,255,255,0.25)',
      },
      button: {
        backgroundColor: '#4C566A',
        minWidth: 48,
        margin: '0 4px',
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
    stylesheet,
  },
}

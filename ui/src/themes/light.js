import blue from '@material-ui/core/colors/blue'

const tLight = {
  300: '#62ec83',
  500: '#48e208',
  900: '#008827',
}
const musicListActions = {
  padding: '1rem 0',
  alignItems: 'center',
  '@global': {
    button: {
      margin: 5,
      border: '1px solid transparent',
      backgroundColor: '#fff',
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
      margin: '1.5rem',
      minWidth: 0,
      padding: 5,
      transition: 'transform .3s ease',
      background: tLight['500'],
      color: '#fff',
      borderRadius: 500,
      border: 0,
      '&:hover': {
        transform: 'scale(2.1)',
        backgroundColor: `${tLight['500']} !important`,
        border: 0,
      },
    },
    'button:first-child>span:first-child': {
      padding: 0,
      color: '#fff',
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
  themeName: 'Light',
  palette: {
    primary: {
      light: tLight['300'],
      main: tLight['500'],
    },
    secondary: {
      main: '#000',
      contrastText: '#fff',
    },
    background: {
      default: '#fff',
      paper: 'inherit',
    },
    text: {
      secondary: '#fff',
    },
  },
  typography: {
    fontFamily: "system-ui, 'Helvetica Neue', Helvetica, Arial",
    h6: {
      fontSize: '1rem',
    },
  },
  overrides: {
    MuiPopover: {
      paper: {
        backgroundColor: tLight['500'],
        '& .MuiListItemIcon-root': {
          color: '#fff',
        },
      },
    },
    MuiDialog: {
      paper: {
        backgroundColor: '#5f5fc4',
      },
    },
    MuiFormGroup: {
      root: {
        color: tLight['500'],
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
    MuiFormLabel: {
      root: {
        color: '#9ab191',
      },
    },
    MuiIconButton: {
      label: {},
    },
    MuiButton: {
      root: {
        background: '#fff',
        color: '#fff',
        border: '1px solid transparent',
        borderRadius: 500,
        '&:hover': {
          background: `${tLight['500']} !important`,
        },
      },
      containedPrimary: {
        backgroundColor: '#fff',
      },
      textPrimary: {
        backgroundColor: tLight['500'],
      },
      textSecondary: {
        border: '1px solid #b3b3b3',
        background: '#fff',
        '&:hover': {
          border: '1px solid #fff !important',
          background: `${tLight['500']} !important`,
        },
      },
      label: {
        color: '#000',
        paddingRight: '1rem',
        paddingLeft: '0.7rem',
        '&:hover': {
          color: '#fff',
        },
      },
    },
    MuiDrawer: {
      root: {
        background: '#48e208',
        paddingTop: '10px',
      },
      '&:hover': {
        backgroundColor: '#000',
      },
    },
    MuiTableRow: {
      root: {
        padding: '10px 0',
        transition: 'background-color .3s ease',
        '&:hover': {
          backgroundColor: '#48e208 !important',
        },
        '@global': {
          'td:nth-child(4)': {
            color: '#5f5f5f !important',
          },
        },
      },
      head: {
        backgroundColor: '#eefbe8',
      },
    },
    MuiTableCell: {
      root: {
        borderBottom: '1px solid #1d1d1d',
        padding: '10px !important',
        color: '#000000b0 !important',
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
        backgroundColor: '#48e208 !important',
        boxShadow: 'none',
      },
    },
    NDAppBar: {
      icon: {
        color: '#fff',
      },
    },
    NDAlbumGridView: {
      albumName: {
        marginTop: '0.5rem',
        fontWeight: 700,
        textTransform: 'none',
        color: '#000000b0',
      },
      albumSubtitle: {
        color: '#000000ad',
        display: 'block',
      },
      albumContainer: {
        backgroundColor: '#02ff0a14',
        borderRadius: '.5rem',
        padding: '.75rem',
        transition: 'background-color .3s ease',
        '&:hover': {
          backgroundColor: '#40ff0266',
        },
      },
      albumPlayButton: {
        backgroundColor: tLight['500'],
        borderRadius: '50%',
        boxShadow: '0 8px 8px rgb(0 0 0 / 30%)',
        padding: '0.35rem',
        transition: 'padding .3s ease',
        color: '#fff',
        '&:hover': {
          background: `${tLight['500']} !important`,
          padding: '0.45rem',
          color: '#fff',
        },
      },
    },
    NDPlaylistDetails: {
      container: {
        background: 'linear-gradient(#edfff4, transparent)',
        borderRadius: 0,
        paddingTop: '2.5rem !important',
        boxShadow: 'none',
      },
      title: {
        fontSize: 'calc(1.5rem + 1.5vw);',
        fontWeight: 700,
        color: '#000000b0',
      },
      details: {
        fontSize: '.875rem',
        color: 'rgba(255,255,255, 0.8)',
      },
    },
    NDAlbumDetails: {
      root: {
        background: 'linear-gradient(#d5f7c8, #ffffff)',
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
    NDSubMenu: {
      icon: {
        color: '#fff',
      },
    },
    NDAudioPlayer: {
      audioTitle: {
        color: '#000',
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
      actions: {
        '& button': {
          backgroundColor: '#79d06f',
        },
      },
      systemNameLink: {
        textDecoration: 'none',
        color: tLight['500'],
      },
      systemName: {
        marginTop: '0.5em',
        marginBottom: '1em',
      },
      icon: {
        backgroundColor: 'transparent',
        width: '100px',
      },
      card: {
        minWidth: 300,
        marginTop: '6em',
        overflow: 'visible',
        backgroundColor: '#ffffffe6',
      },
      avatar: {
        marginTop: '-50px',
      },
      button: {
        backgroundColor: 'green!important',
      },
    },
    RaLayout: {
      content: {
        padding: '0 !important',
        background: 'linear-gradient(#d5f7c8, #ffffff)',
      },
    },
    RaListToolbar: {
      toolbar: {
        padding: '0 .55rem !important',
      },
    },
    RaDatagridHeaderCell: {
      icon: {
        color: '#717171 !important',
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
        '& button': {
          backgroundColor: '#fff',
        },
      },
    },
    RaAutocompleteSuggestionList: {
      suggestionsPaper: {
        backgroundColor: '#fff',
      },
    },
    RaLink: {
      link: {
        color: '#717171',
      },
    },
    RaLogout: {
      icon: {
        color: '#f90000!important',
      },
    },
    RaMenuItemLink: {
      root: {
        color: '#fff',
        '& .MuiListItemIcon-root': {
          color: '#fff',
        },
      },
      active: {
        backgroundColor: '#b6ffc4',
        color: '#000000b0 !important',
        '& .MuiListItemIcon-root': {
          color: '#000000b0',
        },
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
    theme: 'light',
  },
}

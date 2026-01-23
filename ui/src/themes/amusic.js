import stylesheet from './amusic.css.js'

export default {
  themeName: 'AMusic',
  typography: {
    fontFamily:
      '-apple-system, BlinkMacSystemFont, Apple Color Emoji, SF Pro, SF Pro Icons, Helvetica Neue, Helvetica, Arial, sans-serif',
    h6: {
      fontSize: '1rem', // AppBar title
    },
    h5: {
      fontSize: '2em',
      fontWeight: '600',
    },
  },
  palette: {
    primary: {
      main: '#ff4e6b',
    },
    secondary: {
      main: '#D60017',
      contrastText: '#eee',
    },
    background: {
      default: '#1a1a1a',
      paper: '#1a1a1a',
    },
    type: 'dark',
  },
  overrides: {
    MuiFormGroup: {
      root: {
        color: 'white',
      },
    },
    MuiAppBar: {
      positionFixed: {
        backgroundColor: '#1d1d1d !important',
        boxShadow: 'none',
        borderBottom: '1px solid #fff1',
      },
      colorSecondary: {
        color: '#eee',
      },
    },
    MuiDrawer: {
      root: {
        background: '#1d1d1d',
        borderRight: '1px solid #fff1',
      },
    },
    MuiToolbar: {
      root: {
        background: 'transparent !important',
      },
    },
    MuiCardMedia: {
      img: {
        borderRadius: '10px',
        boxShadow: '5px 5px 20px #111',
      },
    },
    MuiButton: {
      root: {
        background: '#D60017',
        color: '#fff',
        borderRadius: '6px',
        paddingRight: '0.5rem',
        paddingLeft: '0.5rem',
        marginLeft: '0.5rem',
        marginBottom: '0.5rem',
        textTransform: 'capitalize',
        fontWeight: 600,
      },
      textPrimary: {
        color: '#eee',
      },
      textSecondary: {
        color: '#eee',
        backgroundColor: '#ff4e6b',
      },
      textSizeSmall: {
        fontSize: '0.8rem',
        paddingRight: '0.5rem',
        paddingLeft: '0.5rem',
      },
      label: {
        paddingRight: '1rem',
        paddingLeft: '0.7rem',
      },
    },
    MuiListItemIcon: {
      root: {
        color: '#ff4e6b',
      },
    },
    MuiChip: {
      root: {
        borderRadius: '6px',
      },
    },
    MuiIconButton: {
      root: {
        color: '#ff4e6b',
      },
    },
    MuiTableBody: {
      root: {
        '&>tr:nth-child(odd)': {
          background: 'rgba(255, 255, 255, 0.025)',
        },
      },
    },
    MuiTableRow: {
      root: {
        background: 'transparent',
      },
    },
    MuiTableCell: {
      root: {
        borderBottom: '0 none !important',
        padding: '10px !important',
        color: '#b3b3b3 !important',
      },
      head: {
        color: '#b3b3b3 !important',
      },
    },
    MuiMenuItem: {
      root: {
        fontSize: '0.875rem',
        borderRadius: '10px',
        color: '#eee',
      },
    },
    NDAlbumGridView: {
      albumName: {
        color: '#eee',
      },
      albumPlayButton: {
        color: '#ff4e6b',
      },
      albumArtistName: {
        color: '#ccc',
      },
      cover: {
        borderRadius: '6px',
      },
    },
    NDLogin: {
      systemNameLink: {
        color: '#ff4e6b',
      },
      welcome: {
        color: '#eee',
      },
      card: {
        minWidth: 300,
        backgroundColor: '#1d1d1d',
      },
      icon: {
        filter: 'hue-rotate(115deg)',
      },
    },
    MuiPaper: {
      elevation1: {
        boxShadow: 'none',
      },
      root: {
        color: '#eee',
      },
      rounded: {
        borderRadius: '6px',
      },
    },
    NDMobileArtistDetails: {
      bgContainer: {
        background: '#1a1a1a',
      },
      artistName: {
        fontWeight: '600',
        fontSize: '2em',
      },
    },
    NDDesktopArtistDetails: {
      artistName: {
        fontWeight: '600',
        fontSize: '2em',
      },
      artistDetail: {
        padding: 'unset',
        paddingBottom: '1rem',
      },
    },
    RaDeleteWithConfirmButton: {
      deleteButton: {
        color: '#fff',
      },
    },
    RaBulkDeleteWithUndoButton: {
      deleteButton: {
        color: '#fff',
      },
    },
    RaPaginationActions: {
      currentPageButton: {
        border: '2px solid #D60017',
        background: 'transparent',
      },
      button: {
        border: '2px solid #D60017',
      },
      actions: {
        '@global': {
          '.next-page': {
            border: '0 none',
          },
          '.previous-page': {
            border: '0 none',
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

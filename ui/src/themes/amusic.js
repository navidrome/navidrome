import stylesheet from './amusic.css.js'

export default {
  themeName: 'AMusic',
  typography: {
    fontFamily:
      '-apple-system, BlinkMacSystemFont, Apple Color Emoji, SF Pro, SF Pro Icons, Helvetica Neue, Helvetica, Arial, sans-serif',
    h6: {
      fontSize: '1rem', // AppBar title
    },
  },
  palette: {
    primary: {
      main: '#ff4e6b',
    },
    secondary: {
      main: '#D60017',
      contrastText: '#fff',
    },
    background: {
      default: '#1f1f1f',
      paper: '#1f1f1f',
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
      },
    },
    MuiDrawer: {
      root: {
        background: '#1d1d1d',
        borderRight: '1px solid rgba(255, 255, 255, 0.12)',
      },
    },
    MuiToolbar: {
      root: {
        background: '#1d1d1d',
      },
    },
    MuiCardMedia: {
      img: {
        borderRadius: '10px',
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
        color: '#fff',
      },
      textSecondary: {
        color: '#ff4e6b',
        backgroundColor: '#ff4e6b',
      },
      textSizeSmall:{
        fontSize: '0.8rem',
        paddingRight: '0.5rem',
        paddingLeft: '0.5rem',
      },
      label: {
        color: '#fff',
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
        borderRadius: '10px',
      },
    },
    MuiIconButton: {
      root: {
        color: '#ff4e6b',
      },
    },
    MuiTableBody: {
      root: {
        '&>tr:nth-child(2)': {
          background: 'rgba(255, 255, 255, 0.025)',
        },
      },
    },
    MuiTableRow: {
      root: {
        background: 'transparent',
      },
    },
    MuiTableHead: {
      root: {},
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
      albumSubtitle: {
        color: '#ccc',
      },
      albumPlayButton: {
          color: `#ff4e6b !important`,
      },
    },
    NDLogin: {
      systemNameLink: {
        color: '#D60017',
      },
      icon: {},
      welcome: {
        color: '#eee',
      },
      card: {
        minWidth: 300,
        backgroundColor: '#3a1b22ed',
      },
      avatar: {},
      button: {
        boxShadow: '3px 3px 5px #D60017a3',
      },
    },
    NDMobileArtistDetails: {
      bgContainer: {
        background:
          'linear-gradient(to bottom, rgba(52, 18, 20, 0.72), rgb(48, 20, 22))!important',
      },
    },
  },
  player: {
    theme: 'dark',
    stylesheet,
  },
}

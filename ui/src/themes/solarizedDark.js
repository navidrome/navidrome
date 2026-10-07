import stylesheet from './solarizedDark.css.js'

// Solarized palette by Ethan Schoonover: https://ethanschoonover.com/solarized/
const background = '#002b36' // base03
const surface = '#073642' // base02
const currentLine = '#0d4251' // base02, slightly lighter for hover and headers
const foreground = '#93a1a1' // base1
const comment = '#839496' // base0
const blue = '#268bd2'
const link = '#2aa198' // cyan
const accent = '#2aa198' // cyan, since violet is too dark on base03
const red = '#dc322f'

// For Album, Playlist play button
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
      backgroundColor: `${blue} !important`,
      color: background,
      borderRadius: 500,
      border: 0,
      '&:hover': {
        transform: 'scale(2.1)',
        backgroundColor: `${blue} !important`,
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
        color: foreground,
      },
  },
}

export default {
  themeName: 'Solarized Dark',
  palette: {
    primary: {
      main: blue,
    },
    secondary: {
      main: accent,
      contrastText: foreground,
    },
    error: {
      main: red,
    },
    type: 'dark',
    background: {
      default: background,
      paper: surface,
    },
  },
  overrides: {
    MuiPaper: {
      root: {
        color: foreground,
        backgroundColor: surface,
      },
    },
    MuiAppBar: {
      positionFixed: {
        backgroundColor: `${surface} !important`,
        boxShadow:
          'rgba(15, 17, 21, 0.25) 0px 4px 6px, rgba(15, 17, 21, 0.1) 0px 5px 7px',
      },
    },
    MuiDrawer: {
      root: {
        background: background,
      },
    },
    MuiButton: {
      textPrimary: {
        color: blue,
      },
      textSecondary: {
        color: foreground,
      },
    },
    MuiIconButton: {
      root: {
        color: foreground,
      },
    },
    MuiChip: {
      root: {
        backgroundColor: currentLine,
      },
    },
    MuiFormGroup: {
      root: {
        color: foreground,
      },
    },
    MuiFormLabel: {
      root: {
        color: comment,
        '&$focused': {
          color: blue,
        },
      },
    },
    MuiFormHelperText: {
      error: {
        color: red,
      },
    },
    MuiToolbar: {
      root: {
        backgroundColor: `${surface} !important`,
      },
    },
    MuiOutlinedInput: {
      root: {
        '& $notchedOutline': {
          borderColor: currentLine,
        },
        '&:hover $notchedOutline': {
          borderColor: comment,
        },
        '&$focused $notchedOutline': {
          borderColor: blue,
        },
      },
    },
    MuiFilledInput: {
      root: {
        backgroundColor: currentLine,
        '&:hover': {
          backgroundColor: comment,
        },
        '&$focused': {
          backgroundColor: currentLine,
        },
      },
    },
    MuiTableRow: {
      root: {
        transition: 'background-color .3s ease',
        '&:hover': {
          backgroundColor: `${currentLine} !important`,
        },
      },
    },
    MuiTableHead: {
      root: {
        color: foreground,
        background: surface,
      },
    },
    MuiTableCell: {
      root: {
        color: foreground,
        background: `${surface} !important`,
        borderBottom: `1px solid ${currentLine}`,
      },
      head: {
        color: `${foreground} !important`,
        background: `${currentLine} !important`,
      },
      body: {
        color: `${foreground} !important`,
      },
    },
    NDAlbumGridView: {
      albumName: {
        marginTop: '0.5rem',
        fontWeight: 700,
        color: foreground,
      },
      albumSubtitle: {
        color: comment,
      },
      albumContainer: {
        backgroundColor: surface,
        borderRadius: '8px',
        padding: '.75rem',
        transition: 'background-color .3s ease',
        '&:hover': {
          backgroundColor: currentLine,
        },
      },
      albumPlayButton: {
        backgroundColor: blue,
        borderRadius: '50%',
        boxShadow: '0 8px 8px rgb(0 0 0 / 30%)',
        padding: '0.35rem',
        transition: 'padding .3s ease',
        '&:hover': {
          background: `${blue} !important`,
          padding: '0.45rem',
        },
      },
    },
    NDPlaylistDetails: {
      container: {
        background: `linear-gradient(${currentLine}, transparent)`,
        borderRadius: 0,
        paddingTop: '2.5rem !important',
        boxShadow: 'none',
      },
      title: {
        fontWeight: 700,
        color: foreground,
      },
      details: {
        fontSize: '.875rem',
        color: comment,
      },
    },
    NDAlbumDetails: {
      root: {
        background: `linear-gradient(${currentLine}, transparent)`,
        borderRadius: 0,
        boxShadow: 'none',
      },
      cardContents: {
        alignItems: 'center',
        paddingTop: '1.5rem',
      },
      recordName: {
        fontWeight: 700,
        color: foreground,
      },
      recordArtist: {
        fontSize: '.875rem',
        fontWeight: 700,
        color: accent,
      },
      recordMeta: {
        fontSize: '.875rem',
        color: comment,
      },
    },
    NDCollapsibleComment: {
      commentBlock: {
        fontSize: '.875rem',
        color: comment,
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
        color: foreground,
        fontSize: '0.875rem',
      },
      songTitle: {
        fontWeight: 400,
      },
      songInfo: {
        fontSize: '0.675rem',
        color: comment,
      },
    },
    NDLogin: {
      systemNameLink: {
        color: blue,
      },
      welcome: {
        color: foreground,
      },
      card: {
        minWidth: 300,
        background: surface,
      },
      button: {
        boxShadow: '3px 3px 5px #00212b',
      },
    },
    NDMobileArtistDetails: {
      bgContainer: {
        background: `linear-gradient(to bottom, rgba(0 43 54 / 72%), ${background})!important`,
      },
    },
    RaLayout: {
      content: {
        padding: '0 !important',
        background: background,
      },
      root: {
        backgroundColor: background,
      },
    },
    RaList: {
      content: {
        backgroundColor: background,
      },
    },
    RaListToolbar: {
      toolbar: {
        backgroundColor: background,
        padding: '0 .55rem !important',
      },
    },
    RaSidebar: {
      fixed: {
        backgroundColor: background,
      },
      drawerPaper: {
        backgroundColor: `${background} !important`,
      },
    },
    RaMenuItemLink: {
      root: {
        color: foreground,
        '&[aria-current="page"]': {
          color: `${blue} !important`,
        },
        '&[aria-current="page"] .MuiListItemIcon-root': {
          color: `${blue} !important`,
        },
      },
      active: {
        color: `${blue} !important`,
        '& .MuiListItemIcon-root': {
          color: `${blue} !important`,
        },
      },
    },
    RaLink: {
      link: {
        color: link,
      },
    },
    RaButton: {
      button: {
        margin: '0 5px 0 5px',
      },
    },
    RaPaginationActions: {
      currentPageButton: {
        border: `2px solid ${blue}`,
      },
      button: {
        backgroundColor: currentLine,
        minWidth: 48,
        margin: '0 4px',
      },
    },
  },
  player: {
    theme: 'dark',
    stylesheet,
  },
}

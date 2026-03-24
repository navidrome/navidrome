import stylesheet from './dracula.css.js'

// Dracula color palette
const background = '#282a36'
const currentLine = '#44475a'
const foreground = '#f8f8f2'
const comment = '#6272a4'
const cyan = '#8be9fd'
const green = '#50fa7b'
const pink = '#ff79c6'
const purple = '#bd93f9'
const orange = '#ffb86c'
const red = '#ff5555'
const yellow = '#f1fa8c'

// Darker shade for surfaces
const surface = '#21222c'

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
      backgroundColor: `${green} !important`,
      color: background,
      borderRadius: 500,
      border: 0,
      '&:hover': {
        transform: 'scale(2.1)',
        backgroundColor: `${green} !important`,
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
  themeName: 'Dracula',
  palette: {
    primary: {
      main: purple,
    },
    secondary: {
      main: currentLine,
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
        backgroundColor: `${currentLine} !important`,
        boxShadow:
          'rgba(20, 21, 28, 0.25) 0px 4px 6px, rgba(20, 21, 28, 0.1) 0px 5px 7px',
      },
    },
    MuiDrawer: {
      root: {
        background: background,
      },
    },
    MuiButton: {
      textPrimary: {
        color: purple,
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
          color: purple,
        },
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
          borderColor: purple,
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
        color: `${yellow} !important`,
        background: `${currentLine} !important`,
      },
      body: {
        color: `${foreground} !important`,
      },
    },
    MuiSwitch: {
      colorSecondary: {
        '&$checked': {
          color: green,
        },
        '&$checked + $track': {
          backgroundColor: green,
        },
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
        backgroundColor: green,
        borderRadius: '50%',
        boxShadow: '0 8px 8px rgb(0 0 0 / 30%)',
        padding: '0.35rem',
        transition: 'padding .3s ease',
        '&:hover': {
          background: `${green} !important`,
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
        color: pink,
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
        color: purple,
      },
      welcome: {
        color: foreground,
      },
      card: {
        minWidth: 300,
        background: background,
      },
      button: {
        boxShadow: '3px 3px 5px #191a21',
      },
    },
    NDMobileArtistDetails: {
      bgContainer: {
        background: `linear-gradient(to bottom, rgba(40 42 54 / 72%), ${surface})!important`,
      },
    },
    RaLayout: {
      content: {
        padding: '0 !important',
        background: surface,
      },
      root: {
        backgroundColor: background,
      },
    },
    RaList: {
      content: {
        backgroundColor: surface,
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
    MuiTableSortLabel: {
      root: {
        color: `${yellow} !important`,
        '&:hover': {
          color: `${orange} !important`,
        },
        '&$active': {
          color: `${orange} !important`,
          '&& $icon': {
            color: `${orange} !important`,
          },
        },
      },
    },
    RaMenuItemLink: {
      root: {
        color: foreground,
        '&[aria-current="page"]': {
          color: `${pink} !important`,
        },
        '&[aria-current="page"] .MuiListItemIcon-root': {
          color: `${pink} !important`,
        },
      },
      active: {
        color: `${pink} !important`,
        '& .MuiListItemIcon-root': {
          color: `${pink} !important`,
        },
      },
    },
    RaLink: {
      link: {
        color: cyan,
      },
    },
    RaButton: {
      button: {
        margin: '0 5px 0 5px',
      },
    },
    RaPaginationActions: {
      currentPageButton: {
        border: `2px solid ${purple}`,
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

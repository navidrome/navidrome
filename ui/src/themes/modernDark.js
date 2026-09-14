const modernColours = {
  lighter: '#a6ccff',
  main: '#0084ff',
  darker: '#0062f6',
  mainBackground: '#14172e',
  lightBackground: '#181d37',
  lighterBackground: '#222541',
  darkerBackgroundHighlight: '#32375b',
  backgroundHighlight: '#464b77',
}

// For Album, Playlist
const musicListActions = {
  padding: '1rem 0',
  alignItems: 'center',
  '@global': {
    button: {
      margin: 5,
      border: '1px solid transparent',
      backgroundColor: 'inherit',
      color: 'rgba(255, 255, 255, 0.8)',
      '&:hover': {
        border: '1px solid rgba(255, 255, 255, 0.8)',
        backgroundColor: 'inherit !important',
      },
    },
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
      background: modernColours['main'],
      color: '#fff',
      borderRadius: 500,
      border: 0,
      '&:hover': {
        transform: 'scale(2.1)',
        backgroundColor: `${modernColours['main']} !important`,
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
  themeName: 'Modern Dark',
  typography: {
    fontFamily: "system-ui, 'Helvetica Neue', Helvetica, Arial",
    h6: {
      fontSize: '1rem', // AppBar title
    },
  },
  palette: {
    primary: {
      light: modernColours['lighter'],
      main: modernColours['main'],
    },
    secondary: {
      main: '#fff',
      contrastText: '#fff',
    },
    background: {
      default: modernColours['mainBackground'],
      paper: modernColours['mainBackground'],
    },
    type: 'dark',
  },
  overrides: {
    MuiFormGroup: {
      root: {
        color: modernColours['main'],
      },
    },
    MuiMenuItem: {
      root: {
        fontSize: '0.875rem',
        paddingTop: '4px',
        paddingBottom: '4px',
        margin: '8px',
        minHeight: '2.25rem !important',
        borderRadius: '5rem',
      },
    },
    MuiDivider: {
      root: {
        margin: '1rem .75rem',
      },
    },
    MuiButton: {
      root: {
        background: 'rgba(0, 0, 0, 0)',
        color: '#fff',
        border: '1px solid transparent',
        borderRadius: '5rem',
        '&:hover': {
          background: modernColours['backgroundHighlight'],
        },
      },
      textSecondary: {
        border: '1px solid rgba(255, 255, 255, 0.8)',
        background: modernColours['mainBackground'],
        '&:hover': {
          border: '1px solid #fff !important',
          background: `${modernColours['mainBackground']} !important`,
        },
      },
      label: {
        color: '#fff',
        paddingRight: '1rem',
        paddingLeft: '0.8rem',
      },
      contained: {
        boxShadow: 'unset !important',
      },
    },
    MuiDrawer: {
      root: {
        background: modernColours['mainBackground'],
      },
      paper: {
        height: 'unset',
      },
    },
    MuiTableHead: {
      root: {
        boxShadow: 'none !important',
        zIndex: '1',
      },
    },
    MuiTableRow: {
      root: {
        padding: '10px 0',
        transition: 'background-color .3s ease',
        '&:hover': {
          backgroundColor: `${modernColours['lightBackground']} !important`,
          borderRadius: '.625rem !important',
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
        borderBottom: 'none',
        padding: '10px !important',
        color: 'rgba(255, 255, 255, 0.8) !important',
      },
      head: {
        borderBottom: 'none',
        fontSize: '0.75rem',
        textTransform: 'uppercase',
        letterSpacing: 1.2,
      },
    },
    MuiTablePagination: {
      toolbar: {
        borderTop: `1px solid ${modernColours['mainBackground']}`,
      },
    },
    MuiAppBar: {
      positionFixed: {
        boxShadow: 'unset',
      },
      colorSecondary: {
        backgroundColor: 'unset',
      },
    },
    MuiPaper: {
      rounded: {
        borderRadius: '.625rem',
      },
      elevation1: {
        boxShadow: '0px',
      },
    },
    MuiOutlinedInput: {
      root: {
        borderRadius: '5rem',
        backgroundColor: modernColours['lighterBackground'],
      },
      notchedOutline: {
        borderColor: 'transparent',
        transition: 'border-color .1s',
      },
      adornedEnd: {
        backgroundColor: modernColours['darkerBackgroundHighlight'],
      },
    },
    PrivateNotchedOutline: {
      legendLabelled: {
        width: '0px',
      },
    },
    MuiInputLabel: {
      outlined: {
        transform: 'translate(0px, -24px) scale(0.9) !important',
      },
    },
    MuiFormControl: {
      marginDense: {
        marginTop: '24px',
        marginBottom: '0px',
      },
    },
    MuiFormHelperText: {
      marginDense: {
        marginTop: '0px',
        marginBottom: '0px',
      },
    },
    MuiAutocomplete: {
      inputRoot: {
        '&[class*="MuiOutlinedInput-root"]': {
          padding: '2px 9px 2px 9px',
        },
      },
      tag: {
        marginLeft: '-3px',
        marginRight: '9px',
      },
    },
    MuiChip: {
      root: {
        backgroundColor: modernColours['backgroundHighlight'],
      },
      outlined: {
        backgroundColor: 'rgba(255, 255, 255, 0.8)',
        color: modernColours['lightBackground'],
      },
    },
    MuiCard: {
      root: {
        margin: '1rem',
      },
    },
    MuiSelect: {
      select: {
        '&:focus': {
          backgroundColor: '#3c455f',
          borderRadius: '5rem',
        },
      },
    },
    MuiListItem: {
      button: {
        transition: 'background-color .3s ease !important',
        '&:hover': {
          backgroundColor: modernColours['lighterBackground'],
        },
      },
      root: {
        '&.Mui-selected': {
          backgroundColor: `${modernColours['backgroundHighlight']} !important`,
        },
      },
    },
    MuiSwitch: {
      switchBase: {
        color: modernColours['lighterBackground'],
      },
      track: {
        width: '84%',
        opacity: '1',
        backgroundColor: modernColours['backgroundHighlight'],
        scale: '140%',
        transform: 'translateX(.1rem)',
      },
      thumb: {
        scale: '60%',
        boxShadow: 'none',
      },
    },
    MuiToolbar: {
      gutters: {
        '@media (min-width: 600px)': {
          paddingLeft: '16px',
          paddingRight: '16px',
        },
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
        color: 'rgba(255, 255, 255, 0.8)',
      },
      albumContainer: {
        backgroundColor: modernColours['mainBackground'],
        borderRadius: '.625rem',
        padding: '.75rem',
        transition: 'background-color .3s ease',
        '&:hover': {
          backgroundColor: modernColours['backgroundHighlight'],
        },
      },
      albumPlayButton: {
        backgroundColor: modernColours['main'],
        borderRadius: '50%',
        boxShadow: '0 8px 8px rgb(0 0 0 / 30%)',
        padding: '0.35rem',
        transition: 'padding .3s ease',
        '&:hover': {
          background: `${modernColours['main']} !important`,
          padding: '0.45rem',
        },
      },
    },
    NDPlaylistDetails: {
      container: {
        background: modernColours['lighterBackground'],
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
        background: modernColours['lighterBackground'],
        borderRadius: 0,
        boxShadow: 'none',
        '@media (min-width: 600px)': {
          paddingLeft: 0,
        },
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
      },
      songTitle: {
        fontWeight: 400,
      },
      songInfo: {
        fontSize: '0.675rem',
        color: 'rgba(255, 255, 255, 0.8)',
      },
      player: {
        border: '10px solid blue',
      },
    },
    NDLogin: {
      main: {
        boxShadow: 'none',
      },
      systemNameLink: {
        color: '#fff',
      },
      card: {
        border: 'none',
        borderRadius: '1rem',
        boxShadow:
          '0px 5px 5px -3px rgba(0,0,0,0.2),0px 8px 10px 1px rgba(0,0,0,0.14),0px 3px 14px 2px rgba(0,0,0,0.12)',
        marginTop: '0',
        position: 'absolute',
        top: '50%',
        transform: 'translateY(-50%)',
      },
      avatar: {
        marginBottom: 0,
      },
      icon: {
        borderRadius: '50%',
        boxShadow:
          '0px 5px 5px -3px rgba(0,0,0,0.2),0px 8px 10px 1px rgba(0,0,0,0.14),0px 3px 14px 2px rgba(0,0,0,0.12)',
      },
    },
    NDSubMenu: {
      sidebarIsClosed: {
        '& a': {
          paddingLeft: '10px',
        },
      },
      icon: {
        marginLeft: '-9px',
      },
    },
    RaMenuItemLink: {
      root: {
        paddingLeft: '10px',
      },
      icon: {
        marginLeft: '-3px',
      },
    },
    RaLayout: {
      content: {
        padding: '0 !important',
        background: `${modernColours['lighterBackground']} !important`,
        '&::before': {
          // Very bodgy!!
          content: '""',
          backgroundColor: modernColours['lighterBackground'],
          color: modernColours['lighterBackground'],
          height: '48px',
          marginTop: '-48px',
          marginLeft: '0px',
          position: 'sticky',
          top: '0',
          zIndex: '999',
          borderBottom: `1px solid ${modernColours['mainBackground']}`,
        },
      },
      root: {
        background: `${modernColours['lighterBackground']} !important`,
      },
      appFrame: {
        '@media (min-width: 0px)': {
          marginTop: '48px',
        },
      },
    },
    RaList: {
      content: {
        backgroundColor: 'inherit',
        border: `1px solid ${modernColours['mainBackground']}`,
      },
    },
    RaDatagrid: {
      headerRow: {
        '&:hover': {
          backgroundColor: 'transparent !important',
        },
      },
      headerCell: {
        '&:first-child': {
          borderTopLeftRadius: '.625rem !important',
        },
        '&:last-child': {
          borderTopRightRadius: '.625rem !important',
        },
      },
    },
    RaListToolbar: {
      toolbar: {
        padding: '0 .55rem !important',
        zIndex: '2',
      },
    },
    RaSearchInput: {
      input: {
        paddingLeft: '.5rem',
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
        border: 'none',
        borderRadius: 500,
        backgroundColor: modernColours['backgroundHighlight'],
      },
      button: {
        minWidth: 48,
        margin: '0 4px',
        border: 'none',
        borderRadius: 500,
        backgroundColor: modernColours['lightBackground'],
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
    RaBulkActionsToolbar: {
      toolbar: {
        backgroundColor: modernColours['backgroundHighlight'],
        borderRadius: '.625rem',
      },
    },
    RaToolbar: {
      toolbar: {
        backgroundColor: modernColours['mainBackground'],
      },
    },
    RaSidebar: {
      root: {
        height: 'initial',
        marginTop: '-48px',
      },
      drawerPaper: {
        marginTop: '48px',
      },
    },
    RaAppBar: {
      title: {
        visibility: 'hidden',
      },
      menuButton: {
        marginLeft: '0.125em',
      },
    },
    makeStyles: {
      cover: {
        '& > $item': {
          borderRadius: '.5rem',
        },
      },
    },
  },
  player: {
    theme: 'dark',
    stylesheet: require('./modernDark.css.js'),
  },
}

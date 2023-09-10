const bLight = {
  300: '#0054df',
  500: '#ffffff',
}
const musicListActions = {
  padding: '1rem 0',
  alignItems: 'center',
  '@global': {
    button: {
      margin: 5,
      border: '1px solid #cccccc',
      backgroundColor: '#fff',
      color: '#b3b3b3',
      '&:hover': {
        border: '1px solid #224bff',
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
      background: bLight['500'],
      color: '#fff',
      borderRadius: 500,
      border: 0,
      '&:hover': {
        transform: 'scale(2.1)',
        backgroundColor: `${bLight['500']} !important`,
        border: 0,
        boxShadow: '0px 0px 4px 0px #5656567d',
      },
    },
    'button:only-child': {
      margin: '1.5rem',
    },
    'button:first-child>span:first-child': {
      padding: 0,
      color: bLight['300'],
    },
    'button:first-child>span:first-child>span': {
      display: 'none',
    },
    'button>span:first-child>span, button:not(:first-child)>span:first-child>svg':
      {
        color: '#656565',
      },
  },
}

export default {
  themeName: 'Ligera',
  palette: {
    primary: {
      light: bLight['300'],
      main: '#464646',
    },
    secondary: {
      main: '#000',
      contrastText: '#fff',
    },
    background: {
      default: '#f0f2f5',
      paper: 'inherit',
    },
    text: {
      secondary: '#232323',
    },
  },
  typography: {
    fontFamily: "system-ui, 'Helvetica Neue', Helvetica, Arial",
    h6: {
      fontSize: '1rem',
    },
  },
  overrides: {
    MuiAutocomplete: {
      popper: {
        background: bLight['500'],
      },
    },
    MuiCard: {
      root: {
        marginLeft: '1%',
        marginRight: '1%',
        background: bLight['500'],
      },
    },
    MuiPopover: {
      paper: {
        backgroundColor: bLight['500'],
        '& .MuiListItemIcon-root': {
          color: '#656565',
        },
      },
    },
    MuiTypography: {
      colorTextSecondary: {
        color: '#0a0a0a',
      },
    },
    MuiDialog: {
      paper: {
        backgroundColor: bLight['500'],
      },
    },
    MuiFormGroup: {
      root: {
        color: '#464646',
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
        color: '#91b1b0',
      },
    },
    MuiCheckbox: {
      root: {
        color: '#616161',
      },
    },
    MuiIconButton: {
      label: {},
    },
    MuiButton: {
      root: {
        background: '#fff',
        color: '#000',
        border: '1px solid transparent',
        borderRadius: 500,
        '&:hover': {
          background: `${bLight['300']} !important`,
          color: '#fff',
        },
      },
      containedPrimary: {
        backgroundColor: '#fff',
      },
      textPrimary: {
        backgroundColor: bLight['300'],
        '& span': {
          color: '#fff',
        },
        '&:hover': {
          backgroundColor: '#3079ff !important',
        },
      },
      textSecondary: {
        border: '1px solid #b3b3b3',
        background: '#fff',
        '&:hover': {
          border: '1px solid #fff !important',
          background: '#dedede !important',
        },
      },
      label: {
        color: '#000',
        paddingRight: '1rem',
        paddingLeft: '0.7rem',
      },
    },
    MuiDrawer: {
      root: {
        background: bLight['500'],
        paddingTop: '10px',
        boxShadow: '-14px -7px 20px black',
      },
    },
    MuiTableRow: {
      root: {
        padding: '10px 0',
        transition: 'background-color .3s ease',
        '&:hover': {
          backgroundColor: '#e4e4e4 !important',
        },
        '@global': {
          'td:nth-child(4)': {
            color: '#3c3c3c !important',
          },
        },
      },
      head: {
        backgroundColor: '#e0efff',
      },
    },
    MuiTableCell: {
      root: {
        borderBottom: '1px solid #1d1d1d',
        padding: '10px !important',
        color: '#656565 !important',
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
        background: `${bLight['500']} !important`,
        boxShadow: '13px -12px 20px 0px #000',
      },
      colorSecondary: {
        color: bLight['300'],
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
        backgroundColor: '#e0efff7d',
        borderRadius: '.5rem',
        padding: '.75rem',
        transition: 'background-color .3s ease',
        '&:hover': {
          backgroundColor: '#c6dbff',
        },
      },
      albumPlayButton: {
        backgroundColor: bLight['500'],
        borderRadius: '50%',
        boxShadow: '0 8px 8px rgb(0 0 0 / 30%)',
        padding: '0.35rem',
        transition: 'padding .3s ease',
        color: bLight['300'],
        '&:hover': {
          background: `${bLight['300']} !important`,
          padding: '0.45rem',
          color: bLight['500'],
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
        color: '#000000b0',
      },
      details: {
        fontSize: '.875rem',
        color: 'rgba(255,255,255, 0.8)',
      },
    },
    NDAlbumDetails: {
      root: {
        borderRadius: 0,
        boxShadow: '-1px 1px 6px 0px #00000057',
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
        color: 'rgb(113 113 113 / 80%)',
      },
      commentBlock: {
        fontSize: '.875rem',
        color: 'rgb(113 113 113 / 80%)',
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
        color: '#656565',
      },
    },
    NDAudioPlayer: {
      audioTitle: {
        color: '#000',
        fontSize: '0.875rem',
      },
      songTitle: {
        fontWeight: 400,
      },
      songInfo: {
        fontSize: '0.675rem',
        color: '#b3b3b3',
      },
      player: {},
    },
    NDLogin: {
      actions: {
        '& button': {
          backgroundColor: '#3c9cff',
        },
      },
      systemNameLink: {
        textDecoration: 'none',
        color: bLight['300'],
      },
      systemName: {
        marginTop: '0.5em',
        marginBottom: '1em',
      },
      icon: {
        backgroundColor: 'transparent',
        width: '100px',
        height: '100px',
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
    },
    NDMobileArtistDetails: {
      bgContainer: {
        background:
          'linear-gradient(to bottom, rgb(255 255 255 / 51%), rgb(240 242 245))!important',
      },
    },
    RaLayout: {
      content: {
        padding: '0 !important',
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
          color: '#0f0f0f',
          backgroundColor: '#fff',
          '& span': {
            color: '#101010',
          },
          '&:hover': {
            backgroundColor: '#dedede !important',
          },
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
        color: '#287eff',
      },
    },
    RaLogout: {
      icon: {
        color: '#f90000!important',
      },
    },
    RaMenuItemLink: {
      root: {
        color: '#232323 !important',
        '& .MuiListItemIcon-root': {
          color: '#656565',
        },
      },
      active: {
        backgroundColor: '#44a0ff1f',
        color: '#232323 !important',
        '& .MuiListItemIcon-root': {
          color: '#0066ff',
        },
      },
    },
    RaSidebar: {
      root: {
        height: 'initial',
      },
      drawerPaper: {
        '@media (min-width: 0px) and (max-width: 599.95px)': {
          backgroundColor: `${bLight['500']} !important`,
        },
      },
    },
    RaBulkActionsToolbar: {
      toolbar: {
        backgroundColor: bLight['500'],
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

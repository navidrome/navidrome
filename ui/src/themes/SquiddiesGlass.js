import stylesheet from './SquiddiesGlass.css.js'

/**
 * Color constants used throughout the Squiddies Glass theme.
 * Provides a consistent color palette with pink, gray, purple, and basic colors.
 * @type {Object}
 */
const colors = {
  pink: {
    100: '#fbe3f4',
    200: '#f5b9e3',
    300: '#ec7cd6',
    400: '#e14ac2',
    500: '#c231ab', // base
    600: '#a31a92',
    700: '#8b0f7e',
    800: '#7a006d',
    900: '#670066',
  },
  gray: {
    50: '#c2c1c2',
    100: '#b3b3b3', // light gray
    200: '#282828', // medium dark
    300: '#1d1d1d', // darker
    400: '#181818', // even darker
    500: '#171717', // darkest
  },
  purple: {
    400: '#524590',
    500: '#4d3249',
    600: '#6d1c5e',
  },
  black: '#000',
  white: '#fff',
  dark: '#121212',
}

/**
 * Shared style object for music list action buttons.
 * Defines common styling for buttons in music lists, including hover effects and responsive scaling.
 * @type {Object}
 */
const musicListActions = {
  padding: '1rem 0',
  alignItems: 'center',
  '@global': {
    button: {
      border: '1px solid transparent',
      backgroundColor: 'inherit',
      color: colors.gray[100],
      '&:hover': {
        border: `1px solid ${colors.gray[100]}`,
        backgroundColor: 'inherit !important',
      },
    },
    'button:first-child:not(:only-child)': {
      '@media screen and (max-width: 720px)': {
        transform: 'scale(1.3)',
        margin: '1em',
        '&:hover': {
          transform: 'scale(1.2) !important',
        },
      },
      transform: 'scale(1.3)',
      margin: '1em',
      minWidth: 0,
      padding: 5,
      transition: 'transform .3s ease',
      background: colors.pink[500],
      color: `${colors.black} !important`,
      borderRadius: 500,
      border: 0,
      '&:hover': {
        transform: 'scale(1.2)',
        backgroundColor: `${colors.pink[500]} !important`,
        border: 0,
      },
    },
    'button:only-child': {
      marginTop: '0.3em',
    },
    'button:first-child>span:first-child': {
      padding: 0,
      color: `${colors.black} !important`,
    },
    'button:first-child>span:first-child>span': {
      display: 'none',
    },
    'button>span:first-child>span, button:not(:first-child)>span:first-child>svg':
      {
        color: colors.gray[100],
      },
  },
}

/**
 * Squiddies Glass theme configuration object.
 * Defines the complete theme structure including typography, palette, component overrides, and player settings.
 * @type {Object}
 */
export default {
  /**
   * The name of the theme.
   * @type {string}
   */
  themeName: 'Squiddies Glass',

  /**
   * Typography settings for the theme.
   * Specifies font family and heading sizes.
   * @type {Object}
   */
  typography: {
    fontFamily: "system-ui, 'Helvetica Neue', Helvetica, Arial, sans-serif",
    h6: {
      fontSize: '1rem', // AppBar title
    },
  },

  /**
   * Color palette configuration.
   * Defines primary, secondary, and background colors for the theme.
   * @type {Object}
   */
  palette: {
    primary: {
      light: colors.pink[300],
      main: colors.pink[500],
    },
    secondary: {
      main: colors.white,
      contrastText: colors.white,
    },
    background: {
      default: colors.dark,
      paper: colors.dark,
    },
    type: 'dark',
  },

  /**
   * Component overrides for Material-UI and custom Navidrome components.
   * Customizes the appearance and behavior of various UI components.
   * @type {Object}
   */
  overrides: {
    // Material-UI Components
    MuiAppBar: {
      positionFixed: {
        backgroundColor: `${colors.black} !important`,
        boxShadow: 'none',
      },
    },
    MuiButton: {
      root: {
        background: colors.pink[500],
        color: colors.white,
        border: '1px solid transparent',
        borderRadius: 500,
        '&:hover': {
          background: `${colors.pink[900]} !important`,
        },
      },
      textSecondary: {
        border: `1px solid ${colors.gray[100]}`,
        background: colors.black,
        '&:hover': {
          border: `1px solid ${colors.white} !important`,
          background: `${colors.black} !important`,
        },
      },
      label: {
        color: colors.white,
        paddingRight: '1rem',
        paddingLeft: '0.7rem',
      },
    },
    MuiCardMedia: {
      root: {
        boxShadow: `0 2px 32px rgba(0,0,0,0.5), 0px 1px 5px rgba(0,0,0,0.1)`,
      },
    },
    MuiDivider: {
      root: {
        margin: '.75rem 0',
      },
    },
    MuiDrawer: {
      root: {
        background: colors.gray[500],
        paddingTop: '10px',
      },
    },
    MuiFormGroup: {
      root: {
        color: colors.pink[500],
      },
    },
    MuiMenuItem: {
      root: {
        fontSize: '0.875rem',
      },
    },
    MuiTableCell: {
      root: {
        borderBottom: `1px solid ${colors.gray[300]}`,
        padding: '10px !important',
        color: `${colors.gray[100]} !important`,
        '& img': {
          filter:
            'brightness(0) saturate(100%) invert(36%) sepia(93%) saturate(7463%) hue-rotate(289deg) brightness(95%) contrast(102%);',
        },
        '& img + span': {
          color: colors.pink[500],
        },
      },
      head: {
        borderBottom: `1px solid ${colors.gray[200]}`,
        fontSize: '0.75rem',
        textTransform: 'uppercase',
        letterSpacing: 1.2,
      },
    },
    MuiTableRow: {
      root: {
        padding: '10px 0',
        transition: 'background-color .3s ease',
        '&:hover': {
          backgroundColor: `${colors.gray[300]} !important`,
        },
        '@global': {
          'td:nth-child(4)': {
            color: `${colors.white} !important`,
          },
        },
      },
    },

    // React Admin Components
    RaBulkActionsToolbar: {
      topToolbar: {
        gap: '8px',
      },
    },
    RaFilter: {
      form: {
        '& .MuiOutlinedInput-input:-webkit-autofill': {
          '-webkit-box-shadow': `0 0 0 100px ${colors.gray[50]} inset`,
          '-webkit-text-fill-color': colors.white,
        },
      },
    },
    RaFilterButton: {
      root: {
        marginRight: '1rem',
      },
    },
    RaLayout: {
      content: {
        padding: '0 !important',
        background: `linear-gradient(${colors.dark}, ${colors.gray[500]})`,
      },
      contentWithSidebar: {
        gap: '2px',
      },
    },
    RaList: {
      content: {
        backgroundColor: 'inherit',
      },
      bulkActionsDisplayed: {
        marginTop: '-20px',
      },
    },
    RaListToolbar: {
      toolbar: {
        padding: '0 .55rem !important',
      },
    },
    RaPaginationActions: {
      currentPageButton: {
        border: `1px solid ${colors.gray[100]}`,
      },
      button: {
        backgroundColor: 'inherit',
        minWidth: 48,
        margin: '0 4px',
        border: `1px solid ${colors.gray[200]}`,
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
    RaSearchInput: {
      input: {
        paddingLeft: '.9rem',
        border: 0,
        '& .MuiInputBase-root': {
          backgroundColor: `${colors.white} !important`,
          borderRadius: '20px !important',
          color: colors.black,
          border: '0px',
          '& fieldset': {
            borderColor: colors.white,
          },
          '&:hover fieldset': {
            borderColor: colors.white,
          },
          '&.Mui-focused fieldset': {
            borderColor: colors.white,
          },
          '& svg': {
            color: `${colors.black} !important`,
          },
          '& .MuiOutlinedInput-input:-webkit-autofill': {
            borderRadius: '20px 0px 0px 20px',
            '-webkit-box-shadow': `0 0 0 100px ${colors.gray[50]} inset`,
            '-webkit-text-fill-color': colors.black,
          },
        },
      },
    },
    RaSidebar: {
      root: {
        height: 'initial',
        borderTopRightRadius: '8px',
        borderTopLeftRadius: '8px',
      },
    },

    // Navidrome Custom Components
    NDAlbumDetails: {
      root: {
        borderTopRightRadius: '8px',
        borderTopLeftRadius: '8px',
        boxShadow: 'none',
        background: `linear-gradient(45deg, ${colors.purple[500]}, ${colors.purple[400]}, ${colors.purple[600]})`,
        backgroundSize: '200% 200%',
        animation: 'gradientFlow 8s ease-in-out infinite',
        position: 'relative',
        '&:before': {
          content: '""',
          position: 'absolute',
          top: '0',
          left: '0',
          width: '100%',
          height: '100%',
          background: `linear-gradient(to bottom, transparent, ${colors.dark})`,
        },
      },
      cardContents: {
        alignItems: 'flex-start',
      },
      coverParent: {
        zIndex: '99999',
      },
      details: {
        zIndex: '99999',
      },
      recordName: {
        fontSize: 'calc(1rem + 1.5vw)',
        fontWeight: 900,
      },
      recordArtist: {
        fontSize: '.875rem',
        fontWeight: 700,
      },
      recordMeta: {
        fontSize: '.875rem',
        color: `rgba(${colors.white}, 0.8)`,
      },
      content: {
        paddingBottom: '0px !important',
        paddingTop: '0px',
      },
    },
    NDAlbumGridView: {
      albumName: {
        marginTop: '0.5rem',
        fontWeight: 700,
        textTransform: 'none',
        color: colors.white,
      },
      albumSubtitle: {
        color: colors.gray[100],
      },
      albumContainer: {
        backgroundColor: colors.gray[400],
        borderRadius: '.5rem',
        padding: '.75rem',
        transition: 'background-color .3s ease',
        '&:hover': {
          backgroundColor: colors.gray[200],
        },
      },
      albumPlayButton: {
        backgroundColor: colors.pink[500],
        borderRadius: '50%',
        boxShadow: '0 8px 8px rgb(0 0 0 / 30%)',
        padding: '0.35rem',
        transition: 'padding .3s ease',
        '&:hover': {
          background: `${colors.pink[500]} !important`,
          padding: '0.45rem',
        },
      },
    },
    NDAlbumShow: {
      albumActions: musicListActions,
    },
    NDArtistShow: {
      actions: {
        padding: '2rem 0',
        alignItems: 'center',
        overflow: 'visible',
        minHeight: '120px',
        '@global': {
          button: {
            border: '1px solid transparent',
            backgroundColor: 'inherit',
            color: colors.gray[100],
            margin: '0 0.5rem',
            '&:hover': {
              border: `1px solid ${colors.gray[100]}`,
              backgroundColor: 'inherit !important',
            },
          },
          // Hide shuffle button label (first button)
          'button:first-child>span:first-child>span': {
            display: 'none',
          },
          // Style shuffle button (first button)
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
            background: colors.pink[500],
            color: colors.white,
            borderRadius: 500,
            border: 0,
            '&:hover': {
              transform: 'scale(2.1)',
              backgroundColor: `${colors.pink[500]} !important`,
              border: 0,
            },
          },
          'button:first-child>span:first-child': {
            padding: 0,
            color: `${colors.black} !important`,
          },
          'button>span:first-child>span, button:not(:first-child)>span:first-child>svg':
            {
              color: colors.gray[100],
            },
        },
      },
      actionsContainer: {
        overflow: 'visible',
      },
    },
    NDAudioPlayer: {
      audioTitle: {
        color: colors.white,
        fontSize: '1.5rem',
        '& span:nth-child(3)': {
          fontSize: '0.8rem',
        },
      },
      songTitle: {
        fontWeight: 900,
      },
      songInfo: {
        fontSize: '0.9rem',
        color: colors.gray[100],
      },
    },
    NDCollapsibleComment: {
      commentBlock: {
        fontSize: '.875rem',
        color: `rgba(${colors.white}, 0.8)`,
      },
    },
    NDLogin: {
      main: {
        boxShadow: `inset 0 0 0 2000px rgba(${colors.black}, .75)`,
      },
      systemNameLink: {
        color: colors.white,
      },
      card: {
        border: `1px solid ${colors.gray[200]}`,
      },
      avatar: {
        marginBottom: 0,
      },
    },
    NDPlaylistDetails: {
      container: {
        background: `linear-gradient(${colors.gray[300]}, transparent)`,
        borderRadius: 0,
        paddingTop: '2.5rem !important',
        boxShadow: 'none',
      },
      title: {
        fontSize: 'calc(1.5rem + 1.5vw)',
        fontWeight: 700,
        color: colors.white,
      },
      details: {
        fontSize: '.875rem',
        color: `rgba(${colors.white}, 0.8)`,
      },
    },
    NDPlaylistShow: {
      playlistActions: musicListActions,
    },
  },

  /**
   * Player configuration settings.
   * Specifies the player theme and associated stylesheet.
   * @type {Object}
   */
  player: {
    theme: 'dark',
    stylesheet,
  },
}

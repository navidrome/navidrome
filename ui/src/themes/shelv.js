/**
 * Shelv Theme for Navidrome
 * A standalone dark theme with a modern violet interface.
 */

const ShelvTheme = {
  themeName: 'Shelv',
  palette: {
    type: 'dark',
    primary: { main: '#7C3AED', contrastText: '#FFFFFF' },
    secondary: { main: '#8B5CF6', contrastText: '#FFFFFF' },
    background: { default: '#000000', paper: '#1B1720' },
    text: { primary: '#F7F5FA', secondary: '#A9A4B0' },
    action: {
      active: '#8B5CF6',
      hover: 'rgba(124, 58, 237, 0.16)',
      selected: 'rgba(124, 58, 237, 0.24)',
    },
    divider: 'rgba(165, 125, 190, 0.16)',
  },
  typography: {
    fontFamily:
      '-apple-system, BlinkMacSystemFont, "SF Pro Display", "SF Pro Text", "Helvetica Neue", Arial, sans-serif',
    h1: {
      fontFamily:
        '-apple-system, BlinkMacSystemFont, "SF Pro Display", "SF Pro Text", "Helvetica Neue", Arial, sans-serif',
      fontWeight: 700,
      letterSpacing: '-0.02em',
    },
    h2: {
      fontFamily:
        '-apple-system, BlinkMacSystemFont, "SF Pro Display", "SF Pro Text", "Helvetica Neue", Arial, sans-serif',
      fontWeight: 700,
      letterSpacing: '-0.02em',
    },
    h3: {
      fontFamily:
        '-apple-system, BlinkMacSystemFont, "SF Pro Display", "SF Pro Text", "Helvetica Neue", Arial, sans-serif',
      fontWeight: 650,
      letterSpacing: '-0.01em',
    },
    h4: {
      fontFamily:
        '-apple-system, BlinkMacSystemFont, "SF Pro Display", "SF Pro Text", "Helvetica Neue", Arial, sans-serif',
      fontWeight: 650,
    },
    h5: {
      fontFamily:
        '-apple-system, BlinkMacSystemFont, "SF Pro Display", "SF Pro Text", "Helvetica Neue", Arial, sans-serif',
      fontWeight: 600,
    },
    h6: {
      fontFamily:
        '-apple-system, BlinkMacSystemFont, "SF Pro Display", "SF Pro Text", "Helvetica Neue", Arial, sans-serif',
      fontWeight: 600,
    },
    subtitle1: { fontWeight: 500 },
    subtitle2: { fontWeight: 500 },
    body1: { fontWeight: 400 },
    body2: { fontWeight: 400 },
    button: {
      fontWeight: 600,
      textTransform: 'none',
      fontFamily:
        '-apple-system, BlinkMacSystemFont, "SF Pro Display", "SF Pro Text", "Helvetica Neue", Arial, sans-serif',
      letterSpacing: '-0.01em',
    },
  },
  shape: { borderRadius: 10 },
  overrides: {
    MuiCssBaseline: {
      '@global': {
        body: { backgroundColor: '#000000' },
        '*': { scrollbarColor: '#27202D transparent' },
      },
    },
    MuiAppBar: {
      root: {
        boxShadow: 'none',
        borderBottom: '1px solid rgba(165, 125, 190, 0.16)',
        backgroundColor: 'rgba(0, 0, 0, 0.9)',
        backdropFilter: 'blur(16px)',
      },
      colorSecondary: {
        backgroundColor: 'rgba(0, 0, 0, 0.9)',
        color: '#F7F5FA',
      },
    },
    MuiToolbar: { root: { backgroundColor: 'transparent' } },
    MuiPaper: {
      root: { backgroundColor: '#1B1720', backgroundImage: 'none' },
      elevation1: { boxShadow: '0 0.0625rem 0.1875rem rgba(0, 0, 0, 0.28)' },
      elevation2: { boxShadow: '0 0.125rem 0.5rem rgba(0, 0, 0, 0.28)' },
    },
    MuiCard: {
      root: {
        backgroundColor: '#000000',
        borderRadius: '0.75rem',
        boxShadow: '0 0.125rem 0.5rem rgba(0, 0, 0, 0.28)',
      },
    },
    MuiButton: {
      root: { borderRadius: 6, textTransform: 'none', fontWeight: 600 },
      contained: { boxShadow: 'none', '&:hover': { boxShadow: 'none' } },
      containedPrimary: {
        backgroundColor: '#7C3AED',
        '&:hover': { backgroundColor: '#7C3AED', filter: 'brightness(0.9)' },
      },
      text: { color: '#7C3AED' },
    },
    MuiIconButton: {
      root: {
        color: '#F7F5FA',
        '&:hover': {
          backgroundColor: 'rgba(124, 58, 237, 0.16)',
          color: '#8B5CF6',
        },
        borderRadius: 999,
        transition: 'background-color 120ms ease, color 120ms ease',
      },
      colorPrimary: { color: '#7C3AED' },
      sizeSmall: { padding: '0.75rem' },
    },
    MuiSvgIcon: { colorPrimary: { color: '#7C3AED' } },
    MuiCheckbox: {
      root: {
        color: 'rgba(165, 125, 190, 0.68)',
        '&$checked': { color: '#8B5CF6' },
        '&:hover': {
          color: '#F7F5FA',
          backgroundColor: 'rgba(124, 58, 237, 0.16)',
        },
      },
    },
    MuiChip: {
      root: {
        backgroundColor: '#1B1720',
        color: '#F7F5FA',
        borderRadius: '1rem',
      },
      colorPrimary: {
        backgroundColor: 'rgba(124, 58, 237, 0.16)',
        color: '#7C3AED',
      },
    },
    MuiTableRow: {
      root: {
        '&:hover': { backgroundColor: '#27202D !important' },
        backgroundColor: '#1B1720',
      },
    },
    MuiTableCell: {
      root: { borderBottomColor: 'rgba(165, 125, 190, 0.16)' },
      head: {
        backgroundColor: '#1B1720',
        color: '#A9A4B0',
        fontWeight: 600,
        fontSize: '0.75rem',
        textTransform: 'uppercase',
        letterSpacing: '0.05em',
      },
      body: { color: '#F7F5FA' },
    },
    MuiListItem: {
      root: {
        color: '#F7F5FA',
        '&:hover': { backgroundColor: 'rgba(124, 58, 237, 0.16)' },
        '&$selected': {
          backgroundColor: 'rgba(124, 58, 237, 0.16)',
          color: '#7C3AED',
          '& .MuiListItemIcon-root': { color: '#7C3AED' },
          '&:hover': { backgroundColor: 'rgba(124, 58, 237, 0.16)' },
        },
      },
      button: {
        color: '#F7F5FA',
        '&:hover': {
          backgroundColor: 'rgba(124, 58, 237, 0.16)',
          color: '#F7F5FA',
        },
      },
    },
    MuiListItemIcon: {
      root: {
        color: '#F7F5FA',
        minWidth: '2.5rem',
        '& svg': { fontSize: '1.25rem' },
      },
    },
    MuiListItemText: { primary: { color: 'inherit' } },
    MuiMenuItem: {
      root: {
        fontSize: '0.875rem',
        paddingTop: '4px',
        paddingBottom: '4px',
        paddingLeft: '10px',
        margin: '5px',
        borderRadius: '0.5rem',
        color: '#F7F5FA',
      },
    },
    MuiDrawer: {
      paper: {
        backgroundColor: '#000000',
        borderRight: '1px solid rgba(165, 125, 190, 0.16)',
      },
    },
    MuiSlider: {
      root: { color: '#7C3AED' },
      track: { backgroundColor: '#7C3AED' },
      thumb: {
        backgroundColor: '#7C3AED',
        '&:hover': { boxShadow: '0 0 0 0.5rem rgba(124, 58, 237, 0.16)' },
      },
      rail: { backgroundColor: '#27202D' },
    },
    MuiLinearProgress: {
      root: { backgroundColor: '#27202D', borderRadius: '0.25rem' },
      bar: { backgroundColor: '#7C3AED', borderRadius: '0.25rem' },
    },
    MuiTabs: {
      root: { borderBottom: '1px solid rgba(165, 125, 190, 0.16)' },
      indicator: {
        backgroundColor: '#7C3AED',
        height: '0.1875rem',
        borderRadius: '0.1875rem 0.1875rem 0 0',
      },
    },
    MuiTab: {
      root: {
        textTransform: 'none',
        fontWeight: 500,
        fontFamily:
          '-apple-system, BlinkMacSystemFont, "SF Pro Display", "SF Pro Text", "Helvetica Neue", Arial, sans-serif',
        '&$selected': { color: '#7C3AED', fontWeight: 600 },
      },
    },
    MuiInputBase: {
      root: { backgroundColor: '#1B1720', borderRadius: '0.625rem' },
    },
    MuiOutlinedInput: {
      root: {
        borderRadius: '0.625rem',
        '& $notchedOutline': { borderColor: 'rgba(165, 125, 190, 0.16)' },
        '&:hover $notchedOutline': { borderColor: '#77717F' },
        '&$focused $notchedOutline': {
          borderColor: '#7C3AED',
          borderWidth: '0.125rem',
        },
      },
    },
    MuiFilledInput: {
      root: {
        backgroundColor: '#1B1720',
        borderRadius: '0.625rem',
        '&:hover': { backgroundColor: '#27202D' },
        '&$focused': { backgroundColor: '#1B1720' },
      },
    },
    MuiFab: {
      primary: {
        backgroundColor: '#7C3AED',
        '&:hover': { backgroundColor: '#7C3AED', filter: 'brightness(0.9)' },
      },
    },
    MuiAvatar: { root: { borderRadius: '0.5rem' } },
    MuiRating: {
      iconFilled: { color: '#7C3AED' },
      iconHover: { color: '#7C3AED' },
    },
    MuiTooltip: {
      tooltip: {
        backgroundColor: '#27202D',
        color: '#F7F5FA',
        fontSize: '0.75rem',
        padding: '0.25rem 0.5rem',
        borderRadius: 8,
        border: '1px solid rgba(165, 125, 190, 0.16)',
      },
    },
    MuiBottomNavigation: {
      root: {
        backgroundColor: 'rgba(27, 23, 32, 0.9)',
        backdropFilter: 'blur(1.25rem)',
        borderTop: '1px solid rgba(165, 125, 190, 0.16)',
      },
    },
    MuiBottomNavigationAction: {
      root: { color: '#A9A4B0', '&$selected': { color: '#7C3AED' } },
      label: {
        fontFamily:
          '-apple-system, BlinkMacSystemFont, "SF Pro Display", "SF Pro Text", "Helvetica Neue", Arial, sans-serif',
        fontSize: '0.65rem',
        '&$selected': { fontSize: '0.65rem' },
      },
    },
    NDAppBar: { root: { color: '#F7F5FA' } },
    NDLogin: {
      main: { backgroundColor: '#000000' },
      card: {
        backgroundColor: '#000000',
        borderRadius: '1rem',
        boxShadow: '0 0.25rem 1.5rem rgba(0, 0, 0, 0.28)',
      },
    },
    NDAlbumGridView: {
      albumContainer: {
        borderRadius: '0.5rem',
        '& img': { borderRadius: '0.5rem' },
      },
      albumTitle: { fontWeight: 600, color: '#F7F5FA' },
      albumSubtitle: { color: '#A9A4B0' },
      albumPlayButton: {
        backgroundColor: '#7C3AED',
        borderRadius: '50%',
        boxShadow: '0 0.5rem 0.5rem rgba(0, 0, 0, 0.15)',
        padding: '0.35rem',
        transition: 'padding .3s ease',
        '&:hover': {
          backgroundColor: '#7C3AED !important',
          padding: '0.45rem',
        },
      },
    },
    NDAlbumDetails: {
      root: {
        '@media (max-width: 599px)': { padding: '0.7em', minWidth: 'unset' },
      },
      cardContents: {
        '@media (max-width: 599px)': {
          flexDirection: 'column',
          alignItems: 'center',
        },
      },
      details: { '@media (max-width: 599px)': { width: '100%' } },
      cover: { borderRadius: '0.5rem' },
      coverParent: {
        marginRight: '1.5rem',
        '@media (max-width: 599px)': {
          width: '100%',
          height: 'auto',
          minWidth: 'unset',
          aspectRatio: '1',
          marginRight: 0,
          marginBottom: '1rem',
        },
        '@media (min-width: 600px)': {
          height: '14em',
          width: '14em',
          minWidth: '14em',
        },
        '@media (min-width: 1280px)': {
          height: '18em',
          width: '18em',
          minWidth: '18em',
        },
      },
      recordName: {
        fontSize: '1.75rem',
        fontWeight: 700,
        marginBottom: '0.15rem',
      },
      recordArtist: { marginBottom: '0.75rem' },
      recordMeta: { marginBottom: '0.5rem' },
      genreList: { marginTop: '0.75rem' },
      loveButton: { marginLeft: '0.5rem' },
    },
    NDAlbumShow: {
      albumActions: {
        padding: '1rem 0',
        alignItems: 'center',
        '@global': {
          button: {
            border: '1px solid transparent',
            backgroundColor: '#1B1720',
            color: '#A9A4B0',
            margin: '0 0.5rem',
            borderRadius: '50%',
            minWidth: 0,
            padding: '1rem',
            position: 'relative',
            '&:hover': {
              backgroundColor: '#27202D !important',
              border: '1px solid transparent',
            },
          },
          'button:first-child:not(:only-child)': {
            '@media screen and (max-width: 720px)': {
              transform: 'scale(1.5)',
              margin: '1rem',
              '&:hover': { transform: 'scale(1.6) !important' },
            },
            transform: 'scale(2)',
            margin: '1.5rem',
            minWidth: 0,
            padding: '0.3125rem',
            transition: 'transform .3s ease',
            background: '#7C3AED',
            color: '#fff',
            borderRadius: '50%',
            border: 0,
            '&:hover': {
              transform: 'scale(2.1)',
              backgroundColor: '#7C3AED !important',
              border: 0,
            },
          },
          'button:only-child': { margin: '1.5rem' },
          'button:first-child>span:first-child': { padding: 0 },
          'button>span:first-child>span': { display: 'none' },
          'button:not(:first-child):hover>span:first-child>span': {
            display: 'inline',
            position: 'absolute',
            bottom: '100%',
            left: '50%',
            transform: 'translateX(-50%)',
            marginBottom: '0.25rem',
            fontSize: '0.75rem',
            whiteSpace: 'nowrap',
            backgroundColor: '#F7F5FA',
            color: '#000000',
            padding: '0.25rem 0.5rem',
            borderRadius: '0.25rem',
            zIndex: 9999,
          },
          'button:not(:first-child)>span:first-child>svg': { color: '#A9A4B0' },
        },
      },
    },
    NDPlaylistShow: {
      playlistActions: {
        padding: '0 8px',
        alignItems: 'center',
        '@global': {
          button: {
            border: '1px solid transparent',
            backgroundColor: '#1B1720',
            color: '#A9A4B0',
            margin: '0 4px',
            borderRadius: '50%',
            minWidth: 0,
            padding: 8,
            position: 'relative',
            '&:hover': {
              backgroundColor: '#27202D !important',
              border: '1px solid transparent',
            },
          },
          'button:first-child:not(:only-child)': {
            '@media screen and (max-width: 720px)': {
              transform: 'scale(1.25)',
              margin: '4px 8px',
              '&:hover': { transform: 'scale(1.3) !important' },
            },
            transform: 'scale(1.35)',
            margin: '4px 10px',
            minWidth: 0,
            padding: 4,
            transition: 'transform .3s ease',
            background: '#7C3AED',
            color: '#fff',
            borderRadius: '50%',
            border: 0,
            '&:hover': {
              transform: 'scale(1.42) !important',
              backgroundColor: '#7C3AED !important',
              border: 0,
            },
          },
          'button:only-child': {
            margin: 4,
            padding: 4,
            backgroundColor: 'transparent',
          },
          'button:first-child>span:first-child': { padding: 0 },
          'button>span:first-child>span': { display: 'none' },
          'button:not(:first-child):hover>span:first-child>span': {
            display: 'inline',
            position: 'absolute',
            bottom: '100%',
            left: '50%',
            transform: 'translateX(-50%)',
            marginBottom: '0.25rem',
            fontSize: '0.75rem',
            whiteSpace: 'nowrap',
            backgroundColor: '#F7F5FA',
            color: '#000000',
            padding: '0.25rem 0.5rem',
            borderRadius: '0.25rem',
            zIndex: 9999,
          },
          'button:not(:first-child)>span:first-child>svg': { color: '#A9A4B0' },
        },
        '& > div': { justifyContent: 'space-between', alignItems: 'center' },
        '& > div > div': { display: 'flex', alignItems: 'center' },
      },
    },
    NDSubMenu: {
      icon: {
        color: '#F7F5FA',
        minWidth: '2.5rem',
        '& svg': { fontSize: '1.25rem' },
      },
      menuHeader: {
        color: '#F7F5FA',
        '& .MuiTypography-root': { color: '#F7F5FA' },
      },
      actionIcon: { marginLeft: '0.5rem' },
    },
    RaMenuItemLink: {
      root: {
        color: '#F7F5FA !important',
        '& .MuiListItemIcon-root': {
          color: '#F7F5FA',
          minWidth: '2.5rem',
          '& svg': { fontSize: '1.25rem' },
        },
        '&[class*="makeStyles-active"]': {
          color: '#7C3AED !important',
          '& .MuiListItemIcon-root': { color: '#7C3AED !important' },
        },
      },
      active: {
        color: '#7C3AED !important',
        '& .MuiListItemIcon-root': { color: '#7C3AED !important' },
      },
    },
    NDDesktopArtistDetails: {
      root: {
        '@media (min-width: 600px)': { padding: '1em' },
        '@media (min-width: 1280px)': { padding: '1em' },
      },
      cover: {
        borderRadius: '0.5rem',
        '@media (min-width: 600px)': {
          height: '14em',
          width: '14em',
          minWidth: '14em',
        },
        '@media (min-width: 1280px)': {
          height: '18em',
          width: '18em',
          minWidth: '18em',
        },
      },
      artistImage: {
        borderRadius: '0.5rem',
        marginRight: '1.5rem',
        '@media (min-width: 600px)': {
          height: '14em',
          width: '14em',
          minWidth: '14em',
          maxHeight: '14em',
          minHeight: '14em',
        },
        '@media (min-width: 1280px)': {
          height: '18em',
          width: '18em',
          minWidth: '18em',
          maxHeight: '18em',
          minHeight: '18em',
        },
      },
      artistName: {
        fontSize: '1.75rem',
        fontWeight: 700,
        marginBottom: '0.5rem',
      },
    },
    NDMobileArtistDetails: {
      cover: { borderRadius: '0.5rem' },
      artistImage: { borderRadius: '0.5rem' },
    },
    RaList: {
      content: {
        overflow: 'hidden',
        marginTop: 0,
        backgroundColor: '#1B1720',
        borderRadius: 0,
        boxShadow: 'none',
      },
      root: {
        backgroundColor: '#1B1720',
        border: '1px solid rgba(165, 125, 190, 0.16)',
        borderRadius: 14,
        overflow: 'hidden',
        '& > .MuiBox-root:last-child': { paddingBottom: 24 },
      },
      toolbar: {
        backgroundColor: '#1B1720',
        alignItems: 'center',
        border: '1px solid rgba(165, 125, 190, 0.16)',
        borderBottom: 0,
        borderRadius: '14px 14px 0 0',
        boxSizing: 'border-box',
        '& + div > .MuiCard-root': {
          borderLeft: '1px solid rgba(165, 125, 190, 0.16)',
          borderRight: '1px solid rgba(165, 125, 190, 0.16)',
        },
        '& ~ .MuiTablePagination-root': {
          borderLeft: '1px solid rgba(165, 125, 190, 0.16)',
          borderRight: '1px solid rgba(165, 125, 190, 0.16)',
          borderBottom: '1px solid rgba(165, 125, 190, 0.16)',
          borderRadius: '0 0 14px 14px',
          overflow: 'hidden',
        },
        '& ~ .MuiCardContent-root': {
          minHeight: 80,
          padding: '20px 24px !important',
          boxSizing: 'border-box',
          backgroundColor: '#1B1720',
          borderLeft: '1px solid rgba(165, 125, 190, 0.16)',
          borderRight: '1px solid rgba(165, 125, 190, 0.16)',
          borderBottom: '1px solid rgba(165, 125, 190, 0.16)',
          borderRadius: '0 0 14px 14px',
          color: '#A9A4B0',
          '& .MuiTypography-root': {
            fontSize: '1rem',
            fontWeight: 400,
            textAlign: 'left',
          },
        },
      },
    },
    RaBulkActionsToolbar: {
      topToolbar: {
        backgroundColor: 'transparent',
        boxShadow: 'none',
        padding: '0.5rem',
        '@global': {
          button: {
            border: '1px solid transparent',
            backgroundColor: '#1B1720',
            color: '#A9A4B0',
            margin: '0 0.25rem',
            borderRadius: '50%',
            minWidth: 0,
            padding: '0.5rem',
            position: 'relative',
            '&:hover': {
              backgroundColor: '#27202D !important',
              border: '1px solid transparent',
            },
          },
          'button>span:first-child>span': { display: 'none' },
          'button:hover>span:first-child>span': {
            display: 'inline',
            position: 'absolute',
            bottom: '100%',
            left: '50%',
            transform: 'translateX(-50%)',
            marginBottom: '0.25rem',
            fontSize: '0.75rem',
            whiteSpace: 'nowrap',
            backgroundColor: '#F7F5FA',
            color: '#000000',
            padding: '0.25rem 0.5rem',
            borderRadius: '0.25rem',
            zIndex: 9999,
          },
          'button>span:first-child>svg': { color: '#A9A4B0' },
        },
      },
    },
    RaPaginationActions: {
      currentPageButton: { backgroundColor: 'rgba(124, 58, 237, 0.16)' },
    },
    RaSaveButton: {
      button: {
        minWidth: 96,
        minHeight: 40,
        padding: '8px 14px',
        borderRadius: '8px !important',
        boxShadow: 'none',
        color: '#FFFFFF !important',
        backgroundColor: '#7C3AED !important',
        border: '1px solid transparent',
        '&:hover': { backgroundColor: '#8B5CF6 !important', boxShadow: 'none' },
        '&.Mui-disabled': {
          color: '#77717F !important',
          backgroundColor: '#27202D !important',
          borderColor: 'rgba(165, 125, 190, 0.16)',
        },
      },
    },
    NDPluginShow: {
      saveButton: {
        minWidth: 96,
        minHeight: 40,
        padding: '8px 14px',
        borderRadius: '8px !important',
        boxShadow: 'none',
        color: '#FFFFFF !important',
        backgroundColor: '#7C3AED !important',
        border: '1px solid transparent',
        '&:hover': { backgroundColor: '#8B5CF6 !important', boxShadow: 'none' },
        '&.Mui-disabled': {
          color: '#77717F !important',
          backgroundColor: '#27202D !important',
          borderColor: 'rgba(165, 125, 190, 0.16)',
        },
      },
    },
    MuiPopover: {
      root: {
        '&[id="menu-appbar"], &[id="panel-activity"], &[id="panel-nowplaying"]':
          {
            '& .MuiPopover-paper': {
              backgroundColor: '#1B1720',
              borderRadius: 10,
              overflow: 'hidden',
            },
            '& .MuiCard-root': {
              backgroundColor: 'transparent',
              borderRadius: 0,
              boxShadow: 'none',
            },
            '& .MuiCardContent-root, & .MuiCardActions-root': {
              backgroundColor: 'transparent',
            },
          },
      },
    },
    RaDatagrid: {
      rowCell: {
        '&&': {
          paddingTop: '14px !important',
          paddingBottom: '14px !important',
        },
      },
    },
    RaEmpty: {
      message: { '&:last-child': { paddingBottom: 24 } },
      toolbar: { paddingBottom: 24 },
    },
    RaListToolbar: {
      toolbar: {
        backgroundColor: '#1B1720',
        paddingLeft: '16px !important',
        paddingRight: '16px !important',
        borderBottom: '1px solid rgba(165, 125, 190, 0.16)',
      },
      actions: { backgroundColor: '#1B1720' },
    },
    RaSimpleList: {
      link: { '& .MuiListItem-root': { paddingTop: 16, paddingBottom: 16 } },
    },
    RaArtistSimpleList: {
      listItem: { padding: 16 },
      rightIcon: { top: '50%' },
    },
    MuiTablePagination: {
      root: {
        backgroundColor: '#1B1720',
        borderTop: '1px solid rgba(165, 125, 190, 0.16)',
        borderBottom: 0,
      },
    },
    RaToolbar: {
      toolbar: {
        backgroundColor: 'transparent',
        boxShadow: 'none',
        '& .MuiButton-root': {
          minWidth: 96,
          minHeight: 40,
          padding: '8px 14px',
          borderRadius: '8px !important',
          boxShadow: 'none',
          fontWeight: 600,
          transition:
            'background-color 120ms ease, border-color 120ms ease, color 120ms ease',
        },
        '& .MuiButton-root .MuiButton-label svg': {
          fontSize: '18px !important',
        },
        '& .ra-delete-button': {
          color: '#FFFFFF !important',
          backgroundColor: '#F44336 !important',
          border: '1px solid transparent',
          '&:hover': {
            color: '#FFFFFF !important',
            backgroundColor: '#C62828 !important',
            borderColor: 'transparent',
          },
          '&.Mui-disabled': {
            color: 'rgba(255, 92, 87, 0.38) !important',
            backgroundColor: '#27202D !important',
            borderColor: 'rgba(165, 125, 190, 0.16)',
          },
        },
      },
      desktopToolbar: {
        width: 'fit-content',
        minHeight: 0,
        padding: '0 16px !important',
        gap: 12,
      },
      defaultToolbar: { justifyContent: 'flex-start', gap: 12 },
      mobileToolbar: { backgroundColor: '#1B1720' },
    },
    RaAppBar: { title: { color: '#8B5CF6' } },
    MuiDialog: {
      paper: {
        borderRadius: 12,
        border: '1px solid rgba(165, 125, 190, 0.16)',
        boxShadow: '0 18px 48px rgba(0, 0, 0, 0.38)',
        '& #config-panel .MuiButton-root + .MuiButton-root': { marginLeft: 12 },
      },
    },
  },
  player: {
    theme: 'dark',
    stylesheet: `
.react-jinke-music-player-main {
        background-color: #000000 !important;
        font-family: -apple-system,BlinkMacSystemFont,"SF Pro Text","Helvetica Neue",Arial,sans-serif !important;
      }

      .react-jinke-music-player-main .music-player-panel {
        background-color: rgba(27, 23, 32, 0.9) !important;
        backdrop-filter: blur(1.25rem) !important;
        -webkit-backdrop-filter: blur(1.25rem) !important;
        border-top: 1px solid rgba(165, 125, 190, 0.16) !important;
        box-shadow: 0 -0.125rem 1.25rem rgba(0, 0, 0, 0.06) !important;
      }

      .react-jinke-music-player-main svg {
        color: #F7F5FA !important;
      }

      .react-jinke-music-player-main svg:hover {
        color: #7C3AED !important;
      }

      .react-jinke-music-player-main .rc-slider-track,
      .react-jinke-music-player-main .rc-slider-handle {
        background-color: #7C3AED !important;
      }

      .react-jinke-music-player-main .rc-slider-handle {
        border-color: #7C3AED !important;
      }

      .react-jinke-music-player-main .rc-slider-rail {
        background-color: #1B1720 !important;
      }

      .react-jinke-music-player-main .rc-slider {
        height: 4px !important;
      }

      .react-jinke-music-player-main .rc-slider-rail,
      .react-jinke-music-player-main .rc-slider-track {
        height: 4px !important;
        border-radius: 2px !important;
      }

      .react-jinke-music-player-main .rc-slider-handle {
        width: 12px !important;
        height: 12px !important;
        margin-top: -4px !important;
      }

      .react-jinke-music-player-main .audio-lists-panel,
      .react-jinke-music-player-main .audio-lists-panel-content {
        background-color: #000000 !important;
      }

      .react-jinke-music-player-main .audio-lists-panel-content .audio-item {
        background-color: transparent !important;
        color: #F7F5FA !important;
      }

      .react-jinke-music-player-main .audio-lists-panel-content .audio-item:hover {
        background-color: rgba(124, 58, 237, 0.16) !important;
      }

      .react-jinke-music-player-main .audio-lists-panel-content .audio-item.playing {
        background-color: rgba(124, 58, 237, 0.16) !important;
        color: #7C3AED !important;
      }

      .react-jinke-music-player-main .lyric-btn-active,
      .react-jinke-music-player-main .play-mode-title {
        color: #7C3AED !important;
      }

      .react-jinke-music-player-main .music-player-panel .player-content .music-player-controller .music-player-info .music-player-title {
        color: #F7F5FA !important;
        font-weight: 600 !important;
        font-family: -apple-system, BlinkMacSystemFont, "SF Pro Display", "SF Pro Text", "Helvetica Neue", Arial, sans-serif !important;
      }

      .react-jinke-music-player-main .music-player-panel .player-content .music-player-controller .music-player-info .music-player-artist {
        color: #A9A4B0 !important;
      }

      .react-jinke-music-player-main.mini-player {
        background-color: rgba(27, 23, 32, 0.9) !important;
        backdrop-filter: blur(1.25rem) !important;
        -webkit-backdrop-filter: blur(1.25rem) !important;
        border-radius: 0.75rem !important;
        box-shadow: 0 0.25rem 1.25rem rgba(165, 125, 190, 0.16) !important;
      }


      .MuiTypography-h1,
      .MuiTypography-h2,
      .MuiTypography-h3,
      .MuiTypography-h4,
      .MuiTypography-h5,
      .MuiTypography-h6 {
        font-family: -apple-system, BlinkMacSystemFont, "SF Pro Display", "SF Pro Text", "Helvetica Neue", Arial, sans-serif !important;
      }

      .react-jinke-music-player-main .music-player-panel .player-content svg {
        font-size: 28px !important;
      }

      .react-jinke-music-player-main .music-player-panel .player-content .play-btn svg,
      .react-jinke-music-player-main .music-player-panel .player-content .play-btn:hover svg,
      .react-jinke-music-player-mobile-toggle .play-btn svg {
        color: #7C3AED !important;
      }

      .react-jinke-music-player-main .music-player-panel .player-content .play-btn svg {
        font-size: 36px !important;
      }

      .react-jinke-music-player-main .music-player-panel .player-content .lyric-btn svg {
        font-size: 22px !important;
      }

      .react-jinke-music-player-main .music-player-panel .player-content .audio-lists-btn {
        min-width: auto !important;
        padding: 0 4px !important;
        background-color: transparent !important;
        box-shadow: none !important;
      }

      .react-jinke-music-player-main .audio-lists-panel,
      .react-jinke-music-player-main .audio-lists-panel-content,
      .audio-lists-panel,
      .audio-lists-panel-content {
        background-color: #27202D !important;
      }

      .react-jinke-music-player-main .audio-lists-panel-header,
      .audio-lists-panel-header {
        background-color: #30263A !important;
        border-bottom: 1px solid rgba(165, 125, 190, 0.16) !important;
        box-shadow: none !important;
      }

      .react-jinke-music-player-main .audio-lists-panel-content .audio-item,
      .audio-lists-panel-content .audio-item {
        background-color: #1B1720 !important;
        border-bottom: 1px solid rgba(165, 125, 190, 0.16) !important;
      }

      .react-jinke-music-player-main .audio-lists-panel-content .audio-item:hover,
      .audio-lists-panel-content .audio-item:hover {
        background-color: rgba(124, 58, 237, 0.16) !important;
      }

      .react-jinke-music-player-main .audio-lists-panel-content .audio-item.playing,
      .audio-lists-panel-content .audio-item.playing {
        background-color: rgba(124, 58, 237, 0.24) !important;
        color: #8B5CF6 !important;
      }
    `,
  },
}

export default ShelvTheme

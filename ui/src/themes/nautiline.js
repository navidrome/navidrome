/**
 * Nautiline Theme for Navidrome
 * Light theme inspired by the Nautiline iOS app
 * 
 * Typography: Unbounded (Google Font) for headings
 */

// ============================================
// ACCENT COLOR OPTIONS - Uncomment one to use
// ============================================

// Forest greens
// const ACCENT_COLOR = '#2E8B57'  // Sea green (current)
// const ACCENT_COLOR = '#228B22'  // Forest green
// const ACCENT_COLOR = '#1B5E3B'  // Dark forest green
// const ACCENT_COLOR = '#2D6A4F'  // Muted forest green
// const ACCENT_COLOR = '#3A7D44'  // Medium forest

// Teal / Blue-greens
// const ACCENT_COLOR = '#2E7D6B'  // Teal green
// const ACCENT_COLOR = '#20796F'  // Dark teal
const ACCENT_COLOR = '#009688'  // Material teal
// const ACCENT_COLOR = '#00897B'  // Dark material teal

// Emerald / Bright greens
// const ACCENT_COLOR = '#10B981'  // Emerald (Tailwind)
// const ACCENT_COLOR = '#059669'  // Emerald dark
// const ACCENT_COLOR = '#34D399'  // Emerald light
// const ACCENT_COLOR = '#22C55E'  // Green (Tailwind)

// Sage / Muted greens
// const ACCENT_COLOR = '#5F8575'  // Sage green
// const ACCENT_COLOR = '#6B8E6B'  // Muted sage
// const ACCENT_COLOR = '#4A7C59'  // Hunter green

// Other colors (if you want to experiment)
// const ACCENT_COLOR = '#0EA5E9'  // Sky blue
// const ACCENT_COLOR = '#8B5CF6'  // Violet
// const ACCENT_COLOR = '#F59E0B'  // Amber
// const ACCENT_COLOR = '#EF4444'  // Red

// ============================================
// COLOR PALETTE GENERATION
// ============================================

// Helper to generate lighter/darker variants
const hexToRgb = (hex) => {
    const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex)
    return result ? {
        r: parseInt(result[1], 16),
        g: parseInt(result[2], 16),
        b: parseInt(result[3], 16)
    } : null
}

const rgb = hexToRgb(ACCENT_COLOR)
const ACCENT_FADED = `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.1)`
const ACCENT_HOVER = `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.15)`

// ============================================
// THEME COLORS
// ============================================

const nautilineColors = {
    accent: {
        main: ACCENT_COLOR,
        faded: ACCENT_FADED,
        hover: ACCENT_HOVER,
    },
    background: {
        primary: '#FFFFFF',
        secondary: '#F5F5F7',
        tertiary: '#E5E5EA',
    },
    text: {
        primary: '#1A1A1A',
        secondary: '#8E8E93',
        tertiary: '#AEAEB2',
    },
    ui: {
        separator: 'rgba(0, 0, 0, 0.08)',
        shadow: 'rgba(0, 0, 0, 0.04)',
        glassBg: 'rgba(255, 255, 255, 0.72)',
        glassBlur: '1.25rem',
    },
}

// ============================================
// THEME DEFINITION
// ============================================

const nautilineTheme = {
    themeName: 'Nautiline',
    palette: {
        type: 'light',
        primary: {
            main: nautilineColors.accent.main,
            contrastText: '#FFFFFF',
        },
        secondary: {
            main: nautilineColors.accent.main,
            contrastText: '#FFFFFF',
        },
        background: {
            default: nautilineColors.background.primary,
            paper: nautilineColors.background.primary,
        },
        text: {
            primary: nautilineColors.text.primary,
            secondary: nautilineColors.text.secondary,
        },
        action: {
            active: nautilineColors.accent.main,
            hover: nautilineColors.accent.faded,
            selected: nautilineColors.accent.faded,
        },
    },
    typography: {
        fontFamily: [
            '-apple-system',
            'BlinkMacSystemFont',
            '"SF Pro Text"',
            '"Helvetica Neue"',
            'Arial',
            'sans-serif',
        ].join(','),
        h1: {
            fontFamily: '"Unbounded", sans-serif',
            fontWeight: 700,
            letterSpacing: '-0.02em',
        },
        h2: {
            fontFamily: '"Unbounded", sans-serif',
            fontWeight: 700,
            letterSpacing: '-0.02em',
        },
        h3: {
            fontFamily: '"Unbounded", sans-serif',
            fontWeight: 600,
            letterSpacing: '-0.01em',
        },
        h4: {
            fontFamily: '"Unbounded", sans-serif',
            fontWeight: 600,
        },
        h5: {
            fontFamily: '"Unbounded", sans-serif',
            fontWeight: 600,
        },
        h6: {
            fontFamily: '"Unbounded", sans-serif',
            fontWeight: 600,
        },
        subtitle1: {
            fontWeight: 500,
        },
        subtitle2: {
            fontWeight: 500,
        },
        body1: {
            fontWeight: 400,
        },
        body2: {
            fontWeight: 400,
            color: nautilineColors.text.secondary,
        },
        button: {
            fontWeight: 500,
            textTransform: 'none',
        },
    },
    shape: {
        borderRadius: '0.75rem',
    },
    overrides: {
        MuiCssBaseline: {
            '@global': {
                '@import': "url('https://fonts.googleapis.com/css2?family=Unbounded:wght@300;400;500;600;700;800&display=swap')",
                body: {
                    backgroundColor: nautilineColors.background.primary,
                },
            },
        },
        MuiAppBar: {
            root: {
                boxShadow: 'none',
                borderBottom: `1px solid ${nautilineColors.ui.separator}`,
            },
            colorSecondary: {
                backgroundColor: nautilineColors.background.primary,
                color: nautilineColors.text.primary,
            },
        },
        MuiToolbar: {
            root: {
                backgroundColor: nautilineColors.background.primary,
            },
        },
        MuiPaper: {
            root: {
                backgroundColor: nautilineColors.background.primary,
            },
            elevation1: {
                boxShadow: `0 0.0625rem 0.1875rem ${nautilineColors.ui.shadow}`,
            },
            elevation2: {
                boxShadow: `0 0.125rem 0.5rem ${nautilineColors.ui.shadow}`,
            },
        },
        MuiCard: {
            root: {
                backgroundColor: nautilineColors.background.primary,
                borderRadius: '0.75rem',
                boxShadow: `0 0.125rem 0.5rem ${nautilineColors.ui.shadow}`,
            },
        },
        MuiButton: {
            root: {
                borderRadius: '0.5rem',
                textTransform: 'none',
                fontWeight: 600,
            },
            contained: {
                boxShadow: 'none',
                '&:hover': {
                    boxShadow: 'none',
                },
            },
            containedPrimary: {
                backgroundColor: nautilineColors.accent.main,
                '&:hover': {
                    backgroundColor: nautilineColors.accent.main,
                    filter: 'brightness(0.9)',
                },
            },
            text: {
                color: nautilineColors.accent.main,
            },
        },
        MuiIconButton: {
            root: {
                color: nautilineColors.text.primary,
                '&:hover': {
                    backgroundColor: nautilineColors.accent.faded,
                },
            },
            colorPrimary: {
                color: nautilineColors.accent.main,
            },
        },
        MuiSvgIcon: {
            colorPrimary: {
                color: nautilineColors.accent.main,
            },
        },
        MuiCheckbox: {
            root: {
                color: 'rgba(0, 0, 0, 0.15)',
                '&$checked': {
                    color: nautilineColors.accent.main,
                },
            },
        },
        MuiChip: {
            root: {
                backgroundColor: nautilineColors.background.secondary,
                color: nautilineColors.text.primary,
                borderRadius: '1rem',
            },
            colorPrimary: {
                backgroundColor: nautilineColors.accent.faded,
                color: nautilineColors.accent.main,
            },
        },
        MuiTableRow: {
            root: {
                '&:hover': {
                    backgroundColor: `${nautilineColors.accent.faded} !important`,
                },
            },
        },
        MuiTableCell: {
            root: {
                borderBottomColor: 'rgba(0, 0, 0, 0.04)',
            },
            head: {
                backgroundColor: nautilineColors.background.secondary,
                color: nautilineColors.text.secondary,
                fontWeight: 600,
                fontSize: '0.75rem',
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
            },
        },
        MuiListItem: {
            root: {
                color: nautilineColors.text.primary,
                '&:hover': {
                    backgroundColor: nautilineColors.accent.faded,
                },
                '&$selected': {
                    backgroundColor: nautilineColors.accent.faded,
                    color: nautilineColors.accent.main,
                    '& .MuiListItemIcon-root': {
                        color: nautilineColors.accent.main,
                    },
                    '&:hover': {
                        backgroundColor: nautilineColors.accent.faded,
                    },
                },
            },
        },
        MuiListItemIcon: {
            root: {
                color: nautilineColors.text.primary,
                minWidth: '2.5rem',
                '& svg': {
                    fontSize: '1.25rem',
                },
            },
        },
        MuiListItemText: {
            primary: {
                color: 'inherit',
            },
        },
        MuiDrawer: {
            paper: {
                backgroundColor: nautilineColors.background.primary,
                borderRight: `1px solid ${nautilineColors.ui.separator}`,
            },
        },
        MuiSlider: {
            root: {
                color: nautilineColors.accent.main,
            },
            track: {
                backgroundColor: nautilineColors.accent.main,
            },
            thumb: {
                backgroundColor: nautilineColors.accent.main,
                '&:hover': {
                    boxShadow: `0 0 0 0.5rem ${nautilineColors.accent.faded}`,
                },
            },
            rail: {
                backgroundColor: nautilineColors.background.tertiary,
            },
        },
        MuiLinearProgress: {
            root: {
                backgroundColor: nautilineColors.background.tertiary,
                borderRadius: '0.25rem',
            },
            bar: {
                backgroundColor: nautilineColors.accent.main,
                borderRadius: '0.25rem',
            },
        },
        MuiTabs: {
            root: {
                borderBottom: `1px solid ${nautilineColors.ui.separator}`,
            },
            indicator: {
                backgroundColor: nautilineColors.accent.main,
                height: '0.1875rem',
                borderRadius: '0.1875rem 0.1875rem 0 0',
            },
        },
        MuiTab: {
            root: {
                textTransform: 'none',
                fontWeight: 500,
                fontFamily: '"Unbounded", sans-serif',
                '&$selected': {
                    color: nautilineColors.accent.main,
                    fontWeight: 600,
                },
            },
        },
        MuiInputBase: {
            root: {
                backgroundColor: nautilineColors.background.secondary,
                borderRadius: '0.625rem',
            },
        },
        MuiOutlinedInput: {
            root: {
                borderRadius: '0.625rem',
                '& $notchedOutline': {
                    borderColor: nautilineColors.ui.separator,
                },
                '&:hover $notchedOutline': {
                    borderColor: nautilineColors.text.tertiary,
                },
                '&$focused $notchedOutline': {
                    borderColor: nautilineColors.accent.main,
                    borderWidth: '0.125rem',
                },
            },
        },
        MuiFilledInput: {
            root: {
                backgroundColor: nautilineColors.background.secondary,
                borderRadius: '0.625rem',
                '&:hover': {
                    backgroundColor: nautilineColors.background.tertiary,
                },
                '&$focused': {
                    backgroundColor: nautilineColors.background.secondary,
                },
            },
        },
        MuiFab: {
            primary: {
                backgroundColor: nautilineColors.accent.main,
                '&:hover': {
                    backgroundColor: nautilineColors.accent.main,
                    filter: 'brightness(0.9)',
                },
            },
        },
        MuiAvatar: {
            root: {
                borderRadius: '0.5rem',
            },
        },
        MuiBottomNavigation: {
            root: {
                backgroundColor: nautilineColors.ui.glassBg,
                backdropFilter: `blur(${nautilineColors.ui.glassBlur})`,
                borderTop: `1px solid ${nautilineColors.ui.separator}`,
            },
        },
        MuiBottomNavigationAction: {
            root: {
                color: nautilineColors.text.secondary,
                '&$selected': {
                    color: nautilineColors.accent.main,
                },
            },
            label: {
                fontFamily: '"Unbounded", sans-serif',
                fontSize: '0.65rem',
                '&$selected': {
                    fontSize: '0.65rem',
                },
            },
        },
        NDLogin: {
            main: {
                backgroundColor: nautilineColors.background.primary,
            },
            card: {
                backgroundColor: nautilineColors.background.primary,
                borderRadius: '1rem',
                boxShadow: `0 0.25rem 1.5rem ${nautilineColors.ui.shadow}`,
            },
        },
        NDAlbumGridView: {
            albumContainer: {
                borderRadius: '0.5rem',
                '& img': {
                    borderRadius: '0.5rem',
                },
            },
            albumTitle: {
                fontWeight: 600,
                color: nautilineColors.text.primary,
            },
            albumSubtitle: {
                color: nautilineColors.text.secondary,
            },
            albumPlayButton: {
                backgroundColor: nautilineColors.accent.main,
                borderRadius: '50%',
                boxShadow: '0 0.5rem 0.5rem rgba(0, 0, 0, 0.15)',
                padding: '0.35rem',
                transition: 'padding .3s ease',
                '&:hover': {
                    backgroundColor: `${nautilineColors.accent.main} !important`,
                    padding: '0.45rem',
                },
            },
        },
        NDAlbumDetails: {
            root: {
                '@media (max-width: 599px)': {
                    padding: '0.7em',
                    width: '100%',
                    minWidth: 'unset',
                },
            },
            cardContents: {
                '@media (max-width: 599px)': {
                    flexDirection: 'column',
                    alignItems: 'center',
                },
            },
            details: {
                '@media (max-width: 599px)': {
                    width: '100%',
                },
            },
            cover: {
                borderRadius: '0.5rem',
            },
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
            recordArtist: {
                marginBottom: '0.75rem',
            },
            recordMeta: {
                marginBottom: '0.5rem',
            },
            genreList: {
                marginTop: '0.75rem',
            },
        },
        NDAlbumShow: {
            albumActions: {
                padding: '1rem 0',
                alignItems: 'center',
                '@global': {
                    button: {
                        border: '1px solid transparent',
                        backgroundColor: nautilineColors.background.secondary,
                        color: nautilineColors.text.secondary,
                        margin: '0 0.5rem',
                        borderRadius: '50%',
                        minWidth: 0,
                        padding: '0.625rem',
                        position: 'relative',
                        '&:hover': {
                            backgroundColor: `${nautilineColors.background.tertiary} !important`,
                            border: '1px solid transparent',
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
                        padding: '0.3125rem',
                        transition: 'transform .3s ease',
                        background: nautilineColors.accent.main,
                        color: '#fff',
                        borderRadius: '50%',
                        border: 0,
                        '&:hover': {
                            transform: 'scale(2.1)',
                            backgroundColor: `${nautilineColors.accent.main} !important`,
                            border: 0,
                        },
                    },
                    'button:only-child': {
                        margin: '1.5rem',
                    },
                    'button:first-child>span:first-child': {
                        padding: 0,
                    },
                    'button>span:first-child>span': {
                        display: 'none',
                    },
                    'button:not(:first-child):hover>span:first-child>span': {
                        display: 'inline',
                        position: 'absolute',
                        bottom: '100%',
                        left: '50%',
                        transform: 'translateX(-50%)',
                        marginBottom: '0.25rem',
                        fontSize: '0.75rem',
                        whiteSpace: 'nowrap',
                        backgroundColor: nautilineColors.text.primary,
                        color: nautilineColors.background.primary,
                        padding: '0.25rem 0.5rem',
                        borderRadius: '0.25rem',
                        zIndex: 9999,
                    },
                    'button:not(:first-child)>span:first-child>svg': {
                        color: nautilineColors.text.secondary,
                    },
                },
            },
        },
        NDPlaylistShow: {
            playlistActions: {
                padding: '1rem 0',
                alignItems: 'center',
                '@global': {
                    button: {
                        border: '1px solid transparent',
                        backgroundColor: nautilineColors.background.secondary,
                        color: nautilineColors.text.secondary,
                        margin: '0 0.5rem',
                        borderRadius: '50%',
                        minWidth: 0,
                        padding: '0.625rem',
                        position: 'relative',
                        '&:hover': {
                            backgroundColor: `${nautilineColors.background.tertiary} !important`,
                            border: '1px solid transparent',
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
                        padding: '0.3125rem',
                        transition: 'transform .3s ease',
                        background: nautilineColors.accent.main,
                        color: '#fff',
                        borderRadius: '50%',
                        border: 0,
                        '&:hover': {
                            transform: 'scale(2.1)',
                            backgroundColor: `${nautilineColors.accent.main} !important`,
                            border: 0,
                        },
                    },
                    'button:only-child': {
                        margin: '1.5rem',
                    },
                    'button:first-child>span:first-child': {
                        padding: 0,
                    },
                    'button>span:first-child>span': {
                        display: 'none',
                    },
                    'button:not(:first-child):hover>span:first-child>span': {
                        display: 'inline',
                        position: 'absolute',
                        bottom: '100%',
                        left: '50%',
                        transform: 'translateX(-50%)',
                        marginBottom: '0.25rem',
                        fontSize: '0.75rem',
                        whiteSpace: 'nowrap',
                        backgroundColor: nautilineColors.text.primary,
                        color: nautilineColors.background.primary,
                        padding: '0.25rem 0.5rem',
                        borderRadius: '0.25rem',
                        zIndex: 9999,
                    },
                    'button:not(:first-child)>span:first-child>svg': {
                        color: nautilineColors.text.secondary,
                    },
                },
            },
        },
        NDSubMenu: {
            icon: {
                color: nautilineColors.text.primary,
                minWidth: '2.5rem',
                '& svg': {
                    fontSize: '1.25rem',
                },
            },
            menuHeader: {
                color: nautilineColors.text.primary,
                '& .MuiTypography-root': {
                    color: nautilineColors.text.primary,
                },
            },
        },
        RaMenuItemLink: {
            root: {
                color: nautilineColors.text.primary,
                '& .MuiListItemIcon-root': {
                    color: nautilineColors.text.primary,
                    minWidth: '2.5rem',
                    '& svg': {
                        fontSize: '1.25rem',
                    },
                },
                '&[class*="makeStyles-active"]': {
                    color: `${nautilineColors.accent.main} !important`,
                    '& .MuiListItemIcon-root': {
                        color: `${nautilineColors.accent.main} !important`,
                    },
                },
            },
            active: {
                color: `${nautilineColors.accent.main} !important`,
                '& .MuiListItemIcon-root': {
                    color: `${nautilineColors.accent.main} !important`,
                },
            },
        },
        NDDesktopArtistDetails: {
            root: {
                '@media (min-width: 600px)': {
                    padding: '1em',
                },
                '@media (min-width: 1280px)': {
                    padding: '1em',
                },
            },
            cover: {
                borderRadius: '0.5rem',
                '@media (min-width: 600px)': {
                    height: '14em',
                    width: '14em',
                },
                '@media (min-width: 1280px)': {
                    height: '18em',
                    width: '18em',
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
            cover: {
                borderRadius: '0.5rem',
            },
            artistImage: {
                borderRadius: '0.5rem',
            },
        },
        RaList: {
            content: {
                overflow: 'visible',
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
                        backgroundColor: nautilineColors.background.secondary,
                        color: nautilineColors.text.secondary,
                        margin: '0 0.25rem',
                        borderRadius: '50%',
                        minWidth: 0,
                        padding: '0.5rem',
                        position: 'relative',
                        '&:hover': {
                            backgroundColor: `${nautilineColors.background.tertiary} !important`,
                            border: '1px solid transparent',
                        },
                    },
                    'button>span:first-child>span': {
                        display: 'none',
                    },
                    'button:hover>span:first-child>span': {
                        display: 'inline',
                        position: 'absolute',
                        bottom: '100%',
                        left: '50%',
                        transform: 'translateX(-50%)',
                        marginBottom: '0.25rem',
                        fontSize: '0.75rem',
                        whiteSpace: 'nowrap',
                        backgroundColor: nautilineColors.text.primary,
                        color: nautilineColors.background.primary,
                        padding: '0.25rem 0.5rem',
                        borderRadius: '0.25rem',
                        zIndex: 9999,
                    },
                    'button>span:first-child>svg': {
                        color: nautilineColors.text.secondary,
                    },
                },
            },
        },
    },
    player: {
        theme: 'light',
        stylesheet: `
      /* Import Unbounded font */
      @import url('https://fonts.googleapis.com/css2?family=Unbounded:wght@300;400;500;600;700;800&display=swap');
      
      /* Main player container */
      .react-jinke-music-player-main {
        background-color: ${nautilineColors.background.primary} !important;
        font-family: -apple-system, BlinkMacSystemFont, "SF Pro Text", "Helvetica Neue", Arial, sans-serif !important;
      }
      
      /* Player panel - frosted glass effect */
      .react-jinke-music-player-main .music-player-panel {
        background-color: ${nautilineColors.ui.glassBg} !important;
        backdrop-filter: blur(${nautilineColors.ui.glassBlur}) !important;
        -webkit-backdrop-filter: blur(${nautilineColors.ui.glassBlur}) !important;
        border-top: 1px solid ${nautilineColors.ui.separator} !important;
        box-shadow: 0 -0.125rem 1.25rem rgba(0, 0, 0, 0.06) !important;
      }
      
      /* Icons and controls */
      .react-jinke-music-player-main svg {
        color: ${nautilineColors.text.primary} !important;
      }
      
      .react-jinke-music-player-main svg:hover {
        color: ${nautilineColors.accent.main} !important;
      }
      
      /* Progress bar track */
      .react-jinke-music-player-main .rc-slider-track {
        background-color: ${nautilineColors.accent.main} !important;
      }
      
      /* Progress bar handle */
      .react-jinke-music-player-main .rc-slider-handle {
        border-color: ${nautilineColors.accent.main} !important;
        background-color: ${nautilineColors.accent.main} !important;
      }
      
      /* Progress bar rail */
      .react-jinke-music-player-main .rc-slider-rail {
        background-color: ${nautilineColors.background.tertiary} !important;
      }
      
      /* Audio list panel */
      .react-jinke-music-player-main .audio-lists-panel {
        background-color: ${nautilineColors.background.primary} !important;
      }
      
      .react-jinke-music-player-main .audio-lists-panel-content {
        background-color: ${nautilineColors.background.primary} !important;
      }
      
      .react-jinke-music-player-main .audio-lists-panel-content .audio-item {
        background-color: transparent !important;
        color: ${nautilineColors.text.primary} !important;
      }
      
      .react-jinke-music-player-main .audio-lists-panel-content .audio-item:hover {
        background-color: ${nautilineColors.accent.faded} !important;
      }
      
      .react-jinke-music-player-main .audio-lists-panel-content .audio-item.playing {
        background-color: ${nautilineColors.accent.faded} !important;
        color: ${nautilineColors.accent.main} !important;
      }
      
      /* Progress bar content */
      .react-jinke-music-player-main .progress-bar-content {
        background-color: ${nautilineColors.background.tertiary} !important;
        border-radius: 0.25rem !important;
      }
      
      .react-jinke-music-player-main .progress-bar-content .progress-bar {
        background-color: ${nautilineColors.accent.main} !important;
        border-radius: 0.25rem !important;
      }
      
      /* Active buttons */
      .react-jinke-music-player-main .lyric-btn-active,
      .react-jinke-music-player-main .play-mode-title {
        color: ${nautilineColors.accent.main} !important;
      }
      
      /* Song info text */
      .react-jinke-music-player-main .music-player-panel .player-content .music-player-controller .music-player-info .music-player-title {
        color: ${nautilineColors.text.primary} !important;
        font-weight: 600 !important;
        font-family: "Unbounded", sans-serif !important;
      }
      
      .react-jinke-music-player-main .music-player-panel .player-content .music-player-controller .music-player-info .music-player-artist {
        color: ${nautilineColors.text.secondary} !important;
      }
      
      /* Mini player */
      .react-jinke-music-player-main.mini-player {
        background-color: ${nautilineColors.ui.glassBg} !important;
        backdrop-filter: blur(${nautilineColors.ui.glassBlur}) !important;
        -webkit-backdrop-filter: blur(${nautilineColors.ui.glassBlur}) !important;
        border-radius: 0.75rem !important;
        box-shadow: 0 0.25rem 1.25rem rgba(0, 0, 0, 0.08) !important;
      }
      
      /* Album cover in player */
      .react-jinke-music-player-main .img-content {
        border-radius: 0.5rem !important;
      }
      
      /* Page titles with Unbounded font */
      .MuiTypography-h1,
      .MuiTypography-h2,
      .MuiTypography-h3,
      .MuiTypography-h4,
      .MuiTypography-h5,
      .MuiTypography-h6 {
        font-family: "Unbounded", sans-serif !important;
      }
    `,
    },
}

export default nautilineTheme

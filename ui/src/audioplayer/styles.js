import { makeStyles } from '@material-ui/core/styles'
import {
  LYRICS_SIDEBAR_RESIZING_BODY_CLASS,
  LYRICS_SIDEBAR_TRANSITION_MS,
} from './lyricsSidebarWidth'

const useStyle = makeStyles(
  (theme) => ({
    audioTitle: {
      textDecoration: 'none',
      color: theme.palette.primary.dark,
    },
    songTitle: {
      fontWeight: 'bold',
      '&:hover + $qualityInfo': {
        opacity: 1,
      },
    },
    songInfo: {
      display: 'block',
      marginTop: '2px',
    },
    songAlbum: {
      fontStyle: 'italic',
      fontSize: 'smaller',
    },
    qualityInfo: {
      marginTop: '-4px',
      opacity: 0,
      transition: 'all 500ms ease-out',
    },
    player: {
      display: (props) => (props.visible ? 'block' : 'none'),
      '@media screen and (max-width:810px)': {
        '& .sound-operation': {
          display: 'none',
        },
      },
      '@media (prefers-reduced-motion)': {
        '& .music-player-panel .panel-content div.img-rotate': {
          animation: 'none',
        },
      },
      '& .progress-bar-content': {
        display: 'flex',
        flexDirection: 'column',
      },
      '& .play-mode-title': {
        'pointer-events': 'none',
      },
      'body.nd-lyrics-sidebar-open & .react-jinke-music-player-main .music-player-panel':
        {
          width: 'calc(100% - var(--nd-lyrics-sidebar-width, 360px))',
        },
      '& .react-jinke-music-player-main .music-player-panel': {
        transition: `width ${LYRICS_SIDEBAR_TRANSITION_MS}ms cubic-bezier(0.22, 1, 0.36, 1)`,
      },
      [`body.${LYRICS_SIDEBAR_RESIZING_BODY_CLASS} & .react-jinke-music-player-main .music-player-panel`]:
        {
          transition: 'none',
        },
      'body.nd-lyrics-sidebar-open & .audio-lists-panel': {
        right: 'calc(33px + var(--nd-lyrics-sidebar-width, 360px))',
      },
      '& .audio-lists-panel': {
        transition: `right ${LYRICS_SIDEBAR_TRANSITION_MS}ms cubic-bezier(0.22, 1, 0.36, 1)`,
      },
      [`body.${LYRICS_SIDEBAR_RESIZING_BODY_CLASS} & .audio-lists-panel`]: {
        transition: 'none',
      },
      '@media (prefers-reduced-motion: reduce)': {
        '& .react-jinke-music-player-main .music-player-panel, & .audio-lists-panel':
          {
            transition: 'none',
          },
      },
      '& .music-player-panel .panel-content div.img-rotate': {
        // Customize desktop player when cover animation is disabled
        animationDuration: (props) => !props.enableCoverAnimation && '0s',
        borderRadius: (props) => !props.enableCoverAnimation && '0',
        // Fix cover display when image is not square
        backgroundSize: 'contain',
        backgroundPosition: 'center',
      },
      '& .react-jinke-music-player-mobile .react-jinke-music-player-mobile-cover':
        {
          // Customize mobile player when cover animation is disabled
          borderRadius: (props) => !props.enableCoverAnimation && '0',
          width: (props) => !props.enableCoverAnimation && '85%',
          maxWidth: (props) => !props.enableCoverAnimation && '600px',
          height: (props) => !props.enableCoverAnimation && 'auto',
          // Fix cover display when image is not square
          aspectRatio: '1/1',
          display: 'flex',
          position: 'relative',
        },
      '& .react-jinke-music-player-mobile .react-jinke-music-player-mobile-cover.nd-mobile-lyrics-active':
        {
          width: '100%',
          maxWidth: '100%',
          height: '100%',
          aspectRatio: 'auto',
          alignItems: 'stretch',
          justifyContent: 'stretch',
        },
      '& .react-jinke-music-player-mobile .react-jinke-music-player-mobile-cover .nd-mobile-lyrics-layer':
        {
          position: 'absolute',
          inset: 0,
          opacity: 0,
          transform: 'scale(0.985)',
          transition:
            'opacity 260ms cubic-bezier(0.22, 1, 0.36, 1), transform 260ms cubic-bezier(0.22, 1, 0.36, 1)',
        },
      '& .react-jinke-music-player-mobile .react-jinke-music-player-mobile-cover .nd-mobile-lyrics-layer[data-entered="true"]':
        {
          opacity: 1,
          transform: 'scale(1)',
        },
      '& .react-jinke-music-player-mobile .react-jinke-music-player-mobile-cover img.cover':
        {
          animationDuration: (props) => !props.enableCoverAnimation && '0s',
          objectFit: 'contain', // Fix cover display when image is not square
          transition:
            'opacity 260ms cubic-bezier(0.22, 1, 0.36, 1), transform 260ms cubic-bezier(0.22, 1, 0.36, 1)',
        },
      '& .react-jinke-music-player-mobile .react-jinke-music-player-mobile-cover.nd-mobile-lyrics-active img.cover':
        {
          opacity: 0,
          transform: 'scale(0.96)',
          pointerEvents: 'none',
        },
      // Hide old singer display
      '& .react-jinke-music-player-mobile .react-jinke-music-player-mobile-singer':
        {
          display: 'none',
        },
      // Hide extra whitespace from switch div
      '& .react-jinke-music-player-mobile .react-jinke-music-player-mobile-switch':
        {
          display: 'none',
        },
      '& .music-player-panel .panel-content .progress-bar-content section.audio-main':
        {
          display: (props) => (props.isRadio ? 'none' : 'inline-flex'),
        },
      '& .react-jinke-music-player-mobile-progress': {
        display: (props) => (props.isRadio ? 'none' : 'flex'),
      },
    },
  }),
  { name: 'NDAudioPlayer' },
)

export default useStyle

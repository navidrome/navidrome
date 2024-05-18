import { makeStyles } from '@material-ui/core/styles'

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
        },
      '& .react-jinke-music-player-mobile .react-jinke-music-player-mobile-cover img.cover':
        {
          animationDuration: (props) => !props.enableCoverAnimation && '0s',
          objectFit: 'contain', // Fix cover display when image is not square
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

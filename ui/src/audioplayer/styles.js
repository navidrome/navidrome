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
    songArtist: {
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
      '& .progress-bar-content': {
        display: 'flex',
        flexDirection: 'column',
      },
      '& .play-mode-title': {
        'pointer-events': 'none',
      },
      // Customize desktop player when cover animation is disabled
      '& .music-player-panel .panel-content div.img-rotate': {
        animationDuration: (props) => !props.enableCoverAnimation && '0s',
        borderRadius: (props) => !props.enableCoverAnimation && '0',
      },
      // Customize mobile player when cover animation is disabled
      '& .react-jinke-music-player-mobile .react-jinke-music-player-mobile-cover': {
        borderRadius: (props) => !props.enableCoverAnimation && '0',
        width: (props) => !props.enableCoverAnimation && '85%',
        maxWidth: (props) => !props.enableCoverAnimation && '600px',
        height: (props) => !props.enableCoverAnimation && 'auto',
      },
      '& .react-jinke-music-player-mobile .react-jinke-music-player-mobile-cover img.cover': {
        animationDuration: (props) => !props.enableCoverAnimation && '0s',
      },
      // Hide old singer display
      '& .react-jinke-music-player-mobile .react-jinke-music-player-mobile-singer': {
        display: 'none',
      },
      // Hide extra whitespace from switch div
      '& .react-jinke-music-player-mobile .react-jinke-music-player-mobile-switch': {
        display: 'none',
      },
    },
  }),
  { name: 'NDAudioPlayer' }
)

export default useStyle

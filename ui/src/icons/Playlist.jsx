import React from 'react'
import SvgIcon from '@material-ui/core/SvgIcon'

const Playlist = (props) => {
  return (
    <SvgIcon xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" {...props}>
      <path d="M 16 3 L 16 14.125 C 15.41 13.977 14.732 13.95 14 14.125 C 11.791 14.654 10 16.60975 10 18.46875 C 10 20.32775 11.791 21.404 14 20.875 C 16.149 20.361 17.87575 18.4985 17.96875 16.6875 L 18 16.6875 L 18 7 L 22 7 L 22 3 L 16 3 z M 2 4 L 2 6 L 15 6 L 15 4 L 2 4 z M 2 9 L 2 11 L 15 11 L 15 9 L 2 9 z M 2 14 L 2 16 L 9.75 16 C 10.242 15.218 10.9735 14.526 11.8125 14 L 2 14 z M 2 19 L 2 21 L 10.125 21 C 9.54 20.473 9.1795 19.785 9.0625 19 L 2 19 z" />
    </SvgIcon>
  )
}

export default Playlist

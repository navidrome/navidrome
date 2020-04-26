import React from 'react'
import { useDispatch } from 'react-redux'
import { playAlbum } from '../audioplayer'
import { useGetList} from 'react-admin'
import IconButton from '@material-ui/core/IconButton'
import PlayIcon from '@material-ui/icons/PlayCircleFilled'

const GridButton = (props) => {
const dispatch = useDispatch()
const { ids, data, loading, error } = useGetList(
	'albumSong', 
	{  }, 
	{ field: 'trackNumber', order: 'ASC' }, 
	{ album_id: props.id},
	)

	if (loading) {
	    return (
	      <IconButton>
	        <PlayIcon/>
	      </IconButton>
	    )
	}

	if (error) {
	    return (
	      <IconButton>
	        <PlayIcon/>
	      </IconButton>
	    )
	}

    return (
      <IconButton onClick={(e) => {
		  e.preventDefault()
		  e.stopPropagation()
		  dispatch(playAlbum(ids[0], data))
	  }}>
        <PlayIcon/>
      </IconButton>
    )
}
export default GridButton
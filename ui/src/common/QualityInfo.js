import React from 'react'
import { makeStyles } from '@material-ui/core/styles';
import Chip from '@material-ui/core/Chip';

const useStyles = makeStyles((theme) => ({
  
}));

export const QualityInfo = (props) => {
    const classes = useStyles();
    let {suffix, bitRate} = props.song ? props.song : props.record
    suffix = suffix.toUpperCase()
    const info = suffix +" "+ bitRate
    return <Chip className={classes.info} size="small" variant="outlined" label={info} />
}





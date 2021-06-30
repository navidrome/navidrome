import Grid from '@material-ui/core/Grid'
import Slider from '@material-ui/core/Slider'
import ViewModuleIcon from '@material-ui/icons/ViewModule'
import { makeStyles } from '@material-ui/core/styles'
import React, { useState } from 'react'
import { useDispatch } from 'react-redux'
import { incrementByAmount } from '../reducers/sliderSlice'

const useStyles = makeStyles((theme) => ({
  slidroot: {
    width: '19ch',
    margin: 10,
    [theme.breakpoints.down('sm')]: {
      display: 'none',
    },
  },
}))

export const Slide = () => {
  const classes = useStyles()
  const dispatch = useDispatch()
  const [value, setValue] = useState('30')
  const handleSliderChange = (event, newValue) => {
    setValue(newValue)
    dispatch(incrementByAmount(Number(value) || 0))
  }

  return (
    <div className={classes.slidroot}>
      <Grid container spacing={2} alignItems="center">
        <Grid item>
          <ViewModuleIcon />
        </Grid>
        <Grid item xs>
          <Slider
            value={typeof value === 'number' ? value : 0}
            onChange={handleSliderChange}
            aria-labelledby="input-slider"
          />
        </Grid>
      </Grid>
    </div>
  )
}

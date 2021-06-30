import { createSlice } from '@reduxjs/toolkit'

export const slice = createSlice({
  name: 'slider',
  initialState: {
    value: 0,
  },
  reducers: {
    incrementByAmount: (state, action) => {
      state.value = action.payload
    },
  },
})

export const { increment, decrement, incrementByAmount } = slice.actions

export const selectCount = (state) => state.slider.value

export default slice.reducer

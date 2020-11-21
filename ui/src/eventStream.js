import { baseUrl } from './utils'
import throttle from 'lodash.throttle'
import { processEvent, serverDown } from './actions'
import { httpClient } from './dataProvider'
import { REST_URL } from './consts'

const defaultIntervalCheck = 20000
const reconnectIntervalCheck = 2000
let currentIntervalCheck = reconnectIntervalCheck
let es = null
let dispatch = null
let timeout = null

const getEventStream = async () => {
  if (es === null) {
    return httpClient(`${REST_URL}/keepalive/`).then(() => {
      es = new EventSource(
        baseUrl(`/app/api/events?jwt=${localStorage.getItem('token')}`)
      )
      return es
    })
  }
  return es
}

// Reestablish the event stream after 20 secs of inactivity
const setTimeout = (value) => {
  currentIntervalCheck = value
  if (timeout != null) {
    window.clearTimeout(timeout)
  }
  timeout = window.setTimeout(() => {
    if (es != null) {
      es.close()
    }
    es = null
    startEventStream(dispatch)
  }, currentIntervalCheck)
}

const stopEventStream = () => {
  if (es) {
    es.close()
  }
  es = null
  if (timeout != null) {
    window.clearTimeout(timeout)
  }
  timeout = null
}

const setDispatch = (dispatchFunc) => {
  dispatch = dispatchFunc
}

const startEventStream = async () => {
  setTimeout(currentIntervalCheck)
  if (!localStorage.getItem('token')) {
    console.log('Cannot create a unauthenticated EventSource connection')
    return
  }
  getEventStream().then((newStream) => {
    newStream.onmessage = throttle(
      (msg) => {
        const data = JSON.parse(msg.data)
        if (data.name !== 'keepAlive') {
          dispatch(processEvent(data))
        }
        setTimeout(defaultIntervalCheck) // Reset timeout on every received message
      },
      100,
      { trailing: true }
    )
    newStream.onerror = (e) => {
      setTimeout(reconnectIntervalCheck)
      dispatch(serverDown())
    }
    es = newStream
  })
}

export { setDispatch, startEventStream, stopEventStream }

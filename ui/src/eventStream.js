import { baseUrl } from './utils'
import throttle from 'lodash.throttle'
import { processEvent, serverDown, streamReconnected } from './actions'
import { REST_URL } from './consts'
import config from './config'

const newEventStream = async () => {
  let url = baseUrl(`${REST_URL}/events`)
  if (localStorage.getItem('token')) {
    url = url + `?jwt=${localStorage.getItem('token')}`
  }
  return new EventSource(url)
}

let eventStream
let reconnectTimer
const RECONNECT_DELAY = 5000

const setupHandlers = (stream, dispatchFn) => {
  stream.addEventListener('serverStart', eventHandler(dispatchFn))
  stream.addEventListener('scanStatus', throttledEventHandler(dispatchFn))
  stream.addEventListener('refreshResource', eventHandler(dispatchFn))
  if (config.enableNowPlaying) {
    stream.addEventListener('nowPlayingCount', eventHandler(dispatchFn))
  }
  stream.addEventListener('keepAlive', eventHandler(dispatchFn))
  stream.onerror = (e) => {
    // eslint-disable-next-line no-console
    console.log('EventStream error', e)
    dispatchFn(serverDown())
    if (stream) stream.close()
    scheduleReconnect(dispatchFn)
  }
}

const scheduleReconnect = (dispatchFn) => {
  if (!reconnectTimer) {
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      connect(dispatchFn)
    }, RECONNECT_DELAY)
  }
}

const connect = async (dispatchFn) => {
  try {
    const stream = await newEventStream()
    eventStream = stream
    setupHandlers(stream, dispatchFn)
    // Dispatch reconnection event to refresh critical data
    dispatchFn(streamReconnected())
    return stream
  } catch (e) {
    // eslint-disable-next-line no-console
    console.log(`Error connecting to server:`, e)
    scheduleReconnect(dispatchFn)
  }
}

const eventHandler = (dispatchFn) => (event) => {
  const data = JSON.parse(event.data)
  if (event.type !== 'keepAlive') {
    dispatchFn(processEvent(event.type, data))
  }
}

const throttledEventHandler = (dispatchFn) =>
  throttle(eventHandler(dispatchFn), 100, { trailing: true })

const startEventStreamLegacy = async (dispatchFn) => {
  return newEventStream()
    .then((newStream) => {
      newStream.addEventListener('serverStart', eventHandler(dispatchFn))
      newStream.addEventListener(
        'scanStatus',
        throttledEventHandler(dispatchFn),
      )
      newStream.addEventListener('refreshResource', eventHandler(dispatchFn))
      if (config.enableNowPlaying) {
        newStream.addEventListener('nowPlayingCount', eventHandler(dispatchFn))
      }
      newStream.addEventListener('keepAlive', eventHandler(dispatchFn))
      newStream.onerror = (e) => {
        // eslint-disable-next-line no-console
        console.log('EventStream error', e)
        dispatchFn(serverDown())
      }
      return newStream
    })
    .catch((e) => {
      // eslint-disable-next-line no-console
      console.log(`Error connecting to server:`, e)
    })
}

const startEventStreamNew = async (dispatchFn) => {
  if (eventStream) {
    eventStream.close()
    eventStream = null
  }
  return connect(dispatchFn)
}

const startEventStream = async (dispatchFn) => {
  if (!localStorage.getItem('is-authenticated')) {
    return Promise.resolve()
  }
  if (config.devNewEventStream) {
    return startEventStreamNew(dispatchFn)
  }
  return startEventStreamLegacy(dispatchFn)
}

export { startEventStream }

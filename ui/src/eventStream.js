import { baseUrl } from './utils'
import throttle from 'lodash.throttle'
import { processEvent, serverDown } from './actions'
import { REST_URL } from './consts'
import config from './config'

const newEventStream = async () => {
  let url = baseUrl(`${REST_URL}/events`)
  if (localStorage.getItem('token')) {
    url = url + `?jwt=${localStorage.getItem('token')}`
  }
  return new EventSource(url)
}

const eventHandler = (dispatchFn) => (event) => {
  const data = JSON.parse(event.data)
  if (event.type !== 'keepAlive') {
    dispatchFn(processEvent(event.type, data))
  }
}

const throttledEventHandler = (dispatchFn) =>
  throttle(eventHandler(dispatchFn), 100, { trailing: true })

const startEventStream = async (dispatchFn) => {
  if (!localStorage.getItem('is-authenticated')) {
    return Promise.resolve()
  }
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

export { startEventStream }

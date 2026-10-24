// jest-dom adds custom jest matchers for asserting on DOM nodes.
// allows you to do things like:
// expect(element).toHaveTextContent(/react/i)
// learn more: https://github.com/testing-library/jest-dom
import '@testing-library/jest-dom'

const localStorageMock = (function () {
  let store = {}

  return {
    getItem: function (key) {
      return store[key] || null
    },
    setItem: function (key, value) {
      store[key] = value.toString()
    },
    removeItem: function (key) {
      delete store[key]
    },
    clear: function () {
      store = {}
    },
  }
})()

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
})

// jsdom has no Web Animations API. The lyrics timeline schedules native
// animations, so record them on the target for assertions instead of running
// a fake timeline.
if (typeof Element !== 'undefined' && !Element.prototype.animate) {
  Element.prototype.animate = function animate(keyframes, options) {
    const animation = {
      keyframes,
      options,
      currentTime: 0,
      playbackRate: 1,
      playState: 'idle',
      play() {
        this.playState = 'running'
      },
      pause() {
        this.playState = 'paused'
      },
      cancel() {
        this.playState = 'idle'
      },
    }
    if (!this.__animations)
      Object.defineProperty(this, '__animations', { value: [] })
    this.__animations.push(animation)
    return animation
  }
}

localStorage.setItem('username', 'admin')

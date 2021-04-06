import { applyMiddleware, combineReducers, compose, createStore } from 'redux'
import { routerMiddleware, connectRouter } from 'connected-react-router'
import createSagaMiddleware from 'redux-saga'
import { all, fork } from 'redux-saga/effects'
import { adminReducer, adminSaga, USER_LOGOUT } from 'react-admin'
import throttle from 'lodash.throttle'
import pick from 'lodash.pick'
import { loadState, saveState } from './persistState'

export default ({
  authProvider,
  dataProvider,
  history,
  customReducers = {},
}) => {
  const reducer = combineReducers({
    admin: adminReducer,
    router: connectRouter(history),
    ...customReducers,
  })
  const resettableAppReducer = (state, action) =>
    reducer(action.type !== USER_LOGOUT ? state : undefined, action)

  const saga = function* rootSaga() {
    yield all([adminSaga(dataProvider, authProvider)].map(fork))
  }
  const sagaMiddleware = createSagaMiddleware()

  const composeEnhancers =
    (process.env.NODE_ENV === 'development' &&
      typeof window !== 'undefined' &&
      window.__REDUX_DEVTOOLS_EXTENSION_COMPOSE__ &&
      window.__REDUX_DEVTOOLS_EXTENSION_COMPOSE__({
        trace: true,
        traceLimit: 25,
      })) ||
    compose

  const persistedState = loadState()
  const store = createStore(
    resettableAppReducer,
    persistedState,
    composeEnhancers(applyMiddleware(sagaMiddleware, routerMiddleware(history)))
  )

  store.subscribe(
    throttle(() => {
      const state = store.getState()
      saveState({
        theme: state.theme,
        queue: pick(state.queue, ['queue', 'volume']),
        albumView: state.albumView,
        settings: state.settings,
        visualizer: state.visualizer,
      })
    }),
    1000
  )

  sagaMiddleware.run(saga)
  return store
}

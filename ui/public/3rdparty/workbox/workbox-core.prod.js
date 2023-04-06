;(this.workbox = this.workbox || {}),
  (this.workbox.core = (function (t) {
    'use strict'
    try {
      self['workbox:core:6.2.4'] && _()
    } catch (t) {}
    const e = (t, ...e) => {
      let n = t
      return e.length > 0 && (n += ' :: ' + JSON.stringify(e)), n
    }
    class n extends Error {
      constructor(t, n) {
        super(e(t, n)), (this.name = t), (this.details = n)
      }
    }
    const r = new Set()
    const o = {
        googleAnalytics: 'googleAnalytics',
        precache: 'precache-v2',
        prefix: 'workbox',
        runtime: 'runtime',
        suffix: 'undefined' != typeof registration ? registration.scope : '',
      },
      s = (t) =>
        [o.prefix, t, o.suffix].filter((t) => t && t.length > 0).join('-'),
      i = {
        updateDetails: (t) => {
          ;((t) => {
            for (const e of Object.keys(o)) t(e)
          })((e) => {
            'string' == typeof t[e] && (o[e] = t[e])
          })
        },
        getGoogleAnalyticsName: (t) => t || s(o.googleAnalytics),
        getPrecacheName: (t) => t || s(o.precache),
        getPrefix: () => o.prefix,
        getRuntimeName: (t) => t || s(o.runtime),
        getSuffix: () => o.suffix,
      }
    function c(t, e) {
      const n = new URL(t)
      for (const t of e) n.searchParams.delete(t)
      return n.href
    }
    let a, u
    function f() {
      if (void 0 === u) {
        const t = new Response('')
        if ('body' in t)
          try {
            new Response(t.body), (u = !0)
          } catch (t) {
            u = !1
          }
        u = !1
      }
      return u
    }
    function l(t) {
      return new Promise((e) => setTimeout(e, t))
    }
    var g = Object.freeze({
      __proto__: null,
      assert: null,
      cacheMatchIgnoreParams: async function (t, e, n, r) {
        const o = c(e.url, n)
        if (e.url === o) return t.match(e, r)
        const s = Object.assign(Object.assign({}, r), { ignoreSearch: !0 }),
          i = await t.keys(e, s)
        for (const e of i) {
          if (o === c(e.url, n)) return t.match(e, r)
        }
      },
      cacheNames: i,
      canConstructReadableStream: function () {
        if (void 0 === a)
          try {
            new ReadableStream({ start() {} }), (a = !0)
          } catch (t) {
            a = !1
          }
        return a
      },
      canConstructResponseFromBodyStream: f,
      dontWaitFor: function (t) {
        t.then(() => {})
      },
      Deferred: class {
        constructor() {
          this.promise = new Promise((t, e) => {
            ;(this.resolve = t), (this.reject = e)
          })
        }
      },
      executeQuotaErrorCallbacks: async function () {
        for (const t of r) await t()
      },
      getFriendlyURL: (t) =>
        new URL(String(t), location.href).href.replace(
          new RegExp('^' + location.origin),
          ''
        ),
      logger: null,
      resultingClientExists: async function (t) {
        if (!t) return
        let e = await self.clients.matchAll({ type: 'window' })
        const n = new Set(e.map((t) => t.id))
        let r
        const o = performance.now()
        for (
          ;
          performance.now() - o < 2e3 &&
          ((e = await self.clients.matchAll({ type: 'window' })),
          (r = e.find((e) => (t ? e.id === t : !n.has(e.id)))),
          !r);

        )
          await l(100)
        return r
      },
      timeout: l,
      waitUntil: function (t, e) {
        const n = e()
        return t.waitUntil(n), n
      },
      WorkboxError: n,
    })
    const w = {
      get googleAnalytics() {
        return i.getGoogleAnalyticsName()
      },
      get precache() {
        return i.getPrecacheName()
      },
      get prefix() {
        return i.getPrefix()
      },
      get runtime() {
        return i.getRuntimeName()
      },
      get suffix() {
        return i.getSuffix()
      },
    }
    return (
      (t._private = g),
      (t.cacheNames = w),
      (t.clientsClaim = function () {
        self.addEventListener('activate', () => self.clients.claim())
      }),
      (t.copyResponse = async function (t, e) {
        let r = null
        if (t.url) {
          r = new URL(t.url).origin
        }
        if (r !== self.location.origin)
          throw new n('cross-origin-copy-response', { origin: r })
        const o = t.clone(),
          s = {
            headers: new Headers(o.headers),
            status: o.status,
            statusText: o.statusText,
          },
          i = e ? e(s) : s,
          c = f() ? o.body : await o.blob()
        return new Response(c, i)
      }),
      (t.registerQuotaErrorCallback = function (t) {
        r.add(t)
      }),
      (t.setCacheNameDetails = function (t) {
        i.updateDetails(t)
      }),
      (t.skipWaiting = function () {
        self.skipWaiting()
      }),
      t
    )
  })({}))
//# sourceMappingURL=workbox-core.prod.js.map

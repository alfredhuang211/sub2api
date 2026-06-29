/// <reference types="vite/client" />

import 'vue-router'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    public?: boolean
    roles?: Array<'base_admin' | 'admin' | 'agent'>
  }
}

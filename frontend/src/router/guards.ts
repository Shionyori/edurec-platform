import type { RouteLocationNormalized } from 'vue-router'

export interface AuthLike {
  isLoggedIn: boolean
  isAdmin: boolean
  user: unknown
  fetchMe: () => Promise<unknown>
  logout: () => void
}

export async function resolveGuard(
  to: RouteLocationNormalized,
  auth: AuthLike,
): Promise<true | { name: string; query?: { redirect?: string } }> {
  if (to.meta.requiresAuth) {
    if (!auth.isLoggedIn) {
      return { name: 'login', query: { redirect: to.fullPath } }
    }
    if (!auth.user) {
      try {
        await auth.fetchMe()
      } catch {
        auth.logout()
        return { name: 'login' }
      }
      if (!auth.isLoggedIn) {
        return { name: 'login' }
      }
    }
    if (to.meta.requiresAdmin && !auth.isAdmin) {
      return { name: 'home' }
    }
  }
  if (to.meta.guestOnly && auth.isLoggedIn) {
    return { name: 'home' }
  }
  return true
}

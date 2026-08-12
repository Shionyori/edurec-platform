const ACCESS_KEY = 'edurec_access_token'
const REFRESH_KEY = 'edurec_refresh_token'

export const tokenStorage = {
  getAccess(): string {
    return localStorage.getItem(ACCESS_KEY) ?? ''
  },
  setAccess(token: string): void {
    localStorage.setItem(ACCESS_KEY, token)
  },
  getRefresh(): string {
    return localStorage.getItem(REFRESH_KEY) ?? ''
  },
  setRefresh(token: string): void {
    localStorage.setItem(REFRESH_KEY, token)
  },
  clear(): void {
    localStorage.removeItem(ACCESS_KEY)
    localStorage.removeItem(REFRESH_KEY)
  },
}

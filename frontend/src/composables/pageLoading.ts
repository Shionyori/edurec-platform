import { readonly, ref } from 'vue'

/**
 * A tiny global page-loading coordinator.
 *
 * The router prefetches the destination page's primary data before it commits
 * the navigation, so the old page normally stays still under this mask while
 * the next page is being prepared. The mask also covers the fallback window
 * when prefetch fails or a page has secondary requests of its own.
 *
 * Rules:
 *  - show immediately on navigation;
 *  - keep visible while the new page has in-flight requests;
 *  - keep a short minimum so fast navigations do not flash;
 *  - never stay longer than MAX_VISIBLE_MS (a failed/crawling request must not trap the user).
 */

const MIN_VISIBLE_MS = 160
const SETTLE_MS = 40
const MAX_VISIBLE_MS = 8000

const visible = ref(false)

let navigating = false
let busy = 0
let navToken = 0
let shownAt = 0
let hideTimer: ReturnType<typeof setTimeout> | null = null
let maxTimer: ReturnType<typeof setTimeout> | null = null

function clearHideTimer() {
  if (hideTimer) {
    clearTimeout(hideTimer)
    hideTimer = null
  }
}

function clearMaxTimer() {
  if (maxTimer) {
    clearTimeout(maxTimer)
    maxTimer = null
  }
}

function scheduleHide() {
  if (!visible.value || navigating || busy > 0) return
  clearHideTimer()
  const remaining = Math.max(0, MIN_VISIBLE_MS - (Date.now() - shownAt)) + SETTLE_MS
  hideTimer = setTimeout(() => {
    hideTimer = null
    if (navigating || busy > 0) return
    visible.value = false
    clearMaxTimer()
  }, remaining)
}

export const pageLoading = {
  visible: readonly(visible),

  /** Called from router.beforeEach; returns a token for the matching afterEach. */
  startNavigation(): number {
    navToken += 1
    navigating = true
    clearHideTimer()
    clearMaxTimer()
    if (!visible.value) {
      visible.value = true
      shownAt = Date.now()
    }
    maxTimer = setTimeout(() => {
      navigating = false
      visible.value = false
    }, MAX_VISIBLE_MS)
    return navToken
  },

  /** Called from router.afterEach; stale tokens from redirects are ignored. */
  endNavigation(token?: number) {
    if (token !== undefined && token !== navToken) return
    navigating = false
    scheduleHide()
  },

  beginRequest() {
    busy += 1
    clearHideTimer()
  },

  endRequest() {
    busy = Math.max(0, busy - 1)
    scheduleHide()
  },
}

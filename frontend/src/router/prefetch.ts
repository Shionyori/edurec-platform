import type { RouteLocationNormalized } from 'vue-router'
import { startPrefetchSession, stopPrefetchSession } from '@/api/client'
import { getRecommendations } from '@/api/recommendation'
import { listCategories } from '@/api/category'
import { getDiscipline, listDisciplines } from '@/api/discipline'
import { getKnowledgePoint } from '@/api/knowledge'
import { getLearningProfile, getLearningPath, getPathResources } from '@/api/learning'
import { listRatings } from '@/api/rating'
import { getResource, listResources } from '@/api/resource'
import { getMe } from '@/api/user'
import { listFavorites, listMyBehaviors } from '@/api/behavior'
import {
  adminGetRecommendationStats,
  adminListRecommendationRuns,
  adminListResources,
  adminListUsers,
} from '@/api/admin'

// The old page stays on screen while these run. Keep the window short: a
// slow backend should still let the new page paint its own skeleton.
const PREFETCH_TIMEOUT_MS = 2600

function queryValue(to: RouteLocationNormalized, key: string): string {
  const raw = to.query[key]
  const value = Array.isArray(raw) ? raw[0] : raw
  return typeof value === 'string' ? value.trim() : ''
}

function numericParam(value: unknown): number | null {
  const id = Number(value)
  return Number.isFinite(id) && id > 0 ? id : null
}

/** Primary GETs for the destination page; the page consumes them from the one-shot cache. */
function buildTasks(to: RouteLocationNormalized): Promise<unknown>[] {
  switch (String(to.name ?? '')) {
    case 'home':
      return [getRecommendations(12), listCategories(), listDisciplines()]

    case 'search': {
      const keyword = queryValue(to, 'keyword')
      const type = queryValue(to, 'type')
      return [
        listCategories(),
        listResources({
          page: 1,
          page_size: 12,
          keyword: keyword || undefined,
          type: type || undefined,
          sort: 'latest',
        }),
      ]
    }

    case 'knowledge':
      return [listDisciplines()]

    case 'discipline': {
      const code = String(to.params.code ?? '')
      return code ? [getDiscipline(code, 12)] : []
    }

    case 'knowledge-detail': {
      const code = String(to.params.code ?? '')
      return code ? [getKnowledgePoint(code, 12)] : []
    }

    case 'learning-path': {
      const id = numericParam(to.params.id)
      return id ? [getLearningPath(id), getPathResources(id, 9)] : []
    }

    case 'questionnaire':
      return [getLearningProfile()]

    case 'user-me':
      return [
        getMe(),
        listFavorites({ page: 1, page_size: 12 }),
        listMyBehaviors({ page: 1, page_size: 10 }),
      ]

    case 'resource-detail': {
      const id = numericParam(to.params.id)
      return id ? [getResource(id), listRatings(id, { page: 1, page_size: 10 })] : []
    }

    case 'resource-upload':
      return [listCategories()]

    case 'admin-dashboard':
      return [
        adminListResources({ page: 1, page_size: 1 }),
        adminListUsers({ page: 1, page_size: 1 }),
      ]

    case 'admin-resources':
      return [adminListResources({ page: 1, page_size: 20 })]

    case 'admin-users':
      return [adminListUsers({ page: 1, page_size: 20 })]

    case 'admin-categories':
      return [listCategories()]

    case 'admin-recommendations':
      return [adminGetRecommendationStats(), adminListRecommendationRuns(20)]

    default:
      return []
  }
}

/**
 * Keep the current page visible until the destination's primary data is ready.
 * Failures and a slow backend are non-blocking: the router proceeds and the new
 * page falls back to its normal skeleton/error states.
 */
export async function prefetchRoute(to: RouteLocationNormalized): Promise<void> {
  const session = startPrefetchSession()
  try {
    const tasks = buildTasks(to)
    if (tasks.length === 0) return
    const all = Promise.all(tasks.map((task) => task.catch(() => null)))
    await new Promise<void>((resolve) => {
      const timer = setTimeout(resolve, PREFETCH_TIMEOUT_MS)
      void all.finally(() => {
        clearTimeout(timer)
        resolve()
      })
    })
  } finally {
    stopPrefetchSession(session)
  }
}

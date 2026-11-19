import { get, post } from './client'
import type { Discipline, DisciplineCrawlStatus, DisciplineDetail } from '@/types'

/** 全部学科（主页面） */
export function listDisciplines() {
  return get<Discipline[]>('/disciplines')
}

/** 学科详情：知识点 + 资源 */
export function getDiscipline(code: string, limit = 12) {
  return get<DisciplineDetail>(`/disciplines/${encodeURIComponent(code)}`, {
    params: { limit },
  })
}

/** 学科无资源时触发自动爬取 */
export function startDisciplineCrawl(code: string) {
  return post<DisciplineCrawlStatus>(`/disciplines/${encodeURIComponent(code)}/crawl`, undefined, { skipPageLoading: true })
}

/** 查询自动爬取状态 */
export function getDisciplineCrawlStatus(code: string) {
  return get<DisciplineCrawlStatus>(`/disciplines/${encodeURIComponent(code)}/crawl`, { skipPageLoading: true })
}

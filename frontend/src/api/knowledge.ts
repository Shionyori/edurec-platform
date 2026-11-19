import { get, post } from './client'
import type {
  GenerationStatus,
  KeywordGenerationResult,
  KnowledgeDetail,
  KnowledgeNode,
  Page,
  Resource,
} from '@/types'

/** 知识点（学科级）列表 */
export function listKnowledgePoints() {
  return get<KnowledgeNode[]>('/knowledge-points')
}

/** 知识点详情：学科先修图 + 概念子图 + 资源 */
export function getKnowledgePoint(code: string, limit = 12) {
  return get<KnowledgeDetail>(`/knowledge-points/${encodeURIComponent(code)}`, {
    params: { limit },
  })
}

/** 知识点下的资源分页 */
export function listKnowledgeResources(code: string, page = 1, pageSize = 12) {
  return get<Page<Resource> & { knowledge_point: KnowledgeNode }>(
    `/knowledge-points/${encodeURIComponent(code)}/resources`,
    { params: { page, page_size: pageSize } },
  )
}

/** 触发某个学科的概念生成（调用 engine，异步执行） */
export function startConceptGeneration(code: string) {
  return post<GenerationStatus>(`/knowledge-points/${encodeURIComponent(code)}/generate`, undefined, { skipPageLoading: true })
}

/** 查询概念生成状态 */
export function getConceptGenerationStatus(code: string) {
  return get<GenerationStatus>(`/knowledge-points/${encodeURIComponent(code)}/generate`, { skipPageLoading: true })
}

/** 搜索栏关键词触发：后端匹配最近的知识点并开始生成 */
export function startGenerationByKeyword(keyword: string) {
  return post<KeywordGenerationResult>('/knowledge-generation', { keyword }, { skipPageLoading: true })
}

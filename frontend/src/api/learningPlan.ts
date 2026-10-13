import { get } from './client'
import type { LearningPlan } from '@/types'

// 学习目标 → 知识点规划。goal 的空值与长度校验在后端进行（400），
// 前端只做即时提示，不作为唯一防线。
export function getLearningPlan(goal: string, knowledgePoints = 8) {
  return get<LearningPlan>('/learning-plan', {
    params: { goal, knowledge_points: knowledgePoints },
  })
}

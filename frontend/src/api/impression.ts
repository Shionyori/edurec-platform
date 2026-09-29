import { post } from './client'

// 曝光场景，与后端 ResourceImpression.Scene 对应
export type ImpressionScene = 'home' | 'search' | 'detail'

// 上报一次推荐展示（批量），由列表渲染后调用。
// 曝光是 CTR 的分母（CTR = click / impression），也是推荐评估与负采样的数据来源；
// 它与 user_behaviors 分开存储，不参与正样本训练。空列表直接跳过，不发请求。
export function recordImpressions(scene: ImpressionScene, resourceIds: number[]) {
  if (resourceIds.length === 0) return Promise.resolve(null)
  return post<null>('/impressions', { scene, resource_ids: resourceIds })
}

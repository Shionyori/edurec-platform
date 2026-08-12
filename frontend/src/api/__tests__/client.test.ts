import { describe, it, expect } from 'vitest'
import { unwrap } from '../client'

describe('client 统一响应解包', () => {
  it('code === 0 时返回 data', () => {
    expect(unwrap({ code: 0, message: 'ok', data: { id: 1 } })).toEqual({ id: 1 })
  })

  it('code !== 0 时抛出业务错误', () => {
    expect(() => unwrap({ code: 10001, message: '参数错误', data: null })).toThrow('参数错误')
  })
})

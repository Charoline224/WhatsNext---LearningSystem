import { describe, expect, it } from 'vitest'
import type { KnowledgeArticle } from '@/types/learning-asset'
import type { ExamPattern } from '@/types/exam'
import { legacyExamKnowledgeArticleBody, resolveKnowledgeArticleBody } from './handbook'

const article: KnowledgeArticle = {
  id: 'article-1',
  node_id: 'node-1',
  title: 'TCP 流量控制',
  body: legacyExamKnowledgeArticleBody,
  source_chunk_id: 'chunk-1',
  sort_order: 0,
  user_edited: false,
}

const pattern: ExamPattern = {
  id: 'pattern-1',
  pattern_key: 'tcp_sliding_window',
  title: 'TCP 滑动窗口与流量控制',
  description: '考察发送窗口、接收窗口与确认机制的联动关系。',
  tested_knowledge: 'rwnd、cwnd、可用窗口和累计确认',
  common_mistakes: '混淆流量控制与拥塞控制。',
  solving_strategy: '画出字节序号区间并标注各区域。',
  occurrence_count: 2,
  frequency_level: 'medium',
  related_nodes: [
    { node_id: 'node-1', node_name: 'TCP 流量控制', confidence: 0.9, relation_reason: '考察该知识' },
  ],
}

describe('resolveKnowledgeArticleBody', () => {
  it('为历史真题占位章节补充知识点概括', () => {
    const body = resolveKnowledgeArticleBody(article, [pattern])
    expect(body).toContain('## 知识点概括')
    expect(body).toContain('rwnd、cwnd、可用窗口和累计确认')
    expect(body).toContain('TCP 滑动窗口与流量控制')
    expect(body).toContain('画出字节序号区间并标注各区域。')
  })

  it('不改写正常手册正文', () => {
    const authoredArticle = { ...article, body: '这是已生成或人工编辑的详细内容。' }
    expect(resolveKnowledgeArticleBody(authoredArticle, [pattern])).toBe(authoredArticle.body)
  })
})

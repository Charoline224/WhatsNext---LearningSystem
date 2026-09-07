import type { KnowledgeArticle } from '@/types/learning-asset'
import type { ExamPattern } from '@/types/exam'

export const legacyExamKnowledgeArticleBody =
  '该知识点由真题题型分析识别，建议结合题型手册和原始资料完成理解与练习。'

export function resolveKnowledgeArticleBody(article: KnowledgeArticle, patterns: ExamPattern[]) {
  if (article.body.trim() !== legacyExamKnowledgeArticleBody) return article.body

  const relatedPatterns = patterns.filter((pattern) =>
    pattern.related_nodes.some((node) => node.node_id === article.node_id),
  )
  if (!relatedPatterns.length) return article.body

  const summaries = [...new Set(relatedPatterns.map((pattern) => pattern.tested_knowledge.trim()).filter(Boolean))]
  const overview = summaries.length
    ? summaries.join('；')
    : relatedPatterns.map((pattern) => pattern.description.trim()).filter(Boolean).join('；')

  const examUses = relatedPatterns
    .map((pattern) => {
      const description = pattern.description.trim() || '结合具体情境检验对该知识点的理解和应用。'
      return `- **${pattern.title}**：${description}`
    })
    .join('\n')

  const strategies = [
    ...new Set(relatedPatterns.map((pattern) => pattern.solving_strategy.trim()).filter(Boolean)),
  ]

  return [
    '## 知识点概括',
    overview || '该知识点是相关题型的核心理论基础，需要理解定义、适用条件与关键机制。',
    '## 在真题中如何考察',
    examUses,
    '## 理解与应用',
    strategies.join('；') || '先明确核心概念和适用条件，再结合题干信息建立完整的推理过程。',
  ].join('\n\n')
}

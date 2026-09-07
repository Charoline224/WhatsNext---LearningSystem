import MarkdownIt from 'markdown-it'
import texmath from 'markdown-it-texmath'
import katex from 'katex'
import 'katex/dist/katex.min.css'

const markdown = new MarkdownIt({
  html: false,
  linkify: true,
  typographer: true,
  breaks: true,
})

markdown.use(texmath, {
  engine: katex,
  delimiters: 'dollars',
  katexOptions: { throwOnError: false, strict: false },
})

function normalizeExponentGroups(formula: string) {
  let result = ''
  for (let index = 0; index < formula.length; index += 1) {
    if (formula[index] !== '^' || formula[index + 1] !== '(') {
      result += formula[index]
      continue
    }
    let depth = 1
    let end = index + 2
    for (; end < formula.length && depth > 0; end += 1) {
      if (formula[end] === '(') depth += 1
      if (formula[end] === ')') depth -= 1
    }
    if (depth !== 0) {
      result += formula[index]
      continue
    }
    const exponent = formula.slice(index + 2, end - 1)
    result += `^{${normalizeExponentGroups(exponent)}}`
    index = end - 1
  }
  return result
}

function normalizeFormula(formula: string) {
  return normalizeExponentGroups(formula)
    .replace(/∫/g, '\\int ')
    .replace(/_([A-Za-z0-9]+)/g, '_{$1}')
    .replace(/(^|[^A-Za-z])d([A-Za-z])\b/g, '$1\\,d$2')
    .replace(/\s+/g, ' ')
    .trim()
}

function looksLikeBareFormula(line: string) {
  const value = line.trim()
  if (!value || /^(#{1,6}|[-*+]\s|\d+\.\s|>|```)/.test(value)) return false
  if (value.includes('$') || value.includes('\\(') || value.includes('\\[')) return false
  return value.includes('=') && /[∫∑√∞^_]|\\(?:frac|int|sum|lim|sqrt)/.test(value)
}

export function normalizeMathMarkdown(source: string) {
  const normalizedDelimiters = source
    .replace(/\\\[([\s\S]*?)\\\]/g, (_, formula: string) => `\n\n$$\n${formula}\n$$\n\n`)
    .replace(/\\\((.+?)\\\)/g, (_, formula: string) => `$${formula}$`)

  return normalizedDelimiters
    .split('\n')
    .map((line) => (looksLikeBareFormula(line) ? `$$\n${normalizeFormula(line)}\n$$` : line))
    .join('\n')
}

export function renderMarkdown(source: string) {
  return markdown.render(normalizeMathMarkdown(source || ''))
}

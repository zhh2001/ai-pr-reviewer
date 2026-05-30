// markdown-it 这一层负责把 LLM 文本变成 HTML，且把 HTML 注入风险关在最小面：
//   - html: false 把内联 <script> / <img onerror> 这种 raw HTML 直接转义。
//   - 默认的 validateLink 已经把 javascript: / data: / vbscript: 拒之门外。
//   - breaks: true 让 DeepSeek 用换行分点的写法在前端能正确换行。
// 浏览器侧还会过一层 DOMPurify (见 markdown.js)；这是 defense-in-depth。
// 这个文件不引 DOMPurify，方便在 Node 环境跑 vitest 而无需 jsdom。
import MarkdownIt from 'markdown-it'

export const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
  typographer: false,
})

export function renderMarkdownUnsanitized(text) {
  if (text == null || text === '') return ''
  return md.render(String(text))
}

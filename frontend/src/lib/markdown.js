import DOMPurify from 'dompurify'
import { renderMarkdownUnsanitized } from './markdown-core.js'

// DOMPurify 走默认 html profile，但显式禁掉一些 markdown 用不到的元素。
const PURIFY_OPTS = {
  USE_PROFILES: { html: true },
  FORBID_TAGS: ['style', 'iframe', 'object', 'embed', 'script', 'form'],
  FORBID_ATTR: ['onerror', 'onload', 'onclick', 'onmouseover', 'style'],
}

// renderMarkdown: 渲染 LLM 返回的 markdown 为安全 HTML。
// 双层防御：markdown-it 已经在解析阶段把 raw HTML 转义、把 javascript: 链接剥光；
// DOMPurify 再扫一遍最终 HTML，作为运行时兜底。
export function renderMarkdown(text) {
  const html = renderMarkdownUnsanitized(text)
  if (typeof window === 'undefined') {
    // SSR / 测试 fallback：markdown-it 的 html:false 已经足够把 raw HTML 转义。
    return html
  }
  return DOMPurify.sanitize(html, PURIFY_OPTS)
}

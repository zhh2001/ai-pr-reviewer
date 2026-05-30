// splitLines: 把"上一次留下的不完整字节" + "新进来的字节"合在一起，按 '\n' 切出
// 已完整的行，剩下的不完整部分作为下次的 buffer 返回。
//   - 不丢空行：'\n\n' 切出两个空字符串行，让调用方自己决定要不要忽略。
//   - 兼容 CRLF：每个 '\n' 前若紧挨着 '\r'，把它一并剥掉。
//   - 末尾不含 '\n' 的字节统统留进 buffer，绝不当成"完整行"提前 parse。
// 纯函数：相同输入永远给相同输出，不依赖外部状态。
export function splitLines(prev, chunk) {
  const buf = (prev || '') + (chunk || '')
  if (buf.length === 0) return { lines: [], buffer: '' }

  const lines = []
  let start = 0
  for (let i = 0; i < buf.length; i++) {
    if (buf.charCodeAt(i) === 10 /* \n */) {
      let end = i
      if (end > start && buf.charCodeAt(end - 1) === 13 /* \r */) {
        end--
      }
      lines.push(buf.slice(start, end))
      start = i + 1
    }
  }
  return { lines, buffer: buf.slice(start) }
}

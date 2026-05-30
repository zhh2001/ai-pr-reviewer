import { splitLines } from './ndjson.js'

// readNDJSONStream: 把 Fetch Response.body 当作 NDJSON 流消费，每行 parse 成一个事件，
// 通过 onEvent 回调推出。
//   - 空行 / 全空白行 → 跳过
//   - 单行 JSON.parse 失败 → console.warn，继续读后续行（流不应被一条坏行打死）
//   - 网络错误 / reader 抛异常 → 直接 throw 出去，让调用方决定怎么提示
//   - 流读完后会把最后残留的 buffer 当成"最后一行"补 parse 一次（应对服务端没补
//     末尾 \n 的情况）
export async function readNDJSONStream(response, onEvent) {
  if (!response.body) {
    throw new Error('response has no body')
  }
  const reader = response.body.getReader()
  const decoder = new TextDecoder('utf-8')
  let buffer = ''
  try {
    while (true) {
      const { value, done } = await reader.read()
      if (done) break
      const chunk = decoder.decode(value, { stream: true })
      const next = splitLines(buffer, chunk)
      buffer = next.buffer
      for (const line of next.lines) {
        emitParsed(line, onEvent)
      }
    }
    const tail = (buffer + decoder.decode()).trim()
    if (tail) emitParsed(tail, onEvent)
  } finally {
    try { reader.releaseLock() } catch { /* ignore */ }
  }
}

function emitParsed(line, onEvent) {
  const trimmed = line.trim()
  if (!trimmed) return
  let obj
  try {
    obj = JSON.parse(trimmed)
  } catch (e) {
    console.warn('ndjson: skipping invalid line:', trimmed)
    return
  }
  onEvent(obj)
}

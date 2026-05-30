// prFilesUrl: PR 的 Files changed 页面。刻意不带行级锚点——GitHub 的 #diff-<sha>
// 锚点对仓库一旦 force-push 或合并就会失效，链到 /files 总是稳的。
export function prFilesUrl(owner, repo, number) {
  if (!owner || !repo || !number || number <= 0) return ''
  return `https://github.com/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}/pull/${Number(number)}/files`
}

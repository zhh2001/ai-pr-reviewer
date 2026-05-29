# ai-pr-reviewer

AI 辅助的 GitHub PR Review 工具：拉取指定 PR 的代码变更，用 LLM
（DeepSeek，走 OpenAI 兼容接口）产出变更总结、风险代码识别与 Review 建议。

> 本仓库目前是**脚手架**阶段：后端只有 `/healthz`，前端只有一个最小输入页面，
> 还没有真实的 GitHub / LLM 调用。

## 目录结构

```text
.
├── backend/                Go 后端
│   ├── cmd/server/         入口 main
│   └── internal/
│       ├── config/         环境变量装载
│       ├── httpserver/     HTTP 路由
│       ├── github/         go-github 封装（占位）
│       └── llm/            go-openai (DeepSeek) 封装（占位）
├── frontend/               Vue 3 + Vite
│   ├── index.html
│   └── src/
│       ├── main.js
│       └── App.vue
├── .env.example            需要配置的环境变量
└── README.md
```

## 环境变量

复制 `.env.example` 为 `.env`，按需填写：

- `GITHUB_TOKEN` —— 访问 GitHub API 用
- `DEEPSEEK_API_KEY` —— 调 DeepSeek（OpenAI 兼容）用
- `ADDR` —— 后端监听地址，默认 `:8080`
- `ANALYZE_TIMEOUT_SECONDS` —— summary/risks/suggestions 三个子任务并发跑的整体超时，默认 90

> 配置只走环境变量，不要把密钥写进代码或提交进仓库。

## 启动后端

需要 Go 1.22+。

```bash
cd backend
export GITHUB_TOKEN=...   # 或 source 一份 .env
export DEEPSEEK_API_KEY=...
go run ./cmd/server
```

健康检查：

```bash
curl http://localhost:8080/healthz
# {"status":"ok"}
```

## 启动前端

需要 Node 18+。

```bash
cd frontend
npm install
npm run dev
```

默认开在 <http://localhost:5173>，已通过 Vite 把 `/api/*` 代理到
`http://localhost:8080`，因此前端代码里直接 `fetch('/api/...')` 即可。

# ai-web2api-stack

把三个网页版 AI 服务反代成 OpenAI 兼容 API，用一份 Docker Compose 管理：

| 服务 | 端口 | 上游 | 说明 |
|---|---|---|---|
| `gemini-web2api` | 8083 | gemini.google.com | 文本 / 生图 / 视频，额外提供 `/v1/images/generations` |
| `chatgpt2api` | 3000 | chatgpt.com | 文本 / 图片，账号用 access_token |
| `deepseek-web2api` | 8090 | chat.deepseek.com | 文本，账号用 token + cookies |

三个服务都使用 `network_mode: host`，方便访问只监听 `127.0.0.1` 的本地代理。

## 目录结构

```
.
├── docker-compose.yml
├── .env.example              # 环境变量模板
├── build.sh                  # 编译 Gemini 自定义版并启动全部服务
├── cookie.example.txt        # Gemini Cookie 字段模板
├── src/                      # gemini-web2api 源码（含自定义生图接口）
├── chatgpt2api/              # chatgpt2api 源码
│   └── config.example.json
└── deepseek/                 # deepseek-web2api 源码
    └── Dockerfile.local
```

运行数据、Cookie、账号文件和 `.env` 都已被 `.gitignore` 排除，不会提交。

## 前置条件

- Docker + Docker Compose v2
- 能访问三个上游的网络（Gemini 通常需要代理）
- 各服务的账号凭证

## 快速开始

```bash
cp .env.example .env
# 编辑 .env，填入自己的 token / key / 密码

# Gemini Cookie（只保留这 7 个字段）
cp cookie.example.txt cookie.txt
# 编辑 cookie.txt

# chatgpt2api 配置
cp chatgpt2api/config.example.json chatgpt2api/config.json
# 按需修改 auth-key 和 proxy

# 编译 Gemini 自定义版并启动
./build.sh
```

如果不需要 Gemini 代理，删除 `docker-compose.yml` 中 `--proxy` 及其下一行。

## Gemini（8083）

- 面板：`http://127.0.0.1:8083/admin/`（注意末尾斜杠）
- 管理 API：`Authorization: Bearer <ADMIN_TOKEN>`
- Cookie 只保留：`SID / HSID / SSID / APISID / SAPISID / __Secure-1PSID / __Secure-1PSIDTS`
  - 多余字段（尤其 `OSID`、`__Secure-OSID`）会导致会话冲突
  - Chrome 151 的 DBSC 会让 `__Secure-1PSIDTS` 很快失效，建议用 Firefox 导出
- 免费账号生图额度按算力计，约 15 张 / 5 小时；文本模型免费号会自动降级
- 单 IP 突发请求过多会触发 Google `sorry` 拦截，建议控制在 10 次/分钟以内

### 自定义生图接口

在原有 Gemini 原生接口之外，`src/` 中新增了 OpenAI 风格的：

```bash
curl http://127.0.0.1:8083/v1/images/generations \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"prompt":"一只戴墨镜的猫","n":1,"response_format":"b64_json"}'
```

- 内部固定使用 `gemini-image`（Nano Banana）
- `n` 最多 4，串行请求上游
- `size / quality / style` 会被忽略
- 默认返回 `b64_json`，`response_format:"url"` 时返回的 `url` 实际是 data URL

## chatgpt2api（3000）

- 面板：`http://127.0.0.1:3000/`
- 鉴权：`Authorization: Bearer <CHATGPT2API_AUTH_KEY>`
- 账号只能导入 `access_token`，不支持网页 Cookie：

```bash
curl -X POST http://127.0.0.1:3000/api/accounts \
  -H "Authorization: Bearer $CHATGPT2API_AUTH_KEY" \
  -H "Content-Type: application/json" \
  -d '{"tokens":["<access_token>"]}'
```

access_token 从浏览器登录后访问 `https://chatgpt.com/api/auth/session` 获取，过期后需要重新导入。

- 免费账号生图约 4 张/天
- 出图接口：`/v1/images/generations`，模型 `gpt-image-2`

## DeepSeek（8090）

- WebUI：`http://127.0.0.1:8090/webui`，密码是 `.env` 里的 `DS_ADMIN_PASSWORD`
- 客户端 API Key：`.env` 里的 `DS_API_KEY`，或登录 WebUI 在「设置」页创建
- 账号通过 WebUI 账号池添加，或直接写 `deepseek/data/accounts.json`：

```json
{
  "version": 1,
  "accounts": [
    {
      "email": "account-label",
      "token": "<chat.deepseek.com 的 token>",
      "cookies": "<完整 Cookie 字符串>"
    }
  ]
}
```

- token 从浏览器 Console 执行 `localStorage.getItem('userToken')` 获取
- 网页反代有封号风险，避免高频并发调用
- 目前映射模型：`deepseek-chat`

## 通用调用

```bash
curl http://127.0.0.1:8083/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"3.5-flash","messages":[{"role":"user","content":"你好"}]}'
```

## 限制与风险

- 三个服务都是网页端反代，不是官方 API，上游改版或风控会导致不可用
- 账号额度、IP 限流、封号风险由上游决定，本项目不提供绕过
- 请只使用自己的账号，并遵守各平台服务条款

## 上游与许可

- [zexadev/gemini-web2api-go](https://github.com/zexadev/gemini-web2api-go) — MIT
- [basketikun/chatgpt2api](https://github.com/basketikun/chatgpt2api) — MIT
- [snake-aabb-wtf/deepseek-web2api-free](https://github.com/snake-aabb-wtf/deepseek-web2api-free) — Unlicense

各目录内保留原始 LICENSE。

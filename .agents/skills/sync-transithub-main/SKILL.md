---
name: sync-transithub-main
description: 将官方 deviseo/transit-hub 的 main 分支同步到本地 main；在同功能冲突中优先保留本地行为，完成冲突处理、项目影响分析、后端/前端/部署验证及中文更新摘要。用户要求拉取、同步、更新或合并 TransitHub 官方主线时使用。
---

# 同步 TransitHub 主线

## 目标

将官方仓库 `deviseo/transit-hub` 的 `main` 同步到当前 fork 的本地 `main`。固定使用 remote 名 `transithub`；`origin` 仍指向用户 fork，不把 `origin/main` 当作官方上游。

若上游与本地修改同一功能，以本地 `main` 的行为约定为准。仅吸收明确兼容且不改变本地行为的上游代码。

## 流程

在仓库根目录执行。

### 1. 预检

先检查仓库状态：

```bash
git rev-parse --show-toplevel
git status --short
git branch --show-current
git rev-parse --verify main
git remote -v
```

确认当前仓库确为 TransitHub，至少应存在 `backend/go.mod`、`frontend/package.json` 和 `deploy/`。

检查 Git 是否正处于 merge、rebase、cherry-pick 或 revert；若是，停止并报告现有操作。若本地 `main` 不存在，停止并询问应同步到哪个分支。

工作区有未提交改动时，不执行切分支或合并，也不擅自 stash、提交或丢弃改动；报告改动并请用户处理。禁止使用 `git reset --hard`、`git checkout -- <path>`、删分支或 force push。

检查 `transithub` remote：

```bash
git remote get-url transithub
```

- 若不存在，添加官方上游：`git remote add transithub https://github.com/deviseo/transit-hub.git`。
- 若存在，确认其 owner/repo 为 `deviseo/transit-hub`；SSH 与 HTTPS 形式均可。
- 若指向其他仓库，停止并报告实际 URL，不擅自覆盖。

### 2. 获取并判断分叉

获取官方主线：

```bash
git fetch transithub main
git rev-parse --short transithub/main
git rev-list --left-right --count main...transithub/main
git log --oneline --left-right --cherry-pick main...transithub/main
```

用 `rev-list` 结果说明本地 `main` 是领先、落后、已分叉或已同步。日志过长时只展示足够判断差异的近期提交。

将合并前本地主线保存为 `LOCAL_BEFORE=$(git rev-parse main)`，将已获取上游保存为 `UPSTREAM_HEAD=$(git rev-parse transithub/main)`。不得把当前其他分支的 `HEAD` 误记为 `LOCAL_BEFORE`。

### 3. 切换本地主线

工作区干净且当前分支不是 `main` 时执行：

```bash
git switch main
```

再次确认 `HEAD` 等于 `LOCAL_BEFORE`。

### 4. 合并官方主线

执行：

```bash
git merge --no-edit -X ours transithub/main
```

`-X ours` 仅令冲突文本块优先采用当前本地 `main`，并非丢弃全部上游改动。记录合并输出，以区分快进、already up-to-date、普通 merge commit 及冲突合并。

本 skill 只同步本地分支；不得自动 push。

### 5. 处理冲突

若仍有冲突，先列出：

```bash
git diff --name-only --diff-filter=U
rg -n '^(<<<<<<<|=======|>>>>>>>)' .
```

逐文件比较 stage 2（本地）与 stage 3（上游），按以下规则处理：

- 两边修改无关区域：合并两者，标记为“无关改动兼容合并”。
- 两边修改同一功能：保留本地行为，仅补入明确兼容的上游注释、测试、翻译、import 或邻近辅助代码，标记为“本地主线优先”。
- rename/delete、二进制、来源不明的生成物、迁移顺序、接口/数据结构/依赖联动或大规模结构调整：不得猜测，标记为“需决策”。

“需决策”时保持冲突现场并报告：文件、本地意图、上游意图、直接保留本地为何仍有风险，以及 2-3 个具体方案；优先推荐能保留本地行为者。

若冲突均已安全解决，仅暂存实际冲突文件并完成 merge commit：

```bash
git add <已解决文件>
git commit --no-edit
```

不得暂存无关文件。完成后再次扫描冲突标记，并运行 `git diff --check "$LOCAL_BEFORE"..HEAD`。

### 6. 分析同步内容

先取得事实：

```bash
git diff --stat "$LOCAL_BEFORE"..HEAD
git diff --name-status "$LOCAL_BEFORE"..HEAD
git log --oneline --no-merges "$LOCAL_BEFORE"..HEAD
```

若 codebase-memory-mcp 可用，优先用 `detect_changes` 分析 `LOCAL_BEFORE` 至当前工作树的影响；索引缺失或过期时先索引仓库。需要查调用关系时用 `search_graph`、`trace_path`、`get_code_snippet`，不要先用文本搜索猜测。字符串、配置及冲突标记仍可用 `rg`。

对重要文件读取代表性 diff，不得仅凭文件名推断：

```bash
git diff "$LOCAL_BEFORE"..HEAD -- <重要路径>
```

按本项目领域归纳：

- 后端入口与基础设施：`backend/cmd/api/`、`backend/internal/config/`、`database/`、`httpserver/`。
- 后端业务：`admin_accounts`、`auth`、`users`、`dashboard`、`group_rates`、`group_rate_campaigns`、`my_sites`、`upstream`、`connection_health`、`tickets`、`settings`、`system`、`health`。
- 前端：`frontend/src/modules/admin/`、`auth/`、`embed/tickets/`、`home/`，以及共享组件、`router.ts`、`i18n.ts`、`locales/`。
- 依赖与构建：`backend/go.mod`、`backend/go.sum`、`frontend/package.json`、`frontend/package-lock.json`、Vite/TypeScript/Tailwind 配置。
- 部署与文档：`deploy/`、`.github/workflows/`、环境变量示例、README 与 docs。

重点指出用户可见行为、API 契约、鉴权/会话、安全、PostgreSQL 迁移、Redis 缓存/调度、配置/环境变量、依赖升级、路由、交互及中英文文案变化。

### 7. 按改动验证

合并改到对应区域时运行：

- 后端 Go、`go.mod` 或 `go.sum`：

  ```bash
  cd backend
  go test ./...
  go vet ./...
  go build ./...
  ```

- 前端源码或构建配置：从 `frontend/` 运行 `npm run build`；该命令已包含 `vue-tsc --noEmit`。若 `package.json` 或 `package-lock.json` 改动，先运行 `npm ci`。
- Compose 改动：从仓库根目录运行 `docker compose -f deploy/docker-compose.yml config` 与 `docker compose -f deploy/docker-compose.prod.yml config`。
- `deploy/Dockerfile` 或跨前后端构建链改动：Docker 可用时运行 `docker build -f deploy/Dockerfile -t transithub:sync-check .`。
- locale 或 i18n 改动：除前端构建外，核对 `zh-CN.ts` 与 `en-US.ts` 的新增键，并实际切换语言检查受影响页面。本项目无 `i18n:sync` 脚本，不得调用旧项目命令。
- 数据库迁移、Redis、外部 upstream 或调度逻辑：能启动本地依赖时做对应集成验证；否则明确写出未验证项与手工检查步骤，不虚报通过。

从改动领域派生回归重点：

- `upstream` / `my_sites`：new-api、sub2api 管理接口，凭据探测、账户/分组/余额同步、缓存与错误映射。
- `group_rates` / `group_rate_campaigns`：倍率快照、自动定价、即时/定时活动及原倍率恢复。
- `connection_health`：模型发现、探测预算、状态机、策略分配、手动动作与 scheduler。
- `auth` / `users` / `admin_accounts`：注册、登录、会话、管理员工作区切换与数据隔离。
- `dashboard`：指标刷新、趋势、分组用量与上游钻取。
- `tickets`：嵌入会话、工单 CRUD、附件上传/预览、host guard 与 sub2api 用户资料。
- 前端：受影响路由及创建/编辑/删除流程、抽屉/弹窗、加载/错误态、响应式布局及双语显示。
- 配置/部署：新装默认值、旧配置升级、PostgreSQL/Redis 连接、Compose 与镜像启动。

任何必需检查失败时，不得宣称同步完成。先判断是上游回归、冲突残留、本地定制不兼容或环境缺失；能在不改变本地行为的前提下修复者继续闭环，否则报告阻塞与选项。

### 8. 中文报告

报告须包含：

- `transithub/main` 的短 commit 与同步前本地 commit；
- 本地/上游分叉状态；
- 结果类型：已同步、快进、普通 merge commit、已解决冲突 merge，或因歧义冲突阻塞；
- 自动解决的冲突及分类；
- 尚需用户决策的冲突与方案；
- 后端、前端、部署/配置/文档的主要更新；
- 基于实际改动的回归重点；
- 所有验证命令、结果及未覆盖项；
- 最终 `git status --short`。

简洁陈述事实。若未 push，明确说明本地同步尚未推送。

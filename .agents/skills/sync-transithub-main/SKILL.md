---
name: sync-transithub-main
description: 将官方 deviseo/transit-hub 的 main 分支同步到本地 main；在同功能冲突中优先保留本地行为，完成冲突处理、项目影响分析、按改动做低成本必要验证及中文更新摘要。用户要求拉取、同步、更新或合并 TransitHub 官方主线时使用。
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
git -c merge.renormalize=true merge --no-edit -X ours transithub/main
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

不得暂存无关文件。完成后再次扫描冲突标记，并运行 `git -c core.whitespace=cr-at-eol diff --check "$LOCAL_BEFORE"..HEAD`。

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

验证须与实际改动和风险成比例，以发现合并错误为目的，不为覆盖率或“多跑一项更放心”而重复测试。默认选择能证明结果的最低成本检查；轻量检查已覆盖风险时立即停止，不升级到更耗时、耗 CPU/内存、拉镜像、启容器或浏览器的验证。

先运行对应区域的必要检查：

- 后端 Go、`go.mod` 或 `go.sum`：

  ```bash
  cd backend
  go test ./...
  ```

- `go test ./...` 已编译全部包时，不例行追加 `go build ./...`。仅当入口、构建标签、嵌入资源或发布构建链变化时再运行 `go build ./...`；仅当改动涉及并发、格式化接口、struct tag 等 `vet` 能发现的风险时再运行 `go vet ./...`。
- 前端源码或构建配置：从 `frontend/` 运行 `npm run build`；该命令已包含 `vue-tsc --noEmit`。仅当依赖清单变化且现有 `node_modules` 缺失或明显不一致时先运行 `npm ci`，不得为例行同步重复安装依赖。
- Compose 改动：从仓库根目录运行 `docker compose -f deploy/docker-compose.yml config` 与 `docker compose -f deploy/docker-compose.prod.yml config`。
- locale 或 i18n 改动：除前端构建外，静态核对 `zh-CN.ts` 与 `en-US.ts` 的新增键。本项目无 `i18n:sync` 脚本，不得调用旧项目命令；默认不启动浏览器只为切换语言。

以下高成本验证默认跳过，并在报告中注明未覆盖项：

- 不为例行同步启动临时 PostgreSQL、Redis、外部 upstream、scheduler 或整套应用；数据库迁移先审查顺序、SQL 与已有相关测试。仅当用户明确要求集成验证，或高风险歧义无法用代码审查和现有测试排除时，才启动最小依赖。
- 不拉取镜像或运行 `docker build` 只为增加验证项。即使 `deploy/Dockerfile` 或跨前后端构建链变化，也先用已有构建、配置解析和 diff 审查；仅当用户明确要求镜像验证，或无更轻量方式判断镜像能否构建时才运行。
- 不安装浏览器自动化依赖，不启动 Chrome/Playwright，不做截图或逐页点击，只为验证 locale、样式或常规路由。仅当用户明确要求 UI/E2E 验证，或构建通过但合并问题只能在运行时复现时才做最小页面检查。
- 不重复运行覆盖同一风险的全量命令，不为测试覆盖率新增测试，不因工具可用就扩大验证范围。

从改动领域派生回归重点：

- `upstream` / `my_sites`：new-api、sub2api 管理接口，凭据探测、账户/分组/余额同步、缓存与错误映射。
- `group_rates` / `group_rate_campaigns`：倍率快照、自动定价、即时/定时活动及原倍率恢复。
- `connection_health`：模型发现、探测预算、状态机、策略分配、手动动作与 scheduler。
- `auth` / `users` / `admin_accounts`：注册、登录、会话、管理员工作区切换与数据隔离。
- `dashboard`：指标刷新、趋势、分组用量与上游钻取。
- `tickets`：嵌入会话、工单 CRUD、附件上传/预览、host guard 与 sub2api 用户资料。
- 前端：受影响路由及创建/编辑/删除流程、抽屉/弹窗、加载/错误态、响应式布局及双语显示。
- 配置/部署：新装默认值、旧配置升级、PostgreSQL/Redis 连接、Compose 与镜像启动。

任何已选择的必要检查失败时，不得宣称同步完成。先判断是上游回归、冲突残留、本地定制不兼容或环境缺失；能在不改变本地行为的前提下修复者继续闭环，否则报告阻塞与选项。不得以失败为由无边界追加高成本测试；先用错误输出、代表性 diff 和最小复现定位。

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

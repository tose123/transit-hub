---
name: release-version
description: 为当前 fork tose123/transit-hub 自动执行完整发布。用户要求“发布版本”、“发版”、“打 tag”、"/release-version" 或 “release-version” 时使用：按中国标准时间自动生成 vYY.M.DHHMM tag，同步 TransitHub 内置版本、GHCR 部署镜像和 Release 链接，完成本地验证，发布到 origin 并创建 GitHub Release；全流程无需逐项确认，不等待或查询 GitHub Actions 状态。
---

# TransitHub 发布版本

## 发布契约

本 skill 只适用于当前 fork：

- Git remote：`origin`，必须指向 `tose123/transit-hub`。
- GitHub Release：`https://github.com/tose123/transit-hub/releases`。
- 容器镜像：`ghcr.io/tose123/transit-hub:<tag>`。
- 发布分支：`main`。

版本 tag 由中国标准时间自动生成，格式固定为：

```text
vYY.M.DHHMM
```

例如 2026 年 7 月 12 日 15:30（中国标准时间）生成 `v26.7.121530`。tag 保留前导 `v`，并且必须与应用内置版本和镜像 tag 完全一致。它是 SemVer 兼容的三段数字版本，而不是手工指定的 `vMAJOR.MINOR.PATCH`。

用户明确要求发布，即视为授权执行本 skill 的完整流程，包括创建本地发布提交和 tag、推送分支和 tag，以及创建 GitHub Release。全流程连续自动执行，无需逐项展示或请求确认；仅在预检、验证或外发操作实际失败时停止并报告。推送 tag 后不等待、不查询 GitHub Actions 状态，也不以远端 GHCR 产物状态阻塞发布完成。

不得发布到 `transithub` upstream 或 `deviseo/transit-hub`。若用户希望发布官方仓库，停止并说明本 skill 的范围不适用。

## 发布版本源

tag 是下列发布内容的唯一版本来源：

| 位置 | 应有值 |
| --- | --- |
| `backend/internal/config/config.go` | `defaultAppVersion = "<tag>"` |
| `deploy/docker-compose.prod.yml` | `image: ghcr.io/tose123/transit-hub:<tag>` |
| `README.md` | fork clone URL 与两处 GHCR 镜像示例 |
| `README_CN.md` | fork clone URL 与两处 GHCR 镜像示例 |
| `frontend/src/modules/admin/layout/AdminLayout.vue` | `https://github.com/tose123/transit-hub` |

`backend/go.mod`、`backend/go.sum`、`frontend/package.json`、`frontend/package-lock.json` 不包含 TransitHub 应用版本。除非本次发布请求明确包含依赖变更，不得为了发版修改它们。

## 工作流

### 1. 预检

从仓库根目录开始。先确认项目结构、Git 状态、分支和发布目标：

```bash
git rev-parse --show-toplevel
test -f backend/go.mod
test -f frontend/package.json
test -f deploy/docker-compose.prod.yml
git status --short
git branch --show-current
git symbolic-ref -q HEAD
git rev-parse --verify main
git rev-parse --short HEAD
git remote get-url origin
git remote get-url --push origin
git remote get-url transithub
```

规则：

- 当前分支必须是 `main`，且不能是 detached `HEAD`。不自动切分支、不重置、不 rebase。
- 工作区和 index 必须干净。存在任何预先的改动时停止；不 stash、提交、丢弃或把它们混入版本提交。
- `origin` 的 fetch 与 push URL 必须指向 `tose123/transit-hub`。允许 `git@github-tose:tose123/transit-hub.git`、`git@github.com:tose123/transit-hub.git`、`ssh://git@github.com/tose123/transit-hub.git` 或对应 HTTPS URL。其他地址停止，不擅自修改 remote。
- `transithub` 只作为上游同步来源，必须仍是 `deviseo/transit-hub`；绝不能作为发布 remote。
- 确认已安装 `git`、`go`、`node`、`npm`、`docker compose`、`gh` 与 Docker。缺失任一必需工具时，先报告缺失项，不创建发布提交或 tag。

获取 fork 的发布分支和 tag，并确保本地发布不会落后于 fork：

```bash
git fetch origin main --tags
git rev-parse --verify origin/main
git merge-base --is-ancestor origin/main HEAD
git log --oneline --decorate -5 origin/main..HEAD
gh auth status
gh repo view tose123/transit-hub --json nameWithOwner,defaultBranchRef,url
```

若 `origin/main` 不是当前 `HEAD` 的祖先，停止。先由用户决定如何整合 fork 上的远端提交；不要在发布流程内 pull、merge、rebase 或 force push。`gh auth status` 必须具备向目标仓库创建 Release 的权限。

### 2. 自动生成并校验时间戳 tag

显式以中国标准时间生成 tag，不依赖机器本地时区：

```bash
tag="$(TZ=Asia/Shanghai date '+v%y.%-m.%-d%H%M')"
printf '%s\n' "$tag"
```

校验格式和日期时间部分：

```bash
TAG="$tag" node - <<'NODE'
const tag = process.env.TAG;
const match = /^v(\d{2})\.(\d+)\.(\d+)$/.exec(tag);
if (!match) throw new Error(`Invalid release tag: ${tag}`);

const month = Number(match[2]);
const dayAndTime = Number(match[3]);
const day = Math.floor(dayAndTime / 10000);
const hour = Math.floor((dayAndTime % 10000) / 100);
const minute = dayAndTime % 100;

if (
  !Number.isInteger(month) || !Number.isInteger(day) ||
  !Number.isInteger(hour) || !Number.isInteger(minute) ||
  month < 1 || month > 12 || day < 1 || day > 31 ||
  hour < 0 || hour > 23 || minute < 0 || minute > 59
) {
  throw new Error(`Invalid China Standard Time release tag: ${tag}`);
}
NODE
```

检查本地和 fork 远端的完整 tag 名：

```bash
git rev-parse -q --verify "refs/tags/$tag" >/dev/null && {
  echo "Local tag already exists: $tag"
  exit 1
}
git ls-remote --exit-code --tags origin "refs/tags/$tag" >/dev/null 2>&1 && {
  echo "Remote tag already exists: $tag"
  exit 1
}
```

同一分钟内已有任一 tag 时停止。等待下一分钟后从本步骤重新生成，重新校验；不得修改、删除、复用或强推已有 tag。

### 3. 同步版本和 fork 发布引用

定义发布值：

```bash
image="ghcr.io/tose123/transit-hub:$tag"
github_repo_url="https://github.com/tose123/transit-hub"
```

先读取这些文件的相关片段，然后只更新 release-owned 字段：

```bash
rg -n 'defaultAppVersion|^    image:' backend/internal/config/config.go deploy/docker-compose.prod.yml
rg -n 'github\.com/(deviseo|tose123)/transit-hub|(?:deviseo/transithub|ghcr\.io/tose123/transit-hub):' README.md README_CN.md frontend/src/modules/admin/layout/AdminLayout.vue
```

更新要求：

- `backend/internal/config/config.go` 的 `defaultAppVersion` 必须为 `"$tag"`。
- `deploy/docker-compose.prod.yml` 的应用镜像必须为 `image: $image`。
- `README.md` 与 `README_CN.md` 的克隆命令改为 `$github_repo_url.git`，部署说明与 Docker build 示例均使用 `$image`。每个 README 中应恰有两处 GHCR 镜像示例。
- `frontend/src/modules/admin/layout/AdminLayout.vue` 的唯一仓库常量必须为 `$github_repo_url`，使后台版本链接指向 fork 的 GitHub Releases。
- 不改动 `update-star-history.yml`、星图链接或任何不属于发布契约的官方来源归属。

使用范围最小的 patch 编辑这些文件。编辑后执行一致性检查：

```bash
TAG="$tag" IMAGE="$image" node - <<'NODE'
const fs = require('fs');

const tag = process.env.TAG;
const image = process.env.IMAGE;
const text = (path) => fs.readFileSync(path, 'utf8');
const expected = [
  ['backend/internal/config/config.go', `defaultAppVersion = "${tag}"`],
  ['deploy/docker-compose.prod.yml', `image: ${image}`],
  ['frontend/src/modules/admin/layout/AdminLayout.vue', "const githubRepoUrl = 'https://github.com/tose123/transit-hub'"],
];

for (const [path, value] of expected) {
  if (!text(path).includes(value)) {
    throw new Error(`${path} does not contain expected release value: ${value}`);
  }
}

for (const path of ['README.md', 'README_CN.md']) {
  const content = text(path);
  if (!content.includes('https://github.com/tose123/transit-hub.git')) {
    throw new Error(`${path} does not clone the release fork`);
  }
  if ((content.match(new RegExp(image.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g')) || []).length !== 2) {
    throw new Error(`${path} must contain exactly two ${image} references`);
  }
}
NODE

rg -n '^git clone https://github\.com/' README.md README_CN.md
rg -n '^    image:' deploy/docker-compose.prod.yml
rg -n 'image tag .*:|镜像 tag.*:|docker build .* -t ' README.md README_CN.md
rg -n "const githubRepoUrl" frontend/src/modules/admin/layout/AdminLayout.vue
git diff --check -- \
  backend/internal/config/config.go \
  deploy/docker-compose.prod.yml \
  README.md \
  README_CN.md \
  frontend/src/modules/admin/layout/AdminLayout.vue
git diff -- \
  backend/internal/config/config.go \
  deploy/docker-compose.prod.yml \
  README.md \
  README_CN.md \
  frontend/src/modules/admin/layout/AdminLayout.vue
```

审阅这三组发布引用的输出：克隆命令、应用镜像、两份 README 的镜像示例和前端仓库常量都必须是 fork 对应值。README 的星图、贡献说明等非发布链接可以继续使用官方仓库地址；若发布引用出现额外或不确定的匹配，停止并报告，不进行提交。

确认仅以上五个 release-owned 文件发生预期修改后，创建受控发布准备提交：

```bash
git status --short
git add \
  backend/internal/config/config.go \
  deploy/docker-compose.prod.yml \
  README.md \
  README_CN.md \
  frontend/src/modules/admin/layout/AdminLayout.vue
git commit -m "chore: prepare $tag release"
git status --short
git rev-parse HEAD
```

若所有字段原本已经匹配且没有需要迁移的 fork 发布引用，不创建空提交。若 `git status --short` 出现任何非这五个路径的改动，停止，不得暂存它们。

### 4. 发布前验证

记录本次发布候选 commit：

```bash
release_commit="$(git rev-parse HEAD)"
git diff --check "$release_commit^".."$release_commit"
```

尾随空格校验仅针对本次发布准备提交，避免重新判定已由上游同步流程验证且需保留 `transithub/main` 原格式的内容；候选整体仍由后续构建验证。

依次完成所有必需检查。任一命令失败时停止，不创建本地 tag；报告原始失败输出，修复后从相应步骤重新执行。

```bash
cd backend
go test ./...
go vet ./...
go build ./...

cd ../frontend
npm ci
npm run build

cd ..
docker compose -f deploy/docker-compose.yml config
docker compose -f deploy/docker-compose.prod.yml config
docker build -f deploy/Dockerfile -t "$image" .
```

Docker daemon 不可用、`npm ci` 无法访问依赖或 Docker build 失败均视为发布阻塞，不得以“未验证”状态继续创建 tag。若镜像 build 成功，可用下列命令确认本地标签存在：

```bash
docker image inspect "$image" --format '{{.Id}}'
```

再次运行版本一致性检查和 `git status --short`，确认候选 commit 仍是所验证的 `release_commit`。

### 5. 创建本地 tag

确认所有验证通过后，创建轻量 tag。TransitHub 既有 tag 是轻量 tag，因此不要改为 annotated tag：

```bash
git tag "$tag" "$release_commit"
git rev-parse "$tag^{commit}"
git rev-parse "$release_commit"
git show --no-patch --format='%D%n%H%n%s' "$tag"
```

两个完整 commit hash 必须一致。若 tag 创建失败或指向不正确，停止；不得删除或重建已有 tag。

### 6. 推送分支和 tag

本地 tag 验证通过后，自动仅推送 `main` 和刚创建的 tag：

```bash
git push origin main
git push origin "$tag"
git ls-remote --exit-code --heads origin refs/heads/main
git ls-remote --exit-code --tags origin "refs/tags/$tag"
```

不得使用 `git push --tags`、force push、删除远端 tag 或重写远端分支。若任一 push 失败，停止并报告实际远端状态；不要尝试覆写或自行修复非快进历史。

### 7. 创建并验证 GitHub Release

确认 push 成功后，先检查同名 GitHub Release：

```bash
if gh release view "$tag" --repo tose123/transit-hub --json url,isDraft,isPrerelease,tagName,targetCommitish; then
  echo "GitHub Release already exists for $tag; stop without changing it."
  exit 1
fi
```

仅在前述 `gh auth status` 和目标仓库查询均已通过，且此命令确认 tag 没有既有 Release 时，才继续创建。若 Release 已存在，停止并报告其 URL 和状态；不要覆盖、编辑或删除已有 Release。若不存在，先准备简洁、基于本次 diff 的发布说明。说明应包含：用户可见变更、后端/前端/部署变更、升级事项、镜像地址 `$image` 与对应 tag；不得编造未验证的迁移或功能。

自动以 title `TransitHub $tag` 和准备好的完整 notes 创建 Release：

```bash
gh release create "$tag" \
  --repo tose123/transit-hub \
  --title "TransitHub $tag" \
  --notes-file /absolute/path/to/release-notes.md
gh release view "$tag" --repo tose123/transit-hub --json url,isDraft,isPrerelease,tagName,targetCommitish
```

Release 必须是非 draft、非 prerelease，`tagName` 必须为 `$tag`，并返回公开 URL。不要上传虚构的二进制附件；GitHub 自动提供 source archive。

### 8. 不跟踪 GitHub Actions

推送 tag 后，`.github/workflows/docker-build.yml` 应在 fork 上构建并推送：

- `ghcr.io/tose123/transit-hub:${tag}-amd64`
- `ghcr.io/tose123/transit-hub:${tag}-arm64`
- `ghcr.io/tose123/transit-hub:$tag` 的多架构 manifest
- 同一镜像的 `latest` manifest

tag push 用于触发上述异步镜像构建，但本 skill 不等待、轮询或查询 GitHub Actions，不执行 `gh workflow view`、`gh run list`、`gh run watch`，也不检查 Actions 日志。无需等待远端 GHCR manifest 或签名产出，不以其状态作为本次发布完成条件。

报告镜像地址时，明确其为 tag push 后由 GitHub Actions 异步构建的预期地址，不声称已验证远端镜像、manifest 架构或签名。

### 9. 中文报告

报告必须基于实际命令结果，包含：

- 中国标准时间自动生成的 tag、发布 commit、是否创建了发布准备提交；
- 推送目标 `origin`、fork 仓库 URL，以及 `transithub` 未被推送的确认；
- 五个版本/发布引用的同步结果；
- 所有验证命令及结果，及未覆盖项；
- branch/tag push、GitHub Release URL，以及由 tag push 异步触发的预期 GHCR 镜像地址；
- 明确说明按发布契约未跟踪 GitHub Actions，未验证远端 GHCR manifest 架构和签名；
- 对已有 tag/Release、工具缺失、权限不足、校验或发布失败的准确阻塞状态和恢复步骤；
- 最终 `git status --short`。

branch/tag 已成功推送且 GitHub Release 已创建并验证后，即可陈述“完整发布完成”。GitHub Actions 与远端 GHCR 状态不属于完成门槛。

# 📦 发布到 GitHub Packages 完整指南

本指南介绍如何将 NOFX 项目的前端、后端和 Docker 镜像发布到 GitHub Packages。

---

## 📋 目录

1. [Docker 镜像发布](#docker-镜像发布) ✅ 已配置
2. [前端 npm 包发布](#前端-npm-包发布)
3. [后端 Go 模块发布](#后端-go-模块发布)
4. [使用已发布的包](#使用已发布的包)

---

## 🐳 Docker 镜像发布

### 当前状态
✅ **已配置完成** - 自动构建和发布

### 工作流文件
`.github/workflows/docker-build.yml`

### 自动触发条件
- 推送到 `main`、`dev`、`Individual`、`release/stable` 分支
- 创建版本标签（`v*`）
- 手动触发

### 发布的镜像
- **后端**: `ghcr.io/hansen1018/nofx/nofx-backend`
- **前端**: `ghcr.io/hansen1018/nofx/nofx-frontend`

### 使用方法
```bash
# 拉取镜像
docker pull ghcr.io/hansen1018/nofx/nofx-backend:latest
docker pull ghcr.io/hansen1018/nofx/nofx-frontend:latest

# 使用 docker-compose
docker-compose up -d
```

### 查看已发布的镜像
- 访问：https://github.com/Hansen1018/nofx/packages

---

## 📦 前端 npm 包发布

### 准备工作

#### 1. 更新 package.json

在 `web/package.json` 中添加发布配置：

```json
{
  "name": "@hansen1018/nofx-web",
  "version": "1.0.0",
  "description": "NOFX AI Trading System - Frontend",
  "publishConfig": {
    "registry": "https://npm.pkg.github.com",
    "@hansen1018:registry": "https://npm.pkg.github.com"
  },
  "repository": {
    "type": "git",
    "url": "https://github.com/Hansen1018/nofx.git"
  }
}
```

#### 2. 创建 .npmrc 文件（可选，用于本地发布）

在 `web/.npmrc` 中配置：

```
@hansen1018:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${GITHUB_TOKEN}
```

### 自动发布（推荐）

#### 使用 GitHub Actions

1. **工作流已创建**: `.github/workflows/publish-npm.yml`

2. **触发发布**:
   - 推送到 `main` 或 `Individual` 分支
   - 创建版本标签（如 `v1.0.0`）
   - 手动触发 workflow

3. **手动触发**:
   - 访问：https://github.com/Hansen1018/nofx/actions
   - 选择 "Publish Frontend to GitHub Packages (npm)"
   - 点击 "Run workflow"
   - 可选：输入版本号

### 手动发布

#### 步骤 1：配置 npm

```bash
cd web

# 创建 .npmrc
echo "@hansen1018:registry=https://npm.pkg.github.com" > .npmrc
echo "//npm.pkg.github.com/:_authToken=YOUR_GITHUB_TOKEN" >> .npmrc
```

#### 步骤 2：获取 GitHub Token

1. 访问：https://github.com/settings/tokens
2. 创建 Personal Access Token (classic)
3. 权限选择：`write:packages`、`read:packages`
4. 复制 token

#### 步骤 3：发布

```bash
# 更新版本号
npm version patch  # 或 minor, major

# 发布
npm publish
```

### 查看已发布的包
- 访问：https://github.com/Hansen1018/nofx/packages
- 或：https://github.com/Hansen1018?tab=packages

### 使用已发布的 npm 包

```bash
# 配置 npm 使用 GitHub Packages
echo "@hansen1018:registry=https://npm.pkg.github.com" >> .npmrc
echo "//npm.pkg.github.com/:_authToken=YOUR_GITHUB_TOKEN" >> .npmrc

# 安装
npm install @hansen1018/nofx-web
```

---

## 🔧 后端 Go 模块发布

### 准备工作

#### 1. 确保 go.mod 配置正确

`go.mod` 应该包含：

```go
module github.com/Hansen1018/nofx
```

#### 2. 创建版本标签

```bash
# 创建并推送标签
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```

### 自动发布（推荐）

#### 使用 GitHub Actions

1. **工作流已创建**: `.github/workflows/publish-go.yml`

2. **触发发布**:
   - 推送到 `main` 或 `Individual` 分支（自动创建开发版本）
   - 创建版本标签（如 `v1.0.0`）

3. **Go 模块自动发布**:
   - Go 模块通过 git tag 自动发布到 GitHub
   - 无需额外配置

### 手动发布

#### 步骤 1：创建版本标签

```bash
# 创建标签
git tag -a v1.0.0 -m "Release version 1.0.0"

# 推送标签
git push origin v1.0.0
```

#### 步骤 2：验证模块

```bash
# 验证模块
go mod verify

# 测试构建
go build ./...
```

#### 步骤 3：发布（自动）

Go 模块通过 git tag 自动发布，无需额外操作。

### 使用已发布的 Go 模块

```bash
# 在另一个 Go 项目中使用
go get github.com/Hansen1018/nofx@v1.0.0

# 或使用最新版本
go get github.com/Hansen1018/nofx@latest
```

在 `go.mod` 中：

```go
require (
    github.com/Hansen1018/nofx v1.0.0
)
```

---

## 📊 发布状态总览

| 包类型 | 状态 | 工作流文件 | 包地址 |
|--------|------|-----------|--------|
| Docker 后端 | ✅ 已配置 | `docker-build.yml` | `ghcr.io/hansen1018/nofx/nofx-backend` |
| Docker 前端 | ✅ 已配置 | `docker-build.yml` | `ghcr.io/hansen1018/nofx/nofx-frontend` |
| npm 包 | 📝 需配置 | `publish-npm.yml` | `@hansen1018/nofx-web` |
| Go 模块 | 📝 需配置 | `publish-go.yml` | `github.com/Hansen1018/nofx` |

---

## 🔐 权限设置

### 确保 Actions 有写入权限

1. 访问：https://github.com/Hansen1018/nofx/settings/actions
2. 在 "Workflow permissions" 中选择 **"Read and write permissions"**
3. 保存设置

### 如果遇到权限错误

参考：`.github/workflows/QUICK_FIX_PERMISSIONS.md`

---

## 🚀 快速开始

### 发布 Docker 镜像（已配置）
```bash
# 推送到分支即可自动发布
git push origin Individual
```

### 发布 npm 包
```bash
# 1. 更新 web/package.json（添加 publishConfig）
# 2. 推送到分支或创建标签
git tag v1.0.0
git push origin v1.0.0
```

### 发布 Go 模块
```bash
# 创建并推送标签
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

---

## 📝 版本管理建议

### Docker 镜像
- `latest` - main 分支
- `individual` - Individual 分支
- `stable` - release/stable 分支
- `v1.0.0` - 版本标签

### npm 包
- 遵循语义化版本（SemVer）
- `1.0.0` - 主版本.次版本.修订版本

### Go 模块
- 使用 git tag 管理版本
- 格式：`v1.0.0`

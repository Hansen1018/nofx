# GitHub Packages 配置说明

## 📦 已配置的包

此仓库已配置自动构建和发布 Docker 镜像到 GitHub Packages：

- **后端镜像**: `ghcr.io/hansen1018/nofx/nofx-backend`
- **前端镜像**: `ghcr.io/hansen1018/nofx/nofx-frontend`

## 🚀 自动触发条件

当以下事件发生时，会自动构建和推送镜像：

1. **推送到分支**:
   - `main` - 会创建 `latest` 标签
   - `dev` - 会创建 `dev` 标签
   - `Individual` - 会创建 `individual` 标签
   - `release/stable` - 会创建 `stable` 标签

2. **创建版本标签**: 推送到 `v*` 格式的标签（如 `v1.0.0`）

3. **手动触发**: 在 GitHub Actions 页面手动运行 workflow

## 📍 如何查看已发布的包

### 方法一：通过 GitHub 网页
1. 访问仓库：https://github.com/Hansen1018/nofx
2. 点击右侧边栏的 **"Packages"** 链接
3. 或直接访问：https://github.com/Hansen1018/nofx/packages

### 方法二：通过个人资料
1. 访问：https://github.com/Hansen1018?tab=packages
2. 查看所有已发布的包

## 🔧 如何使用发布的镜像

### 拉取镜像

```bash
# 拉取最新版本（main分支）
docker pull ghcr.io/hansen1018/nofx/nofx-backend:latest
docker pull ghcr.io/hansen1018/nofx/nofx-frontend:latest

# 拉取Individual分支版本
docker pull ghcr.io/hansen1018/nofx/nofx-backend:individual
docker pull ghcr.io/hansen1018/nofx/nofx-frontend:individual

# 拉取特定分支版本
docker pull ghcr.io/hansen1018/nofx/nofx-backend:Individual-amd64
docker pull ghcr.io/hansen1018/nofx/nofx-frontend:Individual-amd64
```

### 使用镜像（需要认证）

如果包是私有的，需要先登录：

```bash
# 使用 Personal Access Token 登录
echo $GITHUB_TOKEN | docker login ghcr.io -u Hansen1018 --password-stdin

# 然后拉取镜像
docker pull ghcr.io/hansen1018/nofx/nofx-backend:latest
```

### 在 docker-compose.yml 中使用

```yaml
services:
  backend:
    image: ghcr.io/hansen1018/nofx/nofx-backend:latest
    # ... 其他配置

  frontend:
    image: ghcr.io/hansen1018/nofx/nofx-frontend:latest
    # ... 其他配置
```

## 🔐 权限设置

### 公开包（推荐）
1. 访问包页面：https://github.com/Hansen1018/nofx/packages
2. 点击包名称
3. 点击右侧 "Package settings"
4. 在 "Danger Zone" 中点击 "Change visibility"
5. 选择 "Public"

### 私有包
- 默认情况下，包是私有的
- 需要 GitHub 认证才能拉取
- 使用 Personal Access Token (PAT) 进行认证

## 📊 支持的架构

- ✅ **linux/amd64** - Intel/AMD 64位
- ✅ **linux/arm64** - ARM 64位（Apple Silicon, Raspberry Pi 等）

## 🛠️ 故障排除

### 问题：无法拉取镜像
**解决方案**：
1. 确认包是否已成功构建（检查 GitHub Actions）
2. 如果包是私有的，确保已登录：`docker login ghcr.io`
3. 检查镜像标签是否正确

### 问题：构建失败
**解决方案**：
1. 检查 GitHub Actions 日志
2. 确认 Dockerfile 路径正确
3. 检查是否有足够的 GitHub Actions 配额

## 📝 相关文件

- Workflow 配置: `.github/workflows/docker-build.yml`
- 后端 Dockerfile: `docker/Dockerfile.backend`
- 前端 Dockerfile: `docker/Dockerfile.frontend`

## 🔗 相关链接

- [GitHub Packages 文档](https://docs.github.com/zh/packages)
- [GitHub Container Registry 文档](https://docs.github.com/zh/packages/working-with-a-github-packages-registry/working-with-the-container-registry)

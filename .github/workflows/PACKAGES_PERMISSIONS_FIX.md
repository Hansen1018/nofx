# 🔧 修复 GitHub Packages 权限问题

## 问题
```
ERROR: failed to build: failed to solve: failed to push ghcr.io/hansen1018/nofx/nofx-backend:Individual-amd64: denied: permission_denied: write_package
```

## 解决方案

### 方法一：在 GitHub 仓库设置中启用权限（推荐）

1. **访问仓库设置**
   - 打开：https://github.com/Hansen1018/nofx/settings
   - 或：仓库主页 → Settings（设置）

2. **进入 Actions 设置**
   - 左侧菜单点击 **"Actions"** → **"General"**
   - 或直接访问：https://github.com/Hansen1018/nofx/settings/actions

3. **配置 Workflow permissions**
   - 找到 **"Workflow permissions"** 部分
   - 选择 **"Read and write permissions"**（读取和写入权限）
   - 确保勾选 **"Allow GitHub Actions to create and approve pull requests"**（如果需要）
   - 点击 **"Save"**（保存）

4. **验证设置**
   - 确认 "Read and write permissions" 已选中
   - 这允许 workflow 写入 GitHub Packages

### 方法二：使用 Personal Access Token（如果方法一不行）

如果仓库设置无法更改，可以创建 Personal Access Token：

1. **创建 PAT**
   - 访问：https://github.com/settings/tokens
   - 点击 **"Generate new token"** → **"Generate new token (classic)"**
   - 名称：`GitHub Packages Write`
   - 权限选择：
     - ✅ `write:packages` - 写入包
     - ✅ `read:packages` - 读取包
     - ✅ `delete:packages` - 删除包（可选）
   - 点击 **"Generate token"**
   - **复制 token**（只显示一次！）

2. **添加到仓库 Secrets**
   - 访问：https://github.com/Hansen1018/nofx/settings/secrets/actions
   - 点击 **"New repository secret"**
   - Name: `GHCR_TOKEN`
   - Value: 粘贴刚才复制的 token
   - 点击 **"Add secret"**

3. **更新 workflow 文件**
   - 将 `secrets.GITHUB_TOKEN` 改为 `secrets.GHCR_TOKEN`
   - 我已经在 workflow 中添加了备用方案

### 方法三：检查仓库可见性

如果仓库是私有的，确保：
- 您有仓库的管理员权限
- Actions 已启用（Settings → Actions → General → Allow all actions）

## 验证修复

修复后，重新运行 workflow：
1. 访问：https://github.com/Hansen1018/nofx/actions
2. 找到失败的 workflow
3. 点击 **"Re-run all jobs"** 或 **"Re-run failed jobs"**

## 常见问题

### Q: 为什么会出现权限错误？
A: GitHub 默认情况下，Actions 只有读取权限。需要明确授予写入包的权限。

### Q: 我已经设置了权限，还是失败？
A: 
- 检查是否保存了设置
- 等待几分钟让设置生效
- 重新运行 workflow
- 检查是否有其他权限限制

### Q: 可以使用 GITHUB_TOKEN 吗？
A: 可以，但需要确保仓库设置中允许 Actions 写入包。如果不行，使用 PAT。

## 相关链接

- [GitHub Actions 权限文档](https://docs.github.com/zh/actions/security-guides/automatic-token-authentication#permissions-for-the-github_token)
- [GitHub Packages 权限文档](https://docs.github.com/zh/packages/learn-github-packages/about-permissions-for-github-packages)
- [仓库 Actions 设置](https://github.com/Hansen1018/nofx/settings/actions)

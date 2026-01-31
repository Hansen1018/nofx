# ⚡ 快速修复 GitHub Packages 权限问题

## 🚨 错误信息
```
ERROR: failed to build: failed to solve: failed to push ghcr.io/hansen1018/nofx/nofx-backend:Individual-amd64: denied: permission_denied: write_package
```

## ✅ 解决方案（按优先级）

### 方案 1：启用仓库 Actions 写入权限（最简单）⭐

**必须操作！** 这是最直接的解决方案：

1. **打开仓库设置**
   ```
   https://github.com/Hansen1018/nofx/settings/actions
   ```

2. **找到 "Workflow permissions" 部分**
   - 向下滚动到 "Workflow permissions"

3. **选择权限级别**
   - ✅ 选择 **"Read and write permissions"**（读取和写入权限）
   - ❌ 不要选择 "Read repository contents and packages permissions"（只读）

4. **保存设置**
   - 点击 **"Save"** 按钮

5. **重新运行 workflow**
   - 访问：https://github.com/Hansen1018/nofx/actions
   - 找到失败的 workflow
   - 点击 **"Re-run all jobs"**

---

### 方案 2：使用 Personal Access Token（如果方案1不行）

如果方案1不起作用，使用 PAT 作为备用方案：

#### 步骤 1：创建 Personal Access Token

1. **访问 GitHub 设置**
   ```
   https://github.com/settings/tokens
   ```

2. **创建新 Token**
   - 点击 **"Generate new token"** → **"Generate new token (classic)"**
   - Token 名称：`GitHub Packages Write - NOFX`
   - 过期时间：选择合适的时间（建议 90 天或 1 年）

3. **选择权限（Scopes）**
   必须勾选：
   - ✅ `write:packages` - 写入包
   - ✅ `read:packages` - 读取包
   - ✅ `delete:packages` - 删除包（可选，用于清理旧版本）

4. **生成并复制 Token**
   - 点击 **"Generate token"**
   - **立即复制 token**（只显示一次！）
   - 格式类似：`ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`

#### 步骤 2：添加到仓库 Secrets

1. **打开仓库 Secrets 设置**
   ```
   https://github.com/Hansen1018/nofx/settings/secrets/actions
   ```

2. **添加新 Secret**
   - 点击 **"New repository secret"**
   - Name: `GHCR_PAT`
   - Value: 粘贴刚才复制的 token
   - 点击 **"Add secret"**

#### 步骤 3：重新运行 workflow

- workflow 会自动使用 `GHCR_PAT`（如果存在）或回退到 `GITHUB_TOKEN`
- 访问 Actions 页面重新运行失败的 job

---

### 方案 3：检查包的访问控制

如果包已经存在，可能需要调整访问权限：

1. **访问包页面**
   ```
   https://github.com/Hansen1018/nofx/packages
   ```

2. **选择包**
   - 点击 `nofx-backend` 或 `nofx-frontend`

3. **打开包设置**
   - 点击右上角 **"Package settings"**

4. **检查 Actions 访问权限**
   - 找到 **"Manage Actions access"** 部分
   - 确保仓库有写入权限
   - 如果没有，点击 **"Add repository"** 添加仓库

---

## 🔍 验证修复

修复后，检查以下几点：

1. ✅ 仓库 Actions 设置中已启用 "Read and write permissions"
2. ✅ 如果使用 PAT，Secret `GHCR_PAT` 已创建
3. ✅ 重新运行 workflow 后不再出现权限错误

## 📝 注意事项

- **方案 1 是必须的**：即使使用 PAT，也建议启用仓库的写入权限
- **PAT 安全性**：PAT 有完整权限，请妥善保管，不要泄露
- **Token 过期**：PAT 过期后需要重新创建并更新 Secret
- **等待生效**：设置更改后可能需要几分钟才能生效

## 🔗 快速链接

- [仓库 Actions 设置](https://github.com/Hansen1018/nofx/settings/actions)
- [创建 PAT](https://github.com/settings/tokens)
- [仓库 Secrets](https://github.com/Hansen1018/nofx/settings/secrets/actions)
- [Actions 运行历史](https://github.com/Hansen1018/nofx/actions)
- [已发布的包](https://github.com/Hansen1018/nofx/packages)

## ❓ 仍然失败？

如果以上方案都不行，请检查：

1. 仓库是否为私有？私有仓库需要额外配置
2. 账户是否有足够的 GitHub Actions 配额？
3. 是否有组织级别的权限限制？

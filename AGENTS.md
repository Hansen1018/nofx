# NOFX 代码库指南

给 AI 代理：这是一个 AI 自动交易系统，使用 Go 后端 + React 前端。

## 构建与测试命令

### 后端 (Go)
```bash
# 构建
go build -o nofx
go run main.go

# 测试
make test              # 所有测试
make test-backend      # 仅后端测试
go test -v ./...       # 运行所有 Go 测试
go test -v -run TestName ./manager  # 运行单个测试
go test -v ./manager -run TestLoadTraders  # 在 manager 包中运行特定测试

# 格式化与 Lint
go fmt ./...           # 格式化 Go 代码
golangci-lint run      # 运行 linter

# 覆盖率
make test-coverage     # 生成 coverage.html
go test -coverprofile=coverage.out ./...
```

### 前端 (Vite + React + TypeScript)
```bash
cd web

# 开发
npm run dev            # 启动开发服务器 (端口 3000)
npm run build          # 生产构建 (tsc + vite build)
npm run preview        # 预览生产构建

# 测试 (Vitest)
npm test               # 运行所有测试
npm test -- --run      # 单次运行（非 watch 模式）
npm test -- path/to/File.test.tsx  # 运行单个测试文件
npm test -- -t "match me"          # 运行匹配模式的测试

# 格式化与 Lint
npm run lint           # ESLint 检查
npm run lint:fix       # ESLint 自动修复
npm run format         # Prettier 格式化
npm run format:check   # Prettier 检查
```

### 统一命令 (根目录)
```bash
make test              # 运行所有测试（后端 + 前端）
make build             # 构建后端
make build-frontend    # 构建前端
make fmt               # 格式化 Go 代码
make deps              # 安装 Go 依赖
make deps-frontend     # 安装前端依赖
```

## 代码风格指南

### 后端 (Go)

#### 导入风格
```go
import (
    "nofx/api"
    "nofx/config"
    "github.com/gin-gonic/gin"
    "golang.org/x/crypto"
)
```
- 本地包用 `nofx/xxx`
- 标准库按字母顺序

#### 命名约定
- 包名称: 小写单数 (`store`, `trader`, `api`)
- 常量: 大写 + 下划线 (`MAX_RETRY`, `DEFAULT_TIMEOUT`)
- 导出: PascalCase (`NewServer`, `HandleError`)
- 私有: camelCase (`internalVar`, `privateFunc`)
- 接口: 纯行为命名 (`Trader`, `Store`, `Exchange`)

#### 错误处理
```go
// 标准模式
data, err := doSomething()
if err != nil {
    logger.Errorf("Failed: %v", err)
    return fmt.Errorf("action failed: %w", err)  // 使用 %w 包装
}

// 日志 + 错误
if err := st.Save(data); err != nil {
    logger.Fatalf("❌ Failed to initialize: %v", err)
}
```
- 总是检查错误
- 日志记录：`logger.Error()` 用于可恢复错误
- 日志记录：`logger.Fatal()` 用于致命错误
- 用 `%w` 包装错误以保留堆栈

#### 注释风格
```go
// Package store 统一数据库存储层
package store

// New 创建新的 Store 实例
func New(dbPath string) (*Store, error) { ... }

// TraderStore 管理 trader 数据
type TraderStore struct { ... }
```

### 前端 (TypeScript + React)

#### 导入风格
```typescript
import { useContext, useEffect } from 'react'
import { api } from './lib/api'
import type { TraderInfo, AIModel } from './types'
```
- External 库先，internal 后
- type 导入单独分组

#### 命名约定
- 组件: PascalCase (`TraderDashboardPage`, `HeaderBar`)
- 函数: camelCase (`handleClick`, `fetchData`)
- 常量: UPPER_SNAKE_CASE (`MAX_RETRIES`)
- 类型: PascalCase (`SystemStatus`, `AccountInfo`)
- 文件: kebab-case (`trader-dashboard-page.tsx`, `api.ts`)

#### 类型定义
```typescript
// 复杂对象使用 interface
interface TraderConfig {
  id: string
  name: string
  aiModel: string
}

// 联合类型/简单工具使用 type
type Page = 'traders' | 'competition' | 'backtest'
type Status = 'running' | 'stopped'

// 泛型：简洁实用
function fetchUrl<T>(url: string): Promise<T> { ... }
```

#### 错误处理
```typescript
// async/await 模式
try {
  const data = await api.getTraders()
  setData(data)
} catch (error) {
  console.error('Failed to fetch traders:', error)
  setError(error instanceof Error ? error.message : 'Unknown error')
}
```

#### React Hooks
```typescript
function Component() {
  const [data, setData] = useState<DataType[]>([])
  const { user, logout } = useAuth()

  useEffect(() => {
    // 副作用逻辑
  }, [dependencies])

  return ...
}
```

#### 格式化规则 (Prettier)
- 无分号 (`semi: false`)
- 单引号 (`singleQuote: true`)
- 2 空格缩进 (`tabWidth: 2`)
- 换行: LF (`endOfLine: 'lf'`)

## 项目架构

### 后端结构
```
nofx/
├── api/           # HTTP 服务器 (Gin)
├── auth/          # JWT 认证
├── backtest/      # 回测引擎
├── config/        # 环境配置
├── crypto/        # 加密服务 (API keys)
├── manager/       # Trader 管理器
├── market/        # 市场数据
├── provider/      # 交易所集成 (Binance, Bybit, OKX...)
├── store/         # 数据库存储 (GORM)
├── trader/        # 交易逻辑
├── web/           # 前端 (Vite + React)
└── main.go        # 入口
```

### 前端结构 (web/)
```
web/
├── src/
│   ├── components/  # UI 组件
│   ├── pages/       # 页面级组件
│   ├── lib/         # 工具函数，API 客户端
│   ├── contexts/    # React Context (Auth, Language)
│   ├── hooks/       # 自定义 Hooks
│   ├── types/       # TypeScript 类型定义
│   ├── i18n/        # 国际化
│   └── App.tsx      # 根组件
├── vite.config.ts   # Vite 配置 + 代理到 localhost:8080
└── tsconfig.json    # TypeScript 严格模式
```

## Git Hooks
- **pre-commit**: 运行 `lint-staged`（ESLint + Prettier，仅在 web/目录）
- 使用 Husky 管理

## CI/CD (.github/workflows/)
- `test.yml`: Go + 前端测试
- `pr-go-test-coverage.yml`: Go 覆盖率报告
- `docker-build.yml`: Docker 镜像构建

## 数据库
- 默认: SQLite (`data/nofx.db`)
- 支持: PostgreSQL (配置通过 .env)
- ORM: GORM

## 端口
- 后端 API: 8080
- 前端开发: 3000 (代理到 8080/api)

## 重要约束
- **不要使用 `@ts-ignore` 或 `any`** - TypeScript 严格模式
- **不要留空 catch 块**
- **Go 中始终处理错误**
- **前端使用类型安全的 API 调用**
- **遵循现有的错误日志模式**（logger.Error/Errorf/ErrorContext 用于可恢复错误）

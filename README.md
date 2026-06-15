# 海洋测绘声呐图标注协同系统

面向海洋测绘研究院的船载侧扫声呐回波图像在线标注平台，支持多人实时协同标注水下目标（礁石、沉船、管线等）。

## 技术架构

```
┌───────────────────────────────────────────────────────────┐
│                        前端 (Vue3 + TS)                   │
├─────────────────┬─────────────────┬───────────────────────┤
│  Canvas渲染引擎  │  标注工具模块   │  协同状态同步模块      │
│  - 8K大图加载   │  - 矩形标注     │  - WebSocket实时同步   │
│  - 平移缩放     │  - 多边形标注   │  - 本地草稿缓存        │
│  - 性能优化     │  - 分类标签     │  - 版本快照管理        │
├─────────────────┴─────────────────┴───────────────────────┤
│  UI组件：左侧目录树 | 中央绘图画布 | 右侧工具栏/分类面板    │
└───────────────────────────────────────────────────────────┘
                              │
                        HTTP / WebSocket
                              │
┌───────────────────────────────────────────────────────────┐
│                     后端 (Golang + Gin)                   │
├─────────────────┬─────────────────┬───────────────────────┤
│  文件存储接口   │  标注提交校验   │  WebSocket广播服务     │
│  标注CRUD接口   │  参数校验器     │  按文件房间隔离广播    │
│  快照回滚接口   │  事务处理       │  在线用户状态管理      │
└─────────────────┴─────────────────┴───────────────────────┘
                              │
          ┌───────────────────┴───────────────────┐
          │                                       │
  ┌───────────────┐                      ┌────────────────┐
  │  PostgreSQL   │                      │     Redis      │
  │  - 声呐文件   │                      │  - 在线用户    │
  │  - 标注数据   │                      │  - 标注缓存    │
  │  - 目标分类   │                      │  - 协同状态    │
  │  - 快照版本   │                      │                │
  └───────────────┘                      └────────────────┘
```

## 项目结构

```
a62/
├── frontend/                          # 前端 Vue3 + TS 项目
│   ├── src/
│   │   ├── components/                # UI 组件
│   │   │   ├── FileTreePanel.vue      # 左侧文件目录树
│   │   │   └── AnnotationPanel.vue    # 右侧标注工具栏
│   │   ├── composables/               # 组合式函数
│   │   │   ├── useSonarCanvas.ts      # Canvas 渲染引擎
│   │   │   ├── useAnnotationTool.ts   # 标注工具逻辑
│   │   │   └── useWebSocket.ts        # WebSocket 协同
│   │   ├── stores/                    # Pinia 状态管理
│   │   │   └── annotation.ts          # 标注状态与本地缓存
│   │   ├── types/                     # TypeScript 类型定义
│   │   ├── utils/                     # 工具函数
│   │   ├── views/                     # 页面视图
│   │   │   └── EditorView.vue         # 主编辑器页面
│   │   ├── router/                    # 路由配置
│   │   └── styles/                    # 全局样式
│   ├── package.json
│   ├── vite.config.ts
│   └── tsconfig.json
├── backend/                           # 后端 Golang 项目
│   ├── internal/
│   │   ├── config/                    # 配置加载
│   │   ├── models/                    # 数据模型 & GORM 实体
│   │   ├── database/                  # 数据库连接 (PG + Redis)
│   │   ├── handlers/                  # API 处理器
│   │   │   ├── files.go               # 声呐文件接口
│   │   │   ├── annotations.go         # 标注CRUD接口
│   │   │   ├── categories.go          # 分类接口
│   │   │   └── snapshots.go           # 快照版本接口
│   │   └── ws/                        # WebSocket 服务
│   │       └── hub.go                 # 协同广播 Hub
│   ├── pkg/
│   │   ├── validation/                # 参数校验
│   │   └── utils/                     # 工具函数
│   ├── main.go                        # 应用入口
│   └── go.mod
├── database/
│   └── migrations/
│       └── 001_init_schema.sql        # 数据库初始化脚本
├── configs/
│   └── backend.env                    # 后端环境变量配置
└── README.md
```

## 数据流图

```
标注操作流程:
用户点击画布 → useAnnotationTool 捕获点 → 创建标注对象 → 
→ API POST /annotations → 后端校验 → 保存 PG → 生成快照 → 
→ WebSocket 广播 → 其他客户端实时更新 → 本地草稿持久化

协同同步流程:
客户端A标注 → WS消息 → Hub按文件房间广播 → 客户端B/C接收 → 
→ 更新本地状态 → Canvas 重绘 → 200ms内完成同步
```

## 系统要求

- **Node.js**: >= 18.0
- **Golang**: >= 1.21
- **PostgreSQL**: >= 12
- **Redis**: >= 6.0

## 快速开始

### 1. 启动数据库服务

```bash
# PostgreSQL
docker run --name sonar-pg -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=sonar_annotation -p 5432:5432 -d postgres:14

# Redis
docker run --name sonar-redis -p 6379:6379 -d redis:7
```

### 2. 初始化数据库

```bash
# 执行迁移脚本
psql -h localhost -U postgres -d sonar_annotation \
  -f database/migrations/001_init_schema.sql
```

### 3. 启动后端服务

```bash
cd backend

# 安装依赖
go mod download

# 启动服务 (默认端口 8080)
go run main.go
```

### 4. 启动前端服务

```bash
cd frontend

# 安装依赖
npm install

# 启动开发服务器 (默认端口 5173)
npm run dev
```

### 5. 访问应用

打开浏览器访问 `http://localhost:5173`

## API 接口文档

### 声呐文件接口

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/files` | 获取文件列表 |
| POST | `/api/files` | 上传声呐图像 |
| GET | `/api/files/:id` | 获取文件信息 |
| GET | `/api/files/:id/image` | 获取图像数据 |

### 标注接口

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/annotations/file/:fileId` | 获取文件标注列表 |
| POST | `/api/annotations` | 创建标注 |
| PUT | `/api/annotations/:id` | 更新标注 |
| DELETE | `/api/annotations/:id` | 删除标注 |

### 快照接口

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/snapshots/file/:fileId` | 获取版本快照列表 |
| POST | `/api/snapshots/restore/:id` | 回滚到指定版本 |

### 分类接口

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/categories` | 获取目标分类列表 |
| POST | `/api/categories` | 创建新分类 |

### WebSocket 接口

```
ws://localhost:8080/ws/annotate/:fileId?userId=xxx&userName=xxx
```

消息类型:
- `annotation-create`: 创建标注
- `annotation-update`: 更新标注  
- `annotation-delete`: 删除标注
- `cursor-move`: 光标位置同步
- `user-join`: 用户加入
- `user-leave`: 用户离开

## 功能特性

### 前端模块

1. **Canvas 声呐渲染引擎** ([useSonarCanvas.ts](file:///d:/trae3/a62/frontend/src/composables/useSonarCanvas.ts))
   - 支持 8K 超大尺寸灰度图加载
   - requestAnimationFrame 渲染循环，无卡顿
   - 鼠标滚轮缩放（0.1x ~ 5x）
   - Alt+拖动 / 中键拖动平移
   - 双指触控捏合缩放（平板支持）
   - DPR 自适应高清渲染
   - 图像平滑降级处理

2. **标注工具** ([useAnnotationTool.ts](file:///d:/trae3/a62/frontend/src/composables/useAnnotationTool.ts))
   - 矩形标注 (快捷键 R)
   - 多边形标注 (快捷键 P)
   - 自定义目标分类标签
   - 实时绘制预览
   - 标注点磁吸闭合

3. **协同同步** ([useWebSocket.ts](file:///d:/trae3/a62/frontend/src/composables/useWebSocket.ts))
   - 自动重连机制（指数退避）
   - 在线用户实时显示
   - 光标位置同步
   - 操作即时广播

4. **本地草稿缓存** ([annotation.ts](file:///d:/trae3/a62/frontend/src/stores/annotation.ts))
   - Pinia 持久化插件
   - 断网数据不丢失
   - 按文件隔离缓存

5. **UI 组件**
   - 左侧：文件目录树 + 搜索 + 上传
   - 中央：灰度声呐绘图画布 + 缩放控制 + 在线用户
   - 右侧：工具栏 + 分类选择 + 标注列表 + 版本历史 + 快捷键

### 后端模块

1. **声呐文件存储** ([files.go](file:///d:/trae3/a62/backend/internal/handlers/files.go))
   - 大文件分片上传支持（最大 100MB）
   - 自动解析图像尺寸
   - 文件服务静态化

2. **标注提交校验** ([annotation_validator.go](file:///d:/trae3/a62/backend/pkg/validation/annotation_validator.go))
   - 标注类型校验
   - 点坐标几何校验
   - 矩形/多边形规则校验

3. **WebSocket 广播** ([hub.go](file:///d:/trae3/a62/backend/internal/ws/hub.go))
   - 按文件房间隔离
   - 读写 goroutine 分离
   - 心跳检测（54s ping）
   - 带缓冲通道防阻塞
   - 200ms 内同步所有客户端

4. **版本快照管理** ([snapshots.go](file:///d:/trae3/a62/backend/internal/handlers/snapshots.go))
   - 每次操作自动生成快照
   - 保留最近 30 个版本
   - 一键回滚
   - 事务保证一致性

5. **数据库实体映射** ([models.go](file:///d:/trae3/a62/backend/internal/models/models.go))
   - GORM 自动迁移
   - JSONB 存储标注点
   - UUID 主键
   - 自动时间戳

### 性能优化

- **前端**: Canvas 分层渲染、requestAnimationFrame 循环、DPR 自适应、图像按需平滑
- **后端**: Redis 缓存标注数据、数据库连接池、WebSocket 房间广播、批量操作事务
- **数据库**: JSONB 索引、热点查询索引、连接池配置

## 快捷键

| 快捷键 | 功能 |
|--------|------|
| R | 矩形标注工具 |
| P | 多边形标注工具 |
| Esc | 取消绘制 / 退出工具 |
| Enter | 闭合多边形 |
| Delete | 删除选中标注 |
| 滚轮 | 缩放视图 |
| Alt + 拖动 | 平移视图 |

## 默认目标分类

| 分类 | 颜色 | 描述 |
|------|------|------|
| 礁石 | 🔴 #ff4d4f | 水下礁石 |
| 沉船 | 🟠 #faad14 | 沉船残骸 |
| 管线 | 🔵 #1890ff | 海底管线 |
| 锚 | 🟣 #722ed1 | 船锚 |
| 渔网 | 🔵 #13c2c2 | 废弃渔网 |
| 其他 | ⚪ #8c8c8c | 其他目标 |

## 配置说明

后端配置文件: [backend.env](file:///d:/trae3/a62/configs/backend.env)

```env
SERVER_PORT=8080
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=sonar_annotation
REDIS_HOST=localhost
REDIS_PORT=6379
UPLOAD_DIR=./uploads
MAX_UPLOAD_SIZE=104857600  # 100MB
MAX_SNAPSHOTS=30
```

## 开发说明

### 目录规范

- 前端组件放在 `frontend/src/components/`
- 业务逻辑放在 `frontend/src/composables/`
- 状态管理放在 `frontend/src/stores/`
- 后端处理器放在 `backend/internal/handlers/`
- 通用工具放在 `backend/pkg/`

### 代码风格

- 前端: TypeScript Strict 模式
- 后端: Go 官方规范，go fmt 格式化
- 接口: RESTful 风格，统一错误格式

## 多人协同演示

1. 打开两个浏览器窗口访问 `http://localhost:5173`
2. 两个窗口上传同一张声呐图并打开
3. 在窗口 A 创建标注，观察窗口 B 是否在 200ms 内同步显示
4. 查看右侧在线用户列表，确认两个用户都显示
5. 测试快照回滚功能

## 故障排查

### 前端无法连接后端
- 检查 Vite 代理配置 `vite.config.ts`
- 确认后端服务在 8080 端口运行

### WebSocket 连接失败
- 检查 Nginx/代理的 WebSocket 升级配置
- 确认 Redis 服务运行正常

### 大图片加载卡顿
- 确认浏览器支持 `image.decoding = 'async'`
- 检查图片格式是否优化

### 标注不同步
- 检查 Redis 在线用户键 `sonar:online:{fileId}`
- 查看浏览器控制台 WebSocket 消息日志

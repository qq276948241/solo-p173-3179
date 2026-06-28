# 民宿短租订单系统 - 架构文档

## 目录结构

```
project173/
├── main.go                    # HTTP 服务入口，路由注册
├── go.mod / go.sum            # Go 模块依赖
├── config/
│   └── config.go              # 环境变量加载
├── database/
│   └── database.go            # SQLite 连接 + 自动迁移
├── models/
│   ├── user.go                # 用户模型（房东/租客）
│   ├── property.go            # 房源模型
│   ├── order.go               # 订单模型
│   └── review.go              # 评价模型
├── pkg/
│   ├── jwt/
│   │   ├── jwt.go             # JWT 签发/解析
│   │   └── jwt_test.go        # JWT 单元测试
│   └── statemachine/
│       ├── order_statemachine.go   # 订单状态机
│       └── order_statemachine_test.go  # 状态机单元测试
├── middleware/
│   ├── auth.go                # JWT 鉴权 + 角色校验中间件
│   └── auth_test.go           # 中间件单元测试
├── services/
│   └── review_service.go      # 评价业务逻辑层
├── handlers/
│   ├── user_handler.go        # 用户接口（注册/登录/当前用户）
│   ├── property_handler.go    # 房源接口（CRUD/上下架/筛选）
│   ├── order_handler.go       # 订单接口（创建/查询/状态流转）
│   └── review_handler.go      # 评价接口（创建/查询）
└── docs/
    └── architecture.md        # 本文档
```

---

## 一、启动流程（从 main.go 开始）

入口文件：[main.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo173/project173/main.go#L1-L75)

### 1. 加载配置

```go
cfg := config.Load()
```

调用 [config.Load()](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo173/project173/config/config.go#L11-L29)，从环境变量读取：
- `DB_PATH`：SQLite 数据库文件路径，默认 `bnb.db`
- `JWT_SECRET`：JWT 签名密钥，默认 `bnb-secret-key-change-in-production`
- `SERVER_PORT`：HTTP 服务端口，默认 `8080`

环境变量不存在时使用合理默认值，不强制配置。

### 2. 初始化数据库

```go
database.Init(cfg.DBPath)
```

调用 [database.Init()](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo173/project173/database/database.go#L12-L19)：
- 使用 `gorm.io/driver/sqlite` 打开 SQLite 数据库
- 调用 `AutoMigrate` 自动创建/更新 4 张表：`users`、`properties`、`orders`、`reviews`
- 数据库实例存入全局变量 `database.DB` 供全局使用

### 3. 初始化业务组件

```go
jwtService := jwt.NewService(cfg.JWTSecret)
userHandler := handlers.NewUserHandler(jwtService)
propertyHandler := handlers.NewPropertyHandler()
orderHandler := handlers.NewOrderHandler()
reviewService := services.NewReviewService(database.DB)
reviewHandler := handlers.NewReviewHandler(reviewService)
```

- JWT 服务独立封装，通过构造函数注入密钥
- Handler 层通过构造函数注入依赖（`reviewHandler` 依赖 `reviewService`，`reviewService` 依赖 `*gorm.DB`）

### 4. 注册路由

使用 Gin 框架，路由分组如下：

```
/api
├── /auth
│   ├── POST /register       # 公开
│   └── POST /login          # 公开
├── GET  /properties              # 公开
├── GET  /properties/:id          # 公开
├── GET  /properties/:id/reviews  # 公开
└── / (需鉴权 AuthMiddleware)
    ├── GET  /auth/me
    ├── / (需房东角色 RoleMiddleware)
    │   ├── POST   /properties
    │   ├── PUT    /properties/:id
    │   ├── DELETE /properties/:id
    │   └── PATCH  /properties/:id/status
    ├── POST /orders
    ├── GET  /orders
    ├── GET  /orders/:id
    ├── POST /orders/:id/transition
    └── POST /reviews
```

---

## 二、中间件层

### JWT 鉴权中间件 [middleware/auth.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo173/project173/middleware/auth.go#L21-L58)

`AuthMiddleware(jwtService)` 执行流程：
1. 读取 `Authorization` 请求头，校验格式为 `Bearer <token>`
2. 调用 `jwtService.ParseToken()` 解析 token
3. 解析成功后，将 `user_id` 和 `user_role` 注入 Gin 上下文（`c.Set()`）
4. 解析失败直接返回 401，终止请求链

### 角色校验中间件 [middleware/auth.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo173/project173/middleware/auth.go#L60-L82)

`RoleMiddleware("landlord")` 执行流程：
1. 从 Gin 上下文读取已注入的 `user_role`
2. 与允许的角色列表比对，匹配则放行
3. 不匹配返回 403，终止请求链

### 上下文工具函数

- `GetUserID(c)`：从上下文取出当前登录用户 ID
- `GetUserRole(c)`：从上下文取出当前登录用户角色

---

## 三、JWT 服务

实现文件：[pkg/jwt/jwt.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo173/project173/pkg/jwt/jwt.go)

### 签发 Token

```go
token, err := jwtService.GenerateToken(userID, role)
```

- 使用 HS256 签名算法
- Payload 包含：`user_id`（uint）、`role`（string）
- 默认 24 小时过期，可通过 `SetExpireHour()` 调整
- 自定义 `Claims` 结构体嵌入 `jwt.RegisteredClaims`

### 解析 Token

```go
claims, err := jwtService.ParseToken(tokenStr)
```

- 校验签名算法必须是 HMAC（防 `alg: none` 攻击）
- 校验签名正确性、过期时间
- 解析失败返回具体错误：无效 token、过期、签名错误等

### 单元测试

[pkg/jwt/jwt_test.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo173/project173/pkg/jwt/jwt_test.go) 覆盖 7 个场景：
- 正常签发+解析
- 不同角色
- 伪造 token
- 密钥不匹配
- 过期 token
- 错误签名算法
- 过期时间防御性校验

---

## 四、数据模型层

### 4.1 用户模型 [models/user.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo173/project173/models/user.go)

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| `id` | uint | PRIMARY KEY | 用户ID |
| `username` | string | UNIQUE, NOT NULL | 用户名 |
| `password_hash` | string | NOT NULL | bcrypt 哈希后的密码 |
| `role` | string | NOT NULL | `landlord` 或 `tenant` |
| `created_at` / `updated_at` | time.Time | - | GORM 自动维护 |
| `deleted_at` | gorm.DeletedAt | INDEX | 软删除 |

- `SetPassword()`：使用 bcrypt 哈希密码（cost=14）
- `CheckPassword()`：比对密码
- `Validate()`：校验用户名非空、角色合法

### 4.2 房源模型 [models/property.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo173/project173/models/property.go)

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| `id` | uint | PRIMARY KEY | 房源ID |
| `landlord_id` | uint | NOT NULL, INDEX | 房东用户ID，外键→users.id |
| `title` | string | NOT NULL | 房源标题 |
| `description` | string | - | 房源描述 |
| `city` | string | NOT NULL, INDEX | 城市（用于筛选） |
| `address` | string | NOT NULL | 详细地址 |
| `price_per_day` | float64 | NOT NULL, INDEX | 日租金（用于价格筛选） |
| `status` | string | NOT NULL, INDEX | `online` 或 `offline`，默认 offline |

- `Validate()`：校验必填字段、价格为正
- `IsOnline()`：判断房源是否上架

### 4.3 订单模型 [models/order.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo173/project173/models/order.go)

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| `id` | uint | PRIMARY KEY | 订单ID |
| `order_no` | string | UNIQUE, NOT NULL | 订单号（BN+时间戳+随机数） |
| `property_id` | uint | NOT NULL, INDEX | 房源ID，外键→properties.id |
| `tenant_id` | uint | NOT NULL, INDEX | 租客ID，外键→users.id |
| `landlord_id` | uint | NOT NULL, INDEX | 房东ID，外键→users.id |
| `check_in_date` | time.Time | NOT NULL | 入住日期 |
| `check_out_date` | time.Time | NOT NULL | 退房日期 |
| `days` | int | NOT NULL | 入住天数 |
| `total_amount` | float64 | NOT NULL | 总价 = days × price_per_day |
| `status` | string | NOT NULL, INDEX | 订单状态（见状态机） |

### 4.4 评价模型 [models/review.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo173/project173/models/review.go)

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| `id` | uint | PRIMARY KEY | 评价ID |
| `order_id` | uint | NOT NULL, UNIQUE | 订单ID，外键→orders.id，一个订单只能评一次 |
| `property_id` | uint | NOT NULL | 房源ID，外键→properties.id |
| `tenant_id` | uint | NOT NULL, INDEX | 租客ID，外键→users.id |
| `rating` | int | NOT NULL, INDEX | 评分 1-5 |
| `comment` | string | - | 文字评论 |
| `created_at` | time.Time | INDEX | 联合索引 (property_id, created_at DESC) |

### 4.5 表关系与外键约束

```
users (id)
  ├─ properties (landlord_id) → ON DELETE CASCADE（删房东连带删房源）
  ├─ orders (tenant_id)       → ON DELETE RESTRICT（有订单的租客不能删）
  ├─ orders (landlord_id)     → ON DELETE RESTRICT
  └─ reviews (tenant_id)      → ON DELETE RESTRICT

properties (id)
  ├─ orders (property_id)     → ON DELETE CASCADE
  └─ reviews (property_id)    → ON DELETE CASCADE

orders (id)
  └─ reviews (order_id)       → ON DELETE RESTRICT（有评价的订单不能删）
```

---

## 五、订单状态机

实现文件：[pkg/statemachine/order_statemachine.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo173/project173/pkg/statemachine/order_statemachine.go)

### 五个状态

| 状态 | 说明 |
|---|---|
| `unpaid` | 未支付 |
| `paid` | 已支付 |
| `checked_in` | 已入住 |
| `checked_out` | 已退房 |
| `canceled` | 已取消 |

### 状态流转图

```
                    ┌───────────┐
                    │  unpaid   │
                    └─────┬─────┘
                          │
              ┌───────────┴───────────┐
              ▼                       ▼
        ┌───────────┐           ┌───────────┐
        │   paid    │           │ canceled  │
        └─────┬─────┘           └───────────┘
              │
    ┌─────────┴─────────┐
    ▼                   ▼
┌────────────┐    ┌───────────┐
│ checked_in │    │ canceled  │
└──────┬─────┘    └───────────┘
       │
       ▼
┌────────────┐
│checked_out │
└────────────┘
```

### 合法跳转表

| 当前状态 | 事件 | 目标状态 | 操作人 |
|---|---|---|---|
| `unpaid` | `pay` | `paid` | 租客 |
| `unpaid` | `cancel` | `canceled` | 租客 |
| `paid` | `check_in` | `checked_in` | 房东 |
| `paid` | `cancel` | `canceled` | 租客 |
| `checked_in` | `check_out` | `checked_out` | 房东 |
| `checked_out` | - | - | 终态 |
| `canceled` | - | - | 终态 |

### 核心 API

```go
nextStatus, err := statemachine.Transition(currentStatus, event)
// 非法跳转返回 error: "illegal state transition"
```

### 单元测试

[pkg/statemachine/order_statemachine_test.go](file:///d:/code/ai-prompt/solo-chrome-dev-F12/repos/repo173/project173/pkg/statemachine/order_statemachine_test.go) 覆盖 15 个场景：
- 全部合法跳转（含完整下单→入住→退房流程）
- 全部非法跳转（共 11 种，如 unpaid 不能直接 check_in）
- 无效状态输入
- `CanTransition()` 预检
- `AllowedEvents()` 查询当前状态可执行事件

---

## 六、完整请求链路示例：租客登录 → 下单

### 场景
租客 `alice` 登录系统，拿到 JWT 后，为房源 `#1` 创建一笔 3 天的订单。

---

#### 步骤 1：POST /api/auth/login（登录）

**前端请求**：
```json
POST /api/auth/login
Content-Type: application/json

{
  "username": "alice",
  "password": "secret123"
}
```

**请求链路**：
```
1. Gin 路由匹配 → handlers.NewUserHandler.Login()
   └─ 解析 JSON 到 LoginRequest
2. 查库：database.DB.Where("username = ?", "alice").First(&user)
3. 密码校验：user.CheckPassword("secret123") → bcrypt 比对
4. 签发 Token：jwtService.GenerateToken(user.ID, "tenant")
   └─ HS256 签名，24 小时过期
5. 返回响应
```

**后端响应**：
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 42,
    "username": "alice",
    "role": "tenant"
  }
}
```

---

#### 步骤 2：POST /api/orders（下单，需鉴权）

**前端请求**：
```json
POST /api/orders
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "property_id": 1,
  "check_in_date": "2026-07-10T00:00:00Z",
  "check_out_date": "2026-07-13T00:00:00Z"
}
```

**请求链路**：
```
1. Gin 路由匹配 → AuthMiddleware → RoleMiddleware → handlers.NewOrderHandler.Create()

   ┌─ AuthMiddleware ──────────────────────────────────────────────┐
   │ 1. 读取 Authorization 头，提取 Bearer token                 │
   │ 2. jwtService.ParseToken() → 校验签名+过期，得到 claims       │
   │    {user_id: 42, role: "tenant"}                             │
   │ 3. c.Set("user_id", 42)、c.Set("user_role", "tenant")        │
   │ 4. 放行，进入下一层中间件                                     │
   └──────────────────────────────────────────────────────────────┘

   ┌─ RoleMiddleware (tenant) ───────────────────────────────────┐
   │ 1. 从上下文读 user_role = "tenant"                           │
   │ 2. 匹配允许列表，放行                                         │
   └──────────────────────────────────────────────────────────────┘

2. orderHandler.Create() 执行业务逻辑：
   ├─ 从上下文取 tenantID = 42，校验 role == "tenant"
   ├─ 查房源 #1，校验 status == "online"（必须上架）
   ├─ 计算天数 = 3，总价 = 3 × 房源.price_per_day
   ├─ 生成订单号：BN20260628163000 + 4位随机数
   ├─ 构造 Order 结构体，status 默认 "unpaid"
   ├─ order.Validate() 校验
   └─ database.DB.Create(&order) 落库

3. 返回响应
```

**后端响应**：
```json
{
  "id": 1001,
  "order_no": "BN202606281630001234",
  "property_id": 1,
  "tenant_id": 42,
  "landlord_id": 10,
  "check_in_date": "2026-07-10T00:00:00Z",
  "check_out_date": "2026-07-13T00:00:00Z",
  "days": 3,
  "total_amount": 1500.0,
  "status": "unpaid",
  "created_at": "2026-06-28T16:30:00Z"
}
```

---

#### 订单后续状态流转（可选）

| 操作 | 接口 | 状态变化 |
|---|---|---|
| 租客支付 | `POST /api/orders/1001/transition {event: "pay"}` | `unpaid` → `paid` |
| 房东办理入住 | `POST /api/orders/1001/transition {event: "check_in"}` | `paid` → `checked_in` |
| 房东办理退房 | `POST /api/orders/1001/transition {event: "check_out"}` | `checked_in` → `checked_out` |
| 租客评价 | `POST /api/reviews {order_id: 1001, rating: 5}` | 生成评价记录 |

---

## 七、代码分层原则

```
┌───────────────────────────────────────────────────┐
│              HTTP 层 (Gin Handler)                │
│  handlers/*_handler.go                            │
│  - 解析请求参数、绑定校验                          │
│  - 调用 service / 直接查库                        │
│  - 错误映射到 HTTP 状态码                         │
│  - 返回 JSON 响应                                 │
└──────────────────┬────────────────────────────────┘
                   │
┌──────────────────▼────────────────────────────────┐
│              Service 层 (可选)                     │
│  services/review_service.go                       │
│  - 封装业务逻辑（权限校验、状态校验、分页）        │
│  - 定义领域错误（sentinel errors）                │
│  - 数据库操作封装                                  │
└──────────────────┬────────────────────────────────┘
                   │
┌──────────────────▼────────────────────────────────┐
│              领域层                                │
│  pkg/jwt/、pkg/statemachine/                      │
│  - 纯业务逻辑，无 HTTP 依赖                        │
│  - 可独立单元测试                                  │
└──────────────────┬────────────────────────────────┘
                   │
┌──────────────────▼────────────────────────────────┐
│              模型层                                │
│  models/*.go                                      │
│  - 定义表结构、字段约束                            │
│  - 模型级校验（Validate()）                       │
└──────────────────┬────────────────────────────────┘
                   │
┌──────────────────▼────────────────────────────────┐
│              基础设施层                            │
│  database/database.go、config/config.go           │
│  - SQLite 连接、自动迁移                          │
│  - 环境变量加载                                    │
└───────────────────────────────────────────────────┘
```

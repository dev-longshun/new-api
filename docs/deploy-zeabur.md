# Zeabur 部署指南

## 镜像地址

```
ghcr.io/dev-longshun/new-api:latest
```

| 标签 | 说明 |
|------|------|
| `latest` | 最新正式版（打 `v*` tag 时更新） |
| `beta` | 最新 main 分支构建 |
| `vX.Y.Z` | 指定版本，如 `v0.11.9` |

---

## 部署步骤

### 1. 创建服务

Zeabur 控制台 → Add Service → Prebuilt Image → 输入上方镜像地址。

### 2. 添加数据库服务（推荐 PostgreSQL）

在同一 Project 中添加 PostgreSQL 服务：

Add Service → Marketplace → PostgreSQL

记录自动生成的连接信息，后续配置 `SQL_DSN` 时使用。

### 3. 添加 Redis 服务（可选，推荐）

Add Service → Marketplace → Redis

启用后可获得更好的缓存性能和多节点支持。

### 4. 挂载持久化卷

在 new-api 服务的 Volumes 中添加：

- Mount Directory：`/data`
- 作用：保存 SQLite 数据库文件（若使用 PostgreSQL 则非必须，但建议保留用于日志等）

### 5. 配置环境变量

在 new-api 服务的 Variables 中添加以下环境变量：

| 变量名 | 说明 | 示例值 |
|--------|------|--------|
| `SQL_DSN` | 数据库连接串（留空则使用 SQLite） | `postgresql://user:pass@host:5432/new-api` |
| `REDIS_CONN_STRING` | Redis 连接串（留空则禁用 Redis） | `redis://:password@host:6379` |
| `SESSION_SECRET` | Session 加密密钥，**必须修改** | 任意随机字符串 |
| `TZ` | 时区 | `Asia/Shanghai` |
| `BATCH_UPDATE_ENABLED` | 启用批量写入优化 | `true` |

> Zeabur 中 PostgreSQL/Redis 服务的连接信息可在对应服务的 Connect 面板中直接复制。

**可选变量：**

| 变量名 | 说明 |
|--------|------|
| `MEMORY_CACHE_ENABLED` | 启用内存缓存（`true`） |
| `ERROR_LOG_ENABLED` | 启用错误日志（`true`） |
| `SYNC_FREQUENCY` | 定期同步数据库的间隔秒数 |

### 6. 开放网络端口

在 Networking 中开放端口 `3000`，并绑定自定义域名（可选）。

---

## 不需要配置的项

- **Dockerfile** — 使用预构建镜像，无需提供
- **启动命令（Command）** — 镜像内置 `ENTRYPOINT ["/new-api"]`，无需填写

---

## 常见问题

### 首次登录

默认账号：`root` / `123456`，**登录后立即修改密码**。

### 数据库连接失败

确认 `SQL_DSN` 格式正确：
- PostgreSQL：`postgresql://user:password@host:5432/dbname`
- MySQL：`user:password@tcp(host:3306)/dbname`
- 留空：自动使用 SQLite，数据存储在 `/data/new-api.db`

### 重启后数据丢失（SQLite 模式）

确保已挂载 `/data` 持久化卷。未挂载时容器重启会丢失 SQLite 文件。

### 502 / 服务不可用

检查服务日志，常见原因：
1. `SQL_DSN` 指向的数据库尚未就绪 — 等待数据库服务启动完成后重启 new-api
2. 端口未正确开放 — 确认 Networking 中开放了 `3000`

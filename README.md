# go-task-queue

以 Go、PostgreSQL 和 Redis 打造的分散式任務佇列系統。

## 系統架構

```mermaid
flowchart TD
    Req["POST /tasks"] --> API["API Server (:8080)"]

    API -- "push job ID" --> Redis["Redis Queue"]
    API -- "store/fetch" --> Redis

    API --> DB["PostgreSQL (jobs)"]

    Redis -- "BRPOP" --> Worker["Worker"]
    Worker --> DB
```

**資料流程：**
1. `POST /tasks` → 將 job 存入 PostgreSQL，並把 job ID 推進 Redis list
2. Worker 持續運行：從 Redis `BRPop` → 從 PostgreSQL 查詢 job → 更新狀態 → 執行 → 再次更新狀態
3. Redis 只存 job ID，job 的完整資料以 PostgreSQL 為主

## 環境需求

- Docker & Docker Compose
- Go 1.21+

## 初始化設定

```bash
# 啟動 PostgreSQL + Redis
docker compose up -d

# 套用資料庫 schema（只需執行一次）
psql postgres://taskqueue:taskqueue@localhost:5432/taskqueue?sslmode=disable -f migrations/001_create_jobs.sql
```

## 啟動服務

分別在兩個終端機視窗執行：

```bash
# HTTP API 伺服器（port 8080）
go run ./cmd/server

# 背景 Worker
go run ./cmd/worker
```

## API

### 建立任務

```
POST /tasks
Content-Type: application/json

{
  "type": "send_email",
  "payload": { "to": "user@example.com" }
}
```

**回應 `201`：**
```json
{
  "id": "abc123",
  "type": "send_email",
  "payload": { "to": "user@example.com" },
  "status": "pending",
  "retry_count": 0,
  "max_retries": 0,
  "created_at": "2026-04-06T00:00:00Z"
}
```

### 查詢任務狀態

```
GET /tasks/:id
```

回傳指定 job 的完整資料，找不到回傳 `404`。

### 取得下一個任務（外部 Worker 用）

```
GET /tasks/next
```

最多等待 5 秒，若無任務則回傳 `204`。

## Job 狀態流程

```
pending → running → done
                  → failed → retry → running（重試）
                                   → dead（超過上限）
```

| 狀態      | 說明                              |
|-----------|-----------------------------------|
| `pending` | 任務已建立，等待處理              |
| `running` | Worker 已取得，正在執行中         |
| `done`    | 執行成功完成                      |
| `failed`  | 執行失敗                          |
| `dead`    | 超過最大重試次數，永久放棄        |

## 支援的任務類型

| Type           | 說明         |
|----------------|--------------|
| `send_email`   | 模擬寄送郵件 |
| `resize_image` | 模擬調整圖片 |

## 連線設定（預設值）

| 服務       | 連線字串                                                     |
|------------|--------------------------------------------------------------|
| PostgreSQL | `postgres://taskqueue:taskqueue@localhost:5432/taskqueue?sslmode=disable` |
| Redis      | `localhost:6379`                                             |
| Queue key  | `task_queue:default`                                         |

## 建置與測試

```bash
go build ./...
go test ./...
go test ./internal/queue/...   # 單一套件測試
```

## 已知限制

- 任務 handler（`send_email`、`resize_image`）目前以 `time.Sleep` 模擬，尚未接真實邏輯

# TÀI LIỆU THIẾT KẾ KIẾN TRÚC
# HỆ THỐNG QUẢN LÝ SERVER (SMS MICROSERVICES)

## 1. TỔNG QUAN HỆ THỐNG
### 1.1 Mục tiêu hệ thống
Hệ thống Server Management System (SMS) là nền tảng quản trị giúp theo dõi trạng thái hoạt động của hàng vạn (10k+) máy chủ đích (Target Servers) thông qua giao thức ICMP Ping. 
Được tái cấu trúc từ mô hình Monolithic sang **Microservices** và **Event-Driven Architecture (EDA)**, hệ thống giải quyết triệt để các giới hạn về khả năng mở rộng, phân tách quyền sở hữu dữ liệu (Domain-Driven Design), và cung cấp cơ chế tính toán Uptime tự động với độ trễ thấp thông qua Elasticsearch.

### 1.2 Sơ đồ System Context
**[Placeholder: Sơ đồ System Context]**

**Chú thích sơ đồ:**
- **System Administrator (Admin):** Người dùng trực tiếp, thực hiện quản trị, import/export server, xem báo cáo thống kê và nhận thông báo cảnh báo qua email.
- **Server Management System (SMS):** Hệ thống phần mềm trung tâm, đóng vai trò giám sát, thu thập dữ liệu và báo cáo.
- **Target Servers (10k+):** Hàng nghìn máy chủ đích nằm trong hạ tầng cần được giám sát. Hệ thống liên tục gửi gói tin ICMP (Ping) đến các server này.
- **SMTP Server:** Hệ thống gửi email bên ngoài (như MailHog hoặc SendGrid). SMS giao tiếp với SMTP để gửi báo cáo uptime định kỳ.

---

## 2. KIẾN TRÚC TỔNG THỂ
### 2.1 Container Diagram
Hệ thống được thiết kế theo chuẩn C4, bóc tách thành các Container (Microservices) độc lập kết nối qua API Gateway và Event Bus.

**[Placeholder: Sơ đồ Container / Phân rã Dịch vụ]**

**Chú thích sơ đồ Container:**
- **API Gateway (Traefik):** Reverse proxy định tuyến traffic, cung cấp bảo mật tập trung (Rate Limiting, ForwardAuth).
- **Web App (Angular):** Frontend Single Page Application giao tiếp với Backend qua REST/HTTPS.
- **SMS Identity (Go):** Dịch vụ quản lý xác thực, cấp phát JWT và quản lý vòng đời Session/Token.
- **SMS Management (Go):** Dịch vụ quản lý cấu hình máy chủ, CRUD, và phát đi sự kiện qua Outbox Pattern.
- **SMS Monitoring Worker (Go):** Background Worker tự động đồng bộ mục tiêu, thực thi Ping và đẩy log lên Elasticsearch.
- **SMS Reporting (Go):** Dịch vụ tự trị tính toán % Uptime và phát hành báo cáo.
- **PostgreSQL (Database):** CSDL quan hệ lưu metadata.
- **Redis (Streams & Cache):** Event Bus trung tâm (Pub/Sub), Distributed Lock, và Cache trạng thái siêu tốc.
- **Elasticsearch (Log Storage):** CSDL Time-series lưu trữ Observation Logs.

### 2.2 Chi tiết các khối công nghệ
| Lớp / Phân hệ | Công nghệ | Phiên bản | Vai trò |
| :--- | :--- | :--- | :--- |
| **Backend Core** | Go | 1.22+ | Xử lý logic, hiệu năng cao, goroutine pool |
| **API Protocol** | gRPC + grpc-gateway | v2 | Cung cấp song song gRPC native và REST HTTP |
| **API Gateway** | Traefik | v3.0 | Reverse Proxy, TLS, Rate Limit, ForwardAuth |
| **Frontend** | Angular | 19 | Single Page Application (SPA) |
| **Primary DB** | PostgreSQL | 15 | Cơ sở dữ liệu quan hệ (OLTP) cho các service |
| **Broker & Cache** | Redis | 7.x | Redis Streams (Event Bus), Distributed Lock, Session Cache. Cấu hình Persistent (AOF+RDB) |
| **Time-Series DB**| Elasticsearch | 8.17 | Lưu observation logs để query Aggregation uptime |
| **Email** | SMTP / MailHog | - | Dịch vụ gửi email báo cáo |
| **Ping Library** | pro-bing | 0.8 | Hỗ trợ ICMP unprivileged/privileged ping |

---

## 3. KIẾN TRÚC THÀNH PHẦN (COMPONENT)
Hệ thống tuân thủ kiến trúc Microservices. Mỗi service chứa các component chuyên biệt để xử lý luồng nghiệp vụ nội bộ.

### 3.1 SMS Identity
**[Placeholder: Sơ đồ Component của SMS Identity]**
**Mô tả:** Chịu trách nhiệm về Authentication. Chứa các component như ForwardAuth Handler, Auth Server, User Repo, Session Repo, RefreshToken Repo để quản lý phiên bản và Token Blacklist trên Redis.

### 3.2 SMS Management
**[Placeholder: Sơ đồ Component của SMS Management]**
**Mô tả:** Nơi duy nhất quản lý vòng đời Server. Chứa Server Service, Server Repo, và đặc biệt là **Outbox Relay Worker** dùng để đọc các sự kiện từ CSDL cục bộ và đẩy vào Redis Streams (sms.events.server).

### 3.3 SMS Monitoring
**[Placeholder: Sơ đồ Component của SMS Monitoring]**
**Mô tả:** Sử dụng mô hình Producer-Consumer siêu tốc. `Cycle Scheduler` bốc mục tiêu từ Redis Set ném vào Queue, `Worker Pool` liên tục rút Queue để Ping, sau đó `Observation Logger` ghi Bulk Insert lên Elasticsearch.

### 3.4 SMS Reporting
**[Placeholder: Sơ đồ Component của SMS Reporting]**
**Mô tả:** Dịch vụ hoàn toàn tự trị. Chứa `Event Consumer` để tự đồng bộ dữ liệu Server, `Daily Scheduler` để tạo Job, `Uptime Calculator` truy vấn ES, và `SMTP Notifier` để kết xuất HTML Template.

---

## 4. KIẾN TRÚC CHI TIẾT CÁC NGHIỆP VỤ & LUỒNG DỮ LIỆU

### 4.1 Nghiệp vụ Quản lý định danh (Identity)
Nghiệp vụ đảm nhiệm việc chứng thực (Authentication), cấp phát và quản lý vòng đời JWT.
Hệ thống sử dụng cơ chế **Token Theft Detection (Truy vết đánh cắp):** Nếu một Refresh Token đã bị thu hồi (`RevokedAt != nil`) nhưng vẫn được tái sử dụng, hệ thống lập tức xóa sổ toàn bộ Session và ghi cờ đen vào Redis Revocation Store.

#### 4.1.1 Các luồng xử lý
- **Luồng Đăng nhập (Login)**
  Xử lý xác thực thông tin, tạo và trả về Access Token, Refresh Token và thiết lập Cookie an toàn.
- **Quản lý Phiên (Session Management: Logout, LogoutAll, RefreshToken)**
  - **Làm mới Token (Refresh):** Cấp phát lại Access Token khi hết hạn mà không cần đăng nhập lại.
  - **Thu hồi Phiên (Logout/Revocation):** Khi người dùng Logout hoặc LogoutAll, token/session sẽ bị tước quyền và đẩy vào Blacklist (Redis Revocation Store).
- **Luồng Xác thực Token (ForwardAuth Verify)**
  Middleware Traefik luôn check API này để biết token có bị Revoked trong Redis hay không trước khi cho phép request đi tiếp.

#### 4.1.2 Thiết kế Database
**A. PostgreSQL (Schema: identity)**
- **USERS:** Lưu trữ `email`, `password_hash`, `role_code`.
- **AUTH_SESSIONS:** Lưu `id`, `user_id`, `expires_at`, `revoked_at`, `ip_address`, `user_agent`.
- **REFRESH_TOKENS:** Lưu `id`, `session_id`, `token_hash`, `replaced_by`, `revoked_at`.

**B. Redis Cache**
- **`revoked_session:{id}`:** (STRING) Lưu trữ session bị cấm để chặn Anti-Replay Attack cực nhanh.

### 4.2 Nghiệp vụ Quản lý Server (Management)
Cung cấp API CRUD và thao tác hàng loạt. Ứng dụng kỹ thuật Event-Driven để đảm bảo dữ liệu được nhân bản an toàn sang các service khác.

#### 4.2.1 Các luồng xử lý
- **Luồng Thêm/Sửa/Xóa Server (CRUD & Outbox Transaction)**
  Khi thêm Server, dữ liệu được ghi vào bảng SERVERS và OUTBOX_EVENTS trong **cùng một Transaction**. Tránh triệt để lỗi Dual-Write.
- **Xử lý hàng loạt (Bulk Import/Export)**
  - **Nhập Hàng Loạt (Import):** Phân lô batch 100 dòng để tối ưu I/O DB và tránh Out-of-Memory khi import hàng chục ngàn server.
  - **Xuất Hàng Loạt (Export):** Stream file CSV về client, tải dần dữ liệu tránh sập RAM.
- **Tiêu thụ Sự kiện Trạng thái (Status Consumer)**
  Worker lắng nghe trạng thái Server thay đổi từ Monitoring (qua Redis Streams) và cập nhật lại vào PostgreSQL một cách không đồng bộ.

#### 4.2.2 Thiết kế Database
**A. PostgreSQL (Schema: management)**
- **SERVERS:** Lưu thông tin gốc `server_id`, `server_name`, `ipv4`, `current_status`.
- **OUTBOX_EVENTS:** Chứa các sự kiện chưa được phát `event_type`, `payload`, `processed`.

**B. Redis Streams (Event Bus)**
- **`sms.events.server`:** Kênh Pub/Sub phát các sự kiện ServerCreated, ServerUpdated, ServerDeleted.

### 4.3 Nghiệp vụ Giám sát Server (Monitoring)
Chịu trách nhiệm thực thi Ping. Hoàn toàn không phụ thuộc vào Database quan hệ, chỉ dùng Redis để điều phối và ES để ghi log.

#### 4.3.1 Các luồng xử lý
- **Vòng lặp Giám sát (Monitoring Cycle & Flapping Logic)**
  **Quy trình Phân bổ tải & Bầu chọn Producer (Producer Election):**
  - **Distributed Lock:** Scheduler sử dụng lệnh `SET NX PX` với thời gian hết hạn của khóa (expiration) luôn nhỏ hơn chu kỳ Tick một chút (vd: `Tick - 2s`). Điều này đảm bảo khi hệ thống có N Replicas, luôn chỉ có duy nhất 1 Replica giữ vai trò cấp phát công việc (Producer), các Replicas khác sẽ bị chặn lại (Skip).
  - **Snowballing Prevention (Chống bùng nổ hàng đợi):** Ngay cả khi đã lấy được Lock, Producer vẫn phải kiểm tra độ dài hàng đợi hiện tại (`LLen monitoring:queue`). Nếu hàng đợi vẫn còn job (tức là các Worker xử lý không kịp), Producer sẽ lập tức **hủy bỏ chu kỳ đẩy mới** để tránh nhồi nhét thêm, ngăn chặn tình trạng tràn bộ nhớ Redis (Snowballing).
  - **Batch Push:** Nếu hàng đợi trống, Producer mới lấy toàn bộ ID/IP từ `server:all_ids` (lệnh `SMEMBERS`) và đẩy vào `monitoring:queue` thông qua một lệnh `RPUSH` duy nhất, giúp tối ưu tối đa số vòng lặp I/O (RTT).
- **Ping Worker & Ghi Log Observation**
  Sử dụng State Machine: Nếu ping thất bại >= Threshold (2) mới chính thức coi là OFFLINE để tránh Flapping do nhiễu mạng. Worker đẩy log vào Logger để Bulk Insert lên Elasticsearch.

#### 4.3.2 Thiết kế Data Store & Lock
- **Elasticsearch (`sms_observation_logs`):** Ghi bulk log theo chuỗi thời gian (time-series). Gồm `server_id` (keyword), `is_success` (boolean), `timestamp` (date). Cơ chế ghi log có buffer nội bộ và xả (flush) định kỳ, kết hợp `sync.Once` để Graceful Shutdown.
- **Redis Sets (`server:all_ids`):** Lưu danh sách IP cần giám sát (Được đồng bộ ngầm từ Redis Streams).
- **Redis Queue (`monitoring:queue`):** Hàng đợi chứa các jobs cho Worker Pool (500 Goroutines) tiêu thụ bằng kỹ thuật `BLPOP` (Blocking Pop) nhằm tối ưu triệt để CPU Idle.
- **Redis Lock (`lock:monitoring_producer`):** Đảm bảo chỉ 1 Replica làm nhiệm vụ đẩy Job vào hàng đợi. Thời gian hết hạn của khóa (Expiration) được tính toán tự động dựa trên Tick Interval.

### 4.4 Nghiệp vụ Báo cáo tự trị (Reporting)
Hoạt động độc lập nhờ cơ chế Data Replication, có khả năng tự gen HTML và bắn email mà không phụ thuộc vào Management Service.

#### 4.4.1 Các luồng xử lý
- **Quy trình Xuất báo cáo tự trị (Reporting Generate)**
  Sử dụng Uptime Calculator truy vấn thẳng Elasticsearch (Aggregation) để lấy % uptime. Tích hợp sẵn `SMTP Notifier` (trước đây là module Notification độc lập ở Backend cũ) để build cấu trúc Multipart MIME và gửi Email trực tiếp, giảm thiểu độ trễ RPC qua lại giữa các service.
- **Cronjob Gửi Báo cáo (Daily Scheduler)**
  Sử dụng Distributed Lock (`SET NX`) trên Redis để đảm bảo khi scale nhiều Replicas, chỉ có duy nhất 1 instance thực thi việc tạo và gửi email báo cáo mỗi ngày, tránh hiện tượng gửi email trùng lặp.
- **Đồng bộ Dữ liệu Server (Event Consumer)**
  Worker lắng nghe sự kiện CRUD từ Redis Streams để tự động cập nhật bản sao dữ liệu (Data Replication) vào CSDL Reporting, duy trì tính tự trị (autonomy).

#### 4.4.2 Thiết kế Database
**A. PostgreSQL (Schema: reporting)**
- **REPORT_REQUESTS:** Ghi nhận lịch sử xuất báo cáo `status` (PENDING, PROCESSING, COMPLETED, FAILED).
- **REPORTING_SERVERS:** Bản sao dữ liệu Server (Data Replication) chứa `server_id`, `server_name`.

#### 4.4.3 Thuật toán tính Uptime (Elasticsearch Aggregation)
Hệ thống sử dụng Query Range theo `timestamp` và đếm tổng số log có `is_success = true` chia cho tổng số log, xử lý trực tiếp trên Elasticsearch Engine giúp tốc độ phản hồi tính bằng mili-giây.

---

## 5. TỔNG QUAN KIẾN TRÚC LƯU TRỮ (POLYGLOT PERSISTENCE)
Hệ thống sử dụng nhiều công nghệ lưu trữ cho những nhu cầu xử lý riêng biệt.

### 5.1. Chiến lược lưu trữ đa mô hình
- **PostgreSQL (OLTP):** Nguồn dữ liệu gốc (Single Source of Truth) cho Identity, Management, và Reporting. Mỗi service sở hữu một schema/DB riêng rẽ.
- **Redis (In-memory):** Tối ưu hóa ranh giới tốc độ (Caching, Queue) và là Message Broker (Redis Streams) trung tâm kết nối các hệ thống.
- **Elasticsearch (Time-Series):** Bóc tách hoàn toàn thao tác ghi Log Ping ra khỏi DB quan hệ, tối ưu năng lực truy vấn Aggregation.

### 5.2. Sự biến đổi của Bounded Context (DDD)
Thực thể "Server" tồn tại dưới nhiều hình thái khác nhau qua các miền:
- Tại **SMS Management**: Là tài nguyên gốc (`ID, Name, IPv4, CurrentStatus`).
- Tại **SMS Monitoring**: Là `MonitoredEndpoint` nằm trong Redis Cache (chỉ cần `IPv4`, `State`).
- Tại **SMS Reporting**: Là `ReportingServer` được nhân bản (Data Replication) qua Event để đảm bảo service hoạt động tự trị.

#### A. Sơ đồ ERD (Entity-Relationship Diagram)

### 5.4. Các luồng quy trình chính (Sequence Diagrams)

#### 1. Luồng Thêm/Sửa Server (CRUD & Outbox Transaction)
Đảm bảo tính nhất quán dữ liệu bằng Transaction Database và Pattern Outbox.

#### 2. Luồng Import/Export hàng loạt (Bulk Operations)
Xử lý phân lô (batching) để tránh Out-of-Memory khi thao tác với file Excel dung lượng lớn.

#### 3. Luồng Giám sát (Monitoring Worker)
Tách biệt quy trình đồng bộ danh sách Server (từ Redis Streams) và quy trình Ping (qua Redis Queue).

#### 4. Luồng Báo cáo tự trị (Reporting & Uptime)
Tự động sinh HTML và gửi Email định kỳ mà không cần tương tác trực tiếp với Database của Management.

---

## 6. BẢO MẬT & HẠ TẦNG TRIỂN KHAI

### 6.1. Bảo mật đa lớp tại API Gateway (Traefik)
- **Rate Limiting Kép:** Traefik giới hạn ở mức mạng (average 100 req/s), Management giới hạn ở mức ứng dụng (Application-level Limiter bằng Redis).
- **Xác thực ủy quyền & Chống CSRF (ForwardAuth):** Mọi request bị Traefik chặn lại và hỏi ý kiến Identity qua `/verify`. Kiểm tra chéo CSRF Token giữa Cookie và Header. Traefik tự động đẩy Custom Headers (`X-User-Role`) xuống các service phía sau.

### 6.2. Hạ tầng triển khai (Docker Swarm)
- **Zero Downtime Deployments (ZDT):** Cấu hình `update_config: order: start-first` kết hợp với HTTP Healthchecks đảm bảo quá trình cập nhật không gây gián đoạn dịch vụ. Traefik tự động điều hướng traffic (Load Balancing).
- **Native OS Environment Variables:** Loại bỏ hoàn toàn sự phụ thuộc vào file `.env` tĩnh (Docker Configs). Cấu hình động được tiêm trực tiếp qua `environment` block của Swarm, tuân thủ nguyên lý 12-Factor App.
- **Mật khẩu an toàn (Docker Secrets):** Các dữ liệu nhạy cảm (`db_url`, `jwt_secret`) được truyền qua Docker Secrets dưới dạng In-Memory Files (tmpfs), tuyệt đối không phơi bày ra môi trường thô.
- **ICMP Privileged Mode:** Monitoring cho phép cấu hình `ICMP_PRIVILEGED=true/false` để tự động chuyển đổi giữa việc dùng Raw Sockets (cần quyền root) hoặc UDP Datagrams (an toàn hơn).

# 

# 

# 

# 

# 

# 

# 

# **TÀI LIỆU THIẾT KẾ KIẾN TRÚC**

## **HỆ THỐNG QUẢN LÝ SERVER (SMS) \- CHƯƠNG TRÌNH ĐÀO TẠO PASSPORT**

			  
Mã nguồn dự án (Github):  
Backend: [dinhdat07/sms-microservices](https://github.com/dinhdat07/sms-microservices)  
Frontend: [dinhdat07/sms-frontend](https://github.com/dinhdat07/sms-frontend) 

# **1\. TỔNG QUAN HỆ THỐNG**

## 1.1 Mục tiêu hệ thống

Hệ thống Server Management System (SMS) là nền tảng quản trị giúp theo dõi trạng thái hoạt động của hàng vạn (10k+) máy chủ đích (Target Servers) thông qua giao thức ICMP Ping.   
Được tái cấu trúc từ mô hình Modular Monolith sang Microservices và Event-Driven Architecture (EDA), hệ thống giải quyết triệt để các giới hạn về khả năng mở rộng, phân tách quyền sở hữu dữ liệu (Domain-Driven Design), và cung cấp cơ chế tính toán Uptime tự động thông qua Elasticsearch.

Link Github:

* [dinhdat07/sms-microservices: Core backend services for the Server Management System (SMS)](https://github.com/dinhdat07/sms-microservices)  
* [dinhdat07/sms-frontend: Frontend Single Page Application for the Server Management System (SMS)](https://github.com/dinhdat07/sms-frontend) 

## 1.2 Sơ đồ System Context

![][image1]

**Chú thích sơ đồ:**

* **System Administrator (Admin)**: Người dùng trực tiếp của hệ thống, thực hiện các thao tác quản trị, import/export server, xem báo cáo thống kê và nhận thông báo cảnh báo qua email.

* **Server Management System (SMS)**: Hệ thống phần mềm trung tâm, đóng vai trò giám sát, thu thập dữ liệu và báo cáo.

* **Target Servers (10k+)**: Hàng nghìn máy chủ đích nằm trong hạ tầng cần được giám sát. Hệ thống sẽ liên tục gửi gói tin ICMP (Ping) đến các server này để kiểm tra trạng thái ONLINE/OFFLINE.

* **SMTP Server**: Hệ thống gửi email bên ngoài (như MailHog trong môi trường dev hoặc SendGrid trên production). SMS giao tiếp với SMTP để gửi báo cáo uptime định kỳ hoặc manual cho Admin.

# 

# 

# 

# 

# 

# 

# 

# 

# 

# 

# **2\. KIẾN TRÚC TỔNG THỂ** 

## 2.1 Container Diagram

Hệ thống được thiết kế theo chuẩn C4, bóc tách thành các Container (Microservices) độc lập kết nối qua API Gateway và Event Bus.  
![][image2]  
![][image3]

**Chú thích sơ đồ Container:**

* **API Gateway (Traefik)**: Reverse proxy định tuyến traffic, cung cấp bảo mật tập trung (Rate Limiting, ForwardAuth).  
* **Web App (Angular)**: Frontend Single Page Application giao tiếp với Backend qua REST/HTTPS.  
* **SMS Identity (Go)**: Dịch vụ quản lý xác thực, cấp phát JWT và quản lý vòng đời Session/Token.  
* **SMS Management (Go):** Dịch vụ quản lý cấu hình máy chủ, CRUD, và phát đi sự kiện qua Outbox Pattern.  
* **SMS Monitoring Worker (Go)**: Background Worker tự động đồng bộ mục tiêu, thực thi Ping và đẩy log lên Elasticsearch.  
* **SMS Reporting (Go)**: Dịch vụ tính toán % Uptime và quản lý các yêu cầu báo cáo.
* **SMS Notification (Go)**: Dịch vụ độc lập xử lý hàng đợi và gửi email/thông báo qua SMTP.
* **SMS Agent Handler (Go)**: Dịch vụ tiếp nhận heartbeat (Agent Push) từ các máy chủ đích có cài đặt Agent, xác thực qua X-Master-Key.
* **PostgreSQL (Database)**: CSDL quan hệ lưu metadata.  
* **Redis (Streams & Cache)**: Event Bus trung tâm (Pub/Sub), Distributed Lock, và Cache trạng thái siêu tốc.  
* **Elasticsearch (Log Storage)**: CSDL Time-series lưu trữ Observation Logs.

##  2.2 Chi tiết các khối công nghệ

| Lớp / Phân hệ | Công nghệ | Phiên bản | Vai trò |
| :---- | :---- | :---- | :---- |
| **Backend Core** | Go | 1.22+ | Xử lý logic, hiệu năng cao, goroutine pool |
| **API Protocol** | gRPC \+ grpc-gateway | v2 | Cung cấp song song gRPC native và REST HTTP |
| **API Gateway** | Traefik | v3.0 | Reverse Proxy, TLS, Rate Limit, ForwardAuth |
| **Frontend** | Angular | 19 | Single Page Application (SPA) |
| **Primary DB** | PostgreSQL | 15 | Cơ sở dữ liệu quan hệ (OLTP) cho các service, GORM làm ORM |
| **Broker & Cache** | Redis | 7.x | Redis Streams (Event Bus), Distributed Lock, Session Cache |
| **Time-Series DB** | Elasticsearch | 8.17 | Lưu observation logs để query Aggregation uptime |
| **Email** | SMTP / MailHog | \- | Dịch vụ gửi email (cho dev/test) |
| **Ping Library** | pro-bing | 0.8 | Hỗ trợ ICMP unprivileged/privileged ping |
| **E2E Testing** | Bruno | \- | Nền tảng kiểm thử tích hợp API (Automation Test) |
| **CI/CD** | GitHub Actions | \- | Tự động hóa Lint, Unit Test, Integration Test |

# 

# 

# 

# 

# 

# 

# 

# 

# 

# 

# 

# **3\. KIẾN TRÚC THÀNH PHẦN**

Hệ thống tuân thủ kiến trúc Microservices. Mỗi service chứa các component chuyên biệt để xử lý luồng nghiệp vụ nội bộ.

##  3.1 SMS Identity

## ![][image4]

Chịu trách nhiệm về Authentication. Chứa các component như ForwardAuth Handler, Auth Server, User Repo, Session Repo, RefreshToken Repo để quản lý phiên bản và Token Blacklist trên Redis.

##  

## 

## 

## 

## 

## 3.2 SMS Management

![][image5]

Nơi duy nhất quản lý vòng đời Server. Chứa Server Service, Server Repo, và đặc biệt là Outbox Relay Worker dùng để đọc các sự kiện từ CSDL cục bộ và đẩy vào Redis Streams (sms.events.server).

## 

## 3.3 SMS Monitoring

![][image6]  
Sử dụng mô hình Producer-Consumer. Cycle Scheduler bốc mục tiêu từ Redis Set để thêm vào Queue, Worker Pool liên tục rút từ Queue để thực hiện Ping, sau đó Observation Logger ghi Bulk Insert lên Elasticsearch.

## 

## 

## 

## 

## 

## 

## 3.4 SMS Reporting

![][image7]  
Dịch vụ hoàn toàn tự trị. Chứa Event Consumer để tự đồng bộ dữ liệu Server, Daily Scheduler để tạo Job, và Uptime Calculator truy vấn ES. Đã được bóc tách khỏi chức năng gửi thông báo.

## 3.5 SMS Notification
Dịch vụ chuyên biệt tiếp nhận các yêu cầu thông báo từ Message Queue/Broker và đảm nhận việc render HTML Template, gửi cảnh báo qua Email/SMTP. Giúp giảm tải đáng kể cho hệ thống Reporting.

## 3.6 SMS Agent Handler
Dịch vụ quản lý các Agent cài đặt tại máy chủ đích. Khác với ICMP, Agent sẽ chủ động thu thập trạng thái và đẩy (Heartbeat/Push) lên API của Agent Handler. Dịch vụ này thực hiện xác thực thông qua X-Master-Key và cập nhật trạng thái vào Redis Streams.
#  

# 

# 

# 

# 

# 

# 

# **4\. KIẾN TRÚC CHI TIẾT CÁC NGHIỆP VỤ**

## 4.1 Nghiệp vụ Quản lý định danh (Identity)

Nghiệp vụ đảm nhiệm việc chứng thực (Authentication) người dùng, cấp phát và quản lý vòng đời của JWT Access Token và Refresh Token với cơ chế Anti-Replay Attack: Nếu phát hiện một Refresh Token đã bị thu hồi (revoked) nhưng vẫn được sử dụng lại, hệ thống lập tức vô hiệu hóa (LogoutAll) mọi phiên đăng nhập của user và ghi cờ đen vào Redis Revocation Store đó để bảo vệ tài khoản.

### 4.1.1 Các luồng xử lý

#### *A. Luồng Đăng nhập (Login)*

![][image8]Xử lý xác thực thông tin, tạo và trả về Access Token, Refresh Token và thiết lập Cookie an toàn 

#### *B.* Quản lý Phiên (Session Management: Logout, LogoutAll, RefreshToken)

1. Làm mới Token (Refresh): Cấp phát lại Access Token khi hết hạn mà không cần đăng nhập lại.  
   ![][image9]  
2. Thu hồi Phiên (Logout/Revocation): Khi người dùng Logout hoặc LogoutAll, token/session sẽ bị tước quyền và đẩy vào Blacklist (Redis Revocation Store).![][image10]

#### *C. Luồng Xác thực Token (Verify Token Middleware)*

![][image11]  
Middleware Traefik luôn check API này để biết token có bị Revoked trong Redis hay không trước khi cho phép request đi tiếp. 

### 4.1.2 Thiết kế Database

#### *A. Sơ đồ ERD*

![][image12]

#### *B. Data Dictionary*

*Bảng USERS*

| Column | Type | Constraints | Description |
| :---- | :---- | :---- | :---- |
| id | INT/UINT | PK | Khóa chính tự tăng (gorm.Model) |
| email | VARCHAR | UK, Indexed | Dùng để đăng nhập |
| password\_hash | VARCHAR |   | Mật khẩu băm (Bcrypt) |
| role\_code | VARCHAR |   | Phân quyền (VD: ADMIN) |
| created\_at / updated\_at | TIMESTAMP |   | Dấu thời gian tạo/cập nhật |

   
*Bảng AUTH\_SESSIONS*

| Column | Type | Constraints | Description |
| :---- | :---- | :---- | :---- |
| id | UUID | PK | Khóa chính Session |
| user\_id | INT/UINT | FK, Indexed | Khóa ngoại trỏ về USERS |
| expires\_at | TIMESTAMP |   | Thời điểm hết hạn |
| last\_used\_at | TIMESTAMP |   | Thời điểm Session được dùng gần nhất |
| revoked\_at | TIMESTAMP | Nullable | Thời điểm bị admin thu hồi. Nếu NULL là còn hiệu lực. |
| ip\_address | VARCHAR |   | IP của thiết bị đăng nhập |
| user\_agent | VARCHAR |   | Thông tin trình duyệt/app |
| created\_at / updated\_at | TIMESTAMP |   | Dấu thời gian hệ thống |

   
*Bảng REFRESH\_TOKENS*

| Column | Type | Constraints | Description |
| :---- | :---- | :---- | :---- |
| id | UUID | PK | Khóa chính Token |
| session\_id | UUID | FK, Indexed | Khóa ngoại trỏ về AUTH\_SESSIONS |
| user\_id | INT/UINT | FK, Indexed | Khóa ngoại denormalized trỏ về USERS |
| token\_hash | TEXT | UK, Indexed | Chuỗi hash SHA-256 của Refresh Token |
| expires\_at | TIMESTAMP |   | Thời điểm hết hạn |
| revoked\_at | TIMESTAMP | Nullable | Thời điểm bị admin thu hồi |
| replaced\_by | UUID | FK, Nullable | Cơ chế xoay vòng: Trỏ về ID của token mới |
| created\_at / updated\_at | TIMESTAMP |   | Dấu thời gian hệ thống |

#### *C. Redis Cache* 

| Key Pattern | Data Type | TTL | Purpose |
| :---- | :---- | :---- | :---- |
| revoked\_session:{id} | STRING | Theo thời hạn Token | Chặn session bị thu hồi, Anti-Replay Attack. |

## 

## 

## 

## 

## 

## 

## 

## 4.2 Nghiệp vụ Quản lý Server (Server Management)

Cung cấp API CRUD và thao tác hàng loạt. Ứng dụng kỹ thuật Event-Driven để đảm bảo dữ liệu được nhân bản an toàn sang các service khác.

### 4.2.1 Các luồng xử lý 

#### *A. Luồng Thêm/Sửa Server (CRUD & Outbox Transaction)*

![][image13]  
Khi thêm Server, dữ liệu được ghi vào bảng SERVERS và OUTBOX\_EVENTS trong cùng một Transaction. Tránh triệt để lỗi Dual-Write. 

#### *B. Luồng xử lý hàng loạt (Bulk Import/Export)*

1. *Nhập Hàng Loạt (Import): Phân lô batch 100 dòng để tối ưu I/O DB và tránh Out-of-Memory khi import hàng chục ngàn server.*![][image14]  
2. *Xuất Hàng Loạt (Export): Stream file CSV về client, tải dần dữ liệu tránh sập RAM.*![][image15]

#### *C. Luồng tiêu thụ sự kiện trạng thái (Status Consumer)*

![][image16]

Worker lắng nghe trạng thái Server thay đổi từ Monitoring (qua Redis Streams) và cập nhật lại vào PostgreSQL một cách không đồng bộ. 

### 4.2.2 Thiết kế Database

#### *A. Data Dictionary*

*Bảng SERVERS* 

| Column | Type | Constraints | Description |
| :---- | :---- | :---- | :---- |
| server\_id | UUID | PK | Khóa chính định danh server |
| server\_name | VARCHAR(255) | UK, Indexed, Not Null | Tên hiển thị của server (duy nhất) |
| ipv4 | VARCHAR(15) | UK, Indexed, Not Null | Bắt buộc đúng định dạng IPv4 |
| current\_status | VARCHAR |   | ONLINE hoặc OFFLINE |
| consecutive\_failures | BIGINT |   | Số lần ping thất bại liên tiếp |
| created\_at / updated\_at | TIMESTAMP |   | Dấu thời gian hệ thống |

Bảng OUTBOX\_EVENTS 

| Column | Type | Constraints | Description |
| :---- | :---- | :---- | :---- |
| id | VARCHAR(36) | PK | Khóa chính (UUID) |
| aggregate\_type | VARCHAR(50) | Indexed, Not Null | Loại Aggregate Root (VD: SERVER) |
| aggregate\_id | VARCHAR(255) | Indexed, Not Null | ID của Aggregate |
| event\_type | VARCHAR(50) | Not Null | Loại sự kiện (VD: Server Created) |
| payload | JSONB | Not Null | Nội dung sự kiện |
| is\_processed | BOOLEAN | Indexed, Not Null, Default false | Trạng thái đã gửi qua Message Broker hay chưa |
| created\_at | TIMESTAMP | Not Null | Thời điểm tạo sự kiện |

#### 

#### *B. Redis Streams (Event Bus)*

| Key Pattern | Data Type | TTL | Purpose |
| :---- | :---- | :---- | :---- |
| sms.events.server | STREAM | Unbounded | Outbox events phân phối các thay đổi CRUD của Server (Created, Updated, Deleted). Publish bởi sms-management. |

## 4.3 Nghiệp vụ Giám sát Server (Monitoring Worker)

Monitoring Worker chạy theo chu kỳ 30 giây và sử dụng mô hình **Goroutine Worker Pool**. Các yêu cầu ping được phân phối qua buffered channel để các worker xử lý tuần tự theo cơ chế hàng đợi, giúp cân bằng giữa khả năng xử lý đồng thời và mức tiêu thụ tài nguyên

###  4.3.1 Luồng xử lý

*Vòng lặp Giám sát đa phương thức (Healthcheck Methods)*

Hệ thống hỗ trợ 2 phương thức kiểm tra trạng thái:
1. **ICMP (Ping)**: Agentless. Được thực hiện tự động bởi Cycle Scheduler và Ping Worker.
2. **AGENT_PUSH (Heartbeat)**: Cần cài đặt Agent tại máy chủ đích. Agent sẽ chủ động đẩy trạng thái định kỳ lên hệ thống thông qua `sms-agent-handler` với xác thực `X-Master-Key`. Monitoring Worker sẽ quét (sweep) các server dùng phương thức này xem có bị quá hạn (Timeout) hay không để đánh giá OFFLINE.

Quy trình Phân bổ tải & Bầu chọn Producer (Producer Election) cho phương thức ICMP:

* Distributed Lock: Scheduler sử dụng lệnh SET NX PX với thời gian hết hạn của khóa (expiration) luôn nhỏ hơn chu kỳ Tick một chút. Điều này đảm bảo khi hệ thống có N Replicas, luôn chỉ có duy nhất 1 Replica giữ vai trò cấp phát công việc (Producer).
* Snowballing Prevention (Chống bùng nổ hàng đợi): Ngay cả khi đã lấy được Lock, Producer vẫn phải kiểm tra độ dài hàng đợi hiện tại (LLen monitoring:queue). Nếu hàng đợi vẫn còn job, Producer sẽ hủy bỏ chu kỳ đẩy mới để ngăn chặn tình trạng tràn bộ nhớ Redis.
* Batch Push: Lấy toàn bộ ID/IP cần ICMP Ping và đẩy vào monitoring:queue thông qua một lệnh RPUSH duy nhất.

*Ping Worker & Ghi Log Observation:* Worker kéo job bằng `BLPOP`. Sử dụng State Machine, nếu thất bại >= Threshold mới chính thức coi là OFFLINE để tránh Flapping do nhiễu mạng. Worker đẩy log vào Logger để Bulk Insert lên Elasticsearch.

![][image17]

### 4.3.2 Thiết kế Data Store & Lock

#### *A. Elasticsearch*

Dữ liệu ping được tạo ra liên tục với tần suất lớn. Thay vì lưu trữ trực tiếp trong cơ sở dữ liệu quan hệ, toàn bộ Observation Logs được ghi vào Elasticsearch thông qua Bulk API nhằm tối ưu việc lưu trữ và truy vấn log.

**Index sms\_observation\_logs**: Ghi bulk log theo chuỗi thời gian (time-series). Gồm server\_id (keyword), is\_success (boolean), timestamp (date). Cơ chế ghi log có buffer nội bộ và xả (flush) định kỳ, kết hợp sync.Once để Graceful Shutdown. 

| Field | JSON Type | ES Mapping Type | Purpose |
| :---- | :---- | :---- | :---- |
| server\_id | String | keyword | Bắt buộc là keyword để có thể chạy Aggregation Group-by khi tính tỷ lệ Uptime. |
| is\_success | Boolean | boolean | Dùng để filter số lần ping thành công / thất bại. |
| timestamp | String | date | Phục vụ truy vấn khoảng thời gian (Range query) để xuất báo cáo theo tháng/quý. |

 

#### *B. Cache & Streams*

| Key Pattern | Data Type | TTL | Purpose |
| :---- | :---- | :---- | :---- |
| sms.events.server\_status | STREAM | Unbounded | Phân phối các sự kiện thay đổi trạng thái (ONLINE/OFFLINE) của Server sau khi Ping. Publish bởi sms-monitoring. |
| server:all\_ids | SET | Unbounded | Lưu danh sách IP/ID cần giám sát (Được đồng bộ ngầm từ Redis Streams). |
| server:info:{id} | HASH | Unbounded | Lưu trữ thông tin chi tiết (ví dụ ipv4) của một Server cụ thể phục vụ cho Ping nhanh. |
| monitoring:queue | LIST | N/A | Hàng đợi chứa các jobs cho Worker Pool (500 Goroutines) tiêu thụ bằng kỹ thuật BLPOP (Blocking Pop) nhằm tối ưu triệt để CPU Idle. |

#### *C. Lock*

| Key Pattern | Data Type | TTL | Purpose |
| :---- | :---- | :---- | :---- |
| lock:monitoring\_worker | STRING (NX) | Tick Interval | Đảm bảo chỉ 1 Replica làm nhiệm vụ đẩy Job vào hàng đợi (Distributed Lock).  |

## 

## 

## 

## 

## 

## 

## 4.4 Nghiệp vụ Báo cáo và Lập lịch 

**Mô tả:** Các tác vụ xử lý báo cáo như tính toán uptime trên Elasticsearch, render HTML và gửi email qua SMTP được thực hiện bởi Background Worker Process. API Handler chỉ tiếp nhận yêu cầu, đưa job vào buffered channel và phản hồi cho client mà không phải chờ quá trình xử lý hoàn tất. Cách tiếp cận này giúp giảm thời gian phản hồi của API và tách biệt các tác vụ tốn thời gian khỏi luồng xử lý chính. 

### 4.4.1 Các luồng xử lý

#### *A. Quy trình Xuất báo cáo tự trị (Reporting Generate)*

Sử dụng Uptime Calculator truy vấn thẳng Elasticsearch (Aggregation) để lấy % uptime. Tích hợp sẵn SMTP Notifier (trước đây là module Notification độc lập ở Backend cũ) để build cấu trúc Multipart MIME và gửi Email trực tiếp, giảm thiểu độ trễ RPC qua lại giữa các service.

![][image18]

#### *B. Cronjob Gửi Báo cáo (Daily Scheduler)*

![][image19]

Sử dụng Distributed Lock (SET NX) trên Redis để đảm bảo khi scale nhiều Replicas, chỉ có duy nhất 1 instance thực thi việc tạo và gửi email báo cáo mỗi ngày, tránh hiện tượng gửi email trùng lặp. 

#### *C. Đồng bộ Dữ liệu Server (Event Consumer)*

Worker lắng nghe sự kiện CRUD từ Redis Streams để tự động cập nhật bản sao dữ liệu (Data Replication) vào CSDL Reporting, duy trì tính tự trị (autonomy). 

![][image20]

### 

### 4.4.2 Thiết kế Database

#### *A. Data Dictionary*

*Bảng REPORTS* 

| Column | Type | Constraints | Description |
| :---- | :---- | :---- | :---- |
| id | UUID | PK | Khóa chính của yêu cầu xuất báo cáo |
| requestor\_email | VARCHAR(255) | Not Null | Email người yêu cầu báo cáo |
| start\_time / end\_time | TIMESTAMP | Not Null | Khoảng thời gian báo cáo |
| status | VARCHAR(50) | Not Null | Trạng thái báo cáo (PENDING, PROCESSING, COMPLETED, FAILED) |

### 

*Bảng REPORTING\_SERVERS*

 

| Column | Type | Constraints | Description |
| :---- | :---- | :---- | :---- |
| server\_id | VARCHAR(255) | PK | Khóa chính liên kết với server\_id gốc |
| name | VARCHAR(255) |  | Tên Server |
| ipv4 | VARCHAR(45) |  | Địa chỉ IPv4 |
| status | VARCHAR(50) | Default UNKNOWN | Trạng thái của Server |

#### *B. Redis (Cache & Lock)*

| Key Pattern | Data Type | TTL | Purpose |
| :---- | :---- | :---- | :---- |
| lock:daily\_report | STRING | Theo chu kỳ chạy | Đảm bảo chỉ duy nhất 1 Replica thực thi tiến trình gửi email báo cáo (Cronjob Daily Scheduler). |
| sms.events.server | STREAM | Unbounded | Consumer lắng nghe sự kiện để đồng bộ dữ liệu vào reporting\_servers (Data Replication). |
| sms.events.server\_status | STREAM | Unbounded | Lắng nghe thay đổi trạng thái (ONLINE/OFFLINE) để cập nhật reporting\_servers. |

#### *C. Elasticsearch (Time-Series Query)*

Reporting truy vấn trực tiếp vào index sms\_observation\_logs thông qua Elasticsearch Aggregation Query để tính toán % Uptime trong khoảng thời gian báo cáo (tránh full-table scan trên PostgreSQL).

###  4.4.3 Thuật toán tính Uptime

Công thức: Uptime được tính trực tiếp từ các bản ghi Ping thu thập được theo công thức:

Tỷ lệ Uptime   
\= (Tổng số lần Ping thành công / Tổng số lần Ping) × 100

Cơ chế truy vấn (tích hợp Elasticsearch): Để đảm bảo tốc độ phản hồi ổn định, thực hiện truy vấn trực tiếp trên index sms\_observation\_logs của Elasticsearch. Thuật toán thực hiện hai phép đếm:

1. **Tổng số lần Ping:** Truy vấn theo khoảng thời gian (Range) trên trường timestamp từ startTime đến endTime để xác định tổng số lần kiểm tra đã được thực hiện.  
2. **Số lần Ping thành công:** Kết hợp truy vấn Range với điều kiện is\_success \= true để xác định số lần máy chủ phản hồi thành công.

Các trường hợp đặc biệt và xử lý lỗi:

* **Tránh chia cho 0:** Nếu Tổng số lần Ping \= 0 (ví dụ hệ thống vừa khởi tạo hoặc khoảng thời gian được chọn chưa có dữ liệu), thuật toán trả về 0,00%.  
* **Đồng nhất múi giờ:** Toàn bộ mốc thời gian được xử lý theo chuẩn UTC nhằm bảo đảm kết quả tính toán khớp với thời điểm ghi log của Monitoring Worker.

# 

# 

# 

# 

# 

# 

# **5\. TỔNG QUAN KIẾN TRÚC LƯU TRỮ**

Hệ thống Quản lý Server (SMS) áp dụng mô hình Polyglot Persistence, tức là sử dụng nhiều công nghệ lưu trữ khác nhau cho những nhu cầu xử lý riêng biệt, thay vì phụ thuộc hoàn toàn vào một cơ sở dữ liệu duy nhất. Mỗi thành phần được lựa chọn dựa trên đặc điểm dữ liệu và yêu cầu vận hành của hệ thống. 

## 5.1. Chiến lược lưu trữ đa mô hình

PostgreSQL (Relational \- OLTP): Là nguồn dữ liệu chính của hệ thống (Single Source of Truth) cho Identity, Management, và Reporting. Mỗi service sở hữu một schema/DB riêng rẽ, lưu trữ thông tin người dùng, phiên đăng nhập và metadata của server, đồng thời đảm bảo tính nhất quán theo chuẩn ACID.  
   
Redis (In-memory \- Key/Value): Đóng vai trò tối ưu hóa ranh giới tốc độ và tranh chấp tài nguyên:

* *Identity:* Chặn Anti-Replay Attack bằng cách lưu trữ tạm thời các Token vừa bị thu hồi.  
* *Server Management:* Cache tốc độ cao toàn bộ trạng thái Server (Status, IPv4, Failures) với thời gian truy xuất O(1).  
* *Monitoring:* Hoạt động như một Distributed Lock (Mutex) để đảm bảo tiến trình chạy nền không bị giẫm lên nhau.

Elasticsearch (Time-Series / Search Engine): Việc tách hoàn toàn hàng chục triệu bản ghi Log Ping (Observation Logs) ra khỏi DB quan hệ giúp hệ thống tối ưu hóa năng lực truy vấn Aggregation khi cần tính toán tỷ lệ Uptime.

## 5.2. Sự biến đổi của Bounded Context (DDD)

Thực thể "Server" tồn tại dưới nhiều hình thái khác nhau qua các miền:

* **Tại SMS Management**: Là tài nguyên gốc (ID, Name, IPv4, CurrentStatus).  
* **Tại SMS Monitoring**: Là MonitoredEndpoint nằm trong Redis Cache (chỉ cần IPv4, State).  
* **Tại SMS Reporting**: Là ReportingServer được nhân bản (Data Replication) qua Event để đảm bảo service hoạt động tự trị.

## 5.3. Luồng dữ liệu và cơ chế đồng bộ 

Để đảm bảo các thành phần lưu trữ hoạt động đồng bộ trên kiến trúc phân tán, hệ thống áp dụng các cơ chế sau:

* **Đồng bộ Eventual Consistency qua Outbox Pattern & Redis Streams**: 
Thay vì Dual-write tiềm ẩn rủi ro lỗi đồng bộ, hệ thống ghi dữ liệu thay đổi vào bảng nghiệp vụ và bảng outbox\_events trên PostgreSQL trong cùng một Transaction (ở Management Service). Một Outbox Worker sau đó sẽ quét và phát (Publish) sự kiện lên Redis Streams. Các stream được bảo vệ bởi tham số **MAXLEN** (với mốc ~1 triệu sự kiện) để ngăn chặn tràn RAM (OOM). 
Monitoring, Reporting và Notification Service đóng vai trò là Consumer. Đặc biệt, hệ thống triển khai các cơ chế đảm bảo không rò rỉ và tự chữa lành:
  * **Idempotent Consumer (Consumer tự toàn vẹn)**: Cơ chế giao vận "At-least-once" của Redis Stream có thể khiến một Worker nhận cùng một sự kiện 2 lần nếu có lỗi mạng trước khi gửi lệnh xác nhận (ACK). Để chống lại lỗi nhân bản dữ liệu (Data Duplication), mọi thao tác cập nhật cấu hình/trạng thái server từ Event Stream đều được cài đặt dưới dạng `UPSERT` (cụ thể là `ON CONFLICT DO UPDATE` trong PostgreSQL). Dù hệ thống có đẩy sự kiện 1 lần hay 100 lần, trạng thái cuối cùng trong CSDL vẫn duy nhất và chính xác tuyệt đối.
  * **XAUTOCLAIM cho Worker hỏng (Stalled Worker Recovery)**: Các worker liên tục rà soát các message bị kẹt (PEL) bằng lệnh `XAUTOCLAIM` để tiếp quản công việc từ các worker đã chết.
  * **Dead Letter Queue (DLQ)**: Nếu message xử lý lỗi quá nhiều lần (sau 3 lần Retry bất thành do lỗi logic hoặc format dị biệt), thay vì tiếp tục cản đường dòng chảy sự kiện, thông điệp sẽ bị gỡ khỏi luồng chính và đẩy sang stream phụ `*stream_name*:dlq` (Dead Letter Queue) để kỹ sư phân tích thủ công sau.

* **Giảm khuếch đại ghi (Write Amplification Reduction) chống Flapping**: 
Tiến trình Monitoring hoạt động liên tục. Thay vì liên tục đẩy trạng thái về Management, Monitoring chỉ phát sinh sự kiện ServerStatusChanged qua Redis Streams khi trạng thái của server thực sự thay đổi. Cách tiếp cận này giúp Management và Reporting không bị quá tải bởi các cập nhật trạng thái không cần thiết.
* **Ghi log bất đồng bộ lên Elasticsearch (Asynchronous Logging)**: Kết quả của mỗi nhịp Ping được Worker đưa vào bộ đệm nội bộ (Buffered Logger). Dữ liệu này sẽ được gom thành từng lô (Batch) và thực hiện Bulk Insert lên Elasticsearch theo chu kỳ thời gian hoặc khi đầy bộ đệm. Cơ chế này giúp tiến trình Ping không bị ảnh hưởng bởi độ trễ I/O của database, tối đa hóa throughput của hệ thống.

## 5.4. Phân tách trách nhiệm Đọc/Ghi (CQRS Pattern)

Hệ thống được thiết kế hoàn toàn theo mẫu kiến trúc **CQRS (Command Query Responsibility Segregation)**:
* **Command (Ghi/Sửa/Xóa)**: Thuộc về dịch vụ `sms-management` thao tác trực tiếp lên Primary Database (PostgreSQL). Luồng Command yêu cầu tốc độ cao và tính nhất quán (Consistency).
* **Query (Đọc/Phân tích)**: Thuộc về dịch vụ `sms-reporting` thao tác trên ElasticSearch và bản sao cấu hình (Sync Database) riêng của nó. Các truy vấn tính toán tỷ lệ Uptime theo dải thời gian của hàng vạn server là cực kỳ nặng nề về mặt tính toán.

Sự phân tách kiến trúc thành hai nửa Command và Query độc lập tuyệt đối này đảm bảo rằng: Ngay cả khi hàng trăm quản trị viên cùng lúc trích xuất các báo cáo Uptime cực nặng (Full Scan Query), các API phục vụ thao tác thiết lập, chỉnh sửa máy chủ (Command) vẫn hoạt động mượt mà với độ trễ thấp, không bao giờ bị nghẽn (Lock) do tắc nghẽn Database.

# 

# **6\. BẢO MẬT & GIỚI HẠN TÀI NGUYÊN**

## 6.1. Bảo mật đa lớp tại API Gateway (Traefik)

* **Rate Limiting Kép**: Traefik giới hạn ở mức mạng (average 100 req/s), Management giới hạn ở mức ứng dụng (Application-level Limiter bằng Redis).  
* **Xác thực ủy quyền & Chống CSRF (ForwardAuth)**: Mọi request bị Traefik chặn lại và hỏi ý kiến Identity qua /verify. Kiểm tra chéo CSRF Token giữa Cookie và Header. Traefik tự động đẩy Custom Headers (X-User-Role) xuống các service phía sau.

## 6.2. Hạ tầng triển khai (Docker Swarm)![][image21]

* **Zero Downtime Deployments (ZDT) & Graceful Shutdown**: Cấu hình update\_config: order: start-first kết hợp với HTTP Healthchecks. Cấp độ Service đã được chuẩn hóa vòng đời Graceful Shutdown để xả (flush) toàn bộ log/queue trước khi container dừng, đảm bảo cập nhật không gây gián đoạn hay mất mát dữ liệu.
* **Persistent Volumes**: Các Stateful workloads (PostgreSQL, Redis, Elasticsearch) được gắn persistent volume để tránh rủi ro mất dữ liệu khi container tái khởi động hoặc di chuyển giữa các node.
* **Native OS Environment Variables**: Cấu hình động được tiêm trực tiếp qua environment block của Swarm.
* **Mật khẩu an toàn (Docker Secrets)**: Các dữ liệu nhạy cảm được truyền qua Docker Secrets dưới dạng In-Memory Files (tmpfs).
* **ICMP Privileged Mode**: Hỗ trợ ICMP_PRIVILEGED=true/false để chọn Raw Sockets hoặc UDP Datagrams.

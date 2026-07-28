# 

# 

# 

# 

# 

# 

# 

# **TÀI LIỆU MÔ TẢ HỆ THỐNG**  **VÀ HƯỚNG DẪN SỬ DỤNG**

## **HỆ THỐNG QUẢN LÝ SERVER (SMS) \- CHƯƠNG TRÌNH ĐÀO TẠO PASSPORT**

Mã nguồn dự án (Github):  
Backend: [dinhdat07/sms-backend](https://github.com/dinhdat07/server-management-service)  
Frontend: [dinhdat07/sms-frontend](https://github.com/dinhdat07/sms-frontend) 

# 

# **1\. TỔNG QUAN VÀ ĐẶC TÍNH KỸ THUẬT**

## 1.1 Giới thiệu chung

Hệ thống Quản lý Server (SMS) là giải pháp giám sát tập trung, cho phép Quản trị viên quản lý thông tin và theo dõi trạng thái sống/chết (Uptime) của hàng chục nghìn máy chủ theo thời gian thực thông qua giao thức ICMP (Ping).

## 1.2 Các yêu cầu chức năng cốt lõi đã đáp ứng 

Hệ thống giải quyết trọn vẹn các bài toán nghiệp vụ đặt ra, bao gồm:

* **Giám sát trạng thái:** Định kỳ quét (ping) ngầm và cập nhật trạng thái On/Off tập trung cho hàng chục nghìn servers.  
* **Quản lý Dữ liệu:** Tạo, sửa, xóa, tìm kiếm, phân trang và sắp xếp server. Đảm bảo ràng buộc định danh độc nhất và định dạng IPv4 hợp lệ.  
* **Import / Export:** Xử lý nhập/xuất dữ liệu hàng loạt qua file Excel tốc độ cao, cơ chế tự động bỏ qua bản ghi trùng lặp.  
* **Báo cáo tự động & Chủ động:** Tiến trình Cronjob tự động gửi báo cáo Uptime hàng ngày qua Email, kết hợp cùng API cho phép quản trị viên chủ động trích xuất báo cáo theo khoảng thời gian tùy chọn.

## 1.3 Các yêu cầu phi chức năng đã đáp ứng 

Hệ thống được thiết kế và xây dựng tuân thủ nghiêm ngặt các quy chuẩn kỹ thuật. Dưới đây là các minh chứng cụ thể (Evidence) cho từng yêu cầu: 

* **Kiến trúc Dữ liệu (Polyglot Persistence):**  
  * Sử dụng **PostgreSQL** làm cơ sở dữ liệu chính (Primary DB) để lưu trữ định danh.  
  * Sử dụng **Redis** làm bộ đệm tốc độ cao (Cache) và khóa phân tán (Distributed Lock).   
  * Sử dụng **Elasticsearch** chuyên biệt để lưu trữ log ping và tính toán tỷ lệ Uptime siêu tốc. 

  **![][image1]**  
           *Hình 1:*  Danh sách các container (Services) của hệ thống đang chạy trên môi trường Docker, bao gồm Worker giám sát, PostgreSQL, Redis, Elasticsearch và Mailhog. 

* **Bảo mật (Security):**  
  * Toàn bộ API được bảo vệ bằng xác thực **JWT (JSON Web Token)** với chữ ký bảo mật.   
  * Phân quyền chặt chẽ (RBAC) theo Role/Scope riêng cho từng endpoint.  
  * Ngăn chặn triệt để SQL Injection thông qua thư viện ORM (GORM).

  **![][image2]**  
    	*Hình 2: Hệ thống trả về mã lỗi 401 Unauthorized kèm thông điệp "invalid token" khi truy cập không có mã xác thực hợp lệ.* 

  **![][image3]**  
    	*Hình 3: Giao diện trả về mã lỗi 403 Forbidden khi người dùng sử dụng token hợp lệ nhưng không đủ quyền (Scope/Role) để thao tác trên endpoint* 

* **Đặc tả API (API Documentation):**  
  * Hệ thống tự động gen tài liệu **OpenAPI (Swagger)**, định nghĩa rõ ràng Request, Response và Error Code. 

  ![][image4]

  *Hình 4: Tài liệu đặc tả API tự động (OpenAPI/Swagger) hiển thị danh sách các endpoint thuộc dịch vụ ServerManagementService* 

* **Chất lượng mã nguồn:**  
  * Hệ thống đạt mức **Code coverage cao (\>= 90%)** qua các bài Unit Test độc lập.	![][image5]  
    Hình 5: Kết quả chạy script kiểm tra Code Coverage của các module cốt lõi đạt mức 91.96%, chỉ xét trên các mã nguồn logic chính, loại các code mock và gen   
  * Ghi log (Logging) ra file đầy đủ kèm cơ chế xoay vòng log (**Logrotate**) để chống đầy ổ cứng.   
    *![][image6]*  
    *Hình 6: Cấu trúc các file log  tệp tin log cũ được nén tự động theo cơ chế logrotate*

* **Tối ưu hóa bộ nhớ (Memory Efficiency):**
  * Xử lý Import/Export cấu hình hàng chục nghìn máy chủ bằng kỹ thuật luồng (Stream Processing) với `excelize`, đảm bảo không chiếm dụng RAM và chặn đứng nguy cơ lỗi sập ứng dụng (Out-of-Memory).

# **2\. HƯỚNG DẪN SỬ DỤNG CHI TIẾT** 

### **1\. Đăng nhập và Bảo mật (Authentication)**

Tất cả người dùng phải được cấp tài khoản để truy cập hệ thống.

* Nhập **Email** và **Mật khẩu** tại màn hình đăng nhập.  
* Nếu thông tin hợp lệ, hệ thống sẽ cấp JWT Token và chuyển hướng vào trang Quản trị.

![][image7]  
*Hình 7: Giao diện màn hình Đăng nhập (Sign In) tập trung của hệ thống Server Management.* 

### **2\. Quản lý Danh sách Server (View & CRUD Server)**

Phân hệ này cho phép Admin thao tác trực tiếp với dữ liệu Server. Cấu trúc dữ liệu hiển thị tối thiểu bao gồm: server\_id (ẩn/duy nhất), server\_name, ipv4, status, created\_time, last\_updated.

**2.1. Xem danh sách (View Server)**

* Hệ thống hỗ trợ hiển thị danh sách dưới dạng bảng có **Phân trang (Pagination)**.  
* Hỗ trợ **Bộ lọc (Filter)** đa dạng: Tìm kiếm thông minh đồng thời theo Tên hoặc IPv4, lọc theo Trạng thái (Online/Offline), và lọc theo **Khoảng thời gian tạo (Created From \- To)**.  
* Hỗ trợ **Sắp xếp (Sort)** dữ liệu linh hoạt.

**![][image8]**

*Hình 8: Giao diện trang Quản trị danh sách Server dưới dạng bảng, hỗ trợ phân trang, bộ lọc tìm kiếm và hiển thị trạng thái On/Off* 

**2.2. Thêm mới, Sửa, Xóa (CRUD Server)**

* **Thêm mới (Create):** Nhấn nút "Thêm Server". Yêu cầu server\_name không được trùng lặp và ipv4 phải đúng định dạng chuẩn. ID sẽ được hệ thống tự động sinh (UUID). Admin cũng có thể chọn **Phương thức kiểm tra (Healthcheck Method)** là `ICMP` (mặc định) hoặc `AGENT_PUSH` (yêu cầu cấu hình Agent tại máy đích).
* **Cập nhật (Update):** Chọn biểu tượng Sửa trên từng dòng dữ liệu.  
* **Xóa (Delete):** Chọn biểu tượng Xóa. Hệ thống có xác nhận trước khi xóa vĩnh viễn.
**![][image9]**  
*Hình 9: Form nhập liệu khi nhấn nút "Thêm Server" (Add New Server), yêu cầu điền Tên Server và địa chỉ IPv4 hợp lệ* 

**![][image10]**  
*Hình 10: Form cập nhật thông tin Server (Edit Server) cho phép Admin chỉnh sửa trực tiếp thông tin của một bản ghi.* 

**![][image11]**  
*Hình 11: Hộp thoại cảnh báo xác nhận trước khi xóa vĩnh viễn Server khỏi hệ thống để ngăn chặn việc xóa nhầm dữ liệu* 

### **3\. Import / Export Dữ liệu Hàng loạt**

Tính năng giúp tiết kiệm thời gian khi làm việc với hàng nghìn Server.

**3.1. Import Excel**

* Tải file Excel mẫu do hệ thống cung cấp.  
* Điền danh sách Server. Khi upload, hệ thống sẽ xử lý ngầm: tự động **bỏ qua các bản ghi trùng lặp** và báo cáo số dòng thành công/lỗi.


**![][image12]**

*Hình 12: Hộp thoại Import Servers hỗ trợ kéo/thả hoặc chọn file Excel từ máy tính để nhập dữ liệu hàng loạt.*  

**![][image13]**

*Hình 13: Thông báo kết quả Import Excel thành công, hiển thị chi tiết số lượng bản ghi hợp lệ được thêm vào và số lượng dòng bị lỗi/trùng lặp tự động bỏ qua.*   

**3.2. Export Excel**

* Nhấn nút "Export Excel" trên giao diện danh sách.  
* Hệ thống sẽ trả về file .xlsx chứa toàn bộ dữ liệu khớp với bộ lọc hiện hành (bao gồm lọc theo Tên/IP, Trạng thái và Khoảng thời gian tạo).

	![][image14]

![][image15]

*Hình 14: Danh sách sau khi áp dụng bộ lọc trạng thái "Online" và File Excel (.xlsx) trích xuất dữ liệu (Export) tương ứng chứa đầy đủ thông tin server* 

### **4\. Giám sát Trạng thái (Real-time Monitoring)**

Đây là tính năng cốt lõi của hệ thống, hoạt động dựa trên 2 phương thức:
* **ICMP (Quét tự động):** Tiến trình ngầm sẽ định kỳ quét (ping) toàn bộ máy chủ cấu hình ICMP cứ mỗi 30 giây.
* **AGENT_PUSH (Heartbeat):** Các máy chủ đích sẽ chủ động thu thập trạng thái và đẩy (Push) về hệ thống qua Agent chuyên dụng. Đảm bảo cấu hình đúng `X-Master-Key` trên Agent để gửi dữ liệu thành công.
* **Cập nhật tập trung:** Bất kỳ sự thay đổi trạng thái nào (Từ On sang Off và ngược lại, hoặc từ Timeout) đều được cập nhật tự động lên giao diện danh sách Server theo thời gian thực. Bất kể là phương thức kiểm tra nào.

### **5\. Thống kê và Báo cáo (Reporting)**

**5.1. Báo cáo Tự động (Cronjob)**

* Hệ thống có một tiến trình ngầm (Cronjob) chạy định kỳ đúng **1 lần/ngày vào lúc 00:00**.  
* Tiến trình này tự động tổng hợp số lượng Server On/Off và tính toán tỷ lệ Uptime trung bình của ngày hôm trước, sau đó điều phối qua **Dịch vụ Gửi thông báo (Notification Service)** độc lập để đảm bảo gửi Email tới Quản trị viên một cách an toàn và tin cậy nhất.
**![][image16]**  
*Hình 15: Mẫu Email báo cáo tình trạng Server (Server Status Report) do tiến trình ngầm (Cronjob) tự động tổng hợp và gửi định kỳ hàng ngày cho Quản trị viên* 

**5.2. Báo cáo Chủ động (Manual Report)**

* Admin có thể chủ động yêu cầu hệ thống tính toán Uptime cho một giai đoạn bất kỳ thông qua giao diện Báo cáo.  
* **Cách thực hiện:** Chọn Start date, End date, nhập Email nhận và bấm Gửi yêu cầu.  
* Giao diện sẽ hiển thị danh sách các yêu cầu báo cáo cùng trạng thái (Pending, Processing, Completed). Khi hoàn thành, báo cáo cũng sẽ được gửi về Email.

![][image17] 

*Hình 16: Giao diện chức năng Báo cáo chủ động (Uptime Report) cho phép Admin tùy chọn khoảng thời gian (Start/End Date) và nhập Email đích để trích xuất dữ liệu*

**5.3  Phương pháp tính Uptime**

* Cứ mỗi 30 giây (hoặc 60 giây tùy theo cấu hình), Monitoring Worker sẽ cố gắng kết nối ("Ping") tới tất cả các máy chủ đã được đăng ký.  
* Mỗi lần kiểm tra đều được ghi nhận an toàn vào cơ sở dữ liệu chuỗi thời gian (**Elasticsearch**) dưới dạng:  
  * Success: nhận được phản hồi **PONG** từ máy chủ.  
  * Failure: không nhận được phản hồi trong thời gian chờ (**Timeout**).  
* Khi người dùng yêu cầu tạo báo cáo cho một khoảng thời gian cụ thể (ví dụ: 7 ngày gần nhất), hệ thống sẽ thống kê:  
  * Tổng số lần Ping đã được thực hiện trong khoảng thời gian đó.  
  * Số lần Ping thành công.  
* **Công thức tính:**  
  Uptime \= (Tổng số lần Ping thành công / Tổng số lần Ping) × 100

**Lưu ý trường hợp Uptime \= 0,00%:** Nếu báo cáo hiển thị 0,00%, có thể do một trong các nguyên nhân sau:

1. Các máy chủ không hoạt động trong toàn bộ khoảng thời gian được chọn.  
2. Báo cáo được tạo ngay sau khi thêm máy chủ mới, khi hệ thống chưa thu thập đủ dữ liệu Ping.  
3. Backend API không kết nối được tới Elasticsearch khi khởi động. Trong trường hợp này, chỉ cần khởi động lại container API.


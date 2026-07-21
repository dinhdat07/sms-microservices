workspace "SMS Microservices Architecture" "Server Management System" {

    model {
        admin = person "System Administrator" "Performs administrative tasks, manages servers, views reports, and receives alerts."
        smtp = softwareSystem "SMTP Server" "External email system (e.g., MailHog, SendGrid)" "External"
        targetServers = softwareSystem "Target Servers (10k+)" "External infrastructure servers monitored via ICMP" "External"

        sms = softwareSystem "Server Management System (SMS)" "Monitors target servers, collects observation data, and generates uptime reports." {

            traefik = container "API Gateway" "Traefik Proxy: Routes traffic, terminates TLS, enforces Rate Limiting, and triggers ForwardAuth." "Traefik" "Gateway"
            spa = container "Web Application" "Single Page Application (Frontend)." "Angular 19" "Web Browser"
            swaggerUI = container "Swagger UI" "Centralized API Documentation Hub." "Swagger" "Web Browser"
            
            identity = container "Identity Service" "Handles authentication, JWT generation, and token verification. Provides ForwardAuth for Traefik." "Go" "Microservice" {
                forwardAuthHandler = component "ForwardAuth Handler" "Provides /verify endpoint for Traefik to validate tokens and inject user headers." "Go HTTP Handler"
                authServer = component "Auth gRPC/REST Server" "Provides login, logout, and token refresh endpoints." "Go gRPC/HTTP Server"
                authenticator = component "Authenticator" "Validates JWT tokens and handles CSRF." "Go Component"
                identityService = component "Identity Service" "Core business logic for user authentication." "Go Service"
                userRepo = component "User Repository" "Provides access to user records." "Go Repository"
                sessionRepo = component "Session Repository" "Provides access to auth sessions." "Go Repository"
                refreshRepo = component "Refresh Token Repository" "Provides access to refresh tokens." "Go Repository"
                revocationStore = component "Revocation Store (Redis)" "Maintains a blacklist of revoked Session IDs to enable global revocation." "Go/Redis Component"
            }
            
            management = container "Management Service" "Handles CRUD operations and Bulk Import/Export for Server inventories." "Go" "Microservice" {
                serverServer = component "Server gRPC/REST Server" "Provides CRUD endpoints for target servers." "Go gRPC/HTTP Server"
                serverService = component "Server Service" "Core business logic for server management." "Go Service"
                serverRepo = component "Server Repository (PostgreSQL)" "Reads/writes server metadata." "Go Repository"
                outboxRepo = component "Outbox Repository (PostgreSQL)" "Stores domain events for reliable publishing." "Go Repository"
                outboxRelay = component "Outbox Relay Worker" "Polls outbox and publishes events to Redis Stream." "Go Worker"
                statusConsumer = component "Status Consumer Worker" "Consumes ServerStatusChanged events to update DB." "Go Worker"
                rateLimiter = component "Rate Limiter" "Application-level rate limiter to protect heavy operations." "Go/Redis Component"
            }
            
            reporting = container "Reporting Service" "Handles async report generation, HTML rendering, and SMTP communication." "Go" "Microservice" {
                reportHandler = component "Report Handler" "Accepts report requests via API." "Go gRPC/HTTP Server"
                dailyScheduler = component "Daily Scheduler" "Cronjob triggering daily report generation." "Go Cron"
                reportWorker = component "Report Worker" "Consumes in-memory channel to render HTML templates and generate reports." "Go Worker"
                notifier = component "SMTP Notifier" "Builds multipart MIME email and sends it via SMTP." "Go Component"
                eventConsumer = component "Event Consumer Worker" "Listens to server and status events to sync local DB." "Go Worker"
                reportService = component "Report Service" "Core business logic for report generation." "Go Service"
                esUptimeCalc = component "ES Uptime Calculator" "Calculates uptime by querying observation logs." "Go Component"
                reportRepo = component "Reporting Repository (PostgreSQL)" "Manages report request status and synced server data." "Go Repository"
            }
            
            monitoring = container "Monitoring Worker" "Background pool (500 concurrency) continuously executing ICMP pings against target servers." "Go" "Worker" {
                monitorWorker = component "Ping Worker" "Pulls targets from queue and pings servers." "Go Worker"
                monitoringService = component "Monitoring Service" "Evaluates state machine and publishes events." "Go Service"
                observationLogger = component "Observation Logger" "Writes logs to Elasticsearch." "Go Component"
                streamConsumer = component "Stream Consumer Worker" "Consumes ServerCreated/Updated/Deleted events to update Cache." "Go Worker"
                cacheRepo = component "Monitoring Cache (Redis)" "Local high-speed access to server targets." "Redis Set"
                scheduler = component "Cycle Scheduler" "Elects leader and pushes ping targets into a shared queue." "Go Component"
                queueRepo = component "Ping Queue (Redis)" "A shared queue storing targets to be pinged in the current cycle." "Redis List"
            }
            
            postgres = container "Primary Database" "Stores Users, Sessions, and Server Metadata." "PostgreSQL 15" "Database"
            redis = container "Event Broker & Cache" "Redis Streams for pub/sub events, token blacklist, and target cache. (Persistent AOF+RDB)" "Redis 7" "Message Broker"
            es = container "Time-Series DB" "Stores millions of ICMP Observation Logs for fast uptime aggregation." "Elasticsearch 8" "Database"
        }

        # Context level relationships
        admin -> sms "Manages servers, requests reports, and views status via"
        sms -> targetServers "Pings continuously using ICMP protocol"
        sms -> smtp "Sends email reports and alerts via"
        smtp -> admin "Delivers emails to"

        # Container level relationships
        admin -> traefik "Makes API requests to" "HTTPS/REST"
        admin -> spa "Loads frontend application from" "HTTPS"
        admin -> swaggerUI "Views API documentation on" "HTTPS"
        spa -> traefik "Consumes APIs via" "HTTPS/REST"
        swaggerUI -> traefik "Fetches OpenAPI specs (/openapi/*) via" "HTTP/REST"

        traefik -> identity "Routes Auth API requests" "HTTP/gRPC"
        traefik -> identity "Verifies tokens via ForwardAuth (/verify)" "HTTP"
        traefik -> management "Routes Management API requests (injecting X-User-* headers)" "HTTP/gRPC"
        traefik -> reporting "Routes Reporting API requests (injecting X-User-* headers)" "HTTP/gRPC"

        identity -> postgres "Reads/Writes User & Session data" "TCP/5432"
        identity -> redis "Checks revoked tokens (Anti-Replay)" "TCP/6379"

        management -> postgres "Reads/Writes Server & Outbox data" "TCP/5432"
        management -> redis "Publishes Server CRUD events / Consumes Status events" "Redis Stream"

        reporting -> postgres "Reads/Writes Report Requests & Synced Servers" "TCP/5432"
        reporting -> redis "Consumes Server CRUD & Status events" "Redis Stream"
        reporting -> es "Aggregates uptime data from" "HTTP/9200"
        reporting -> smtp "Sends emails via" "SMTP"

        monitoring -> redis "Reads targets, publishes Status events, consumes CRUD events" "Redis Stream / Cache"
        monitoring -> es "Bulk inserts observation logs to" "HTTP/9200"
        monitoring -> targetServers "Pings" "ICMP"
        
        # Component level relationships (Identity)
        traefik -> forwardAuthHandler "Calls /verify" "HTTP"
        traefik -> authServer "Routes /api/v1/auth/*" "HTTP/gRPC"
        forwardAuthHandler -> authenticator "Authenticates token"
        authServer -> identityService "Delegates to"
        identityService -> userRepo "Uses"
        identityService -> sessionRepo "Uses"
        identityService -> refreshRepo "Uses"
        identityService -> revocationStore "Blacklists revoked sessions in"
        authenticator -> revocationStore "Checks token blacklist in"
        userRepo -> postgres "Queries" "TCP/5432"
        sessionRepo -> postgres "Queries" "TCP/5432"
        refreshRepo -> postgres "Queries" "TCP/5432"
        revocationStore -> redis "Commands" "TCP/6379"
        
        # Component level relationships (Management)
        traefik -> serverServer "Routes /api/v1/servers/*" "HTTP/gRPC"
        serverServer -> rateLimiter "Checks quota"
        rateLimiter -> redis "Increments counters" "TCP/6379"
        serverServer -> serverService "Delegates to"
        serverService -> serverRepo "Writes Server Metadata"
        serverService -> outboxRepo "Writes Outbox Events (Transactionally)"
        outboxRelay -> outboxRepo "Polls unpublished events"
        outboxRelay -> redis "Publishes to stream sms.events.server" "XADD"
        redis -> statusConsumer "Delivers stream sms.events.server_status" "XREADGROUP"
        statusConsumer -> serverRepo "Updates Server status"
        serverRepo -> postgres "Queries" "TCP/5432"
        outboxRepo -> postgres "Queries" "TCP/5432"

        # Component level relationships (Monitoring)
        redis -> streamConsumer "Delivers stream sms.events.server" "XREADGROUP"
        streamConsumer -> cacheRepo "Updates targets"
        scheduler -> redis "Acquires dynamic lock (SET NX PX)" "TCP/6379"
        scheduler -> cacheRepo "Reads all targets (SMEMBERS)"
        scheduler -> queueRepo "Pushes targets (LPUSH)"
        monitorWorker -> queueRepo "Pulls targets (BLPOP)"
        monitorWorker -> targetServers "Pings" "ICMP"
        monitorWorker -> monitoringService "Evaluates ping result"
        monitoringService -> observationLogger "Logs observation"
        observationLogger -> es "Bulk inserts logs" "HTTP/9200"
        monitoringService -> cacheRepo "Gets and Sets Server State"
        monitoringService -> redis "Publishes to stream sms.events.server_status" "XADD"
        cacheRepo -> redis "Commands" "TCP/6379"
        queueRepo -> redis "Commands" "TCP/6379"

        # Component level relationships (Reporting)
        traefik -> reportHandler "Routes /api/v1/servers/report/*" "HTTP/gRPC"
        dailyScheduler -> redis "Acquires lock (SET NX) to prevent duplicate emails" "TCP/6379"
        dailyScheduler -> reportService "Triggers Daily Reporting"
        reportHandler -> reportService "Delegates to"
        reportService -> reportRepo "Creates report request"
        reportService -> reportWorker "Enqueues job"
        redis -> eventConsumer "Delivers streams sms.events.server & sms.events.server_status" "XREADGROUP"
        eventConsumer -> reportRepo "Syncs reporting_servers table"
        reportWorker -> reportRepo "Updates request status & reads synced servers"
        reportWorker -> esUptimeCalc "Delegates uptime calculation"
        reportWorker -> notifier "Delegates email sending"
        esUptimeCalc -> es "Queries Aggregation (sms_observation_logs)" "HTTP/9200"
        notifier -> smtp "Sends compiled email"
        reportRepo -> postgres "Queries" "TCP/5432"
        
        # Deployment Environment (Docker Swarm)
        deploymentEnvironment "Production" {
            deploymentNode "Docker Swarm Cluster" "Manager & Worker Nodes" "Docker Swarm" {
                
                deploymentNode "Manager Node" "Traefik must run on manager to listen to Docker socket" "Docker Engine" {
                    gatewayInstance = containerInstance traefik
                }
                
                deploymentNode "Worker Nodes" "Application workloads" "Docker Engine" {
                    identityInstance = containerInstance identity
                    managementInstance = containerInstance management
                    reportingInstance = containerInstance reporting
                    monitoringInstance = containerInstance monitoring
                    spaInstance = containerInstance spa
                    swaggerInstance = containerInstance swaggerUI
                }
                
                deploymentNode "Stateful Nodes" "Databases and Message Brokers" "Docker Engine" {
                    pgInstance = containerInstance postgres
                    redisInstance = containerInstance redis
                    esInstance = containerInstance es
                }
            }
        }
    }

    views {
        systemContext sms "SystemContext" {
            include *
            autoLayout
        }

        container sms "Containers" {
            include *
            autoLayout
        }
        
        component identity "IdentityComponents" {
            include *
            autoLayout
        }
        
        component management "ManagementComponents" {
            include *
            autoLayout
        }

        component monitoring "MonitoringComponents" {
            include *
            autoLayout
        }

        component reporting "ReportingComponents" {
            include *
            autoLayout
        }

        deployment sms "Production" "SwarmDeployment" {
            include *
            autoLayout
        }

        dynamic identity "Identity_Login" "Detailed sequence for user login" {
            spa -> traefik "1. POST /api/v1/auth/login"
            traefik -> authServer "2. Routes request"
            authServer -> identityService "3. Login(email, password)"
            identityService -> userRepo "4. FindByEmail(email)"
            userRepo -> postgres "5. SELECT FROM users"
            identityService -> sessionRepo "6. Create(session)"
            sessionRepo -> postgres "7. INSERT INTO auth_sessions"
            identityService -> refreshRepo "8. Create(refreshToken)"
            refreshRepo -> postgres "9. INSERT INTO refresh_tokens"
            identityService -> authServer "10. Return Tokens"
            authServer -> traefik "11. Return 200 OK"
            traefik -> spa "12. Return JWT + Set-Cookie"
            autoLayout
        }
        
        dynamic identity "Identity_Verify" "Detailed sequence for ForwardAuth validation" {
            spa -> traefik "1. Request Protected API"
            traefik -> forwardAuthHandler "2. GET /verify (Passes Headers/Cookies)"
            forwardAuthHandler -> authenticator "3. ValidateToken(jwt)"
            authenticator -> revocationStore "4. Check if session is revoked"
            revocationStore -> redis "5. EXISTS revoked_session:{id}"
            authenticator -> forwardAuthHandler "6. Return Principal"
            forwardAuthHandler -> traefik "7. Return 200 OK + Inject X-User Headers"
            traefik -> management "8. Forward request to downstream service"
            autoLayout
        }

        dynamic management "Management_Create" "Detailed sequence for creating a server with Outbox Pattern" {
            spa -> traefik "1. POST /api/v1/servers"
            traefik -> serverServer "2. Routes request"
            serverServer -> rateLimiter "3. Check quota"
            rateLimiter -> redis "4. INCR"
            serverServer -> serverService "5. CreateServer(input)"
            serverService -> serverRepo "6. BEGIN Transaction"
            serverService -> serverRepo "7. INSERT INTO servers"
            serverRepo -> postgres "8. Execute SQL"
            serverService -> outboxRepo "9. INSERT INTO outbox_events (ServerCreated)"
            outboxRepo -> postgres "10. Execute SQL"
            serverService -> serverRepo "11. COMMIT Transaction"
            serverService -> serverServer "12. Return Success"
            serverServer -> traefik "13. Return 201 Created"
            traefik -> spa "14. Return Response"
            
            outboxRelay -> outboxRepo "15. [Async] Fetch unpublished events"
            outboxRepo -> postgres "16. SELECT FROM outbox_events"
            outboxRelay -> redis "17. XADD sms.events.server"
            outboxRelay -> outboxRepo "18. Mark as published"
            outboxRepo -> postgres "19. UPDATE outbox_events"
            autoLayout
        }

        dynamic monitoring "Monitoring_Cycle" "Detailed sequence for distributed ICMP ping cycle" {
            scheduler -> redis "1. SETNX lock:monitoring_producer (Leader Election)"
            scheduler -> cacheRepo "2. SMEMBERS server:all_ids"
            cacheRepo -> redis "3. Fetch targets"
            scheduler -> queueRepo "4. RPUSH monitoring:queue"
            queueRepo -> redis "5. Enqueue targets"
            
            monitorWorker -> queueRepo "6. LPOP monitoring:queue (Consume Job)"
            queueRepo -> redis "7. Dequeue target"
            monitorWorker -> targetServers "8. ICMP Ping"
            monitorWorker -> monitoringService "9. Evaluate(serverID, success)"
            monitoringService -> observationLogger "10. LogObservation [fire-and-forget]"
            observationLogger -> es "11. Buffered Bulk Write"
            monitoringService -> cacheRepo "12. Get & Set ServerState"
            cacheRepo -> redis "13. HGET / HSET"
            monitoringService -> redis "14. [If State Changed] XADD sms.events.server_status"
            autoLayout
        }

        dynamic reporting "Reporting_Generate" "Detailed sequence for automated report generation" {
            dailyScheduler -> reportService "1. Trigger daily report"
            reportService -> reportRepo "2. CreateReportRequest(PENDING)"
            reportRepo -> postgres "3. INSERT"
            reportService -> reportWorker "4. Enqueue to jobQueue (In-memory Channel)"
            
            reportWorker -> reportRepo "5. Dequeue job & UpdateReportStatus(PROCESSING)"
            reportRepo -> postgres "6. UPDATE report_requests"
            reportWorker -> reportRepo "7. GetServerCountByStatus()"
            reportRepo -> postgres "8. SELECT FROM reporting_servers"
            reportWorker -> esUptimeCalc "9. CalculateUptime()"
            esUptimeCalc -> es "10. Elasticsearch Aggregation Query"
            reportWorker -> notifier "11. SendReportEmail(Render HTML)"
            notifier -> smtp "12. Send SMTP Email"
            reportWorker -> reportRepo "13. UpdateReportStatus(COMPLETED)"
            reportRepo -> postgres "14. UPDATE report_requests"
            autoLayout
        }

        styles {
            element "Person" {
                color #ffffff
                background #08427b
                shape Person
            }
            element "Software System" {
                background #1168bd
                color #ffffff
            }
            element "Container" {
                background #438dd5
                color #ffffff
            }
            element "Database" {
                shape Cylinder
                background #22649b
                color #ffffff
            }
            element "Gateway" {
                shape Pipe
            }
            element "Worker" {
                shape Hexagon
            }
            element "Message Broker" {
                shape Pipe
                background #ff9900
                color #000000
            }
            element "Component" {
                background #85bbf0
                color #000000
            }
            element "External" {
                background #999999
                color #ffffff
            }
        }
    }
}

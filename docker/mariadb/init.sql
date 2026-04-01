-- Applad database initialization
-- All schema changes beyond this are handled by the migrations system.

CREATE DATABASE IF NOT EXISTS n8n CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
GRANT ALL PRIVILEGES ON n8n.* TO 'applad'@'%';
FLUSH PRIVILEGES;

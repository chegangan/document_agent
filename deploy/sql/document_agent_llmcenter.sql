-- 设置客户端连接的字符集为 utf8mb4
SET NAMES utf8mb4;
-- 暂时禁用外键约束检查，以便顺利创建或删除表
SET FOREIGN_KEY_CHECKS = 0;

-- --------------------------------------------------
-- LLM Service Database and User Setup
-- --------------------------------------------------

-- 创建一个新的数据库，专门用于 LLM 服务
CREATE DATABASE IF NOT EXISTS `document_agent_llmcenter` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

-- 创建一个名为 'llmcenter' 的新用户，并为其设置一个强密码。
-- 出于安全考虑，请在生产环境中替换为更复杂的密码。
CREATE USER 'llmcenter'@'%' IDENTIFIED BY 'G7v@pL2#xQ9!sT8z';

-- 授予 'llmcenter' 用户对 'document_agent_llmcenter' 数据库的全部权限。
GRANT ALL PRIVILEGES ON `document_agent_llmcenter`.* TO 'llmcenter'@'%';

-- 刷新权限，以确保上述授权更改立即生效。
FLUSH PRIVILEGES;

-- 切换到新创建的数据库上下文
USE `document_agent_llmcenter`;

-- --------------------------------------------------
-- Table structure for conversations (会话表)
-- --------------------------------------------------
DROP TABLE IF EXISTS `conversations`;
CREATE TABLE `conversations` (
  `conversation_id` VARCHAR(32) NOT NULL COMMENT '会话ID (主键, ULID)',
  `user_id`         VARCHAR(255) NOT NULL COMMENT '关联的用户ID',
  `title`           VARCHAR(255) NOT NULL DEFAULT '' COMMENT '会话标题',
  `metadata`        JSON DEFAULT NULL COMMENT '存储额外的数据，例如模型设置等',
  `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',
  PRIMARY KEY (`conversation_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='会话表';

-- --------------------------------------------------
-- Table structure for messages (消息表)
-- --------------------------------------------------
DROP TABLE IF EXISTS `messages`;
CREATE TABLE `messages` (
  `message_id`      VARCHAR(32) NOT NULL COMMENT '消息ID (主键, ULID)',
  `conversation_id` VARCHAR(32) NOT NULL COMMENT '关联的会话ID (外键)',
  `role`            ENUM('user', 'assistant') NOT NULL COMMENT '角色: "user" 或 "assistant"',
  `content`         TEXT NOT NULL COMMENT '消息的具体内容',
  `content_type`    VARCHAR(30) NOT NULL DEFAULT 'text' COMMENT '内容类型: "text", "document_outline" 等',
  `metadata`        JSON DEFAULT NULL COMMENT '存储额外的数据，例如引用的文档ID等',
  `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '消息创建时间',
  PRIMARY KEY (`message_id`),
  -- 为 conversation_id 创建索引以优化查询性能
  KEY `idx_conversation_id` (`conversation_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='消息表';

-- 重新启用外键约束检查
SET FOREIGN_KEY_CHECKS = 1;

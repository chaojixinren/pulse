-- Pulse 迁移：移除旧的绑定码表。
--
-- 旧的 App 生成一次性绑定码（device_bind_codes）流程整体下线，
-- 此迁移删除遗留表。IF EXISTS 保证在全新安装（002 已不再建该表）上也幂等。

DROP TABLE IF EXISTS device_bind_codes;

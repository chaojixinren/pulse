-- Pulse 迁移：移除 device_pairing_requests 表。
--
-- 硬件回退为「纯上传 + token 手抄」，硬件主导配对（device_pairing_requests）
-- 整体下线。本迁移删除遗留表；IF EXISTS 保证在全新安装（不再建该表）上也幂等。

DROP TABLE IF EXISTS device_pairing_requests;

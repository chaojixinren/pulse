-- Pulse 设备级鉴权迁移
-- DeviceAuth 中间件按 SHA256(device_token) 反查设备，每个设备请求都会走这条查询，
-- 没有索引就是全表扫描。此处用普通索引而非 UNIQUE：token 由 32 字节随机数生成，
-- 碰撞不是现实威胁，而 UNIQUE 会让任何历史脏数据直接卡住迁移。
CREATE INDEX idx_devices_token_hash ON devices (device_token_hash);

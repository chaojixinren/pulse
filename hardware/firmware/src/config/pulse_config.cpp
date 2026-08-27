/**
 * @file pulse_config.cpp
 * @brief 配置加载实现：编译期默认 → /config.json → NVS
 */
#include "pulse_config.h"

#include <ArduinoJson.h>
#include <FS.h>
#include <Preferences.h>
#include <SD_MMC.h>
#include <WiFi.h>

#include <string.h>

static const char *TAG = "cfg";
static const char *NVS_NAMESPACE = "pulse";
static const char *CONFIG_PATH = "/config.json";

static pulse_config_t g_cfg;
static bool g_defaults_applied = false;

/* 安全字符串拷贝：始终以 NUL 结尾，避免 strncpy 截断后无终止符。 */
static void cfg_strlcpy(char *dst, const char *src, size_t cap) {
    if (!dst || cap == 0) return;
    if (!src) { dst[0] = '\0'; return; }
    size_t n = strlen(src);
    if (n >= cap) n = cap - 1;
    memcpy(dst, src, n);
    dst[n] = '\0';
}

/* ─────────────── 编译期默认值 ───────────────
 * 所有数值都可被 config.json 覆盖，这里只保证"没有配置文件也能跑起来"。 */
static void apply_defaults(pulse_config_t *c) {
    memset(c, 0, sizeof(*c));

    /* WiFi：默认留空，等配网写入 NVS */
    c->wifi.connect_timeout_ms = 15000;
    c->wifi.reconnect_min_ms = 1000;
    c->wifi.reconnect_max_ms = 60000;

    /* 云端：路径对齐后端 router.go 的 /api/v1/audio/upload */
    cfg_strlcpy(c->cloud.base_url, "http://192.168.1.100:8080", CFG_URL_LEN);
    cfg_strlcpy(c->cloud.upload_path, "/api/v1/audio/upload", CFG_STR_LEN);
    cfg_strlcpy(c->cloud.auth_scheme, "Bearer", sizeof(c->cloud.auth_scheme));
    c->cloud.request_timeout_ms = 30000;
    c->cloud.max_retry = 5;
    c->cloud.retry_backoff_ms = 2000;
    c->cloud.verify_tls = false;

    /* 音频：30s / 960KB 两个阈值等价（32000 B/s），先到先切 */
    c->audio.dma_buf_count = 8;
    c->audio.dma_buf_len = 512;
    c->audio.gain_shift = 0;
    c->audio.dc_removal = true;
    c->audio.segment_ms = 30000;
    c->audio.segment_max_bytes = 960000;

    /* VAD：阈值必须现场实测，此处仅为可启动的初值 */
    c->vad.enabled = true;
    c->vad.energy_threshold = 1500;
    c->vad.zcr_min = 5;
    c->vad.zcr_max = 120;
    c->vad.speech_confirm_ms = 150;
    c->vad.hangover_ms = 800;
    c->vad.preroll_ms = 300;
    c->vad.min_segment_ms = 1000;

    /* 存储 */
    cfg_strlcpy(c->storage.spool_dir, "/rec/pending", CFG_STR_LEN);
    cfg_strlcpy(c->storage.failed_dir, "/rec/failed", CFG_STR_LEN);
    c->storage.max_spool_mb = 2048;
    c->storage.min_free_mb = 64;
    c->storage.sd_freq_mhz = 20; /* 保守值：GPIO45/46 若有上拉需降频 */
    c->storage.sd_use_1bit = false;
    c->storage.fallback_to_psram = true;
    c->storage.psram_buffer_kb = 4096;

    /* 电源：三级功耗。deep_sleep_ms 默认 0 = 永不自动深睡，
     * 避免"静音就深睡"导致设备在白天聋掉。 */
    c->power.screen_dim_ms = 30000;
    c->power.deep_sleep_ms = 0;
    c->power.sleep_requires_empty_spool = true;
    c->power.battery_warn_pct = 10;
    c->power.battery_crit_pct = 5;
    c->power.backlight_pct = 70;
    c->power.pmu_poll_ms = 1000;

    /* 界面 */
    c->ui.rotation = 0;
    c->ui.lvgl_tick_ms = 5;
    cfg_strlcpy(c->ui.timezone, "CST-8", CFG_STR_LEN);
    cfg_strlcpy(c->ui.ntp_server, "pool.ntp.org", CFG_STR_LEN);

    g_defaults_applied = true;
}

/* ─────────────── JSON 取值助手 ───────────────
 * 字段缺失时保留原值（即默认值），不报错。 */
static void js_str(JsonVariantConst v, char *dst, size_t cap) {
    if (!v.isNull() && v.is<const char *>()) cfg_strlcpy(dst, v.as<const char *>(), cap);
}
static void js_u32(JsonVariantConst v, uint32_t *dst) {
    if (!v.isNull() && v.is<uint32_t>()) *dst = v.as<uint32_t>();
}
static void js_u16(JsonVariantConst v, uint16_t *dst) {
    if (!v.isNull() && v.is<uint16_t>()) *dst = v.as<uint16_t>();
}
static void js_u8(JsonVariantConst v, uint8_t *dst) {
    if (!v.isNull() && v.is<uint8_t>()) *dst = v.as<uint8_t>();
}
static void js_i16(JsonVariantConst v, int16_t *dst) {
    if (!v.isNull() && v.is<int16_t>()) *dst = v.as<int16_t>();
}
static void js_bool(JsonVariantConst v, bool *dst) {
    if (!v.isNull() && v.is<bool>()) *dst = v.as<bool>();
}

/* 解析 /config.json。文件不存在或格式错误返回 false，g_cfg 保持默认值。 */
static bool load_from_file(pulse_config_t *c) {
    /* SD 未挂载时（app 启动早期 storage_init 尚未执行）直接跳过。
     * 用 cardType() 判断而非 exists()：后者在未挂载时会触发库的
     * "File system is not mounted" 误导日志。挂载成功后 storage_init
     * 会再次调用 config_load() 补读文件配置。 */
    if (SD_MMC.cardType() == CARD_NONE) {
        return false;
    }
    if (!SD_MMC.exists(CONFIG_PATH)) {
        log_w("%s: %s 不存在，使用内置默认值", TAG, CONFIG_PATH);
        return false;
    }
    File f = SD_MMC.open(CONFIG_PATH, FILE_READ);
    if (!f) {
        log_e("%s: 打开 %s 失败", TAG, CONFIG_PATH);
        return false;
    }

    /* 8KB：当前 config.json 约 2.4KB、47 个键，ArduinoJson 的 DOM + 字符串
     * 副本约需输入的 1.5~2 倍，4KB 处于临界值。留足余量并显式限制上限，
     * 避免配置文件被恶意放大导致 OOM。 */
    DynamicJsonDocument doc(8192);
    DeserializationError err = deserializeJson(doc, f);
    f.close();
    if (err) {
        log_e("%s: config.json 解析失败 (%s)，回退默认值", TAG, err.c_str());
        return false;
    }
    /* 解析成功但 DOM 内存不足时 deserializeJson 可能不报错，必须单独查。 */
    if (doc.overflowed()) {
        log_e("%s: config.json 超出 %d 字节容量，配置被截断，回退默认值", TAG,
              (int)doc.capacity());
        return false;
    }

    JsonObjectConst r = doc.as<JsonObjectConst>();

    JsonObjectConst w = r["wifi"];
    js_str(w["ssid"], c->wifi.ssid, CFG_STR_LEN);
    js_str(w["password"], c->wifi.password, CFG_STR_LEN);
    js_u32(w["connect_timeout_ms"], &c->wifi.connect_timeout_ms);
    js_u32(w["reconnect_min_ms"], &c->wifi.reconnect_min_ms);
    js_u32(w["reconnect_max_ms"], &c->wifi.reconnect_max_ms);

    JsonObjectConst cl = r["cloud"];
    js_str(cl["base_url"], c->cloud.base_url, CFG_URL_LEN);
    js_str(cl["upload_path"], c->cloud.upload_path, CFG_STR_LEN);
    js_str(cl["device_id"], c->cloud.device_id, CFG_STR_LEN);
    js_str(cl["auth_scheme"], c->cloud.auth_scheme, sizeof(c->cloud.auth_scheme));
    js_str(cl["auth_token"], c->cloud.auth_token, CFG_TOKEN_LEN);
    js_u32(cl["request_timeout_ms"], &c->cloud.request_timeout_ms);
    js_u8(cl["max_retry"], &c->cloud.max_retry);
    js_u32(cl["retry_backoff_ms"], &c->cloud.retry_backoff_ms);
    js_bool(cl["verify_tls"], &c->cloud.verify_tls);

    JsonObjectConst a = r["audio"];
    js_u16(a["dma_buf_count"], &c->audio.dma_buf_count);
    js_u16(a["dma_buf_len"], &c->audio.dma_buf_len);
    js_i16(a["gain_shift"], &c->audio.gain_shift);
    js_bool(a["dc_removal"], &c->audio.dc_removal);
    js_u32(a["segment_ms"], &c->audio.segment_ms);
    js_u32(a["segment_max_bytes"], &c->audio.segment_max_bytes);

    JsonObjectConst v = r["vad"];
    js_bool(v["enabled"], &c->vad.enabled);
    js_u32(v["energy_threshold"], &c->vad.energy_threshold);
    js_u16(v["zcr_min"], &c->vad.zcr_min);
    js_u16(v["zcr_max"], &c->vad.zcr_max);
    js_u32(v["speech_confirm_ms"], &c->vad.speech_confirm_ms);
    js_u32(v["hangover_ms"], &c->vad.hangover_ms);
    js_u32(v["preroll_ms"], &c->vad.preroll_ms);
    js_u32(v["min_segment_ms"], &c->vad.min_segment_ms);

    JsonObjectConst s = r["storage"];
    js_str(s["spool_dir"], c->storage.spool_dir, CFG_STR_LEN);
    js_str(s["failed_dir"], c->storage.failed_dir, CFG_STR_LEN);
    js_u32(s["max_spool_mb"], &c->storage.max_spool_mb);
    js_u32(s["min_free_mb"], &c->storage.min_free_mb);
    js_u8(s["sd_freq_mhz"], &c->storage.sd_freq_mhz);
    js_bool(s["sd_use_1bit"], &c->storage.sd_use_1bit);
    js_bool(s["fallback_to_psram"], &c->storage.fallback_to_psram);
    js_u32(s["psram_buffer_kb"], &c->storage.psram_buffer_kb);

    JsonObjectConst p = r["power"];
    js_u32(p["screen_dim_ms"], &c->power.screen_dim_ms);
    js_u32(p["deep_sleep_ms"], &c->power.deep_sleep_ms);
    js_bool(p["sleep_requires_empty_spool"], &c->power.sleep_requires_empty_spool);
    js_u8(p["battery_warn_pct"], &c->power.battery_warn_pct);
    js_u8(p["battery_crit_pct"], &c->power.battery_crit_pct);
    js_u8(p["backlight_pct"], &c->power.backlight_pct);
    js_u32(p["pmu_poll_ms"], &c->power.pmu_poll_ms);

    JsonObjectConst u = r["ui"];
    js_u8(u["rotation"], &c->ui.rotation);
    js_u32(u["lvgl_tick_ms"], &c->ui.lvgl_tick_ms);
    js_str(u["timezone"], c->ui.timezone, CFG_STR_LEN);
    js_str(u["ntp_server"], c->ui.ntp_server, CFG_STR_LEN);

    log_i("%s: 已从 %s 载入配置", TAG, CONFIG_PATH);
    return true;
}

/* NVS 覆盖：只存运行时写入的凭据，优先级高于 config.json。 */
static bool load_from_nvs(pulse_config_t *c) {
    Preferences prefs;
    if (!prefs.begin(NVS_NAMESPACE, /*readOnly=*/true)) return false;

    bool got = false;
    char buf[CFG_TOKEN_LEN];

    if (prefs.isKey("ssid")) {
        prefs.getString("ssid", buf, CFG_STR_LEN);
        if (buf[0]) { cfg_strlcpy(c->wifi.ssid, buf, CFG_STR_LEN); got = true; }
    }
    if (prefs.isKey("pass")) {
        prefs.getString("pass", buf, CFG_STR_LEN);
        if (buf[0]) { cfg_strlcpy(c->wifi.password, buf, CFG_STR_LEN); got = true; }
    }
    if (prefs.isKey("scheme")) {
        prefs.getString("scheme", buf, sizeof(c->cloud.auth_scheme));
        if (buf[0]) { cfg_strlcpy(c->cloud.auth_scheme, buf, sizeof(c->cloud.auth_scheme)); got = true; }
    }
    if (prefs.isKey("token")) {
        prefs.getString("token", buf, CFG_TOKEN_LEN);
        if (buf[0]) { cfg_strlcpy(c->cloud.auth_token, buf, CFG_TOKEN_LEN); got = true; }
    }
    if (prefs.isKey("backlight")) {
        /* 5~100 才接受，避免 0（全黑看不见屏）或脏数据 */
        uint8_t bl = prefs.getUChar("backlight", 0);
        if (bl >= 5 && bl <= 100) { c->power.backlight_pct = bl; got = true; }
    }
    prefs.end();
    if (got) log_i("%s: NVS 覆盖已应用", TAG);
    return got;
}

/* device_id 缺省用 MAC 生成，保证每台设备唯一且重启后稳定。 */
static void ensure_device_id(pulse_config_t *c) {
    if (c->cloud.device_id[0]) return;
    uint8_t mac[6] = {0};
    esp_read_mac(mac, ESP_MAC_WIFI_STA);
    snprintf(c->cloud.device_id, CFG_STR_LEN, "pulse-%02x%02x%02x%02x%02x%02x",
             mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]);
}

bool config_load(void) {
    apply_defaults(&g_cfg);
    /* SD 卡可能未挂载（M0 测试场景），load_from_file 会自行判定并回退。 */
    g_cfg.loaded_from_file = load_from_file(&g_cfg);
    g_cfg.loaded_from_nvs = load_from_nvs(&g_cfg);
    ensure_device_id(&g_cfg);
    return g_cfg.loaded_from_file;
}

const pulse_config_t *config_get(void) {
    if (!g_defaults_applied) apply_defaults(&g_cfg);
    return &g_cfg;
}

bool config_save_wifi(const char *ssid, const char *password) {
    Preferences prefs;
    if (!prefs.begin(NVS_NAMESPACE, /*readOnly=*/false)) return false;
    prefs.putString("ssid", ssid ? ssid : "");
    prefs.putString("pass", password ? password : "");
    prefs.end();
    cfg_strlcpy(g_cfg.wifi.ssid, ssid, CFG_STR_LEN);
    cfg_strlcpy(g_cfg.wifi.password, password, CFG_STR_LEN);
    return true;
}

bool config_save_token(const char *scheme, const char *token) {
    Preferences prefs;
    if (!prefs.begin(NVS_NAMESPACE, /*readOnly=*/false)) return false;
    if (scheme && scheme[0]) prefs.putString("scheme", scheme);
    prefs.putString("token", token ? token : "");
    prefs.end();
    if (scheme && scheme[0]) cfg_strlcpy(g_cfg.cloud.auth_scheme, scheme, sizeof(g_cfg.cloud.auth_scheme));
    cfg_strlcpy(g_cfg.cloud.auth_token, token, CFG_TOKEN_LEN);
    return true;
}

bool config_save_backlight(uint8_t percent) {
    if (percent < 5) percent = 5;
    if (percent > 100) percent = 100;
    Preferences prefs;
    if (!prefs.begin(NVS_NAMESPACE, /*readOnly=*/false)) return false;
    prefs.putUChar("backlight", percent);
    prefs.end();
    g_cfg.power.backlight_pct = percent;
    log_i("%s: 背光亮度已持久化 %u%%", TAG, percent);
    return true;
}

/* ─────────────── 脱敏打印 ───────────────
 * 密码与 token 只显示长度和首尾字符，避免串口日志泄露凭据。 */
static void print_secret(const char *label, const char *s) {
    size_t n = s ? strlen(s) : 0;
    if (n == 0) {
        Serial.printf("  %-22s (未设置)\n", label);
    } else if (n <= 4) {
        Serial.printf("  %-22s **** (%u 字节)\n", label, (unsigned)n);
    } else {
        Serial.printf("  %-22s %c%c****%c%c (%u 字节)\n", label, s[0], s[1], s[n - 2], s[n - 1],
                      (unsigned)n);
    }
}

void config_dump(void) {
    const pulse_config_t *c = config_get();

    Serial.println();
    Serial.println("================ Pulse 配置 ================");
    Serial.printf("固件版本   %s\n", PULSE_FW_VERSION);
    Serial.printf("配置来源   config.json=%s  NVS=%s\n",
                  c->loaded_from_file ? "是" : "否(默认值)", c->loaded_from_nvs ? "是" : "否");

    Serial.println("\n[wifi]");
    Serial.printf("  %-22s %s\n", "ssid", c->wifi.ssid[0] ? c->wifi.ssid : "(未设置)");
    print_secret("password", c->wifi.password);
    Serial.printf("  %-22s %u ms\n", "connect_timeout_ms", c->wifi.connect_timeout_ms);
    Serial.printf("  %-22s %u ~ %u ms\n", "reconnect_backoff", c->wifi.reconnect_min_ms,
                  c->wifi.reconnect_max_ms);

    Serial.println("\n[cloud]");
    Serial.printf("  %-22s %s\n", "base_url", c->cloud.base_url);
    Serial.printf("  %-22s %s\n", "upload_path", c->cloud.upload_path);
    Serial.printf("  %-22s %s\n", "device_id", c->cloud.device_id);
    Serial.printf("  %-22s %s\n", "auth_scheme", c->cloud.auth_scheme);
    print_secret("auth_token", c->cloud.auth_token);
    Serial.printf("  %-22s %u ms\n", "request_timeout_ms", c->cloud.request_timeout_ms);
    Serial.printf("  %-22s %u\n", "max_retry", c->cloud.max_retry);
    Serial.printf("  %-22s %u ms\n", "retry_backoff_ms", c->cloud.retry_backoff_ms);
    Serial.printf("  %-22s %s\n", "verify_tls", c->cloud.verify_tls ? "true" : "false");

    Serial.println("\n[audio] (采样参数由后端锁定，不可配置)");
    Serial.printf("  %-22s %u Hz / %u bit / %u ch = %u B/s\n", "format(固定)", AUDIO_SAMPLE_RATE,
                  AUDIO_BITS_PER_SAMPLE, AUDIO_CHANNELS, AUDIO_BYTES_PER_SEC);
    Serial.printf("  %-22s %u x %u 采样点\n", "dma", c->audio.dma_buf_count, c->audio.dma_buf_len);
    Serial.printf("  %-22s %d\n", "gain_shift", c->audio.gain_shift);
    Serial.printf("  %-22s %s\n", "dc_removal", c->audio.dc_removal ? "true" : "false");
    Serial.printf("  %-22s %u ms\n", "segment_ms", c->audio.segment_ms);
    Serial.printf("  %-22s %u 字节 (=%.1f s)\n", "segment_max_bytes", c->audio.segment_max_bytes,
                  (double)c->audio.segment_max_bytes / AUDIO_BYTES_PER_SEC);

    Serial.println("\n[vad]");
    Serial.printf("  %-22s %s\n", "enabled", c->vad.enabled ? "true" : "false");
    Serial.printf("  %-22s %u\n", "energy_threshold", c->vad.energy_threshold);
    Serial.printf("  %-22s %u ~ %u\n", "zcr", c->vad.zcr_min, c->vad.zcr_max);
    Serial.printf("  %-22s %u ms\n", "speech_confirm_ms", c->vad.speech_confirm_ms);
    Serial.printf("  %-22s %u ms\n", "hangover_ms", c->vad.hangover_ms);
    Serial.printf("  %-22s %u ms\n", "preroll_ms", c->vad.preroll_ms);
    Serial.printf("  %-22s %u ms\n", "min_segment_ms", c->vad.min_segment_ms);

    Serial.println("\n[storage]");
    Serial.printf("  %-22s %s\n", "spool_dir", c->storage.spool_dir);
    Serial.printf("  %-22s %s\n", "failed_dir", c->storage.failed_dir);
    Serial.printf("  %-22s %u MB\n", "max_spool_mb", c->storage.max_spool_mb);
    Serial.printf("  %-22s %u MB\n", "min_free_mb", c->storage.min_free_mb);
    Serial.printf("  %-22s %u MHz\n", "sd_freq_mhz", c->storage.sd_freq_mhz);
    Serial.printf("  %-22s %s\n", "sd_bus_width", c->storage.sd_use_1bit ? "1-bit" : "4-bit");
    Serial.printf("  %-22s %s (%u KB)\n", "fallback_to_psram",
                  c->storage.fallback_to_psram ? "true" : "false", c->storage.psram_buffer_kb);

    Serial.println("\n[power] 三级功耗: Active -> Idle(灭屏仍录音) -> DeepSleep");
    Serial.printf("  %-22s %u ms\n", "screen_dim_ms", c->power.screen_dim_ms);
    Serial.printf("  %-22s %s\n", "deep_sleep_ms",
                  c->power.deep_sleep_ms == 0 ? "0 (永不自动深睡)" : String(c->power.deep_sleep_ms).c_str());
    Serial.printf("  %-22s %s\n", "sleep_needs_empty_spool",
                  c->power.sleep_requires_empty_spool ? "true" : "false");
    Serial.printf("  %-22s %u%% / %u%%\n", "battery warn/crit", c->power.battery_warn_pct,
                  c->power.battery_crit_pct);
    Serial.printf("  %-22s %u%%\n", "backlight_pct", c->power.backlight_pct);
    Serial.printf("  %-22s %u ms\n", "pmu_poll_ms", c->power.pmu_poll_ms);

    Serial.println("\n[ui]");
    Serial.printf("  %-22s %u\n", "rotation", c->ui.rotation);
    Serial.printf("  %-22s %u ms\n", "lvgl_tick_ms", c->ui.lvgl_tick_ms);
    Serial.printf("  %-22s %s\n", "timezone", c->ui.timezone);
    Serial.printf("  %-22s %s\n", "ntp_server", c->ui.ntp_server);
    Serial.println("===========================================");
    Serial.println();
}

/* ─────────────── 运行时校验 ───────────────
 * 只检查会导致运行期故障的自洽性问题，逐条打印，返回问题数。 */
int config_validate(void) {
    const pulse_config_t *c = config_get();
    int bad = 0;

    Serial.println("---------------- 配置校验 ----------------");

    if (c->cloud.base_url[0] == '\0') {
        Serial.println("  [错误] cloud.base_url 为空，无法上传");
        bad++;
    } else if (strncmp(c->cloud.base_url, "http://", 7) != 0 &&
               strncmp(c->cloud.base_url, "https://", 8) != 0) {
        Serial.println("  [错误] cloud.base_url 必须以 http:// 或 https:// 开头");
        bad++;
    }
    if (c->cloud.upload_path[0] != '/') {
        Serial.println("  [错误] cloud.upload_path 必须以 / 开头");
        bad++;
    }
    if (c->cloud.auth_token[0] == '\0') {
        Serial.println("  [警告] cloud.auth_token 未设置，上传会被后端 401 拒绝");
    }

    /* 分段时长与字节上限应大致等价，否则其中一个永远不会触发。 */
    uint32_t bytes_of_ms = (uint32_t)((uint64_t)c->audio.segment_ms * AUDIO_BYTES_PER_SEC / 1000);
    if (c->audio.segment_max_bytes < bytes_of_ms / 4) {
        Serial.printf("  [警告] segment_max_bytes(%u) 远小于 segment_ms 对应的 %u 字节，"
                      "实际分段将由字节数主导，每段仅约 %.1f 秒\n",
                      c->audio.segment_max_bytes, bytes_of_ms,
                      (double)c->audio.segment_max_bytes / AUDIO_BYTES_PER_SEC);
    }
    if (c->audio.segment_ms < 1000) {
        Serial.println("  [错误] audio.segment_ms 小于 1000，分段过碎");
        bad++;
    }
    if (c->audio.dma_buf_count < 2) {
        Serial.println("  [错误] audio.dma_buf_count 至少为 2，否则 DMA 会丢帧");
        bad++;
    }

    if (c->vad.zcr_min >= c->vad.zcr_max) {
        Serial.println("  [错误] vad.zcr_min 必须小于 zcr_max");
        bad++;
    }
    if (c->vad.enabled && c->vad.min_segment_ms > c->audio.segment_ms) {
        Serial.println("  [错误] vad.min_segment_ms 大于 audio.segment_ms，所有段都会被丢弃");
        bad++;
    }

    if (c->storage.spool_dir[0] != '/') {
        Serial.println("  [错误] storage.spool_dir 必须为绝对路径");
        bad++;
    }
    if (strcmp(c->storage.spool_dir, c->storage.failed_dir) == 0) {
        Serial.println("  [错误] spool_dir 与 failed_dir 不能相同");
        bad++;
    }
    if (c->storage.sd_freq_mhz == 0 || c->storage.sd_freq_mhz > 40) {
        Serial.println("  [错误] storage.sd_freq_mhz 应在 1~40 之间");
        bad++;
    }

    if (c->power.battery_crit_pct >= c->power.battery_warn_pct) {
        Serial.println("  [错误] battery_crit_pct 必须小于 battery_warn_pct");
        bad++;
    }
    if (c->power.backlight_pct > 100) {
        Serial.println("  [错误] power.backlight_pct 不得大于 100");
        bad++;
    }
    /* deep_sleep_ms 非 0 时必须大于灭屏时间，否则会跳过 Idle 直接深睡。 */
    if (c->power.deep_sleep_ms != 0 && c->power.deep_sleep_ms <= c->power.screen_dim_ms) {
        Serial.println("  [警告] deep_sleep_ms 不大于 screen_dim_ms，Idle 级实际会被跳过");
    }

    if (c->ui.rotation > 3) {
        Serial.println("  [错误] ui.rotation 只能是 0~3");
        bad++;
    }
    if (c->ui.lvgl_tick_ms == 0 || c->ui.lvgl_tick_ms > 50) {
        Serial.println("  [错误] ui.lvgl_tick_ms 应在 1~50 之间");
        bad++;
    }

    if (bad == 0) {
        Serial.println("  全部通过");
    } else {
        Serial.printf("  发现 %d 处错误\n", bad);
    }
    Serial.println("-----------------------------------------");
    return bad;
}

/**
 * @file pulse_app.cpp
 * @brief 应用编排层实现：模块初始化 + FreeRTOS 任务 + power hooks
 */
#include "pulse_app.h"

#include <Arduino.h>
#include <freertos/FreeRTOS.h>
#include <freertos/task.h>

#include <string.h>

#include "audio/pulse_audio.h"
#include "board_pins.h"
#include "config/pulse_config.h"
#include "hal/display/pulse_display.h"
#include "hal/pmu/pulse_pmu.h"
#include "hal/rtc/pulse_rtc.h"
#include "hal/storage/pulse_storage.h"
#include "hal/touch/pulse_touch.h"
#include "net/upload/pulse_upload.h"
#include "net/wifi/pulse_wifi.h"
#include "power/pulse_power.h"
#include "ui/lv_port/lv_port.h"
#include "ui/screens/pulse_ui.h"

static const char *TAG = "app";

/* ─────────── 跨任务共享状态（仅原子/受锁保护）─────────── */

/* 界面状态由各任务填充，UI 任务持锁读取并刷新 */
static pulse_ui_state_t g_ui;
static SemaphoreHandle_t g_ui_mutex = nullptr;

/* 事件标志：通知 UI 任务有新状态要刷新 */
static EventGroupHandle_t g_evt = nullptr;
#define EVT_UI_REFRESH (1 << 0)

/* ─────────── 共享状态更新（供各任务调用）─────────── */

static void ui_state_lock(void) {
    if (g_ui_mutex) xSemaphoreTake(g_ui_mutex, portMAX_DELAY);
}
static void ui_state_unlock(void) {
    if (g_ui_mutex) xSemaphoreGive(g_ui_mutex);
}
static void ui_signal_refresh(void) {
    if (g_evt) xEventGroupSetBits(g_evt, EVT_UI_REFRESH);
}

/* 由各任务更新自己的那份状态字段，再发刷新信号 */
static void update_rec_state(ui_rec_state_t s, uint32_t dur) {
    ui_state_lock();
    g_ui.rec_state = s;
    g_ui.rec_duration_s = dur;
    ui_state_unlock();
    ui_signal_refresh();
}

/* ─────────── 音频任务（Core 1，实时域）─────────── */

#define AUDIO_FRAME_SAMPLES 320
static int16_t g_audio_frame[AUDIO_FRAME_SAMPLES];
static int16_t g_preroll_buf[AUDIO_SAMPLE_RATE];

static void audio_task(void *arg) {
    (void)arg;
    log_i("%s: 音频任务启动 (Core %d)", TAG, xPortGetCoreID());

    storage_segment_t *seg = nullptr;
    uint32_t seg_start_ms = 0;
    uint32_t seg_count = 0;

    const pulse_config_t *c = config_get();

    vad_reset();
    audio_clear_preroll();

    while (true) {
        size_t got = 0;
        vad_result_t vr;
        if (!audio_read_with_vad(g_audio_frame, AUDIO_FRAME_SAMPLES, &got, &vr, 200)) {
            vTaskDelay(pdMS_TO_TICKS(5));
            continue;
        }
        if (got == 0) {
            /* 读空（正常不应发生：i2s_read 会阻塞到有数据或超时），
             * 加 1ms 让步避免极少数情况下 tight loop 占满 CPU。 */
            vTaskDelay(pdMS_TO_TICKS(1));
            continue;
        }

        /* 语音起点：开新段 + 补 pre-roll */
        if (vr.speech_started && !seg) {
            uint32_t stamp = pulse_rtc_unix();
            if (stamp == 0) stamp = (uint32_t)(millis() / 1000);
            seg = storage_segment_open(stamp);
            if (seg) {
                seg_start_ms = millis();
                size_t pn = audio_get_preroll(g_preroll_buf, AUDIO_SAMPLE_RATE);
                if (pn) storage_segment_write(seg, g_preroll_buf, pn * sizeof(int16_t));
                audio_clear_preroll();
                log_i("%s: 段 %u 开始", TAG, (unsigned)seg_count + 1);
            }
        }

        /* 录音中写帧 */
        if (seg && (vr.state == VAD_SPEECH || vr.state == VAD_HANGOVER)) {
            storage_segment_write(seg, g_audio_frame, got * sizeof(int16_t));
        }

        /* 结段 */
        bool hit_time = seg && (millis() - seg_start_ms >= c->audio.segment_ms);
        bool hit_size = seg && (storage_segment_size(seg) >= c->audio.segment_max_bytes);
        if (seg && (vr.speech_ended || hit_time || hit_size)) {
            uint32_t dur = millis() - seg_start_ms;
            bool keep = dur >= c->vad.min_segment_ms;
            storage_segment_close(seg, keep);
            seg = nullptr;
            if (keep) {
                seg_count++;
                log_i("%s: 段 %u 结束 (%ums%s)", TAG, (unsigned)seg_count, dur,
                      hit_time ? " 达时长" : hit_size ? " 达字节" : "");
            }
            /* 连续说话则续段 */
            if ((hit_time || hit_size) && !vr.speech_ended) {
                uint32_t stamp = pulse_rtc_unix();
                if (stamp == 0) stamp = (uint32_t)(millis() / 1000);
                seg = storage_segment_open(stamp);
                if (seg) seg_start_ms = millis();
            }
        }

        /* 上报录音状态 + 语音活动 */
        ui_rec_state_t rs = (vr.state == VAD_SPEECH)       ? UI_REC_SPEECH
                            : (vr.state == VAD_HANGOVER)   ? UI_REC_HANGOVER
                            : (seg != nullptr)             ? UI_REC_IDLE
                                                           : UI_REC_IDLE;
        update_rec_state(rs, seg ? (millis() - seg_start_ms) / 1000 : 0);
        if (vr.state == VAD_SPEECH) pulse_power_notify_speech();

        vTaskDelay(pdMS_TO_TICKS(1));
    }
}

/* ─────────── 网络任务（Core 0）─────────── */

static void net_task(void *arg) {
    (void)arg;
    log_i("%s: 网络任务启动 (Core %d)", TAG, xPortGetCoreID());

    /* 先尝试从 RTC 恢复时间，再联网，联网成功后补 SNTP 校时 */
    if (pulse_rtc_init()) {
        log_i("%s: RTC %s", TAG, pulse_rtc_source_str(pulse_rtc_source()));
    } else {
        log_w("%s: RTC 初始化失败（I2C 无应答或硬件异常）", TAG);
        /* 在屏幕上弹出告警，避免用户发现录音时间戳全部错乱后无法排查。
         * LVGL 此时已就绪（pulse_app_init 中 lv_port_init 在此任务创建前完成）。 */
        if (lv_port_lock(50)) {
            pulse_ui_toast("RTC 硬件异常\n录音时间戳不可靠");
            lv_port_unlock();
        }
    }

    pulse_wifi_init();

    uint32_t last_sntp = 0;
    bool sntp_done = false;

    while (true) {
        pulse_wifi_tick();

        /* 无凭据（首次开机）时自动启动 AP 配网门户。
         * start_provisioning 会把状态从 NO_CREDENTIALS 切到 AP_PROVISION，
         * 之后不再触发，无需额外防重入。 */
        if (pulse_wifi_state() == PULSE_WIFI_NO_CREDENTIALS) {
            pulse_ui_set_provision_mode(false); /* 初次配网 */
            pulse_wifi_start_provisioning();
        }

        pulse_upload_tick();

        /* 联网且 RTC 未 SNTP 校时过，则定期尝试校时 */
        if (pulse_wifi_is_connected() && !sntp_done && pulse_rtc_is_ready()) {
            uint32_t now = millis();
            if (last_sntp == 0 || now - last_sntp >= 300000) { /* 每 5 分钟 */
                last_sntp = now;
                if (pulse_rtc_sync_sntp(10000)) sntp_done = true;
            }
        }

        /* 更新 UI 网络/上传状态 */
        pulse_wifi_status_t ws;
        pulse_wifi_get_status(&ws);
        upload_status_t us;
        pulse_upload_get_status(&us);

        ui_state_lock();
        g_ui.wifi_connected = ws.connected;
        g_ui.rssi = ws.rssi;
        g_ui.upload_state = (ui_upload_state_t)us.state;
        g_ui.pending_count = (uint32_t)storage_pending_count();
        g_ui.sent_ok = us.sent_ok;
        g_ui.sent_failed = us.sent_failed;
        ui_state_unlock();
        ui_signal_refresh();

        vTaskDelay(pdMS_TO_TICKS(200));
    }
}

/* ─────────── UI 任务（Core 0）─────────── */

/* 背光回调：设置页滑块触发。commit=false 拖动预览（只调背光），
 * true 松开确认（同时持久化到 NVS）。此回调在 LVGL 锁内被调用。 */
static void backlight_cb(uint8_t pct, bool commit) {
    display_set_backlight(pct);
    if (commit) config_save_backlight(pct);
}

/* WiFi 设置回调：设置页点 WiFi 按钮触发，启动 AP 配网流程。
 * 注意：此回调在 LVGL 锁内被调用，start_provisioning 会切 WiFi 模式、
 * 开软 AP，可能阻塞数百 ms —— 低频用户操作，可接受。 */
static void wifi_cb(void) {
    /* 用户从设置页主动点 WiFi 按钮 = 想换网（当前可能已连接）*/
    pulse_ui_set_provision_mode(true);
    pulse_wifi_start_provisioning();
}

static void ui_task(void *arg) {
    (void)arg;
    log_i("%s: UI 任务启动 (Core %d)", TAG, xPortGetCoreID());

    if (!lv_port_lock(1000)) {
        log_e("%s: LVGL 锁获取失败", TAG);
        vTaskDelete(nullptr);
        return;
    }
    pulse_ui_init();
    pulse_ui_set_backlight_cb(backlight_cb);
    pulse_ui_set_wifi_cb(wifi_cb);
    pulse_ui_set_backlight(config_get()->power.backlight_pct);
    lv_port_unlock();

    while (true) {
        /* 驱动 LVGL */
        lv_port_task_handler();

        /* 刷新界面（持锁） */
        if (lv_port_lock(50)) {
            ui_state_lock();
            pulse_ui_state_t snapshot = g_ui;
            ui_state_unlock();
            pulse_ui_update(&snapshot);

            /* 配网提示：无凭据或正在配网（AP 已开）时显示 */
            pulse_wifi_state_t ws = pulse_wifi_state();
            bool need_prov = (ws == PULSE_WIFI_NO_CREDENTIALS || ws == PULSE_WIFI_AP_PROVISION);
            if (need_prov != pulse_ui_provision_hint_visible()) {
                pulse_ui_show_provision_hint(need_prov);
            }
            if (need_prov) {
                /* 更新热点名：AP_PROVISION 时 ssid 是真实 AP 名，其余时刻为空 */
                pulse_wifi_status_t wst;
                pulse_wifi_get_status(&wst);
                pulse_ui_set_provision_ap(wst.ssid[0] ? wst.ssid : nullptr);
            }
            lv_port_unlock();
        }

        /* 电源 tick 低频跑在这里 */
        pulse_power_tick();

        vTaskDelay(pdMS_TO_TICKS(5));
    }
}

/* ─────────── 时间/电量刷新（挂在 UI 任务里）─────────── */

static void refresh_time_and_battery(void) {
    /* 时间 */
    struct tm t;
    if (pulse_rtc_get_utc(&t)) {
        ui_state_lock();
        g_ui.time_valid = true;
        g_ui.hour = t.tm_hour;
        g_ui.minute = t.tm_min;
        g_ui.second = t.tm_sec;
        g_ui.year = t.tm_year + 1900;
        g_ui.month = t.tm_mon + 1;
        g_ui.day = t.tm_mday;
        ui_state_unlock();
    }

    /* 电量 */
    pmu_status_t ps;
    if (pmu_read(&ps)) {
        ui_state_lock();
        g_ui.battery_present = ps.battery_present;
        g_ui.battery_percent = ps.percent;
        g_ui.charging = ps.charging;
        ui_state_unlock();
    }

    /* 存储 */
    storage_info_t si;
    if (storage_stat(&si)) {
        ui_state_lock();
        g_ui.storage_kind = (si.backend == STORAGE_BACKEND_PSRAM) ? UI_STORAGE_PSRAM
                            : (si.backend == STORAGE_BACKEND_NONE) ? UI_STORAGE_NONE
                                                                   : UI_STORAGE_SD;
        g_ui.storage_free_mb = si.free_bytes / 1048576;
        g_ui.storage_total_mb = si.total_bytes / 1048576;
        ui_state_unlock();
    }
    ui_signal_refresh();
}

/* ─────────── power hooks 实现 ─────────── */

static void hook_enter_idle(void) {
    /* 灭屏：背光 PWM=0。显示面板保持上电以便快速恢复。 */
    display_set_backlight(0);
    log_i("%s: 灭屏（背光=0），录音继续", TAG);
}

static void hook_exit_idle(void) {
    display_set_backlight(config_get()->power.backlight_pct);
    log_i("%s: 亮屏，恢复背光 %u%%", TAG, config_get()->power.backlight_pct);
}

static bool hook_can_sleep(void) {
    const pulse_config_t *c = config_get();
    if (!c->power.sleep_requires_empty_spool) return true;
    bool empty = pulse_upload_queue_empty();
    if (!empty) {
        log_w("%s: 深睡被否决：spool 还有 %d 段未上传", TAG, storage_pending_count());
    }
    return empty;
}

static void hook_before_sleep(void) {
    /* 收尾顺序：先停音频（关 I2S），再关触摸/显示，最后断网 */
    audio_stop();
    update_rec_state(UI_REC_STOPPED, 0); /* 录音已停，界面（若可见）显示「已停止」 */
    touch_sleep();
    display_on_off(false);
    display_set_backlight(0);
    pulse_wifi_shutdown();
    log_i("%s: 深睡收尾完成", TAG);
}

static void hook_battery_warn(int pct) {
    log_w("%s: 低电 %d%%", TAG, pct);
    if (lv_port_lock(50)) {
        pulse_ui_toast("电量低于 10%，请充电");
        lv_port_unlock();
    }
}

static void hook_battery_crit(int pct) {
    log_e("%s: 电量危急 %d%%，即将深睡", TAG, pct);
    /* 危急时 pulse_power 会以 force=true 调深睡，这里不重复触发 */
}

/* ─────────── 模块初始化 ─────────── */

bool pulse_app_init(void) {
    config_load();

    /* 1. PMU：最早，因为后续模块可能依赖它的供电 */
    pmu_init();

    /* 2. 存储：音频任务依赖，降级 PSRAM 也能继续 */
    storage_init();

    /* 3. 显示 + 触摸 + LVGL（顺序敏感：共用复位 → 显示 → 触摸 → LVGL）*/
    display_shared_reset();
    if (!display_init()) {
        log_e("%s: 显示初始化失败", TAG);
        return false;
    }
    display_set_backlight(config_get()->power.backlight_pct);
    touch_init();
    if (!lv_port_init()) {
        log_e("%s: LVGL 初始化失败", TAG);
        return false;
    }

    /* 4. 音频 */
    if (!audio_init()) {
        log_e("%s: 音频初始化失败", TAG);
        return false;
    }

    /* 5. 上传（不依赖 RTC 已校时）*/
    pulse_upload_init();

    /* 6. 电源 hooks */
    pulse_power_hooks_t hooks = {};
    hooks.on_enter_idle = hook_enter_idle;
    hooks.on_exit_idle = hook_exit_idle;
    hooks.can_sleep = hook_can_sleep;
    hooks.before_sleep = hook_before_sleep;
    hooks.on_battery_warn = hook_battery_warn;
    hooks.on_battery_crit = hook_battery_crit;
    pulse_power_init(&hooks);

    log_i("%s: 应用初始化完成", TAG);
    return true;
}

void pulse_app_start(void) {
    g_ui_mutex = xSemaphoreCreateMutex();
    g_evt = xEventGroupCreate();

    memset(&g_ui, 0, sizeof(g_ui));
    g_ui.storage_kind = UI_STORAGE_SD;
    g_ui.battery_present = false;

    /* 任务划分：
     *   音频 → Core 1（app core，实时域，与 WiFi 协议栈的 Core 0 隔离）
     *   网络 → Core 0（WiFi/BT 协议栈默认所在核，减少跨核调度）
     *   UI   → Core 0（与网络共核，两者都不是实时任务）
     *
     * 栈大小说明：音频任务栈大（写 SD 会深调用），网络任务要跑 HTTP
     * （TLS 握手栈很深），UI 任务跑 LVGL。都用 PSRAM 不行的 —— FreeRTOS
     * 任务栈默认在内部分配，这里显式给足。 */
    /* 栈大小：net_task 要跑 HTTPS 上传（mbedTLS 握手栈很深，5~8KB），
     * 给 16384 留足余量；audio/UI 各 8192 足够。 */
    xTaskCreatePinnedToCore(audio_task, "audio", 8192, nullptr, 6, nullptr, 1);
    xTaskCreatePinnedToCore(net_task, "net", 16384, nullptr, 4, nullptr, 0);
    xTaskCreatePinnedToCore(ui_task, "ui", 8192, nullptr, 2, nullptr, 0);

    log_i("%s: 三个任务已创建（audio@core1, net@core0, ui@core0）", TAG);
    log_i("%s: Arduino loop 空闲运行在 core1，仅做时间/电量低频刷新", TAG);
}

/* ─────────── Arduino 入口转发（setup/loop 在 main.cpp）─────────── */

void pulse_app_bootstrap(void) {
    if (!pulse_app_init()) {
        Serial.println("!!!! 初始化失败，进入错误循环 !!!!");
        while (true) {
            delay(1000);
        }
    }

    /* 打印整体诊断 */
    Serial.println();
    pmu_dump_rails();
    storage_dump();
    pulse_rtc_dump();
    pulse_wifi_dump();
    pulse_upload_dump();
    pulse_power_dump();
    lv_port_dump_buffers();

    pulse_app_start();
}

/* loop 空闲，跑在 core1（与音频任务同核）。只做低频刷新，不阻塞。 */
void pulse_app_loop(void) {
    static uint32_t last = 0;
    uint32_t now = millis();
    if (now - last >= 1000) {
        last = now;
        refresh_time_and_battery();
    }
    vTaskDelay(pdMS_TO_TICKS(20));
}

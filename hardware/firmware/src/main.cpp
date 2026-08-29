/**
 * @file main.cpp
 * @brief Pulse 固件入口
 *
 * 本文件只做入口转发，真正的编排在 app/pulse_app.cpp。
 * setup()/loop() 是 Arduino 框架的约定入口，保持精简，
 * 避免把初始化逻辑堆在这里难以维护。
 *
 * 任务布局（详见 pulse_app.cpp）：
 *   Core 1  audio_task  I2S PDM RX + VAD + 段写入
 *   Core 0  net_task    WiFi 状态机 + 上传状态机 + SNTP
 *   Core 0  ui_task     LVGL 渲染 + 触摸 + 电源 tick
 *   loop               （core1）时间/电量低频刷新
 */
#include <Arduino.h>

#include "app/pulse_app.h"
#include "config/pulse_config.h"

void setup() {
    Serial.begin(115200);
    uint32_t t0 = millis();
    while (!Serial && (millis() - t0) < 3000) {
        delay(50);
    }
    delay(200);

    Serial.println();
    Serial.println("##########################################");
    Serial.printf("#   Pulse 固件 v%s\n", PULSE_FW_VERSION);
    Serial.println("##########################################");

    pulse_app_bootstrap();
}

void loop() {
    pulse_app_loop();
}

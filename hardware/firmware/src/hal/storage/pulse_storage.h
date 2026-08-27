/**
 * @file pulse_storage.h
 * @brief 存储层 HAL：TF 卡 spool 队列，无卡时降级 PSRAM 环形缓冲
 *
 * 降级设计（重要）
 * ----------------
 * 你的板子 GPIO45/46 焊了外部上拉，而这两脚正是 SD_D0/SD_D1：
 *   - GPIO45 被拉高会把 VDD_SPI 切到 1.8V，而 S3R8 封装内 PSRAM 是 3.3V
 *   - 因此 SD 能否挂载存在不确定性
 *
 * 为了让 M5/M8/M10 不被 SD 阻塞，本层把存储做成**可降级**：
 *   SD 4-bit → SD 1-bit → PSRAM 环形缓冲 → 只读失败态
 * 上层只调用 storage_* 接口，不关心底层是卡还是内存。
 *
 * PSRAM 模式下容量有限（默认 4MB ≈ 2 分钟音频），满了丢最旧的段，
 * 并在 UI 上给出「未插卡」提示。这是应急路径，不是正常工作模式。
 */
#ifndef PULSE_STORAGE_H
#define PULSE_STORAGE_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef enum {
    STORAGE_BACKEND_NONE = 0, /* 初始化失败，无法录音 */
    STORAGE_BACKEND_SD_4BIT,
    STORAGE_BACKEND_SD_1BIT,
    STORAGE_BACKEND_PSRAM, /* 降级模式：容量小且掉电丢失 */
} storage_backend_t;

typedef struct {
    storage_backend_t backend;
    uint64_t total_bytes;
    uint64_t used_bytes;
    uint64_t free_bytes;
    uint32_t pending_count; /* 待上传段数 */
    uint64_t pending_bytes;
    bool writable; /* 剩余空间是否仍高于 min_free_mb */
} storage_info_t;

/* 段写入句柄。对上层屏蔽 SD 文件与 PSRAM 缓冲的差异。 */
typedef struct storage_segment storage_segment_t;

/**
 * 初始化存储。按 4-bit → 1-bit → PSRAM 顺序降级，不会因无卡而失败。
 * @return false 仅当所有后端都不可用（连 PSRAM 都分配不出）
 */
bool storage_init(void);

storage_backend_t storage_backend(void);
const char *storage_backend_str(storage_backend_t b);

/** 查询容量与队列状态 */
bool storage_stat(storage_info_t *out);

/**
 * 新建一个待上传段。
 * @param recorded_at_unix 录制起始时间（用于文件名与 recorded_at 字段）
 * @return NULL 表示空间不足或后端不可用
 */
storage_segment_t *storage_segment_open(uint32_t recorded_at_unix);

/** 向段追加 PCM 数据。返回实际写入字节数，小于 len 表示空间不足。 */
size_t storage_segment_write(storage_segment_t *seg, const void *data, size_t len);

/** 当前段已写入的 PCM 字节数（不含 WAV 头）*/
size_t storage_segment_size(const storage_segment_t *seg);

/**
 * 关闭段并补齐 WAV 头。
 * @param keep false 表示丢弃该段（如 VAD 判定过短）
 * @return false 表示写入或重命名失败
 */
bool storage_segment_close(storage_segment_t *seg, bool keep);

/**
 * 遍历待上传段。idx 从 0 开始递增直到返回 false。
 * @param path_out 输出路径（SD 模式）或标识符（PSRAM 模式）
 */
bool storage_pending_at(int idx, char *path_out, size_t path_cap, size_t *size_out);

/** 待上传段数量 */
int storage_pending_count(void);

/** 读取某个待上传段的完整内容到调用方缓冲（用于小文件或测试）*/
size_t storage_read_file(const char *path, void *buf, size_t cap);

/**
 * 从指定偏移读取一块数据。供流式上传使用。
 *
 * 为什么需要它：一段 30 秒音频约 960KB，整块读进内存会直接 OOM
 * （内部 RAM 只有 320KB）。上传层用它按 4KB 粒度边读边发。
 *
 * @param offset 文件内偏移（含 WAV 头）
 * @return 实际读取字节数；0 表示到达文件尾或出错
 */
size_t storage_read_chunk(const char *path, size_t offset, void *buf, size_t len);

/** 取某个待上传段的字节大小；0 表示不存在 */
size_t storage_file_size(const char *path);

/** 上传成功后删除该段 */
bool storage_delete(const char *path);

/** 上传反复失败后移入 failed 目录，避免堵塞队列 */
bool storage_move_to_failed(const char *path);

/** 空间不足时淘汰最旧的段，返回释放的字节数 */
uint64_t storage_evict_oldest(uint64_t need_bytes);

/** 串口打印存储状态诊断 */
void storage_dump(void);

#ifdef __cplusplus
}
#endif

#endif /* PULSE_STORAGE_H */

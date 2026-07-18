# FunctionGraph Fault Knowledge Base

> 本知识库汇总 FunctionGraph 常见故障模式、根因定位速查与跨 skill 联动,基于 `monitoring.md` 中定义的真实指标信号。命令示例以 `hcloud` CLI 为主,不确定的参数以 `{{user.*}}` / `{{env.*}}` 占位。

---

## 常见故障模式

| 故障ID | 症状 | 根因 | 排查步骤 | 恢复动作 |
|---|---|---|---|---|
| FG-01 | 失败率升高 | 代码异常/依赖故障 | 查 `fail_count > 5%`;查日志 | 修复代码;回滚版本 |
| FG-02 | 执行超时 | 逻辑慢/下游慢 | 查 `max_duration > 90%` 阈值 | 优化逻辑或调高超时阈值 |
| FG-03 | 触发限流 | 并发超配额 | 查 `reject_count > 0` | 提升并发配额;削峰 |
| FG-04 | 并发逼近上限 | 流量突增 | 查 `concurrent_executions > 80%` 上限 | 扩容并发上限;限流入口 |
| FG-05 | OOM | 内存配置不足 | 查 `duration` + `fail_count` 同时抬升 | 调大内存规格 |
| FG-06 | 冷启动冲击 | 实例回收频繁 | 查 `cold_starts/invocations > 0.3` | 预留实例;减少实例回收 |
| FG-07 | P99 突增 | 下游劣化/资源争抢 | 查 P99 `> 3×` 基线 | 定位下游依赖;扩容 |
| FG-08 | 调用量骤降 | 触发器失效/上游断流 | 查 `invocations` 下降 `> 50%` | 检查触发器与上游链路 |

---

## 根因定位速查

- **失败率 > 5%** → 优先看日志与下游依赖(APIG/OBS/SMN 等),区分代码 bug 与依赖故障。
- **超时 + OOM 并存** → 多为内存不足导致 GC/交换,直接调大内存规格。
- **冷启动占比高** → 预留实例是最快缓解手段,而非改代码。
- **调用量骤降 > 50%** → 多为 OBS/APIG/Timer 等触发器断流,先查上游。

---

## 跨 skill 联动

| 故障场景 | 联动 Skill | 目的 |
|---|---|---|
| 共享 OBS 触发 | `huaweicloud-obs-ops` | 检查桶与事件通知 |
| APIG 触发异常 | `huaweicloud-elb-ops`(APIG 链路) | 检查网关健康 |
| SMN 触发堆积 | `huaweicloud-dms-ops` | 检查消息投递 |
| 定时器失效 | `huaweicloud-cts-ops` | 查审计与调用记录 |
| 数据库依赖 | `huaweicloud-rds-ops` | 查连接与慢查询 |
| 权限/AK 失效 | `huaweicloud-iam-ops` | 查委托与 AK/SK |
| 配额/费用 | `huaweicloud-billing-ops` | 查并发配额与计费 |

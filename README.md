# dcsy-server

## 数据点业务编码

前端、业务后端和独立解析程序统一使用以下英文数据点 ID：

| 中文含义 | 数据点 ID |
| --- | --- |
| 剩余放电时长 | `alg_remaining_discharge_time` |
| 热失控风险等级 | `alg_thermal_runaway_risk_level` |
| 温差 | `ditoo_temperature_difference` |
| 压差 | `ditoo_voltage_difference` |
| 累计放电量 | `tt_total_discharge_capacity` |
| 循环次数 | `cycle_count` |
| 累计充电量 | `tt_total_charge_capacity` |

前端常量集中在 `yewuCloud/src/utils/battery-point-ids.js`；业务后端和独立解析程序分别在 `dicloud/util/pointid`、`deal/util/pointid` 维护相同编码，保持独立构建。实际通道数据点继续使用 TTDC 地址前缀，例如 `02_ditoo_temperature_difference`、`02_cycle_count`。42 模板自动生成的循环次数 ID 与业务常量使用同一个 `cycle_count` 定义。

前端优先按英文 ID 及通道地址前缀识别这七项固定业务数据点。为保留现有数据库配置，累计充电量和累计放电量在未匹配到英文 ID 时，允许按当前设备已绑定标签的完整名称读取其真实数据点；实时卡片、历史选项及曲线沿用原标签 ID，不改写现有数据。后端大屏汇总和定时统计也对这两项采用相同的英文 ID 优先、名称回退规则，分别识别充电与放电，沿用已有网关及数据点绑定；通道名称允许两位十六进制地址前缀。同一指标不混用英文点和名称回退点，多个有效候选沿用最新读数选择规则。其余固定业务点的识别规则保持不变。四个固定算法点的 `algorithm_key` 为空或与其英文数据点 ID 一致，不能将其他数据点 ID 映射到这四个算法结果。中文显示名称和已有汇总数据中的统计代码保持原样。

代码更新不会自动改写已有数据库配置、Redis 或时序历史数据。上线前须同步核对模板、实际通道、网关数据点、数据池标签及告警规则中的 ID 值，具体步骤见 [已有数据点配置对齐说明](dicloud/scripts/align-battery-point-ids.md)。数据库表名、字段名及记录的数值主键保持原样。历史时序数据需要继续按新 ID 查询时，应另行安排迁移，不能仅通过更新配置获得连续历史。

部署时同步更新 `deal`、`dicloud` 中的 `business`、`tolink`、`statistic` 服务及完整前端产物；已有数据配置与新 ID 对齐后再恢复数据处理和统计任务。

## 数据库初始化顺序

首次部署或升级时，先以单实例方式启动 `dicloud` 的 `auth` 程序。只有认证程序会创建共享的缺失表和执行经过审核的增量迁移；`business`、`tolink`、`trigger`、`statistic`、`apiserver` 只读校验数据库结构。认证程序初始化成功后，再启动告警程序及其他服务。

迁移过程会自动创建并使用 `dcsy_schema_migration_v1` 锁表，串行执行后续结构迁移。已有业务表不会执行通用 `AutoMigrate`；缺字段或关键字段类型不兼容时会停止启动并提示人工审核，不会自动删除约束或改写业务数据。

## 季度强制修改密码

季度密码策略分别配置在 `dicloud/configs/auth.yml`、`business.yml` 和 `tolink.yml` 的 `security.password` 下，各服务只从自己的配置文件读取这部分规则，不使用环境变量覆盖。默认 `enabled: true`，按北京时间每年 1月1日、4月1日、7月1日、10月1日零点进入新季度；每季度须修改一次密码，最高权限账号同样执行。9月30日改密后，10月1日仍需再次修改。修改时应保证这三个文件的 `security.password` 内容一致，并重启对应服务。

升级时需同时部署 `auth`、`business`、`tolink` 的新程序，在各自的配置文件中补充 `security.password`，并部署完整前端 `yewuCloud/dist`。先启动 `auth`，它通过幂等增量迁移补充 `users.password_changed_at`、`users.must_change_password`，并在需要时扩容令牌列，保留原账号、密码密文、角色和业务数据；迁移成功后再启动其他服务。新增改密时间保存绝对时刻，按北京时间计算季度。

老账号没有改密记录，开启策略后首次登录必须改密；在线账号下一次请求也会受限。管理员创建或重置的密码为初始密码，用户首次使用时须本人修改。受限登录只允许本人状态查询、改密和退出，业务及采集管理接口均由后端拦截；原短信验证继续执行。改密成功后旧令牌失效，前端清理会话，须使用新密码重新登录。`enabled: false` 只关闭季度限制，管理员创建/重置后的首次改密要求仍保留。

设备接入、MQTT、解析、告警后台任务及独立机器接口不参与用户季度改密。平台通用配置导入/删除接口现需登录及 `commondata:import` 权限，权限字典自动补齐，普通用户由管理员明确授权。

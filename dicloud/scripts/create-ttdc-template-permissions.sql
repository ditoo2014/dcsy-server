-- TTDC public-template permissions for the collection platform (plat_id = 2).
-- This script only inserts permission configuration data and is safe to re-run.

INSERT INTO "permission" (permission_name, url, type, level, parent_id, plat_id)
SELECT 'TTDC公共模板', '', 1, 0, 0, 2
WHERE NOT EXISTS (
    SELECT 1 FROM "permission" WHERE permission_name = 'TTDC公共模板' AND plat_id = 2 AND type = 1
);

INSERT INTO "permission" (permission_name, url, type, level, parent_id, plat_id)
SELECT item.permission_name, item.url, 2, 1, parent.id, 2
FROM (
    SELECT MIN(id) AS id
    FROM "permission"
    WHERE permission_name = 'TTDC公共模板' AND plat_id = 2 AND type = 1
) parent
CROSS JOIN (
    VALUES
        ('查询模板组', 'ttdcgroup:findpage'),
        ('查询模板组详情', 'ttdcgroup:detail'),
        ('查询可用模板组', 'ttdcgroup:options'),
        ('新增模板组', 'ttdcgroup:add'),
        ('修改模板组', 'ttdcgroup:update'),
        ('修改模板组状态', 'ttdcgroup:state'),
        ('删除模板组', 'ttdcgroup:delete'),
        ('查询通道模板', 'ttdctemplate:findpage'),
        ('查询通道模板详情', 'ttdctemplate:detail'),
        ('查询可用通道模板', 'ttdctemplate:options'),
        ('查询模板解析配置', 'ttdctemplate:profiles'),
        ('查询算法数据点', 'ttdctemplate:algorithms'),
        ('新增通道模板', 'ttdctemplate:add'),
        ('修改通道模板', 'ttdctemplate:update'),
        ('删除通道模板', 'ttdctemplate:delete'),
        ('修改通道模板状态', 'ttdctemplate:enable'),
        ('导入通道模板', 'ttdctemplate:import'),
        ('导出通道模板', 'ttdctemplate:export'),
        ('查询告警规则', 'ttdcalarmrule:findpage'),
        ('查询告警规则详情', 'ttdcalarmrule:detail'),
        ('新增告警规则', 'ttdcalarmrule:add'),
        ('修改告警规则', 'ttdcalarmrule:update'),
        ('修改告警规则状态', 'ttdcalarmrule:state'),
        ('删除告警规则', 'ttdcalarmrule:delete')
) item(permission_name, url)
WHERE parent.id IS NOT NULL
  AND NOT EXISTS (
      SELECT 1 FROM "permission" existing
      WHERE existing.url = item.url AND existing.plat_id = 2
  );

-- Repair parent/type information when permissions were inserted manually before this script.
UPDATE "permission"
SET parent_id = (
        SELECT MIN(id) FROM "permission"
        WHERE permission_name = 'TTDC公共模板' AND plat_id = 2 AND type = 1
    ),
    type = 2,
    level = 1
WHERE plat_id = 2
  AND url IN (
      'ttdcgroup:findpage', 'ttdcgroup:detail', 'ttdcgroup:options',
      'ttdcgroup:add', 'ttdcgroup:update', 'ttdcgroup:state', 'ttdcgroup:delete',
      'ttdctemplate:findpage', 'ttdctemplate:detail', 'ttdctemplate:options',
      'ttdctemplate:profiles', 'ttdctemplate:algorithms', 'ttdctemplate:add',
      'ttdctemplate:update', 'ttdctemplate:delete', 'ttdctemplate:enable',
      'ttdctemplate:import', 'ttdctemplate:export',
      'ttdcalarmrule:findpage', 'ttdcalarmrule:detail', 'ttdcalarmrule:add',
      'ttdcalarmrule:update', 'ttdcalarmrule:state', 'ttdcalarmrule:delete'
  );

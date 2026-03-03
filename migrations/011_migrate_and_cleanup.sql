-- 迁移11：数据迁移和清理
-- 1. 为现有babies添加用户名字段
-- 2. 创建默认管理员账号
-- 3. 设置权限默认值

-- 1. 为users表添加姓名字段
ALTER TABLE users ADD COLUMN IF NOT EXISTS name VARCHAR(100);

-- 2. 为现有family_members创建初始权限记录
INSERT INTO user_permissions (user_id, baby_id, permission_type, permission_value)
SELECT m.user_id, m.baby_id, 'view_records', TRUE
FROM family_members m
WHERE NOT EXISTS (
    SELECT 1 FROM user_permissions up
    WHERE up.user_id = m.user_id AND up.baby_id = m.baby_id AND up.permission_type = 'view_records'
);

-- 为创建者添加管理员权限
INSERT INTO user_permissions (user_id, baby_id, permission_type, permission_value)
SELECT m.user_id, m.baby_id, 'manage_family', TRUE
FROM family_members m
WHERE NOT EXISTS (
    SELECT 1 FROM user_permissions up
    WHERE up.user_id = m.user_id AND up.baby_id = m.baby_id AND up.permission_type = 'manage_family'
);

-- 3. 创建默认管理员账号（需要先获取第一个创建者）
-- 注意：这需要手动执行或通过应用逻辑处理

-- 4. 为现有成员创建初始邀请码
INSERT INTO invitations (code, baby_id, created_by, status, max_uses)
SELECT
    md5(id::text || now()::text) as code,
    baby_id,
    user_id,
    'active',
    10
FROM family_members
WHERE NOT EXISTS (
    SELECT 1 FROM invitations i WHERE i.baby_id = family_members.baby_id
);

-- 5. 更新growth_records表添加user_id外键约束
-- 首先检查现有数据
DO $$
DECLARE
    r RECORD;
BEGIN
    -- 为所有growth_records设置user_id为该记录所属的家庭成员的user_id
    FOR r IN
        SELECT gr.id, gr.baby_id, fm.user_id
        FROM growth_records gr
        LEFT JOIN family_members fm ON gr.baby_id = fm.baby_id
        WHERE gr.user_id IS NULL OR fm.user_id IS NULL
    LOOP
        IF r.user_id IS NULL AND r.user_id IS NOT NULL THEN
            UPDATE growth_records
            SET user_id = r.user_id
            WHERE id = r.id;
        END IF;
    END LOOP;
END $$;

-- 添加外键约束
ALTER TABLE growth_records
ADD CONSTRAINT fk_growth_records_user_id
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- 6. 添加growing_up和is_pinned字段到growth_records
ALTER TABLE growth_records
ADD COLUMN IF NOT EXISTS growing_up BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS is_pinned BOOLEAN DEFAULT FALSE;

-- 7. 创建用于计算宝宝年龄的视图
CREATE OR REPLACE VIEW baby_age_view AS
SELECT
    b.id as baby_id,
    b.name as baby_name,
    b.birth_date,
    b.user_id,
    EXTRACT(DAY FROM AGE(NOW(), b.birth_date)) as days,
    EXTRACT(MONTH FROM AGE(NOW(), b.birth_date)) as months,
    EXTRACT(YEAR FROM AGE(NOW(), b.birth_date)) as years
FROM babies b;

-- 8. 创建用于统计家庭成员贡献的视图
CREATE OR REPLACE VIEW member_contributions_view AS
SELECT
    m.id as member_id,
    m.name as member_name,
    m.role,
    m.baby_id,
    COUNT(gr.id) as record_count,
    MAX(gr.created_at) as last_record_date
FROM family_members m
LEFT JOIN growth_records gr ON m.baby_id = gr.baby_id
    AND gr.user_id = m.user_id
GROUP BY m.id, m.name, m.role, m.baby_id;

-- 9. 创建里程碑检查函数
CREATE OR REPLACE FUNCTION check_milestones(baby_id_param INTEGER)
RETURNS TABLE (milestone_name VARCHAR(100), milestone_days INTEGER) AS $$
BEGIN
    RETURN QUERY
    SELECT mt.name, mt.days_offset
    FROM milestone_tags mt
    WHERE mt.days_offset > 0
    ORDER BY mt.days_offset;
END;
$$ LANGUAGE plpgsql;

-- 10. 创建定期清理过期邀请码的函数
CREATE OR REPLACE FUNCTION cleanup_expired_invitations()
RETURNS INTEGER AS $$
DECLARE
    expired_count INTEGER;
BEGIN
    UPDATE invitations
    SET status = 'expired'
    WHERE status = 'active'
    AND max_uses > 0
    AND used_count >= max_uses;

    GET DIAGNOSTICS expired_count = ROW_COUNT;

    RETURN expired_count;
END;
$$ LANGUAGE plpgsql;

-- 创建定期任务（PostgreSQL 13+）
-- 注意：实际生产环境应使用pg_cron扩展或外部调度工具
-- 这里只是创建一个手动调用的函数

-- 添加存储配额设置（未来扩展用）
CREATE TABLE IF NOT EXISTS storage_quotas (
    plan VARCHAR(50) PRIMARY KEY,
    max_members INTEGER DEFAULT 4,
    storage_limit_gb INTEGER DEFAULT 1,
    supports_multi_baby BOOLEAN DEFAULT FALSE,
    has_pdf_export BOOLEAN DEFAULT FALSE,
    price_monthly NUMERIC DEFAULT 0
);

INSERT INTO storage_quotas (plan, max_members, storage_limit_gb, supports_multi_baby, has_pdf_export, price_monthly)
VALUES
    ('free', 4, 1, FALSE, FALSE, 0),
    ('premium', 10, 5, TRUE, TRUE, 19.90),
    ('family', 20, 10, TRUE, TRUE, 49.90)
ON CONFLICT (plan) DO NOTHING;

SELECT 'Migration 11 completed!' as status;

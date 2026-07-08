-- fix_existing.sql
-- 安全补全已有数据库中缺失的列（幂等操作，使用 IF NOT EXISTS）
-- 适用于从旧迁移升级到 001_init.up.sql 完整 schema

-- ==================== users ====================
ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) DEFAULT '爸爸';
ALTER TABLE users ADD COLUMN IF NOT EXISTS name VARCHAR(100) DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_name ON users(name);

-- ==================== babies ====================
ALTER TABLE babies ADD COLUMN IF NOT EXISTS avatar_url VARCHAR(500) DEFAULT NULL;
CREATE INDEX IF NOT EXISTS idx_babies_avatar_url ON babies(avatar_url);

-- ==================== family_members ====================
ALTER TABLE family_members ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE family_members ADD CONSTRAINT IF NOT EXISTS uk_family_members_baby_role UNIQUE (baby_id, role);
CREATE INDEX IF NOT EXISTS idx_family_members_user_id ON family_members(user_id);

-- ==================== growth_records ====================
ALTER TABLE growth_records ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE growth_records ADD COLUMN IF NOT EXISTS media TEXT DEFAULT '[]';
ALTER TABLE growth_records ADD COLUMN IF NOT EXISTS growing_up BOOLEAN DEFAULT FALSE;
ALTER TABLE growth_records ADD COLUMN IF NOT EXISTS is_pinned BOOLEAN DEFAULT FALSE;
CREATE INDEX IF NOT EXISTS idx_growth_records_user_id ON growth_records(user_id);
CREATE INDEX IF NOT EXISTS idx_growth_records_media ON growth_records(media);

-- ==================== invitations (如果表存在但缺列) ====================
-- invitation 表如果存在可能是旧 schema 缺少 reason 列
ALTER TABLE invitations ADD COLUMN IF NOT EXISTS reason TEXT;

-- ==================== likes ====================
CREATE TABLE IF NOT EXISTS likes (
    id SERIAL PRIMARY KEY,
    growth_record_id INTEGER NOT NULL REFERENCES growth_records(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(growth_record_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_likes_growth_record_id ON likes(growth_record_id);
CREATE INDEX IF NOT EXISTS idx_likes_user_id ON likes(user_id);

-- ==================== comments ====================
CREATE TABLE IF NOT EXISTS comments (
    id SERIAL PRIMARY KEY,
    growth_record_id INTEGER NOT NULL REFERENCES growth_records(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id INTEGER REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_comments_growth_record_id ON comments(growth_record_id);
CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments(user_id);
CREATE INDEX IF NOT EXISTS idx_comments_parent_id ON comments(parent_id);

-- ==================== user_stats ====================
CREATE TABLE IF NOT EXISTS user_stats (
    id SERIAL PRIMARY KEY,
    user_id INTEGER UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_records INTEGER DEFAULT 0,
    total_likes INTEGER DEFAULT 0,
    total_comments INTEGER DEFAULT 0,
    storage_used_bytes BIGINT DEFAULT 0,
    baby_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_user_stats_user_id ON user_stats(user_id);

-- ==================== subscriptions ====================
CREATE TABLE IF NOT EXISTS subscriptions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan VARCHAR(50) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    start_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    end_date TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_active ON subscriptions(is_active);

-- ==================== storage_quotas ====================
CREATE TABLE IF NOT EXISTS storage_quotas (
    plan VARCHAR(50) PRIMARY KEY,
    max_members INTEGER DEFAULT 4,
    storage_limit_gb INTEGER DEFAULT 1,
    supports_multi_baby BOOLEAN DEFAULT FALSE,
    has_pdf_export BOOLEAN DEFAULT FALSE,
    price_monthly NUMERIC(10,2) DEFAULT 0
);

-- ==================== 触发器 ====================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 仅为已存在的表添加触发器
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'users') THEN
        CREATE TRIGGER IF NOT EXISTS update_users_updated_at BEFORE UPDATE ON users
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'babies') THEN
        CREATE TRIGGER IF NOT EXISTS update_babies_updated_at BEFORE UPDATE ON babies
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'family_members') THEN
        CREATE TRIGGER IF NOT EXISTS update_family_members_updated_at BEFORE UPDATE ON family_members
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'growth_records') THEN
        CREATE TRIGGER IF NOT EXISTS update_growth_records_updated_at BEFORE UPDATE ON growth_records
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'milestone_tags') THEN
        CREATE TRIGGER IF NOT EXISTS update_milestone_tags_updated_at BEFORE UPDATE ON milestone_tags
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'invitations') THEN
        CREATE TRIGGER IF NOT EXISTS update_invitations_updated_at BEFORE UPDATE ON invitations
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'membership_approvals') THEN
        CREATE TRIGGER IF NOT EXISTS update_approvals_updated_at BEFORE UPDATE ON membership_approvals
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'user_permissions') THEN
        CREATE TRIGGER IF NOT EXISTS update_permissions_updated_at BEFORE UPDATE ON user_permissions
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'comments') THEN
        CREATE TRIGGER IF NOT EXISTS update_comments_updated_at BEFORE UPDATE ON comments
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'user_stats') THEN
        CREATE TRIGGER IF NOT EXISTS update_user_stats_updated_at BEFORE UPDATE ON user_stats
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'subscriptions') THEN
        CREATE TRIGGER IF NOT EXISTS update_subscriptions_updated_at BEFORE UPDATE ON subscriptions
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;

-- 插入默认数据
INSERT INTO milestone_tags (name, days_offset) VALUES
    ('第一周', 7), ('第二周', 14), ('第三周', 21), ('满月', 30),
    ('六周', 42), ('八周', 56), ('百日', 100), ('百日快乐', 100),
    ('三个月', 90), ('半岁', 180), ('九个月', 270), ('一周岁', 365)
ON CONFLICT (name) DO NOTHING;

INSERT INTO storage_quotas (plan, max_members, storage_limit_gb, supports_multi_baby, has_pdf_export, price_monthly) VALUES
    ('free', 4, 1, FALSE, FALSE, 0),
    ('premium', 10, 5, TRUE, TRUE, 19.90),
    ('family', 20, 10, TRUE, TRUE, 49.90)
ON CONFLICT (plan) DO NOTHING;
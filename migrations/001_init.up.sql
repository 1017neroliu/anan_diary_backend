-- 001_init.up.sql
-- 宝宝时光轴完整数据库初始化脚本
-- 合并了所有历史迁移（001-013），统一所有列与约束
-- 适用于全新数据库

-- 1. users 用户表
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT '爸爸',
    name VARCHAR(100) DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. babies 宝宝档案表
CREATE TABLE IF NOT EXISTS babies (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    birth_date VARCHAR(10) NOT NULL,
    baby_id VARCHAR(50) UNIQUE NOT NULL,
    avatar_url VARCHAR(500) DEFAULT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, baby_id)
);

-- 3. family_members 家庭成员表
CREATE TABLE IF NOT EXISTS family_members (
    id SERIAL PRIMARY KEY,
    baby_id INTEGER NOT NULL REFERENCES babies(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    role VARCHAR(20) NOT NULL,
    is_paid BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(baby_id, role)
);

-- 4. growth_records 成长记录表
CREATE TABLE IF NOT EXISTS growth_records (
    id SERIAL PRIMARY KEY,
    baby_id INTEGER NOT NULL REFERENCES babies(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    tags TEXT DEFAULT '[]',
    media TEXT DEFAULT '[]',
    growing_up BOOLEAN DEFAULT FALSE,
    is_pinned BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL,
    UNIQUE(baby_id, created_at, content)
);

-- 5. milestone_tags 里程碑标签表
CREATE TABLE IF NOT EXISTS milestone_tags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    days_offset INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 6. invitations 邀请码表
CREATE TABLE IF NOT EXISTS invitations (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    baby_id INTEGER NOT NULL REFERENCES babies(id) ON DELETE CASCADE,
    created_by INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'expired', 'revoked')),
    max_uses INTEGER DEFAULT 10,
    used_count INTEGER DEFAULT 0,
    reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 7. invitation_uses 邀请码使用记录表
CREATE TABLE IF NOT EXISTS invitation_uses (
    id SERIAL PRIMARY KEY,
    invitation_id INTEGER NOT NULL REFERENCES invitations(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip_address INET,
    used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 8. membership_approvals 成员审批表
CREATE TABLE IF NOT EXISTS membership_approvals (
    id SERIAL PRIMARY KEY,
    baby_id INTEGER NOT NULL REFERENCES babies(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    reason TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
    reviewed_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMP,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 9. user_permissions 用户权限表
CREATE TABLE IF NOT EXISTS user_permissions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    baby_id INTEGER NOT NULL REFERENCES babies(id) ON DELETE CASCADE,
    permission_type VARCHAR(50) NOT NULL,
    permission_value BOOLEAN NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, baby_id, permission_type)
);

-- 10. likes 点赞表
CREATE TABLE IF NOT EXISTS likes (
    id SERIAL PRIMARY KEY,
    growth_record_id INTEGER NOT NULL REFERENCES growth_records(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(growth_record_id, user_id)
);

-- 11. comments 评论表
CREATE TABLE IF NOT EXISTS comments (
    id SERIAL PRIMARY KEY,
    growth_record_id INTEGER NOT NULL REFERENCES growth_records(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id INTEGER REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 12. user_stats 用户统计表
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

-- 13. subscriptions 会员订阅表
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

-- 14. storage_quotas 存储配额表
CREATE TABLE IF NOT EXISTS storage_quotas (
    plan VARCHAR(50) PRIMARY KEY,
    max_members INTEGER DEFAULT 4,
    storage_limit_gb INTEGER DEFAULT 1,
    supports_multi_baby BOOLEAN DEFAULT FALSE,
    has_pdf_export BOOLEAN DEFAULT FALSE,
    price_monthly NUMERIC(10,2) DEFAULT 0
);

-- ==================== 索引 ====================
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_name ON users(name);

CREATE INDEX IF NOT EXISTS idx_babies_user_id ON babies(user_id);
CREATE INDEX IF NOT EXISTS idx_babies_baby_id ON babies(baby_id);

CREATE INDEX IF NOT EXISTS idx_family_members_baby_id ON family_members(baby_id);
CREATE INDEX IF NOT EXISTS idx_family_members_user_id ON family_members(user_id);
CREATE INDEX IF NOT EXISTS idx_family_members_role ON family_members(role);

CREATE INDEX IF NOT EXISTS idx_growth_records_baby_id ON growth_records(baby_id);
CREATE INDEX IF NOT EXISTS idx_growth_records_user_id ON growth_records(user_id);
CREATE INDEX IF NOT EXISTS idx_growth_records_created_at ON growth_records(created_at);

CREATE INDEX IF NOT EXISTS idx_invitations_code ON invitations(code);
CREATE INDEX IF NOT EXISTS idx_invitations_baby_id ON invitations(baby_id);
CREATE INDEX IF NOT EXISTS idx_invitations_status ON invitations(status);

CREATE INDEX IF NOT EXISTS idx_invitation_uses_invitation_id ON invitation_uses(invitation_id);
CREATE INDEX IF NOT EXISTS idx_invitation_uses_user_id ON invitation_uses(user_id);

CREATE INDEX IF NOT EXISTS idx_approvals_baby_id ON membership_approvals(baby_id);
CREATE INDEX IF NOT EXISTS idx_approvals_user_id ON membership_approvals(user_id);
CREATE INDEX IF NOT EXISTS idx_approvals_status ON membership_approvals(status);

CREATE INDEX IF NOT EXISTS idx_permissions_user_id ON user_permissions(user_id);
CREATE INDEX IF NOT EXISTS idx_permissions_baby_id ON user_permissions(baby_id);
CREATE INDEX IF NOT EXISTS idx_permissions_type ON user_permissions(permission_type);

CREATE INDEX IF NOT EXISTS idx_likes_growth_record_id ON likes(growth_record_id);
CREATE INDEX IF NOT EXISTS idx_likes_user_id ON likes(user_id);

CREATE INDEX IF NOT EXISTS idx_comments_growth_record_id ON comments(growth_record_id);
CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments(user_id);
CREATE INDEX IF NOT EXISTS idx_comments_parent_id ON comments(parent_id);

CREATE INDEX IF NOT EXISTS idx_user_stats_user_id ON user_stats(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_active ON subscriptions(is_active);

-- ==================== 触发器（自动更新 updated_at） ====================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_babies_updated_at BEFORE UPDATE ON babies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_family_members_updated_at BEFORE UPDATE ON family_members
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_growth_records_updated_at BEFORE UPDATE ON growth_records
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_milestone_tags_updated_at BEFORE UPDATE ON milestone_tags
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_invitations_updated_at BEFORE UPDATE ON invitations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_approvals_updated_at BEFORE UPDATE ON membership_approvals
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_permissions_updated_at BEFORE UPDATE ON user_permissions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_comments_updated_at BEFORE UPDATE ON comments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_stats_updated_at BEFORE UPDATE ON user_stats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_subscriptions_updated_at BEFORE UPDATE ON subscriptions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ==================== 默认数据 ====================
INSERT INTO milestone_tags (name, days_offset) VALUES
    ('第一周', 7),
    ('第二周', 14),
    ('第三周', 21),
    ('满月', 30),
    ('六周', 42),
    ('八周', 56),
    ('百日', 100),
    ('百日快乐', 100),
    ('三个月', 90),
    ('半岁', 180),
    ('九个月', 270),
    ('一周岁', 365)
ON CONFLICT (name) DO NOTHING;

INSERT INTO storage_quotas (plan, max_members, storage_limit_gb, supports_multi_baby, has_pdf_export, price_monthly) VALUES
    ('free', 4, 1, FALSE, FALSE, 0),
    ('premium', 10, 5, TRUE, TRUE, 19.90),
    ('family', 20, 10, TRUE, TRUE, 49.90)
ON CONFLICT (plan) DO NOTHING;
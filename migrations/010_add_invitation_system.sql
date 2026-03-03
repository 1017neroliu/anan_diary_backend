-- 迁移10: 添加邀请码系统
-- 实现方案：宝宝创建者生成邀请码，其他家庭成员通过邀请码申请加入

-- 1. 创建邀请码表
CREATE TABLE IF NOT EXISTS invitations (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,               -- 邀请码（永久有效，可重置）
    baby_id INTEGER NOT NULL REFERENCES babies(id) ON DELETE CASCADE,
    created_by INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'expired', 'revoked')),
    max_uses INTEGER DEFAULT 10,                    -- 最大使用次数（用于一次性邀请码）
    used_count INTEGER DEFAULT 0,                   -- 已使用次数
    description TEXT,                               -- 邀请说明（可选）
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- 关联关系
    creator User ON DELETE SET NULL,

    -- 约束：同一宝宝的邀请码只能有一个活跃状态
    UNIQUE(baby_id, status) WHERE status = 'active'
);

-- 创建邀请码使用记录表
CREATE TABLE IF NOT EXISTS invitation_uses (
    id SERIAL PRIMARY KEY,
    invitation_id INTEGER NOT NULL REFERENCES invitations(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip_address INET,                                -- IP地址（可选，用于安全审计）
    used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. 创建审批系统表
CREATE TABLE IF NOT EXISTS membership_approvals (
    id SERIAL PRIMARY KEY,
    baby_id INTEGER NOT NULL REFERENCES babies(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,                      -- 申请的角色（爸爸/妈妈等）
    name VARCHAR(100) NOT NULL,                     -- 申请人姓名
    reason TEXT,                                    -- 申请理由（可选）
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
    reviewed_by INTEGER REFERENCES users(id) ON DELETE SET NULL, -- 审批人ID
    reviewed_at TIMESTAMP,                          -- 审批时间
    notes TEXT,                                     -- 审批备注
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- 约束：同一用户对同一宝宝的同一角色只能有一个待审批的申请
    UNIQUE(baby_id, user_id, role, status) WHERE status = 'pending'
);

-- 3. 创建用户权限表
CREATE TABLE IF NOT EXISTS user_permissions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    baby_id INTEGER NOT NULL REFERENCES babies(id) ON DELETE CASCADE,
    permission_type VARCHAR(50) NOT NULL,          -- 权限类型
    permission_value BOOLEAN NOT NULL,             -- 权限值（true/false）

    -- 唯一约束：同一用户对同一宝宝同一权限只能有一条记录
    UNIQUE(user_id, baby_id, permission_type),

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 4. 创建点赞表
CREATE TABLE IF NOT EXISTS likes (
    id SERIAL PRIMARY KEY,
    growth_record_id INTEGER NOT NULL REFERENCES growth_records(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- 唯一约束：同一用户对同一条记录只能点赞一次
    UNIQUE(growth_record_id, user_id),

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 5. 创建评论表
CREATE TABLE IF NOT EXISTS comments (
    id SERIAL PRIMARY KEY,
    growth_record_id INTEGER NOT NULL REFERENCES growth_records(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id INTEGER REFERENCES comments(id) ON DELETE CASCADE, -- 父评论ID（支持嵌套评论）
    content TEXT NOT NULL,

    -- 约束：用户对同一条记录只能回复一次作为顶级评论
    UNIQUE(growth_record_id, user_id) WHERE parent_id IS NULL,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- 关联关系
    commenter User ON DELETE SET NULL,
    parent Comment ON DELETE SET NULL
);

-- 6. 创建用户统计表
CREATE TABLE IF NOT EXISTS user_stats (
    id SERIAL PRIMARY KEY,
    user_id INTEGER UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_records INTEGER DEFAULT 0,                -- 总记录数
    total_likes INTEGER DEFAULT 0,                 -- 总点赞数
    total_comments INTEGER DEFAULT 0,               -- 总评论数
    storage_used_bytes BIGINT DEFAULT 0,           -- 已使用存储空间（字节）
    baby_count INTEGER DEFAULT 0,                  -- 绑定的宝宝数
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 7. 创建会员订阅表
CREATE TABLE IF NOT EXISTS subscriptions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan VARCHAR(50) NOT NULL,                      -- plan: free, premium, family
    is_active BOOLEAN DEFAULT FALSE,
    starts_at TIMESTAMP NOT NULL,
    ends_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- 唯一约束：同一用户同一时间只能有一条活跃订阅
    UNIQUE(user_id, is_active) WHERE is_active = TRUE
);

-- 创建索引
CREATE INDEX idx_invitations_code ON invitations(code);
CREATE INDEX idx_invitations_baby_id ON invitations(baby_id);
CREATE INDEX idx_invitations_status ON invitations(status);
CREATE INDEX idx_invitation_uses_invitation_id ON invitation_uses(invitation_id);
CREATE INDEX idx_invitation_uses_user_id ON invitation_uses(user_id);

CREATE INDEX idx_approvals_baby_id ON membership_approvals(baby_id);
CREATE INDEX idx_approvals_user_id ON membership_approvals(user_id);
CREATE INDEX idx_approvals_status ON membership_approvals(status);

CREATE INDEX idx_permissions_user_id ON user_permissions(user_id);
CREATE INDEX idx_permissions_baby_id ON user_permissions(baby_id);

CREATE INDEX idx_likes_growth_record_id ON likes(growth_record_id);
CREATE INDEX idx_likes_user_id ON likes(user_id);

CREATE INDEX idx_comments_growth_record_id ON comments(growth_record_id);
CREATE INDEX idx_comments_user_id ON comments(user_id);
CREATE INDEX idx_comments_parent_id ON comments(parent_id);

CREATE INDEX idx_user_stats_user_id ON user_stats(user_id);
CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(is_active);

-- 创建触发器
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

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

SELECT 'Invitation system tables created!' as status;

-- 宝宝时光轴数据库初始化脚本
-- PostgreSQL 14+

-- 创建数据库（如果不存在）
CREATE DATABASE anan_diary;

-- 使用anan_diary数据库
\c anan_diary;

-- 创建users表
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建babies表
CREATE TABLE babies (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    birth_date VARCHAR(10) NOT NULL,
    baby_id VARCHAR(50) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, baby_id)
);

-- 创建family_members表
CREATE TABLE family_members (
    id SERIAL PRIMARY KEY,
    baby_id INTEGER NOT NULL REFERENCES babies(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    role VARCHAR(20) NOT NULL,
    is_paid BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建growth_records表
CREATE TABLE growth_records (
    id SERIAL PRIMARY KEY,
    baby_id INTEGER NOT NULL REFERENCES babies(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    tags TEXT DEFAULT '[]',
    created_at TIMESTAMP NOT NULL,
    UNIQUE(baby_id, created_at, content)
);

-- 创建milestone_tags表
CREATE TABLE milestone_tags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    days_offset INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_babies_user_id ON babies(user_id);
CREATE INDEX idx_babies_baby_id ON babies(baby_id);
CREATE INDEX idx_family_members_baby_id ON family_members(baby_id);
CREATE INDEX idx_family_members_role ON family_members(role);
CREATE INDEX idx_growth_records_baby_id ON growth_records(baby_id);
CREATE INDEX idx_growth_records_created_at ON growth_records(created_at);
CREATE INDEX idx_growth_records_unique ON growth_records(baby_id, created_at, content);

-- 插入默认里程碑标签
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
    ('一周岁', 365);

-- 创建更新时间戳的函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 创建触发器
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_babies_updated_at BEFORE UPDATE ON babies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_family_members_updated_at BEFORE UPDATE ON family_members
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_milestone_tags_updated_at BEFORE UPDATE ON milestone_tags
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 完成提示
SELECT 'Database initialization completed!' as status;

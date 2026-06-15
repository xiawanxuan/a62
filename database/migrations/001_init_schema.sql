-- 声呐图标注系统数据库初始化脚本
-- PostgreSQL 12+

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 声呐文件表
CREATE TABLE IF NOT EXISTS sonar_files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    path VARCHAR(512) NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    size BIGINT NOT NULL,
    annotation_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sonar_files_created_at ON sonar_files(created_at DESC);

-- 目标分类表
CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    color VARCHAR(20) NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_categories_name ON categories(name);

-- 标注表
CREATE TABLE IF NOT EXISTS annotations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID NOT NULL REFERENCES sonar_files(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('rectangle', 'polygon')),
    points JSONB NOT NULL,
    category_id UUID NOT NULL REFERENCES categories(id),
    label VARCHAR(100),
    color VARCHAR(20),
    created_by VARCHAR(100) NOT NULL,
    confidence DOUBLE PRECISION,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_annotations_file_id ON annotations(file_id);
CREATE INDEX IF NOT EXISTS idx_annotations_category_id ON annotations(category_id);
CREATE INDEX IF NOT EXISTS idx_annotations_created_at ON annotations(created_at);

-- 快照表
CREATE TABLE IF NOT EXISTS snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID NOT NULL REFERENCES sonar_files(id) ON DELETE CASCADE,
    annotations JSONB NOT NULL,
    created_by VARCHAR(100) NOT NULL,
    message VARCHAR(255),
    version INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_snapshots_file_id ON snapshots(file_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_snapshots_version ON snapshots(file_id, version DESC);

-- 初始化默认分类
INSERT INTO categories (name, color, description) VALUES
    ('礁石', '#ff4d4f', '水下礁石'),
    ('沉船', '#faad14', '沉船残骸'),
    ('管线', '#1890ff', '海底管线'),
    ('锚', '#722ed1', '船锚'),
    ('渔网', '#13c2c2', '废弃渔网'),
    ('其他', '#8c8c8c', '其他目标')
ON CONFLICT (name) DO NOTHING;

-- 更新时间戳触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_sonar_files_updated_at ON sonar_files;
CREATE TRIGGER update_sonar_files_updated_at
    BEFORE UPDATE ON sonar_files
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_annotations_updated_at ON annotations;
CREATE TRIGGER update_annotations_updated_at
    BEFORE UPDATE ON annotations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

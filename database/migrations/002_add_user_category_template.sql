-- 声呐图标注系统 - 用户自定义分类模板迁移脚本
-- PostgreSQL 12+

-- 1. 给 categories 表增加 user_id 字段
--    user_id 为 NULL 表示系统全局默认分类
--    user_id 有值表示特定用户自定义分类
ALTER TABLE categories
    ADD COLUMN IF NOT EXISTS user_id VARCHAR(100);

-- 2. 取消原有的唯一约束 (name)，改为 (user_id, name) 复合唯一
--    系统默认分类 user_id=NULL 下的 name 仍需唯一；用户自定义分类在该用户下唯一
ALTER TABLE categories
    DROP CONSTRAINT IF EXISTS categories_name_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_user_name
    ON categories(user_id, name)
    WHERE user_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_global_name
    ON categories(name)
    WHERE user_id IS NULL;

-- 3. 索引加速按用户查询
CREATE INDEX IF NOT EXISTS idx_categories_user_id ON categories(user_id);

-- 4. 增加 updated_at 字段（如果不存在），记录分类修改时间
ALTER TABLE categories
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- 5. 给 categories 增加 is_builtin 标记，标识是否内置不可删除
ALTER TABLE categories
    ADD COLUMN IF NOT EXISTS is_builtin BOOLEAN NOT NULL DEFAULT FALSE;

-- 6. 将现有默认分类标记为内置 + 全局 (user_id=NULL, is_builtin=TRUE)
UPDATE categories
SET is_builtin = TRUE, user_id = NULL
WHERE user_id IS NULL AND name IN ('礁石', '沉船', '管线', '锚', '渔网', '其他');

-- 7. 更新 updated_at 触发器（如果 categories 没有）
DROP TRIGGER IF EXISTS update_categories_updated_at ON categories;
CREATE TRIGGER update_categories_updated_at
    BEFORE UPDATE ON categories
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 8. 删除分类时处理外键：改为 SET NULL，保留标注数据（标注的颜色/label 已冗余存）
--    先看原约束
ALTER TABLE annotations
    DROP CONSTRAINT IF EXISTS annotations_category_id_fkey;

ALTER TABLE annotations
    ALTER COLUMN category_id DROP NOT NULL;

ALTER TABLE annotations
    ADD CONSTRAINT annotations_category_id_fkey
    FOREIGN KEY (category_id) REFERENCES categories(id)
    ON DELETE SET NULL;

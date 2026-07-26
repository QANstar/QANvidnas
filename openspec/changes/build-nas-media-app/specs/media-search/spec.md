# media-search

媒体搜索与筛选。支持按标签筛选、全文搜索文本信息。

## ADDED Requirements

### Requirement: 标签筛选

用户 SHALL 能按标签筛选媒体列表。选择多个标签时为"与"逻辑（同时拥有所有选中标签的媒体）。

#### Scenario: 单标签筛选

- **WHEN** 用户选择一个标签进行筛选
- **THEN** 系统返回所有包含该标签的媒体

#### Scenario: 多标签筛选

- **WHEN** 用户同时选择多个标签
- **THEN** 系统返回同时包含所有选中标签的媒体（AND 逻辑）

#### Scenario: 清除标签筛选

- **WHEN** 用户取消所有标签选择
- **THEN** 系统返回未筛选的完整媒体列表

### Requirement: 全文搜索

系统 SHALL 支持通过 EF Core LINQ Contains 对媒体的名称、描述、标签名称进行全文搜索。搜索结果按创建时间排序。

#### Scenario: 搜索匹配名称

- **WHEN** 用户搜索关键词且该词出现在某媒体的名称中
- **THEN** 系统返回该媒体，匹配的搜索词高亮显示

#### Scenario: 搜索匹配标签

- **WHEN** 用户搜索关键词且该词是某媒体的标签名称
- **THEN** 系统返回该媒体

#### Scenario: 搜索匹配描述

- **WHEN** 用户搜索关键词且该词出现在某媒体的描述中
- **THEN** 系统返回该媒体

#### Scenario: 无匹配结果

- **WHEN** 搜索关键词未匹配任何内容
- **THEN** 系统返回空列表，前端显示"未找到相关内容"

#### Scenario: 搜索支持中文分词

- **WHEN** 用户输入中文关键词
- **THEN** 系统通过 Contains 模糊匹配正确匹配中文内容

### Requirement: 组合筛选与搜索

标签筛选和全文搜索 SHALL 可以组合使用：搜索结果取两者交集。

#### Scenario: 标签 + 关键词组合

- **WHEN** 用户同时设置标签筛选和关键词搜索
- **THEN** 系统返回同时满足两个条件的媒体

### Requirement: 按媒体类型筛选

用户 SHALL 能按媒体类型（视频/音频/全部）筛选浏览。

#### Scenario: 只看视频

- **WHEN** 用户选择"视频"类型筛选
- **THEN** 系统仅返回 type=video 的媒体

#### Scenario: 只看音频

- **WHEN** 用户选择"音频"类型筛选
- **THEN** 系统仅返回 type=audio 的媒体

# WhatsNext 产品设计文档

版本：v0.1  
状态：产品概念确定，进入 MVP 设计阶段

---

# 1. 产品概述

## 1.1 产品名称

WhatsNext

## 1.2 一句话介绍

WhatsNext 是一个 Personal Learning Agent，帮助用户管理从当前状态到目标状态的学习路径，并根据学习反馈动态调整下一步行动。

核心问题：

> 用户不是不知道有什么知识，而是不知道「下一步应该做什么」。

---

# 2. 产品背景

当前学习场景存在两个核心问题：

## 2.1 短期学习问题（考试冲刺）

例如大学期末：

- 教材、PPT、笔记、真题资料过多
- 不知道哪些内容重要
- 不知道自己的薄弱点
- 不知道有限时间如何安排

用户需要：

> 在有限时间内最大化考试收益。

---

## 2.2 长期学习问题（能力成长）

例如：

- 学习计算机
- 成为后端工程师
- 学习新的技术方向

问题：

- 没有明确教材
- 没有固定考试时间
- 不知道学习路径
- 学了很多但不知道是否成长

用户需要：

> 根据长期目标持续规划能力成长路线。

---

# 3. 产品核心理念

WhatsNext 不只是：

- AI问答工具
- PDF总结工具
- 学习计划生成器

而是：

> 一个维护用户学习状态，并持续规划下一步行动的 Agent 系统。

核心闭环：
```

目标 Goal  
|  
↓  
学习模型 Learning Model  
|  
↓  
Planner Agent  
|  
↓  
学习行动  
|  
↓  
用户反馈  
|  
↓  
状态更新  
|  
↓  
Replanning

```

---

# 4. 产品最终形态

WhatsNext 支持两种学习模式。

## 4.1 Exam Mode（考试模式）

目标：

> 短时间内提高考试表现。

特点：

- 有明确时间
- 有固定资料
- 有考试重点
- 有题目反馈


输入：

- 教材
- PPT
- 真题
- 考试日期


输出：

- 精简知识手册
- 题型总结
- 知识地图
- 动态复习路径


---

## 4.2 Growth Mode（长期成长模式）

目标：

> 长期提升个人能力。

例如：

成为后端工程师。


输入：

- 目标方向
- 当前能力
- 已有经验
- 兴趣


输出：

- 能力成长路线
- 阶段目标
- 学习节点
- 项目实践建议


---

# 5. 核心抽象设计

## 5.1 Learning Space

学习空间。

类似 Plane 中的 Workspace。

一个学习空间代表一个学习目标。

例如：
```

空间1：

计算机网络期末复习

模式：  
Exam

空间2：

后端工程师成长路线

模式：  
Growth

```

数据：
```

LearningSpace

id

user_id

name

mode

goal

```

---

# 5.2 Learning Node（核心对象）

Learning Node 是系统最重要的抽象。

定义：

> 一个需要被理解、掌握、实践或完成的学习对象。

它不是简单知识点。


类型：
```

Knowledge  
知识

Skill  
能力

Practice  
练习

Milestone  
阶段目标

Project  
项目实践

```


例：

考试：
```

TCP可靠传输

type:  
Knowledge

```


长期：
```

完成Reactor服务器

type:  
Project

```


---

# 5.3 Learning Graph

Learning Node 之间存在关系。

例如：

短期：
```

运输层

↓

TCP

↓

可靠传输

↓

TCP题型训练

```


长期：
```

Linux网络

↓

Socket编程

↓

Reactor服务器

```


关系类型：
```

prerequisite  
前置关系

related  
关联关系

contains  
包含关系

```

---

# 6. AI生成课程资料（Exam Mode）

上传：

- 教材
- PPT
- 真题
- 笔记


生成三个核心资产。

---

## 6.1 知识手册 Knowledge Book

目标：

替代教材阅读。


特点：

- 去除废话
- 去除课堂讨论
- 去除习题
- 保留核心知识


例如：
```

TCP可靠传输

定义：

TCP通过序号、确认、重传机制保证可靠传输。

核心机制：

1. 序号机制
2. 确认机制
3. 超时重传

```

---

## 6.2 题型手册 Question Pattern Book

目标：

回答：

> 考试怎么考？


例如：
```

TCP序号确认号计算

考试频率：  
高

考察：  
TCP可靠传输机制

常见错误：  
混淆seq和ack

```

---

## 6.3 知识地图 Knowledge Map

目标：

展示知识结构。


例如：
```

计算机网络

├── 网络层

├── 运输层

│ ├── TCP

│ └── UDP

└── 应用层

```

---

# 7. Planner Agent设计

## 7.1 Planner定位

Planner 不是：

> 生成一段学习计划文本。

而是：

> 管理 Learning Workflow，并根据反馈调整学习路径。

---

# 7.2 Planner输入

## Goal

用户目标。

例如：
```

计算机网络期末80分

剩余10天

```


## Learning Graph

知识/能力结构。


## User State

用户状态：

例如：
```

TCP可靠传输

掌握度:  
40%

错误次数:  
5

```


## History

历史学习情况：
```

昨天任务完成60%

TCP仍困难

```

---

# 7.3 Planner输出

不是 Todo。

而是：

Learning Plan Graph。


例如：
```

目标：

掌握运输层

节点：

TCP基础

↓

TCP可靠传输

↓

TCP题型训练

```

---

# 8. Agent Tool设计

Planner Agent拥有工具：

## 查询知识结构
```

query_learning_graph()

```


## 查询用户状态
```

query_user_state()

```


## 计算优先级
```

calculate_priority()

```


## 修改计划
```

update_plan_node()

```


## 创建新节点
```

create_learning_node()

```

---

# 9. Feedback机制

用户反馈不会直接修改资料。

资料保持稳定。


变化的是：

User State。


例如：

原：
```

TCP可靠传输

掌握度:  
50%

```


反馈：
```

滑动窗口仍不会

```


更新：
```

TCP可靠传输

掌握度:  
30%

状态:  
need_review

```


Planner 根据状态重新调整。

---

# 10. 短期和长期模式区别

底层数据模型统一。

区别：

Planner Strategy。


---

## Exam Planner

优化：

考试收益。


计算：
```

priority =  
考试权重  
×  
薄弱程度  
×  
剩余时间

```


---

## Growth Planner

优化：

能力成长。


计算：
```

priority =  
目标相关性  
×  
能力缺口  
×  
成长价值

```

---

# 11. 最终系统架构
```

```
             Frontend

                |

           Backend

                |

        Learning Agent

                |

         Planner Agent

                |

   -------------------------

   |           |           |
```

Knowledge State Graph

Tool Tool Tool

```
                |

         Learning Plan

                |

          User Feedback

                |

         Memory Update
```

```

---

# 12. MVP范围

目标：

完成一个可展示的 Agent 闭环。

选择：

## Exam Mode


原因：

- 用户痛点明确
- 数据容易获取
- 容易验证效果


---

# MVP用户流程

## Step 1

创建学习空间
```

计算机网络期末复习

考试日期:  
10天后

```


---

## Step 2

上传资料
```

教材  
PPT  
真题  
笔记

```


---

## Step 3

生成课程模型

输出：
```

知识手册

题型总结

知识地图

```


---

## Step 4

Planner生成Learning Plan


例如：
```

阶段：

运输层掌握

节点：

TCP基础

TCP可靠传输

TCP题型

```

---

## Step 5

用户学习并反馈


反馈：
```

TCP滑动窗口不会

```


---

## Step 6

Replanner调整


例如：

新增：
```

滑动窗口专项学习

```


调整：
```

HTTP学习延期

```

---

# 13. MVP暂不实现

以下功能暂缓：

- 多Agent通信
- 自动寻找学习资源
- 自动判断粗心
- 复杂技能图谱
- 自动执行学习任务
- 完整长期成长模式
- 复杂推荐算法

---

# 14. 当前待解决问题

## 14.1 Learning Node详细设计

需要确定：

- Node状态有哪些？
- Node如何评价完成？
- Knowledge、Skill、Project是否需要不同字段？

---

## 14.2 Planner内部结构

需要确定：

是否拆分：
```

Planner

Evaluator

Replanner

```

还是：
```

一个Planner Agent  
+  
多个Tool

```

---

## 14.3 Feedback设计

需要确定：

用户反馈方式：

- 手动评价？
- 做题结果？
- 项目完成情况？
- 自我描述？


---

## 14.4 MVP技术方案

需要确定：

- LangChain是否使用
- Go Backend + Python AI Service是否采用
- Vector DB选择
- 数据库设计


---

# 总结

WhatsNext 的核心不是生成学习资料，而是：

> 建立一个关于“用户、目标、知识、行动”的动态模型，并通过 Planner Agent 持续回答：What’s next?

MVP首先验证：
```

资料理解  
+  
学习状态  
+  
Planner  
+  
反馈重规划

是否能帮助用户更有效学习。

这份文档可以作为后续 PRD、架构设计、开发拆分的基础版本。下一步建议不要继续扩功能，而是基于这份文档确定 **MVP 的数据库 Schema 和 Agent Pipeline**。
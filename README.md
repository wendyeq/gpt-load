# PR 对照图：①分组详情（手机/窄桌面适配）

所有图均「左 = 基线 addd6987，右 = 修复（本 PR，含第三轮改动）」，同脚本、同视口、同 mock 数据、同 DPR（手机 2 / 桌面 1）、同等待时间；顶部标注基线/修复。宽度均 ≤1600px，单张 <1MB（PNG）。由 `/workspace/gpt-load-adapt/toolkit/run-page.sh group-detail ...` 生成。

## 手机整页长图（应用里是 `main.modern-content` 在滚；运行时展开滚动容器后截图，视口大小不变）
| 文件 | 含义 |
|---|---|
| `mobile-long-390x664-sub25.png` | 390x664，/groups/2（订阅账号 25 条）。基线：列表被压成极小内滚窗（43px），整页只能看到头部、工具栏，卡片几乎不可见；修复：列表随整页展开，账号卡片完整出现。 |
| `mobile-long-390x664-longname.png` | 390x664，/groups/5（超长分组名）。基线：h1 占 7 行、头部 480px；修复：h1 限 2 行、头部 272px，卡片可见。 |
| `mobile-long-360x640-sub25.png` | 360x640，/groups/2。同上（小屏）。 |
| `mobile-long-360x640-longname.png` | 360x640，/groups/5。同上（基线 h1 8 行、头部 518px；修复 2 行、272px）。 |
注：修复版整页高 8120/3445/8448/3445 css px，**长图截取前 3000px 并在图底注明**；基线整页高仅 1470–1494px（因为列表只是个小窗），所以右侧比左侧长。

## 桌面首屏对照（`desktop-*`）
| 文件 | 含义 |
|---|---|
| `desktop-1024x768-sub25-light-first.png` / `-longname-` | 1024x768（容器<980 走堆叠布局）：基线列表是 327px 小窗内滚，修复整页滚动，列表不再被固定窗口限制。 |
| `desktop-1280x720-sub25-light-first.png` / `-longname-` | 1280x720：列表窗略增高（258→306 / 197→274）。注意 /groups/2 此视口窗内仍无法放下整行订阅卡（需窗高≥322）。 |
| `desktop-1440x900-sub25-light-first.png` / `-longname-` | 1440x900：布局基本不变；超长名 h1 变单行省略（悬停有 tooltip 显示全名）。 |

## 深色
| 文件 | 含义 |
|---|---|
| `dark-390x664-sub25-dark-first.png` | 深色 390x664 /groups/2 首屏。 |
| `dark-1440x900-sub25-dark-first.png` | 深色 1440x900 /groups/2 首屏。 |

## M08 额度窗口表（凭据详情，390x844，/groups/2?credential=2001）
| 文件 | 含义 |
|---|---|
| `detail-elem-390x844-quota.png` | 初始状态：基线 6 列挤在 347px 内（每列 34–66px）；修复列宽 120/77/72/72/72/94px，表可横滚（前 4 列同屏，后 2 列需横滑）。 |
| `detail-elem-390x844-quota-scrolled-right.png` | 横向滚到最右：修复版首列（窗口名）保持 sticky 贴左（left≈0），后几列可见。基线无横滚。 |

## 数据口径（DOM 度量，非肉眼）
见 `/workspace/gpt-load-adapt/shots/p1-evidence-r2/REPORT.md`。关键点：手机 5 个视口滚动对象由「layout/列表小窗」变为整页 `main.modern-content`；390x664、360x640 首屏仍没有完整卡片（需上滑一次）；1280x720 /groups/2 窗内无完整卡。

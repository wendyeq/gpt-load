# pr-assets — images for the PR description (left = baseline addd6987, right = fix **40d25bd8** for all phone images; desktop images were made at 9134a10d and reused — the later commit only adds `overscroll-behavior-y:auto` inside ≤700px/≤760px rules, desktop pixels re-verified at 40d25bd8; mock data; Chromium headless, phone = touch + DPR2)

All files are PNG, each ≤ 1.3 MB. Phone images re-shot at 40d25bd8.

| file | what it shows |
|---|---|
| m04-mobile-long-390x664-before-after.png | M04 `/groups/2` phone full-length image (390x664 viewport, scroll containers expanded at runtime; **fix side capped at the first 4000 css px of ~8110**). Baseline: header + a 43px list window; fix: single page scroll, 20 full-height cards. |
| m05-mobile-long-390x664-before-after.png | M05 `/groups/5` phone full-length image. Baseline: 99-char name wraps to 7 lines at 24px, header 479.5px, list window 0px. Fix: 4 lines at 14px, header 269.1px, cards visible. |
| m26-mobile-long-390x664-before-after.png | M26 `/monitor/health` phone full-length image. Baseline: list window 35px, 0 data rows. Fix: ≥2 full data rows on the first screen. |
| m26-mobile-sticky-first-column-scrolled-right-390x664-before-after.png | M26 after real horizontal scroll to the far right (scrollLeft = max): baseline name column scrolled out of view; fix keeps the 140px name column pinned and shows the action column (clickable). |
| m04-mobile-after-one-swipe-390x664-before-after.png | M04 after one 300px upward touch swipe **that starts above the list** (filter area): fix shows one full card. (At 40d25bd8 a swipe that starts on the cards scrolls the page too.) |
| m05-mobile-after-one-swipe-390x664-before-after.png | M05 same, 2 full cards after one swipe from above the list. |
| m04-desktop-1280x720-first-screen-before-after.png | M04 desktop 1280x720 first screen — identical layout; only the right-side sparkline differs (mock random data). |
| m05-desktop-1280x720-first-screen-before-after.png | M05 desktop 1280x720 first screen — identical layout. |
| m26-desktop-1280x720-first-screen-before-after.png | M26 desktop 1280x720 first screen — identical (0 px pixel difference). |
| m04-dark-mobile-390x664-first-screen-before-after.png | M04 dark theme, phone first screen. |
| m05-dark-desktop-1440x900-first-screen-before-after.png | M05 dark theme, desktop 1440x900 first screen. |
| m26-dark-mobile-390x664-first-screen-before-after.png | M26 dark theme, phone first screen. |
| m04-mobile-reach-save-by-real-swipes-390x664-before-after.png | M04 after 14 real touch swipes that start on the cards (fix): the 保存修改 / 删除分组 bar reachable; baseline needs 2 swipes. |

Numbers behind every image: `../MANIFEST.md`, `../metrics/key-numbers.md`.

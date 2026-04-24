# Ví dụ Output — Haravan MCP Skill

Mỗi ví dụ dưới đây cho thấy: (1) user hỏi gì, (2) gọi tools gì, (3) output mẫu.

---

## Ví dụ 1: Store Pulse — Tổng quan cửa hàng

**User**: "Tình hình cửa hàng tuần này"

**Tools gọi (3 SONG SONG)**:
```
hrv_orders_summary(date_from="2026-03-31", date_to="2026-04-06", compare_prior=true)
hrv_customer_segments()
hrv_inventory_health()
```

**Output**:

```markdown
## 📊 Store Pulse — 31/03 → 06/04/2026

### Doanh thu & Đơn hàng
- **Tổng doanh thu**: 187,350,000 VND (↑12.3% vs tuần trước)
- **Số đơn hàng**: 423 (↑8.1%)
- **AOV**: 442,900 VND (↑3.9%)
- **Tỷ lệ đơn hủy**: 3.8% ⚠️ (tuần trước: 2.9%)

### Phân bổ kênh bán (từ orders_by_source)
| Kênh | Đơn | % tổng |
|------|-----|--------|
| Web | 278 | 65.7% |
| POS | 98 | 23.2% |
| Mobile | 47 | 11.1% |

### Khách hàng (RFM)
| Segment | Số lượng | % | Doanh thu |
|---------|----------|---|-----------|
| 🏆 Champions | 45 | 4.9% | 89,200,000 VND |
| ⚠️ At Risk | 128 | 13.9% | 34,500,000 VND |
| 💀 Lost | 312 | 33.9% | 0 VND |

### Tồn kho
| Trạng thái | SKU | % catalog |
|------------|-----|-----------|
| ✅ Healthy | 245 | 72% |
| ⚠️ Sắp hết (<5 units) | 34 | 10% |
| 🔴 Hết hàng | 18 | 5% |
| 💀 Dead stock | 45 | 13% |

Dead stock value: **23,450,000 VND**

### 🎯 3 điều cần chú ý
1. **Cancel rate tăng 2.9% → 3.8%**: 6/16 đơn hủy lý do "inventory" (hết hàng). 3 SKU best-seller hết gồm Áo thun trắng M, Quần jean đen 32, Sneaker 42 — mất ~8.5M VND DT/tuần. **→ Nhập khẩn 3 SKU này**
2. **128 khách At Risk** giữ 34.5M VND tiềm năng (mua >3 lần, >90 ngày chưa quay lại). Win-back 20% = 6.9M VND. **→ Gửi email mã giảm 15%, deadline 7 ngày**
3. **AOV tăng 3.9% nhờ POS** (AOV POS 573k vs Web 404k). Nhân viên cửa hàng upsell tốt. **→ Đào tạo script upsell POS cho team online**
```

---

## Ví dụ 2: Order Pipeline — Phân tích bottleneck

**User**: "Đơn hàng tháng này tắc ở đâu?"

**Tools gọi (2 SONG SONG)**:
```
hrv_orders_summary(date_from="2026-03-01", date_to="2026-03-31")
hrv_order_cycle_time(date_from="2026-03-01", date_to="2026-03-31")
```

**Output**:

```markdown
## 📦 Order Pipeline — Tháng 3/2026

### Tốc độ xử lý
| Metric | Median | P90 | Đánh giá |
|--------|--------|-----|----------|
| Time-to-Confirm | 2.1h | 8.4h | ✅ Tốt (median <4h) |
| Time-to-Close | 52h | 96h | ⚠️ Chậm (p90 >72h) |

### Đơn hàng bị kẹt
- **Chờ xác nhận >48h**: 7 đơn
- **Đã paid chưa giao >24h**: 12 đơn
- **Revenue at risk**: 12 × 442,900 = ~5.3M VND

### Phân tích đơn hủy (từ orders_summary)
- Tổng hủy: 67/1,847 = **3.6%** ⚠️
- Cancel reasons:
  - 👤 Khách đổi ý (customer): 38 đơn (57%)
  - 📦 Hết hàng (inventory): 18 đơn (27%) 🔴
  - 💳 Thanh toán lỗi (declined): 8 đơn (12%)
  - ❓ Khác: 3 đơn (4%)

### 🚨 Hành động
1. **12 đơn paid-unfulfilled >24h**: Liên hệ kho kiểm tra ngay — 5.3M VND đang risk
2. **p90 Time-to-Close = 96h**: Bottleneck ở khâu giao hàng. Kiểm tra nhà vận chuyển
3. **18 đơn hủy do hết hàng**: Cross-check với inventory_health → nhập gấp SKU thiếu
```

---

## Ví dụ 3: Stock Health — Báo cáo tồn kho

**User**: "Kho đang thế nào? Cần nhập gì?"

**Tools gọi (3 SONG SONG)**:
```
hrv_inventory_health(low_stock_threshold=5, days_for_dead_stock=90)
hrv_stock_reorder_plan(lead_time_days=7, safety_factor=1.3)
hrv_inventory_imbalance()
```

**Output**:

```markdown
## 📦 Stock Health Report

### Phân loại tồn kho
| Trạng thái | SKU | % catalog |
|------------|-----|-----------|
| ✅ Healthy | 245 | 72% |
| ⚠️ Sắp hết (<5) | 34 | 10% |
| 🔴 Hết hàng | 18 | 5% |
| 💀 Dead stock (0 bán 90 ngày) | 45 | 13% |

**Dead stock value: 23,450,000 VND** (vốn chết)

### 🔴 Top 5 cần nhập KHẨN CẤP
| Sản phẩm | Variant | Còn | Bán/ngày | Còn (ngày) | Nhập đề xuất |
|----------|---------|-----|----------|------------|-------------|
| Áo thun trắng | Size M | 2 | 3.1 | **0.6 ngày** 🔴 | 28 units |
| Quần jean đen | Size 32 | 0 | 2.4 | **0** 🔴 | 22 units |
| Sneaker classic | Size 42 | 1 | 1.8 | 0.6 ngày 🔴 | 16 units |
| Áo khoác denim | Size L | 3 | 1.2 | 2.5 ngày | 11 units |
| Túi canvas | Default | 4 | 0.9 | 4.4 ngày | 8 units |

### 💀 Top 5 Dead Stock (giải phóng vốn)
| Sản phẩm | Variant | Tồn | Giá | Vốn chết |
|----------|---------|-----|-----|----------|
| Áo len đỏ | Size XXL | 45 | 399k | 17,955,000 VND |
| Sandal da | Size 37 | 20 | 189k | 3,780,000 VND |
| Khăn choàng | Xám | 12 | 89k | 1,068,000 VND |

### 🔄 Mất cân bằng kho
| Sản phẩm | Kho thừa | Qty | Kho thiếu | Qty | Chuyển |
|----------|----------|-----|-----------|-----|--------|
| Áo thun trắng M | Kho HN | 47 | Kho HCM | 2 | 22 units HN→HCM |

### 🎯 Hành động
1. 🔴 **KHẨN CẤP**: Nhập ngay 3 SKU hết hàng (66 units tổng). Mất ~15M VND DT/tuần nếu chậm
2. 🔄 **CHUYỂN KHO**: 22 units Áo thun M từ HN→HCM (HCM sắp hết, HN dư 47)
3. 💀 **XẢ DEAD STOCK**: Flash sale 45 áo len XXL + 20 sandal 37 → giải phóng 21.7M VND vốn
```

---

## Ví dụ 4: Customer RFM — Phân khúc khách hàng

**User**: "Phân tích khách hàng, khách nào sắp mất?"

**Tools gọi (1 tool)**:
```
hrv_customer_segments()
```

**Output**:

```markdown
## 👥 Customer Intelligence

### Phân khúc RFM
| Segment | Khách | % | Doanh thu | AOV | Marketing |
|---------|-------|---|-----------|-----|-----------|
| 🏆 Champions | 45 | 4.9% | 89.2M | 523k | Loyalty program, early access |
| 💎 Loyal | 67 | 7.3% | 45.1M | 412k | Cross-sell, tăng AOV |
| 🌱 Potential | 89 | 9.7% | 22.3M | 289k | Nurture: voucher mua lần 2-3 |
| 🆕 New | 156 | 16.9% | 18.9M | 221k | Welcome series |
| ⚠️ At Risk | 128 | 13.9% | 34.5M | 467k | **Win-back mã 15%** |
| 😴 Hibernating | 125 | 13.6% | 8.2M | 198k | Re-engagement offer |
| 💀 Lost | 312 | 33.9% | 0 | — | Archive → lookalike audience |

### Key Metrics (Claude tự tính từ segment data)
- **Tổng khách**: 922
- **Repeat Purchase Rate**: (922 - 156 - 312) / 922 = **49.2%** ✅ (>30%)
- **Top 5% khách** (Champions): chiếm **47.6%** doanh thu ⚠️ (tập trung cao)
- **Revenue mới vs cũ**: New 8.7% | Returning 91.3% ✅

### 🚨 128 khách At Risk — Hành động ngay
- **Tổng DT tiềm năng mất**: 34,500,000 VND
- **AOV At Risk**: 467k (cao hơn TB shop 443k!)
- **Win-back target**: Recover 20% = 25 khách × 467k = **11.7M VND**
- **→ Gửi email cá nhân**: "Chúng tôi nhớ bạn" + mã giảm 15% + deadline 7 ngày
```

---

## Ví dụ 5: Operations Scorecard

**User**: "Chấm điểm tổng thể cửa hàng"

**Tools gọi (4 SONG SONG)**:
```
hrv_orders_summary(date_from="2026-03-01", date_to="2026-03-31", compare_prior=true)
hrv_order_cycle_time(date_from="2026-03-01", date_to="2026-03-31")
hrv_inventory_health()
hrv_customer_segments()
```

**Output** (Claude tự chấm điểm từ data):

```markdown
## 🏅 Operations Scorecard — Tháng 3/2026

### Tổng điểm: **6.9 / 10** ⚠️

| Chỉ số | Điểm | Giá trị | Benchmark | Status |
|--------|-------|---------|-----------|--------|
| 📈 Revenue Growth | 8/10 | +12.3% | >5% | ✅ |
| 🛒 AOV | 8/10 | 443k VND | >300k | ✅ |
| ⚡ Processing Speed | 8/10 | 2.1h median | <4h | ✅ |
| ❌ Cancel Rate | 6/10 | 3.6% | <3% | ⚠️ |
| 📦 Stock-out Rate | 4/10 | 5.3% SKU | <5% | ⚠️ |
| 💀 Dead Stock | 4/10 | 13% SKU | <5% | 🔴 |
| 🔄 Repeat Rate | 8/10 | 49.2% | >30% | ✅ |

### 🏆 Top 3 điểm mạnh
1. **Revenue Growth 12.3%** (8/10): Tăng trưởng tốt, AOV tăng nhờ upsell POS hiệu quả
2. **Repeat Rate 49.2%** (8/10): Gần 1/2 khách quay lại — customer retention rất tốt
3. **Processing Speed 2.1h** (8/10): Xác nhận đơn nhanh, team vận hành phản hồi tốt

### 🚨 Top 3 cần cải thiện
1. **Dead Stock 13%** (4/10): 45 SKU, 23.4M VND vốn chết. **→ Flash sale xả kho, target giảm xuống <5%**
2. **Stock-out 5.3%** (4/10): 18 SKU hết hàng gồm 3 best sellers. **→ Nhập gấp, ước tính recover 15M VND DT/tuần**
3. **Cancel Rate 3.6%** (6/10): 27% do hết hàng → fix stock-out sẽ giảm cancel xuống <3%
```

---

## Ví dụ 6: Xử lý lỗi

**User**: "Tình hình doanh thu"

**Khi 1 tool fail**:

```markdown
## 📊 Store Pulse — 31/03 → 06/04

### Doanh thu & Đơn hàng
- **Tổng doanh thu**: 187,350,000 VND (↑12.3% vs tuần trước)
- **Số đơn hàng**: 423
- **AOV**: 442,900 VND

### Tồn kho
⚠️ Không thể lấy dữ liệu tồn kho. Lỗi: 429 Rate Limited
Tool: hrv_inventory_health
Cách khắc phục: Thử lại sau 30 giây

*Dữ liệu doanh thu và khách hàng vẫn chính xác — chỉ thiếu phần tồn kho.*
```

# Công thức & Insights nâng cao — Haravan MCP

Claude tự tính tất cả 25 insights dưới đây từ data trả về bởi 7 smart tools + base tools.
MCP server chỉ là lớp data — Claude AI là lớp phân tích, công thức, và khuyến nghị.

Quy ước ký hiệu:
- `[field]` = field trực tiếp từ tool output
- `→` = Claude tự tính/suy luận
- `≈` = ước tính/proxy

---

## I. ORDER OPERATIONS (6 Insights)

### Insight 1: Order Cycle Time Breakdown

**Data source**: `hrv_order_cycle_time`

**Công thức**:
```
Time-to-Confirm = confirmed_at − created_at
  → Đã tính sẵn: time_to_confirm_hours.median + p90

Time-to-Close = closed_at − created_at
  → Đã tính sẵn: time_to_close_hours.median + p90

Outlier Ratio = p90 / median
  → Nếu > 3× → có đơn kẹt cực lâu, không chỉ là chậm đều

SLA Breach Rate = stuck_orders.total_stuck / total_orders_analyzed × 100
```

**Ngưỡng / Benchmarks**:
| Metric | Xuất sắc | Tốt | Trung bình | Chậm | Nghiêm trọng |
|--------|----------|-----|-----------|------|-------------|
| Time-to-Confirm (median) | <1h | 1-2h | 2-4h | 4-12h | >12h |
| Time-to-Close (median) | <24h | 24-48h | 48-72h | 72-96h | >96h |
| Outlier Ratio (p90/median) | <2× | 2-3× | 3-5× | 5-10× | >10× |
| Stuck Orders % | <0.5% | 0.5-1% | 1-2% | 2-5% | >5% |

**Diễn giải**:
- Median tốt nhưng p90 cao → không phải chậm đều, mà có nhóm đơn cụ thể bị kẹt (shipper vùng xa, sản phẩm cần xử lý đặc biệt)
- Time-to-Confirm tốt nhưng Time-to-Close chậm → bottleneck ở fulfillment/shipping, không phải ops team
- Time-to-Confirm chậm + Time-to-Close chậm → vấn đề hệ thống toàn chuỗi

**Hành động**:
- `paid_not_fulfilled_gt_24h > 0` → Revenue at risk = `stuck_count × AOV` → Liên hệ kho ngay
- `p90/median > 5×` → Điều tra nhóm đơn outlier: filter `haravan_orders_list` bằng status=open + created_at
- Time-to-Confirm >4h median → Review quy trình xác nhận đơn, tự động hóa rule

---

### Insight 2: Order Defect Rate (ODR)

**Data source**: `hrv_orders_summary` → `orders_by_status` + `cancel_reasons`

**Công thức**:
```
ODR = (cancelled + refunded) / total_orders × 100%

Cancel Rate = cancelled / total_orders × 100%
Refund Rate = refunded / total_orders × 100%

Cancel by reason (%) = cancel_reasons.[reason] / total_cancelled × 100%

Supply Chain ODR = cancel_reasons.inventory / total_orders × 100%
Customer ODR = cancel_reasons.customer / total_orders × 100%
Fraud ODR = cancel_reasons.fraud / total_orders × 100%
```

**Ngưỡng / Benchmarks E-commerce Việt Nam**:
| ODR | Đánh giá | Hành động |
|-----|----------|-----------|
| <1% | Xuất sắc | Duy trì, tập trung tăng trưởng |
| 1-3% | Bình thường | Monitor, tìm pattern |
| 3-5% | Cần cải thiện | Phân tích root cause, action plan |
| 5-10% | Nghiêm trọng | Ưu tiên xử lý ngay |
| >10% | Khủng hoảng | Dừng quảng cáo, review toàn bộ quy trình |

**Diễn giải theo root cause**:
- `cancel_reasons.inventory > 25% total_cancel` → Vấn đề supply chain. Cross-check với `hrv_inventory_health`
- `cancel_reasons.customer > 60% total_cancel` → Vấn đề kỳ vọng khách. Review mô tả sản phẩm, hình ảnh
- `cancel_reasons.fraud > 10% total_cancel` → Cần fraud detection rules mạnh hơn
- `cancel_reasons.declined > 15% total_cancel` → Cổng thanh toán có vấn đề, hoặc giỏ hàng friction
- Cancel rate Web >> POS → Vấn đề trải nghiệm online (mô tả mơ hồ, không khớp thực tế)

**Hành động**:
- ODR >3%: Phân tích cancel_reasons → fix root cause ưu tiên theo volume
- inventory cancels cao → Nhập hàng + set up inventory alert
- customer cancels cao → A/B test mô tả sản phẩm, thêm hình ảnh thực tế

---

### Insight 3: Revenue at Risk

**Data source**: `hrv_orders_summary` + `hrv_order_cycle_time`

**Công thức**:
```
Lost Revenue (Cancelled) = orders_by_status.cancelled × AOV
  → Revenue đã mất, không recover được trong kỳ

Pending Revenue Risk = orders_by_status.pending × AOV
  → Chưa thu được, risk tùy collection rate

Fulfillment Revenue at Risk = stuck_orders.paid_not_fulfilled_gt_24h × AOV
  → Đã thu tiền nhưng chưa giao → risk hoàn tiền, phàn nàn

COD Return Risk (Vietnam specific) = estimated_COD_orders × 15-25% fail rate × AOV
  → Đặc thù VN: COD chiếm 60-80% orders, fail rate cao
  → Dùng khi: biết % COD từ orders_by_source hoặc orders_list

Total Revenue at Risk = Fulfillment Risk + Pending Risk × (1 - collection_rate_est)
```

**Ngưỡng**:
- `Fulfillment Risk > 5% weekly revenue` → Cần action ngay
- `Pending Revenue > 10% total revenue` → Collection efficiency thấp
- `COD Return Risk > 20% COD revenue` → Cần thắt chặt COD rules theo tỉnh

**Diễn giải**:
- Revenue at Risk là số tiền "đang treo" có thể mất — khác với revenue đã mất (cancelled)
- VN đặc thù: COD chiếm lớn → return risk thường là nguồn lớn nhất của revenue at risk
- Paid-unfulfilled >24h tăng nhanh → dấu hiệu logistics/kho có bottleneck

**Hành động**:
- Fulfillment risk: List `paid_not_fulfilled_gt_24h` orders, liên hệ kho giao ngay
- COD risk cao: Xem COD Monitor (Insight 5) → giới hạn COD tỉnh fail rate cao
- Pending cao: Review payment collection process, chase pending orders

---

### Insight 4: Payment Collection Efficiency

**Data source**: `hrv_orders_summary` → `orders_by_status`

**Công thức**:
```
Collection Rate = paid / (total_orders − cancelled) × 100%

Outstanding Amount = pending × AOV  [ước tính]

Collection Gap = 100% − Collection Rate
  → % đơn đã xử lý nhưng chưa thu tiền

COD Impact Factor:
  Collection Rate thấp (<90%) + COD channel lớn
  → Nguyên nhân chính là COD fail/return
  → Cross-check: orders_by_source.web/pos COD proportion
```

**Ngưỡng**:
| Collection Rate | Đánh giá |
|----------------|----------|
| >97% | Xuất sắc |
| 95-97% | Tốt |
| 90-95% | Trung bình |
| 85-90% | Cần cải thiện |
| <85% | Nghiêm trọng |

**Diễn giải**:
- Shop thuần POS (offline): Collection rate thường >98% (thanh toán trực tiếp)
- Shop online với COD cao: Collection rate 80-90% là bình thường — nhưng cần tối ưu
- Pending rate tăng theo thời gian → cash flow ngày càng tệ → giới hạn COD hoặc yêu cầu đặt cọc

**Hành động**:
- Collection Rate <90%: Audit pending orders → follow-up, hoặc review COD rules
- Outstanding lớn: Prioritize thu hồi (phone call / reminder SMS)
- POS collection rate thấp: Kiểm tra quy trình thu tiền tại quầy

---

### Insight 5: COD Risk Profile

**Data source**: `hrv_orders_summary` (tổng quát) + `haravan_orders_list` (chi tiết by province)

**Công thức**:
```
COD Orders % = (web COD + mobile COD) / total_orders × 100%
  → Ước tính từ orders_by_source nếu không có gateway data trực tiếp
  → Chính xác hơn: haravan_orders_list + filter gateway_code contains "cod"

COD Fail Rate = cancelled_COD_orders / total_COD_orders × 100%

COD Fail Rate by Province:
  1. haravan_orders_list(status=any, fields=id,financial_status,gateway_code,shipping_address,total_price)
  2. Claude filter: gateway_code contains "cod"
  3. Group by shipping_address.province
  4. Per province: fail_rate = cancelled / (cancelled + paid)

COD Risk Score (1-10) per province:
  fail_rate <10%  → Score 1-2 (thấp)
  fail_rate 10-20% → Score 3-4
  fail_rate 20-30% → Score 5-6
  fail_rate 30-40% → Score 7-8
  fail_rate >40%  → Score 9-10 (nghiêm trọng)
```

**Benchmark COD Việt Nam**:
| COD Fail Rate | Đánh giá | Hành động |
|--------------|----------|-----------|
| <10% | Xuất sắc | Mở rộng COD |
| 10-15% | Tốt | Duy trì |
| 15-25% | Cảnh báo | Tăng phí COD hoặc yêu cầu confirm trước |
| 25-35% | Cao | Giới hạn giá trị đơn COD (VD: max 500k) |
| >35% | Rất cao | Dừng COD tỉnh đó hoặc yêu cầu đặt cọc |

**Tỉnh rủi ro cao điển hình**: Các tỉnh vùng xa, miền núi, TP nhỏ thường fail rate cao hơn HCM/HN.

**Diễn giải**:
- COD fail rate cao theo tỉnh thường tương quan với: thu nhập thấp hơn, hạ tầng logistics kém, văn hóa "thử xem rồi quyết"
- Thứ 6 - Chủ nhật: COD fail rate cao hơn (~5-10%) do khách bận/không ở nhà
- Đơn giá trị cao (>1M VND) + COD → fail rate tăng gấp 2-3 lần

**Hành động**:
- Map tỉnh theo risk score → Tỉnh score >7: yêu cầu đặt cọc 30% hoặc chuyển khoản
- Tỉnh score 5-6: Tăng phí COD 15-20k để filter khách chưa chắc chắn
- Tỉnh score <4: Có thể offer free COD để tăng conversion

---

### Insight 6: Discount Effectiveness

**Data source**: `hrv_orders_summary.discount_usage` (tổng quan) + `haravan_orders_list` (per-code)

**Công thức tổng quan** (từ `hrv_orders_summary`):
```
Penetration Rate = orders_with_discount / total_orders × 100%

Discount Depth = total_discount_value / total_revenue × 100%
  → Bao nhiêu % doanh thu bị "ăn" bởi giảm giá

Avg Discount per Order = total_discount_value / orders_with_discount
  → AOV with discount vs AOV without = hiệu quả upsell

Revenue Protected Rate = 1 - (discount_value / (revenue + discount_value)) × 100%
  → Nếu không giảm, doanh thu gốc là bao nhiêu
```

**Công thức per-code** (từ `haravan_orders_list` + parse `discount_codes`):
```
Code Usage Count = COUNT orders where discount_codes contains [code]
Revenue from Code = SUM total_price where discount_codes contains [code]
Discount Given by Code = SUM discount_codes.amount where code = [code]
Code ROI = Revenue from Code / Discount Given by Code
  → ROI > 10 = rất tốt | 5-10 = tốt | <5 = xem xét lại

Abuse Detection:
  1 customer dùng 1 code >2 lần → abuse
  Code dùng >X lần/ngày so với plan → over-use
```

**Benchmarks**:
| Penetration Rate | Đánh giá |
|-----------------|----------|
| <10% | Quá thấp — không khai thác hết giảm giá |
| 10-20% | Lý tưởng — giảm giá có chọn lọc |
| 20-30% | Chấp nhận được |
| 30-40% | Cảnh báo — phụ thuộc discount |
| >40% | Nghiêm trọng — "nghiện" giảm giá, margin bị ăn mòn |

**Diễn giải**:
- Penetration cao + revenue growth thấp → discount không tạo thêm đơn, chỉ giảm margin
- Penetration thấp nhưng discount depth lớn → có code "khủng" (50-70%) cần kiểm tra
- Discount chỉ trên sản phẩm tồn nhiều (dead stock) = tốt. Trên best-seller = xem xét lại
- Cannibalization: Nếu top discount periods trùng với thời điểm naturally high demand → đang giảm giá không cần thiết

**Hành động**:
- Penetration >40%: Giảm tần suất code, tăng threshold sử dụng (VD: chỉ áp dụng đơn >500k)
- Code ROI <5: Dừng code đó, thử offer khác (free ship thay vì % off thường hiệu quả hơn)
- Dead stock discount: Tính break-even discount depth = cost price / retail price → không discount dưới đây

---

## II. PRODUCT & INVENTORY (8 Insights)

### Insight 7: ABC-FSN Analysis

**Data source**: `hrv_top_products` (doanh thu) + `hrv_inventory_health` (velocity) + `hrv_stock_reorder_plan` (DSR)

**Công thức ABC (Revenue-based)**:
```
Tổng DT = SUM tất cả products revenue trong kỳ

A-items: Top SKUs chiếm cộng dồn 80% tổng DT
  → Thường là top 10-20% SKUs

B-items: SKUs chiếm cộng dồn 80-95% tổng DT
  → Thường là 20-50% SKUs tiếp theo

C-items: SKUs còn lại (95-100% DT)
  → Thường là 50% SKUs dưới cùng

Revenue Share per SKU = product_revenue / total_revenue × 100%
Cumulative Revenue Share → Sort DESC, tính cộng dồn → xác định threshold A/B/C
```

**Công thức FSN (Velocity-based)**:
```
Fast Moving: daily_sales_rate ≥ 1 unit/ngày
  (từ hrv_stock_reorder_plan.daily_sales_rate)

Slow Moving: 0.1 ≤ daily_sales_rate < 1 unit/ngày
  (≈ 3-30 units/tháng)

Non Moving: daily_sales_rate < 0.1 unit/ngày OR 0 sales trong 90 ngày
  (từ hrv_inventory_health.dead_stock)
```

**Ma trận ABC × FSN và hành động**:
| | Fast (≥1/ngày) | Slow (<1/ngày) | Non-moving (0/90 ngày) |
|---|---|---|---|
| **A (top 80% DT)** | KING — Luôn đầy kho, priority tuyệt đối | Theo dõi, có thể seasonal | Cảnh báo: sản phẩm A bỗng không bán — điều tra ngay |
| **B (80-95% DT)** | Duy trì tốt, safety stock chuẩn | OK — monitor | Xem xét discontinue hoặc bundle |
| **C (<5% DT)** | Nhỏ nhưng chạy đều — OK | Tối ưu hoặc bỏ | Loại bỏ — dead weight |

**Thực hành**:
- Gọi `hrv_top_products(top_n=50)` để có đủ data ABC
- Cross với `hrv_stock_reorder_plan` để lấy daily_sales_rate (FSN)
- Cross với `hrv_inventory_health` để xác nhận Non-moving

---

### Insight 8: Inventory Velocity

**Data source**: `hrv_stock_reorder_plan`

**Công thức**:
```
DSR (Daily Sales Rate) = daily_sales_rate  [có sẵn trong output]

DOS (Days of Stock) = qty_available / daily_sales_rate = days_of_stock  [có sẵn]

Reorder Point = lead_time_days × daily_sales_rate × safety_factor  [có sẵn]

Reorder Quantity = (DOS_target × daily_sales_rate) − qty_available
  → DOS_target thường = lead_time × 2 + safety buffer
  → reorder_qty_suggested  [có sẵn]

Inventory Turnover (proxy) = 30 / DOS  [tính tháng]
  → Turnover >6 lần/tháng = rất tốt | 2-6 = tốt | <2 = chậm
```

**Alert Matrix**:
| DOS vs Lead Time | Status | Icon | Hành động |
|-----------------|--------|------|-----------|
| DOS = 0 | HẾT HÀNG | 🔴 | Đặt hàng KHẨN + thông báo khách |
| DOS < lead_time | SẼ HẾT TRƯỚC KHI HÀNG VỀ | 🔴 CRITICAL | Đặt hàng ngay hôm nay |
| lead_time ≤ DOS < 2×lead_time | ĐỀ XUẤT ĐẶT HÀNG | ⚠️ | Lên PO trong 2-3 ngày tới |
| 2×lead_time ≤ DOS < 90 | BÌNH THƯỜNG | ✅ | Theo dõi |
| DOS ≥ 90 | CHẬM / DEAD | 💀 | Xem xét hành động xả hàng |

**Diễn giải**:
- Safety factor 1.3 = buffer 30% cho demand spike (sale, viral, mùa vụ)
- DSR dao động lớn theo ngày/tuần → cân nhắc tăng safety_factor lên 1.5-2.0 cho mùa cao điểm
- Sản phẩm mới (<30 ngày) → DSR không đáng tin → dùng target DSR dựa trên benchmark tương tự

---

### Insight 9: Dead Stock Analysis

**Data source**: `hrv_inventory_health` → `top_10_dead_stock` + `summary.total_dead_stock_value`

**Công thức**:
```
Dead Stock Detection:
  qty_available > 0 AND last_sale_days_ago > days_for_dead_stock (default: 90)

Dead Stock % = dead_stock_count / total_variants_analyzed × 100%

Dead Stock Value = SUM(qty × cost_price)  [hoặc SUM(qty × retail_price × 0.5) nếu không có cost]
  → total_dead_stock_value  [có sẵn — dùng giá bán, không phải giá vốn]

Monthly Holding Cost = dead_stock_value × 2% / month  [rule of thumb: 2% giá trị/tháng cho kho]

Break-even Discount = (retail_price − cost_price) / retail_price × 100%
  → Giảm tối đa không lỗ vốn = gross margin%
```

**Phân loại và hành động**:
| Dead Stock Value / Item | Thời gian dead | Hành động đề xuất |
|------------------------|----------------|-------------------|
| Cao (>2M VND) | 90-180 ngày | Flash sale 20-30% off |
| Cao | >180 ngày | Bundle + flash sale 40-50% off |
| Thấp (<500k) | 90-180 ngày | Bundle với best-seller |
| Thấp | >180 ngày | Donate/thanh lý/tiêu hủy |

**Diễn giải**:
- Dead stock >10% catalog = vấn đề nghiêm trọng về forecasting hoặc mua quá mức
- Dead stock value / monthly revenue >50% → vốn chết đang ảnh hưởng cash flow
- Seasonal dead stock (mùa đông/hè) → normal, nhưng cần dự báo tốt hơn năm sau

**Hành động theo thứ tự ưu tiên**:
1. High-value dead stock → Flash sale (30-50% off) trong 7-14 ngày
2. Bundle with best-seller → Tăng AOV + giải phóng dead stock
3. Return to vendor nếu còn trong hạn warranty/return policy
4. Donate/CSR nếu giá trị thấp → PR tốt hơn tiêu hủy

---

### Insight 10: Product Lifecycle

**Data source**: `hrv_top_products` (2 kỳ) hoặc `haravan_orders_list` + group by product

**Công thức**:
```
Revenue Growth Rate (MoM) = (revenue_this_month − revenue_last_month) / revenue_last_month × 100%

Velocity Trend = DSR_this_period / DSR_last_period  [từ hrv_stock_reorder_plan so 2 kỳ]
```

**5 Stages và thresholds**:
| Stage | Revenue Trend | Velocity Trend | Tín hiệu bổ sung |
|-------|-------------|----------------|-----------------|
| Launch | N/A (mới) | N/A | <30 ngày tuổi, DSR tăng dần |
| Growth | >+20% MoM | >+15% MoM | Rank cải thiện, review tích cực |
| Maturity | -10% đến +10% MoM | Stable | Rank ổn định, sẵn margin |
| Decline | -10% đến -30% MoM | <-15% MoM | Rank tụt, cần action |
| Dead | <-30% MoM hoặc 0 bán | DSR ≈ 0 | No movement 90+ ngày |

**Hành động theo lifecycle**:
- **Launch**: Tăng visibility, test pricing, theo dõi sát 2 tuần/lần
- **Growth**: Đảm bảo stock dồi dào, tăng ads, bundle, upsell
- **Maturity**: Tối ưu margin, cross-sell, tìm variant mới
- **Decline**: Giảm giá có kiểm soát, phân tích lý do, quyết định continue/discontinue
- **Dead**: Thanh lý hoặc refresh sản phẩm (new photos, new desc, new price)

---

### Insight 11: Variant Performance Matrix

**Data source**: `hrv_top_products` → `variant_breakdown`

**Công thức**:
```
Variant Revenue Share = variant.revenue / product.total_revenue × 100%

Variant Qty Share = variant.qty / product.total_quantity × 100%

Price Premium = variant.revenue / variant.qty vs product avg = ASP_variant / ASP_product
  → Premium variant: ASP >115% product avg
  → Discount magnet: ASP <85% product avg

Sell-through Rate (nếu có inventory):
  STR = qty_sold / (qty_sold + qty_available) × 100%
  → Thường cần cross variant_breakdown với haravan_inventory_locations

Variant Concentration:
  Top 1 variant chiếm >50% revenue = sản phẩm phụ thuộc 1 variant
  Top 3 variants chiếm >80% = catalog quá tập trung
```

**Hành động**:
- Variant chiếm >40% DT nhưng hay hết hàng → Tăng safety stock riêng cho variant này
- Variant <2% DT sau 90+ ngày → Candidate discontinue
- Size/màu "chết" liên tục → Rút khỏi production cycle

---

### Insight 12: Catalog Health Score

**Data source**: `haravan_products_list` → Claude tự score

**Công thức (100 điểm)**:
```
Per product, Claude áp 12 tiêu chí:

CONTENT (30 điểm):
  +10: title.length > 10 ký tự  (tên đủ mô tả)
  +15: body_html.length > 50 ký tự  (có mô tả)
  +5: body_html.length > 200 ký tự  (mô tả đầy đủ — BONUS)

MEDIA (15 điểm):
  +10: images.length >= 1  (có ảnh)
  +5: images.length >= 3  (đủ ảnh)

TAXONOMY (15 điểm):
  +5: product_type != null && product_type != ""
  +5: vendor != null && vendor != ""
  +5: tags.length > 0

VARIANT DATA (30 điểm):
  +15: ALL variants có sku != null && sku != ""
  +10: ALL variants có barcode != null && barcode != ""
  +5: ALL variants có price > 0

PRICING (10 điểm):
  +5: ANY variant có compare_at_price > 0  (giá gốc)
  +5: inventory_management == "haravan"  (quản lý tồn kho)

Total: max 110 điểm → normalize về 100
```

**Grading**:
| Score | Hạng | Ý nghĩa |
|-------|------|---------|
| 80-100 | Excellent | Catalog chuẩn, SEO tốt, quản lý được |
| 60-79 | Good | Chấp nhận được, cải thiện từng phần |
| 40-59 | Needs Work | Thiếu nhiều thông tin quan trọng |
| <40 | Poor | Cần cải thiện toàn diện trước khi scale |

**Báo cáo**:
```
Avg Score = SUM(all scores) / product_count
Products by grade: Excellent X%, Good X%, Needs Work X%, Poor X%
Top issues (by frequency): [missing SKU, no images, no description...]
```

---

### Insight 13: Shrinkage Detection

**Data source**: `haravan_inventory_adjustments_list` + `haravan_inventory_locations`

**Công thức**:
```
Shrinkage = (opening_stock + received − sold) − closing_stock

Shrinkage Rate = shrinkage / (opening_stock + received) × 100%

Shrinkage Value = shrinkage_units × cost_price

Type Distribution:
  Negative adjustment without reason → Unknown shrinkage (theft/damage)
  Adjustment reason = "damage" → Damage shrinkage
  Adjustment reason = "theft" → Theft
  Qty counted < system qty → Counting error or actual loss
```

**Benchmarks Retail/E-commerce**:
| Shrinkage Rate | Đánh giá |
|---------------|----------|
| <0.5% | Xuất sắc |
| 0.5-1% | Bình thường — retail industry average |
| 1-2% | Cần điều tra |
| >2% | Nghiêm trọng |

**Diễn giải**:
- Shrinkage tập trung 1 location → vấn đề cụ thể tại chi nhánh đó (theft, damage, counting)
- Shrinkage đồng đều nhiều location → quy trình kiểm kê yếu
- Shrinkage cao sau kỳ high sales → thường là do fulfillment errors (giao nhầm, thiếu hàng)

---

### Insight 14: Multi-Location Imbalance

**Data source**: `hrv_inventory_imbalance`

**Công thức**:
```
Imbalance Ratio = max_location_qty / min_location_qty
  → Đã có sẵn: imbalance_ratio

Transfer Urgency Score (Claude tính):
  Base: imbalance_ratio / 10  (scale 1-10)
  Bonus: +3 nếu min_location_qty = 0 (hết hàng tại điểm nhận)
  Bonus: +2 nếu variant là A-item (cross với hrv_top_products)
  Cap: 10

Transfer Priority = Sort by urgency_score DESC

Optimal Transfer Qty = (total_qty / location_count) − min_location_qty
  → suggested_transfer.qty  [có sẵn, dùng luôn]

Transfer Cost Estimate:
  Internal logistics VN: ~20-50k VND/lần vận chuyển/location
  → Chỉ transfer nếu imbalance_value > 5× transfer_cost
```

**Diễn giải**:
- Imbalance ratio >10× thường do nhận hàng không đều giữa các kho
- HCM thường có DSR cao hơn → cần stock nhiều hơn HN cho cùng SKU
- Transfer không phải luôn tối ưu: nếu lead time nhà cung cấp ngắn (<3 ngày), nhập trực tiếp cho location đó

**Hành động**:
- Imbalance ratio >5× + destination_qty <5 → Transfer ngay
- Imbalance ratio 2-5× → Schedule transfer trong tuần
- Imbalance ratio <2× → Không cần transfer (natural variation)

---

## III. CUSTOMER INTELLIGENCE (5 Insights)

### Insight 15: RFM Scoring

**Data source**: `hrv_customer_segments`

Server đã tính quintile RFM và phân segment. Claude đọc kết quả và diễn giải.

**Phương pháp Quintile**:
```
R (Recency): 5 = mua gần nhất, 1 = lâu nhất không mua
F (Frequency): 5 = mua nhiều nhất, 1 = ít nhất
M (Monetary): 5 = chi tiêu cao nhất, 1 = thấp nhất

Quintile = chia đều khách hàng thành 5 nhóm bằng nhau theo mỗi chiều
Composite RFM Score = R×100 + F×10 + M×1  (đọc như số)
```

**7 Segments đầy đủ**:
| Segment | R range | F range | M range | Ý nghĩa | Marketing Action |
|---------|---------|---------|---------|---------|-----------------|
| **Champions** | 4-5 | 4-5 | 4-5 | Mua gần đây + thường xuyên + nhiều | Loyalty program, referral rewards, early access sản phẩm mới |
| **Loyal** | 3-5 | 4-5 | 3-5 | Khách lâu năm, mua thường xuyên | Cross-sell, upsell, tăng AOV, VIP tier |
| **Potential Loyalists** | 4-5 | 2-3 | 2-3 | Mới mua gần đây, tiềm năng | Nurture: incentive mua lần 2-3, membership perks |
| **New Customers** | 4-5 | 1 | any | Mới mua lần đầu | Welcome series, voucher mua lần 2 (giới hạn 7-14 ngày) |
| **At Risk** | 1-2 | 3-5 | 3-5 | Từng mua nhiều/thường, lâu không quay lại | **WIN-BACK NGAY**: mã cá nhân, "Chúng tôi nhớ bạn", deadline rõ |
| **Hibernating** | 1-2 | 1-2 | any | Ít mua + lâu không quay lại | Re-engagement offer mạnh, survey "tại sao bạn rời đi" |
| **Lost** | 1 | 1 | 1-2 | Mua 1-2 lần, rất lâu không quay lại | Archive, dùng làm lookalike audience cho quảng cáo |

**Win-back ROI calculation**:
```
Win-back Investment = At_Risk.count × discount_offer_value
Win-back Revenue = At_Risk.count × recovery_rate × At_Risk.avg_order_value
  → recovery_rate thực tế VN ≈ 15-25% (dùng 20% làm baseline)
Win-back ROI = (Win-back Revenue − Win-back Investment) / Win-back Investment × 100%
```

---

### Insight 16: Customer Concentration Risk

**Data source**: `hrv_customer_segments`

**Công thức**:
```
Pareto Check:
  Champions_revenue_share = Champions.total_revenue / SUM(all_segments.total_revenue) × 100%

Top 1% check:
  Champions.count / total_customers ≈ 1% nếu Champions.pct < 2%
  → Nếu <2% khách chiếm >50% DT: rủi ro CAO

Top 5% revenue share:
  (Champions + Loyal).total_revenue / total_revenue × 100%

Top 20% revenue share:
  (Champions + Loyal + Potential + At_Risk).total_revenue / total_revenue × 100%
  → Nếu xấp xỉ 80% (Pareto Law) = bình thường
  → Nếu >90% = tập trung quá cao
```

**Risk Thresholds**:
| Metric | An toàn | Cảnh báo | Nguy hiểm |
|--------|---------|----------|-----------|
| Top 1% chiếm DT | <30% | 30-50% | >50% |
| Top 5% chiếm DT | <50% | 50-70% | >70% |
| Champions count | >5% customers | 2-5% | <2% |

**Diễn giải**:
- Tập trung cao = nếu mất 1-2 khách VIP, doanh thu sụt mạnh
- Champions giảm sút theo thời gian = vấn đề retention nghiêm trọng
- Cần diversify customer base: tăng Loyal và Potential Loyalists

**Hành động**:
- Champions revenue >50%: Chương trình loyalty VIP riêng, dedicated account manager
- At Risk có AOV cao (như Champions): Ưu tiên win-back trước

---

### Insight 17: Acquisition vs Retention

**Data source**: `hrv_customer_segments`

**Công thức**:
```
New Revenue = New_Customers.total_revenue
Returning Revenue = SUM(all except New_Customers and Lost).total_revenue
  = Champions + Loyal + Potential + At_Risk + Hibernating revenue

New Revenue % = New_Revenue / total_revenue × 100%
Returning Revenue % = Returning_Revenue / total_revenue × 100%

Repeat Purchase Rate (RPR):
  RPR = (total_customers − New_Customers.count − Lost.count) / total_customers × 100%
  → Hoặc: (Champions + Loyal + Potential + At_Risk + Hibernating).count / total × 100%

Customer Acquisition Cost proxy:
  Không có trực tiếp — cần data marketing spend bên ngoài

Retention Health Index = Champions% + Loyal% + Potential%  (tổng 3 nhóm active tốt)
  → >25% = healthy | 15-25% = OK | <15% = cần cải thiện retention
```

**Benchmarks**:
| Metric | Xuất sắc | Tốt | Trung bình | Cần cải thiện |
|--------|----------|-----|-----------|--------------|
| Returning Revenue % | >60% | 50-60% | 40-50% | <40% |
| Repeat Purchase Rate | >40% | 30-40% | 20-30% | <20% |
| New Revenue % | 20-40% | 10-20% | 40-60% | >70% (quá phụ thuộc acquisition) |

**Diễn giải**:
- New Revenue% >70%: Business phụ thuộc marketing mới, không bền vững khi quảng cáo tắt
- New Revenue% <10%: Đang bão hòa, cần acquisition strategy mạnh hơn
- RPR tăng theo thời gian = retention getting better = CAC hiệu quả hơn

---

### Insight 18: Purchase Gap Analysis

**Data source**: `hrv_customer_segments` → `avg_days_since_last_order` per segment

**Công thức**:
```
Avg Purchase Gap per Segment = avg_days_since_last_order  [có sẵn]

Segment Health Classification:
  Active: avg_days_since_last_order < 30 ngày
  Cooling: 30-60 ngày
  At Risk: 60-90 ngày
  Dormant: 90-180 ngày
  Lost: >180 ngày

Gap Deviation = segment_avg_gap / shop_avg_gap
  → shop_avg_gap = SUM(segment_avg_gap × segment_count) / total_customers

Win-back Window (Vietnam e-commerce):
  60-90 ngày = cửa sổ win-back tối ưu (còn nhớ brand)
  >180 ngày = khó win-back (<5% success rate)
```

**Diễn giải**:
- At_Risk segment với avg_days >90: Đây là nhóm ưu tiên win-back nhất
- Hibernating với avg_days 120-180: Cần offer mạnh hơn At_Risk để kích hoạt
- Champions avg_days <30: Đang rất active, cơ hội upsell cao

---

### Insight 19: Geographic Revenue Density

**Data source**: `haravan_orders_list` → Claude group by `shipping_address.province`

**Công thức**:
```
-- Yêu cầu gọi: haravan_orders_list với fields=id,shipping_address,total_price,financial_status
-- Claude tự group:

Revenue per Province = SUM(total_price) where province = X
Order Count per Province = COUNT orders where province = X
AOV per Province = Revenue / Order_Count
Cancel Rate per Province = COUNT(cancelled) / total × 100%

Revenue Penetration Proxy:
  Province Revenue Share = province_revenue / total_revenue × 100%
  Top 3 tỉnh thường chiếm 60-80% tổng DT Việt Nam

AOV Differential:
  Province_AOV / National_AOV × 100% = mức sống tương đối

Untapped Potential Score:
  Tỉnh AOV cao (>national avg) + volume thấp (<2% DT) = TIỀM NĂNG KHAI THÁC
```

**Context Việt Nam**:
- HCM thường 35-45% total revenue, HN 15-25%, còn lại rải đều
- AOV Hà Nội thường cao hơn HCM 10-20% (sản phẩm khác nhau)
- Tỉnh miền Tây: volume cao nhưng AOV thấp, COD fail rate cao
- Tây Nguyên, Tây Bắc: volume thấp, logistics khó, AOV trung bình

**Hành động**:
- Province có AOV cao + volume thấp: Tăng quảng cáo targeted địa lý
- Province cancel rate cao: Xem COD Monitor → điều chỉnh policy
- Province mới (revenue tăng >50% MoM): Cơ hội expand, tăng stock sẵn sàng

---

## IV. OMNICHANNEL (3 Insights)

### Insight 20: Channel Profitability

**Data source**: `hrv_orders_summary` → `orders_by_source`

**Công thức**:
```
Per Channel:
  Revenue Share = channel.revenue / total_revenue × 100%
  Order Share = channel.count / total_orders × 100%
  Channel AOV = channel.revenue / channel.count

Channel AOV Index = channel_aov / total_aov × 100%
  → POS AOV Index thường 120-150% (VN context)
  → Mobile AOV Index thường 80-95%

Channel Cancel Rate (proxy):
  Cần haravan_orders_list filter + group by source_name
  Hoặc: dùng cancel_reasons pattern (inventory cancel thường web, fraud thường online)

Channel Mix Trend (so 2 kỳ):
  Mobile Revenue % growing = đang shift đúng hướng
  POS Revenue % shrinking = cần đầu tư omnichannel
```

**Context Việt Nam đặc thù**:
- POS AOV thường cao hơn web 30-50% vì: nhân viên upsell, khách xem trực tiếp
- Mobile conversion thấp nhất nhưng traffic nhiều nhất → cải thiện mobile UX = quick win
- Android > iPhone về volume ở VN (market share Android ~75%)
- "Other" source thường là API, wholesale, custom integration

**Benchmarks**:
| Channel | Typical Revenue Mix | AOV vs Avg |
|---------|-------------------|------------|
| Web | 50-65% | 100% (baseline) |
| POS | 20-35% | 130-150% |
| Mobile (iOS+Android) | 10-20% | 85-95% |

**Hành động**:
- POS AOV cao → Đào tạo upsell script POS và áp dụng cho web/chat sales
- Mobile tăng trưởng → Optimize mobile checkout, thêm Apple/Google Pay
- Channel skewed >80% một kênh → Rủi ro tập trung, đa dạng hóa

---

### Insight 21: Location Efficiency

**Data source**: `haravan_orders_list` + `haravan_locations_list` → Claude group by `location_id`

**Công thức**:
```
-- haravan_orders_list với fields=id,location_id,total_price,financial_status,created_at,closed_at
-- Claude group by location_id, cross với haravan_locations_list

Revenue per Location = SUM(total_price) where location_id = X
Order Count per Location = COUNT where location_id = X
AOV per Location = revenue / order_count
Fulfillment Rate = paid_fulfilled / total_assigned × 100%

Processing Speed = AVG(closed_at − created_at) per location
  → Thường cần order_cycle_time breakdown by location (không có sẵn trong smart tool)
  → Proxy: tính từ orders_list timestamps

GMROI proxy (Gross Margin Return on Inventory Investment):
  GMROI = (revenue × gross_margin%) / avg_inventory_value
  → avg_inventory_value cần từ inventory_locations per location
  → gross_margin% cần từ product cost data (không phải luôn có)
  → Simplified: Revenue per Location / Estimated_Inventory_Value
```

**Diễn giải**:
- Location AOV thấp + high volume → nhân viên chưa upsell đúng cách
- Location Fulfillment Rate thấp → thiếu stock hoặc quy trình xử lý chậm
- Processing Speed chậm tại 1 location → tập huấn hoặc tăng nhân sự

---

### Insight 22: Staff Performance (Plus plan)

**Data source**: `haravan_users_list` + `haravan_orders_list` (filter by assigned staff)

**Công thức**:
```
-- Chỉ khả dụng khi shop có Haravan Plus
-- haravan_orders_list với fields chứa user_id / assigned_to

Revenue per Staff = SUM(total_price) where user_id = X
Orders per Staff = COUNT where user_id = X
AOV per Staff = revenue / orders
Avg Processing Time per Staff = AVG(closed_at − confirmed_at) per staff
Cancel Rate per Staff = cancelled / total × 100%

Staff Efficiency Index = orders_per_day / team_average_orders_per_day
```

**Hành động**:
- Staff có AOV thấp hơn average >20% → Coaching upsell
- Staff có cancel rate cao → Kiểm tra quy trình confirm đơn, tránh nhận đơn khi không có hàng
- Staff có processing time chậm → Tập huấn workflow, tăng quyền truy cập công cụ

---

## V. COMPOSITE (3 Insights)

### Insight 23: Operations Scorecard

**Data source**: 4 smart tools song song: `hrv_orders_summary`, `hrv_order_cycle_time`, `hrv_inventory_health`, `hrv_customer_segments`

**10 Dimensions và Công thức Chấm Điểm (1-10)**:

| Dimension | Metric | 9-10 | 7-8 | 5-6 | 3-4 | 1-2 |
|-----------|--------|------|-----|-----|-----|-----|
| Revenue Growth | % MoM change | >20% | 5-20% | 0-5% | -10–0% | <-10% |
| AOV | VND | >500k | 300-500k | 200-300k | 100-200k | <100k |
| Processing Speed | Median confirm (h) | <1h | 1-2h | 2-4h | 4-12h | >12h |
| Fulfillment Speed | Median close (h) | <24h | 24-48h | 48-72h | 72-96h | >96h |
| Cancel Rate | % | <1% | 1-3% | 3-5% | 5-10% | >10% |
| Collection Rate | % | >97% | 95-97% | 90-95% | 85-90% | <85% |
| Stock-out Rate | % SKU hết hàng | <2% | 2-5% | 5-10% | 10-20% | >20% |
| Dead Stock % | % SKU dead | <2% | 2-5% | 5-10% | 10-15% | >15% |
| Customer Retention | RPR % | >40% | 30-40% | 20-30% | 10-20% | <10% |
| Customer Health | At_Risk+Hibernating % | <15% | 15-25% | 25-35% | 35-45% | >45% |

**Tổng điểm**:
```
Scorecard Total = MEAN(all 10 dimensions)  → X.X / 10

Weighted Version (tùy chọn):
  Revenue Growth × 1.5
  Cancel Rate × 1.3
  Stock-out Rate × 1.2
  Còn lại × 1.0
  Weighted Score = SUM(dim × weight) / SUM(weights)
```

**Grading**:
| Score | Grade | Hành động |
|-------|-------|-----------|
| 8.0-10 | Excellent — Kinh doanh xuất sắc | Duy trì, mở rộng |
| 6.0-7.9 | Good — Đang tốt, còn cải thiện | Fix top 3 weak points |
| 4.0-5.9 | Fair — Nhiều điểm cần cải thiện | Action plan cho bottom 5 |
| <4.0 | Poor — Cần cải thiện toàn diện | Đánh giá lại chiến lược |

---

### Insight 24: Anomaly Detection

**Data source**: Tất cả smart tools + baseline từ kỳ trước

**Công thức**:
```
Baseline = avg(last 4 periods same metric)
Std Dev = STDEV(last 4 periods)
Upper Bound = Baseline + 2 × Std_Dev
Lower Bound = Baseline − 2 × Std_Dev

Anomaly = current_value < Lower_Bound OR current_value > Upper_Bound
  → Mức độ: |current - baseline| / std_dev = Z-score
  → Z > 2: Anomaly cảnh báo | Z > 3: Anomaly nghiêm trọng
```

**9 Anomaly Types cụ thể**:
| # | Anomaly | Metric | Ngưỡng | Nguyên nhân thường gặp |
|---|---------|--------|--------|----------------------|
| 1 | Revenue Spike | Daily revenue | >Baseline + 2σ | Campaign, viral, sự kiện |
| 2 | Revenue Drop | Daily revenue | <Baseline - 2σ | Technical issue, hết hàng, mùa thấp |
| 3 | Cancel Surge | Daily cancel rate | >5% sudden spike | Hết hàng, QC issue, UX bug |
| 4 | AOV Drop | Daily AOV | <-20% vs baseline | Discount abuse, wrong pricing |
| 5 | Processing Slowdown | Daily avg confirm time | >2× baseline | System overload, thiếu nhân sự |
| 6 | Stock-out Spike | # SKUs out of stock | >2× baseline | Demand surge, supply failure |
| 7 | New Customer Drop | New_Customers count | <-30% vs baseline | Marketing stop, tracking issue |
| 8 | COD Fail Surge | Province COD fail rate | >baseline + 15pp | Logistics issue, fraud wave |
| 9 | Refund Wave | Refund rate | >3× baseline | Quality issue, wrong product shipped |

**Diễn giải**:
- Single anomaly → theo dõi thêm 1-2 ngày
- 2+ anomalies cùng lúc → tương quan, điều tra ngay
- Anomaly tích cực (Revenue Spike) → điều tra NGAY để replicate (campaign nào hiệu quả?)

---

### Insight 25: Business Health Dashboard

**Data source**: Tổng hợp từ tất cả insights trước

**Composite Score Calculation**:
```
Health Score = WEIGHTED AVERAGE of:
  Operations Efficiency (40%):
    = AVG(Processing_Speed, Cancel_Rate, Collection_Rate, Fulfillment_Speed) scores
  
  Inventory Health (30%):
    = AVG(Stock_out_Rate, Dead_Stock_pct, Reorder_Coverage) scores
  
  Customer Health (30%):
    = AVG(RPR, At_Risk_pct, New_vs_Returning) scores

Final Health Score = X.X / 10
```

**Dashboard Format**:
```
BUSINESS HEALTH: X.X/10 [GRADE]
├── Operations: X.X/10
│   ├── Processing Speed: X/10
│   ├── Cancel Rate: X/10
│   ├── Collection Rate: X/10
│   └── Fulfillment Speed: X/10
├── Inventory: X.X/10
│   ├── Stock-out Rate: X/10
│   ├── Dead Stock %: X/10
│   └── Reorder Coverage: X/10
└── Customers: X.X/10
    ├── Retention Rate: X/10
    ├── At-Risk Volume: X/10
    └── Acquisition Mix: X/10
```

**Trending**:
- So sánh Health Score tuần/tháng → Trend up/down
- Delta > +0.5: Cải thiện đáng kể
- Delta < -0.5: Xuống cấp, cần điều tra

**Action Priority Matrix**:
```
Score < 4 trong bất kỳ dimension nào → Red Alert, fix ngay
Score 4-6 → Plan improvement trong 30 ngày
Score > 7 → Monitor + optimize
```

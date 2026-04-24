# Anti-Patterns & Tối Ưu — Haravan MCP

---

## I. 8 Anti-Patterns

### 1. Manual Pagination — Tự loop pages

```
❌ SAI:
haravan_orders_list(page=1, limit=50)  → 50 orders
haravan_orders_list(page=2, limit=50)  → 50 orders
haravan_orders_list(page=3, limit=50)  → 50 orders
... × 40 lần = 200,000 tokens, 40 API calls

✅ ĐÚNG:
hrv_orders_summary(date_from, date_to)  → 300 tokens, 1 call
```

**Tại sao sai**: Claude không quản lý rate limiting, tốn context window, chậm, dễ bị 429.

### 2. Detail Loop — Gọi detail tool nhiều lần để đếm

```
❌ SAI:
haravan_products_get(id=1001)  → check images
haravan_products_get(id=1002)  → check images
haravan_products_get(id=1003)  → check images
... × 100 sản phẩm = 100 API calls

✅ ĐÚNG:
haravan_products_list(fields="id,title,images,variants")  → 1-2 calls
→ Claude tự score từ response
```

### 3. Duplicate Calls — Gọi cùng tool 2 lần cùng params

```
❌ SAI:
Turn 1: hrv_orders_summary(date_from="2026-03-01", date_to="2026-03-31")
Turn 2: User hỏi "cancel rate bao nhiêu?"
→ Gọi lại hrv_orders_summary(date_from="2026-03-01", date_to="2026-03-31")

✅ ĐÚNG:
Turn 2: Dùng data từ Turn 1 → orders_by_status.cancelled / total_orders
```

### 4. Missing Date Range — Không truyền date_from/date_to

```
❌ SAI:
hrv_orders_summary()  → server lấy default 30 ngày, user hỏi "tháng 3" nhưng data sai

✅ ĐÚNG:
hrv_orders_summary(date_from="2026-03-01", date_to="2026-03-31")
```

**Luôn tính date range từ câu hỏi user**:
- "hôm nay" → date_from = today 00:00, date_to = now
- "tuần này" → date_from = Monday, date_to = now
- "tháng trước" → date_from = first day prev month, date_to = last day prev month

### 5. Unconfirmed Writes — Tự ý create/update/delete

```
❌ SAI:
User: "hủy đơn 12345"
→ Gọi ngay haravan_orders_cancel(order_id=12345)

✅ ĐÚNG:
User: "hủy đơn 12345"
→ Claude: "Tôi sẽ hủy đơn #12345 với lý do: customer. Xác nhận? (y/n)"
→ User: "y"
→ Gọi haravan_orders_cancel(order_id=12345, reason="customer")
```

### 6. Generic Insights — Insight chung chung không số liệu

```
❌ SAI:
"Doanh thu đang tốt"
"Nên cải thiện tồn kho"
"Khách hàng cần chăm sóc"

✅ ĐÚNG:
"Doanh thu tăng 12% nhờ AOV tăng 350k→420k — upsell bundle đang hiệu quả"
"18 SKU hết hàng gồm 3 best sellers — mất ~12M VND DT/tuần"
"128 khách At Risk giữ 34.5M VND tiềm năng — win-back 20% = 6.9M VND"
```

**Công thức insight**: [Metric] + [Con số] + [So sánh] + [Nguyên nhân] → [Hành động] + [Impact]

### 7. Over-calling — Gọi >6 tools cho 1 câu hỏi

```
❌ SAI:
hrv_orders_summary + hrv_top_products + hrv_order_cycle_time
+ hrv_customer_segments + hrv_inventory_health + hrv_stock_reorder_plan
+ hrv_inventory_imbalance + haravan_shop_get = 8 calls

✅ ĐÚNG:
Scorecard cần max 4 tools:
hrv_orders_summary + hrv_order_cycle_time + hrv_inventory_health + hrv_customer_segments
→ Claude tự tính các dimension còn lại từ data có sẵn
```

### 8. Ignoring Existing Data — Không tái sử dụng data

```
❌ SAI:
Turn 1: hrv_orders_summary → có orders_by_source
Turn 2: User hỏi "kênh nào bán tốt nhất?"
→ Gọi tool mới (không cần!)

✅ ĐÚNG:
Turn 2: Đọc orders_by_source từ Turn 1 → "Web chiếm 65% đơn, POS 23%, Mobile 12%"
```

---

## II. Token Optimization

### So sánh chi phí

| Scenario | Không tối ưu | Tối ưu | Tiết kiệm |
|----------|-------------|--------|-----------|
| "Doanh thu tháng này" | 40 calls, 250K tokens, ~$7.5 | 1 call, 300 tokens, $0.01 | 99.9% |
| "Phân tích khách hàng RFM" | 100+ calls, 500K tokens, ~$15 | 1 call, 800 tokens, $0.02 | 99.8% |
| "Scorecard toàn diện" | 200+ calls, 1M tokens, ~$30 | 4 calls, 2K tokens, $0.06 | 99.8% |
| "Full session 10 câu hỏi" | 500+ calls, 5M tokens, ~$75 | 10-15 calls, 15K tokens, $0.23 | 99.7% |

### Nguyên tắc tiết kiệm token
1. **Smart tools trả summary** (~300-800 tokens) thay vì raw data (50K-500K tokens)
2. **Fields parameter**: chỉ request fields cần thiết → giảm 60-80% response size
3. **Tái sử dụng data**: turn trước có → đừng gọi lại
4. **Max 6 calls/question**: nếu cần nhiều hơn → đang gọi sai

---

## III. Fields Parameter — Chọn đúng fields

| Use case | Fields nên request |
|----------|-------------------|
| Revenue summary | `id,total_price,financial_status,cancelled_status,source_name,created_at` |
| Product analysis | `id,line_items,created_at` |
| Cycle time | `id,created_at,confirmed_at,closed_at,financial_status,fulfillment_status` |
| Cancel analysis | `id,cancelled_status,cancel_reason,source_name,total_price,created_at` |
| Customer lookup | `id,email,first_name,last_name,orders_count,total_spent,last_order_date` |
| Product scoring | `id,title,body_html,images,variants,product_type,vendor,tags` |
| Inventory check | `id,title,variants` (rồi call inventory_locations per variant) |

**Impact**: Full order response = 3,000-5,000 bytes. Optimized = 200-400 bytes. Giảm 80-90%.

---

## IV. Rate Limit Compliance

### Haravan API Limits
- **Bucket**: 80 requests
- **Drain rate**: 4 req/s
- **Header**: `X-Haravan-Api-Call-Limit: 32/80`

### Smart tools tự quản lý
- Sleep 250ms giữa pages (safe rate ~3 req/s)
- Sleep 1s khi bucket usage > 60/80
- Sleep 2s khi bucket usage > 75/80
- Auto-retry khi 429 (check Retry-After header)

### Claude manual calls
- Max 3-4 calls liên tiếp rồi chờ kết quả
- Gọi song song tối đa 6 tools (MCP server queue)
- Nếu 429 → báo user, chờ 5-10s, thử lại

---

## V. Data Reuse Patterns

### Từ hrv_orders_summary đã có sẵn:

| Cần phân tích | Data sẵn có | ĐỪNG gọi thêm |
|---------------|------------|----------------|
| Channel breakdown | `orders_by_source` (web/pos/mobile) | ~~hrv_channel_mix~~ (đã xóa) |
| Cancel analysis | `orders_by_status.cancelled` + `cancel_reasons` | ~~hrv_orders_cancel_analysis~~ (đã xóa) |
| Discount overview | `discount_usage.orders_with_discount` + `total_discount_value` | ~~hrv_discount_analysis~~ (đã xóa) |
| ODR | `(cancelled + refunded) / total_orders` | Tự tính |
| Collection rate | `paid / (total - cancelled)` | Tự tính |

### Từ hrv_customer_segments đã có sẵn:

| Cần | Data sẵn có | ĐỪNG gọi |
|-----|------------|-----------|
| Repeat Purchase Rate | `(total - New.count - Lost.count) / total` | Tự tính |
| Customer concentration | `Champions.total_revenue / sum(all.total_revenue)` | Tự tính |
| At Risk revenue | `At_Risk.total_revenue` | Tự tính |
| Win-back ROI estimate | `At_Risk.count × At_Risk.avg_order_value × 20%` | Tự tính |

### Từ hrv_inventory_health đã có sẵn:

| Cần | Data sẵn có | ĐỪNG gọi |
|-----|------------|-----------|
| Stock-out rate | `out_of_stock / total_variants` | Tự tính |
| Dead stock value | `total_dead_stock_value` | Có sẵn |
| Revenue loss estimate | `out_of_stock_count × avg_daily_revenue` | Tự tính kết hợp orders_summary |

---

## VI. Khi nào NÊN gọi base tool

| Tình huống | Tool | Lý do |
|-----------|------|-------|
| User hỏi chi tiết 1 đơn cụ thể | `haravan_orders_get(id)` | Smart tool chỉ trả summary |
| User muốn tìm khách theo email | `haravan_customers_search(query)` | Smart tool không search |
| User muốn sửa/tạo/xóa | `haravan_*_create/update/delete` | Smart tool chỉ read |
| User cần data tỉnh cụ thể | `haravan_orders_list(status=any)` + filter | Smart tool không group by province |
| User cần catalog scoring | `haravan_products_list` | Claude tự score 12 criteria |
| Drill-down sau smart tool | `haravan_*_get(id)` | Xem chi tiết item từ summary |

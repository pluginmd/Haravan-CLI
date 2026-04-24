# Haravan MCP — Danh mục Tool đầy đủ

Kiến trúc 2 lớp: **MCP Server** (7 smart tools) xử lý pagination lớn + aggregation phức tạp. **Claude Skill** (bạn) phân tích, filter, so sánh, tạo insight từ data trả về.

Nguyên tắc then chốt: Smart tools chỉ làm heavy lifting mà Claude KHÔNG thể làm hiệu quả (1000+ orders, full-population RFM, 200 inventory calls). Mọi thứ còn lại — Claude tự làm.

---

## PHẦN I: 7 SMART TOOLS (hrv_*)

### 1. hrv_orders_summary

**Mục đích**: Tổng hợp doanh thu, đơn hàng, phân bổ theo status/source/lý do hủy/discount — từ toàn bộ orders trong kỳ (có thể 10,000+ đơn, server tự pagination).

**Input params**:
| Param | Type | Default | Mô tả |
|-------|------|---------|-------|
| `date_from` | string (ISO 8601) | 30 ngày trước | Ngày bắt đầu, VD: "2026-03-01" |
| `date_to` | string (ISO 8601) | Hôm nay | Ngày kết thúc, VD: "2026-03-31" |
| `compare_prior` | boolean | true | Tự động so sánh cùng kỳ trước (cùng số ngày) |

**Output — cấu trúc JSON đầy đủ**:
```json
{
  "period": { "from": "2026-03-01", "to": "2026-03-31", "days": 31 },
  "total_orders": 1847,
  "total_revenue": 819234500,
  "aov": 443400,
  "orders_by_status": {
    "paid": 1620,
    "pending": 134,
    "refunded": 26,
    "cancelled": 67
  },
  "orders_by_source": {
    "web": { "count": 1124, "revenue": 478200000 },
    "pos": { "count": 512, "revenue": 289300000 },
    "iphone": { "count": 145, "revenue": 36200000 },
    "android": { "count": 58, "revenue": 12100000 },
    "other": { "count": 8, "revenue": 3434500 }
  },
  "cancel_reasons": {
    "customer": 38,
    "inventory": 18,
    "fraud": 3,
    "declined": 8,
    "other": 0
  },
  "discount_usage": {
    "orders_with_discount": 412,
    "total_discount_value": 18650000,
    "unique_discount_codes": 23
  },
  "comparison": {
    "revenue_change_pct": 12.3,
    "orders_change_pct": 8.1,
    "aov_change_pct": 3.9
  }
}
```

**Claude tự làm từ output này (KHÔNG cần gọi thêm tool)**:
- **Channel breakdown**: Tính % mỗi kênh = `orders_by_source.web.count / total_orders × 100`. AOV per channel = revenue / count
- **Cancel rate**: `orders_by_status.cancelled / total_orders × 100`
- **ODR (Order Defect Rate)**: `(cancelled + refunded) / total_orders × 100`
- **Cancel analysis by reason**: `cancel_reasons` đã breakdown sẵn — tính %, xác định root cause
- **Cancel by channel**: Nếu cần → tính proxy từ cancel_reasons pattern (inventory cancel thường từ web)
- **Discount penetration**: `discount_usage.orders_with_discount / total_orders × 100`
- **Discount depth**: `total_discount_value / total_revenue × 100`
- **Collection Rate**: `paid / (total_orders - cancelled) × 100`
- **Outstanding estimate**: `pending × aov`
- **Revenue comparison**: `comparison.revenue_change_pct` đã có sẵn
- **COD overview tổng quát**: Ước tính từ `orders_by_source` (POS thường ít COD hơn web)

**Khi nào DÙNG**: Mọi câu hỏi liên quan doanh thu, đơn hàng tổng quát, so sánh kỳ, phân bổ kênh, tỷ lệ hủy, tổng quan discount.

**Khi nào KHÔNG dùng**: Khi cần chi tiết từng đơn cụ thể (→ `haravan_orders_get`), khi cần filter theo province/shipping (→ `haravan_orders_list`), khi cần per-code discount ROI (→ `haravan_orders_list` + parse).

---

### 2. hrv_top_products

**Mục đích**: Xếp hạng sản phẩm theo doanh thu trong kỳ, kèm variant breakdown. Server xử lý aggregation qua tất cả line_items của 1000+ đơn.

**Input params**:
| Param | Type | Default | Mô tả |
|-------|------|---------|-------|
| `date_from` | string | 30 ngày trước | Ngày bắt đầu |
| `date_to` | string | Hôm nay | Ngày kết thúc |
| `top_n` | integer | 10 | Số sản phẩm trả về (max: 50) |

**Output — cấu trúc JSON**:
```json
{
  "period": { "from": "2026-03-01", "to": "2026-03-31" },
  "total_products_sold": 187,
  "products": [
    {
      "rank": 1,
      "product_id": "12345678",
      "product_title": "Áo thun basic unisex",
      "total_quantity": 342,
      "total_revenue": 94050000,
      "revenue_share_pct": 11.5,
      "variant_breakdown": [
        { "variant_title": "M / Trắng", "qty": 145, "revenue": 39875000 },
        { "variant_title": "L / Đen", "qty": 98, "revenue": 26950000 },
        { "variant_title": "S / Trắng", "qty": 67, "revenue": 18425000 }
      ]
    }
  ]
}
```

**Claude tự làm từ output này**:
- **Revenue concentration**: Top 3 products / total_revenue = mức độ phụ thuộc sản phẩm
- **Variant hot nhất**: `variant_breakdown[0]` của mỗi product
- **Price sweet spot**: `total_revenue / total_quantity` = ASP (Average Selling Price) per product
- **ABC phân loại sơ bộ**: Top 20% products theo revenue → A-items, middle 30% → B, bottom 50% → C
- **Sell-through proxy**: So sánh qty_sold vs inventory (nếu có từ inventory_health)
- **Lifecycle signal**: Product rank thay đổi lớn so kỳ trước → Growth/Decline stage
- **Bundle opportunities**: Products hay bán cùng nhau (cần orders_list để xác nhận)

**Khi nào DÙNG**: Sản phẩm bán chạy nhất, revenue concentration, phân tích variant, ABC analysis sơ bộ.

**Khi nào KHÔNG dùng**: Khi cần tất cả sản phẩm (kể cả không bán) → `haravan_products_list`. Khi cần catalog health scoring → `haravan_products_list`. Khi cần thông tin chi tiết sản phẩm (images, SEO, variants đầy đủ) → `haravan_products_get`.

---

### 3. hrv_order_cycle_time

**Mục đích**: Đo tốc độ xử lý đơn hàng (confirm + close), phát hiện đơn bị kẹt. Server tính từ timestamps của toàn bộ đơn trong kỳ.

**Input params**:
| Param | Type | Default | Mô tả |
|-------|------|---------|-------|
| `date_from` | string | 30 ngày trước | Ngày bắt đầu |
| `date_to` | string | Hôm nay | Ngày kết thúc |

**Output — cấu trúc JSON**:
```json
{
  "period": { "from": "2026-03-01", "to": "2026-03-31" },
  "total_orders_analyzed": 1847,
  "time_to_confirm_hours": {
    "median": 2.1,
    "p90": 8.4,
    "mean": 3.2
  },
  "time_to_close_hours": {
    "median": 52.0,
    "p90": 96.0,
    "mean": 58.4
  },
  "stuck_orders": {
    "unconfirmed_gt_48h": 7,
    "paid_not_fulfilled_gt_24h": 12,
    "total_stuck": 19
  }
}
```

**Claude tự làm từ output này**:
- **Processing Speed score**: So median với benchmark (<2h=10, 2-4h=8, 4-8h=6, >8h=3)
- **Revenue at risk từ stuck**: `stuck_orders.paid_not_fulfilled_gt_24h × AOV` (AOV lấy từ orders_summary)
- **Outlier detection**: `p90 / median > 3×` = có outlier đơn kẹt cực lâu, cần điều tra
- **Bottleneck diagnosis**: Nếu time_to_confirm tốt nhưng time_to_close chậm → bottleneck ở fulfillment/shipping, không phải ops team
- **SLA breach rate**: `stuck_orders.total_stuck / total_orders_analyzed × 100`
- **Trend comparison**: Nếu gọi 2 kỳ → so sánh median để đánh giá cải thiện/xuống cấp

**Khi nào DÙNG**: Phân tích pipeline, bottleneck xử lý đơn, đơn kẹt, tốc độ vận hành.

**Khi nào KHÔNG dùng**: Khi chỉ cần tổng số đơn/doanh thu → `hrv_orders_summary`. Khi cần xem đơn cụ thể bị kẹt → `haravan_orders_list(status=open)`.

---

### 4. hrv_customer_segments

**Mục đích**: Phân khúc toàn bộ khách hàng theo RFM (Recency-Frequency-Monetary) bằng phương pháp quintile. Server cần xử lý toàn bộ customer + order history — không thể làm phía Claude.

**Input params**:
| Param | Type | Default | Mô tả |
|-------|------|---------|-------|
| `min_orders` | integer | 0 | Lọc chỉ khách có ≥N đơn (0 = tất cả) |

**Output — cấu trúc JSON**:
```json
{
  "total_customers": 922,
  "analysis_date": "2026-04-06",
  "segments": [
    {
      "name": "Champions",
      "count": 45,
      "pct": 4.9,
      "total_revenue": 89200000,
      "avg_order_value": 523000,
      "avg_orders": 8.2,
      "avg_days_since_last_order": 12,
      "rfm_ranges": { "r": "4-5", "f": "4-5", "m": "4-5" },
      "action_suggestion": "Loyalty program, early access, referral rewards"
    },
    {
      "name": "Loyal",
      "count": 67,
      "pct": 7.3,
      "total_revenue": 45100000,
      "avg_order_value": 412000,
      "avg_orders": 5.1,
      "avg_days_since_last_order": 28,
      "rfm_ranges": { "r": "3-5", "f": "4-5", "m": "3-5" },
      "action_suggestion": "Cross-sell, upsell, tăng AOV"
    },
    {
      "name": "Potential_Loyalists",
      "count": 89,
      "pct": 9.7,
      "total_revenue": 22300000,
      "avg_order_value": 289000,
      "avg_orders": 2.3,
      "avg_days_since_last_order": 21,
      "rfm_ranges": { "r": "4-5", "f": "2-3", "m": "2-3" },
      "action_suggestion": "Nurture: voucher mua lần 2-3, membership tier"
    },
    {
      "name": "New_Customers",
      "count": 156,
      "pct": 16.9,
      "total_revenue": 18900000,
      "avg_order_value": 221000,
      "avg_orders": 1.0,
      "avg_days_since_last_order": 15,
      "rfm_ranges": { "r": "4-5", "f": "1", "m": "1-3" },
      "action_suggestion": "Welcome series, voucher mua lần 2"
    },
    {
      "name": "At_Risk",
      "count": 128,
      "pct": 13.9,
      "total_revenue": 34500000,
      "avg_order_value": 467000,
      "avg_orders": 4.8,
      "avg_days_since_last_order": 112,
      "rfm_ranges": { "r": "1-2", "f": "3-5", "m": "3-5" },
      "action_suggestion": "Win-back NGAY: mã cá nhân, deadline rõ ràng"
    },
    {
      "name": "Hibernating",
      "count": 125,
      "pct": 13.6,
      "total_revenue": 8200000,
      "avg_order_value": 198000,
      "avg_orders": 2.1,
      "avg_days_since_last_order": 198,
      "rfm_ranges": { "r": "1-2", "f": "1-2", "m": "1-3" },
      "action_suggestion": "Re-engagement offer mạnh, kèm deadline"
    },
    {
      "name": "Lost",
      "count": 312,
      "pct": 33.9,
      "total_revenue": 0,
      "avg_order_value": 145000,
      "avg_orders": 1.2,
      "avg_days_since_last_order": 420,
      "rfm_ranges": { "r": "1", "f": "1", "m": "1-2" },
      "action_suggestion": "Archive, dùng làm lookalike audience ads"
    }
  ]
}
```

**Claude tự làm từ output này**:
- **Repeat Purchase Rate**: `(total - New_Customers.count - Lost.count) / total × 100`
- **Revenue concentration risk**: `Champions.total_revenue / SUM(all.total_revenue) × 100` — nếu >50% là rủi ro
- **At Risk value**: Giá trị tiềm năng mất = `At_Risk.total_revenue`
- **Win-back ROI estimate**: `At_Risk.count × At_Risk.avg_order_value × 20%` (20% recovery rate thực tế VN)
- **New vs Returning split**: `New_Customers.total_revenue / total_revenue × 100` vs phần còn lại
- **Retention health**: Champions% + Loyal% + Potential% / total = tỷ lệ khách đang tích cực
- **Churn risk metric**: `(At_Risk + Hibernating).count / total × 100`
- **LTV proxy per segment**: `avg_order_value × avg_orders` per segment

**Khi nào DÙNG**: RFM phân khúc, phân tích khách VIP, khách sắp mất, retention rate, acquisition vs retention split.

**Khi nào KHÔNG dùng**: Tìm kiếm khách cụ thể → `haravan_customers_search`. Xem lịch sử mua của 1 khách → `haravan_customers_get` + `haravan_orders_list`. Geography khách hàng → `haravan_customers_list` rồi group by province.

---

### 5. hrv_inventory_health

**Mục đích**: Phân loại toàn bộ variants theo tình trạng tồn kho. Server cần 100-200 inventory API calls để fetch qty per variant — không thể làm phía Claude trong giới hạn token.

**Input params**:
| Param | Type | Default | Mô tả |
|-------|------|---------|-------|
| `low_stock_threshold` | integer | 5 | Ngưỡng "sắp hết" (units) |
| `days_for_dead_stock` | integer | 90 | Số ngày không bán = "dead stock" |

**Output — cấu trúc JSON**:
```json
{
  "summary": {
    "total_variants_analyzed": 342,
    "out_of_stock": 18,
    "low_stock": 34,
    "healthy": 245,
    "dead_stock": 45,
    "total_dead_stock_value": 23450000
  },
  "top_10_low_stock": [
    {
      "product_id": "11111",
      "product_title": "Áo thun trắng",
      "variant_id": "22222",
      "variant_title": "M",
      "sku": "ATT-M",
      "qty_available": 2,
      "daily_sales_rate": 3.1,
      "days_of_stock": 0.6
    }
  ],
  "top_10_dead_stock": [
    {
      "product_id": "33333",
      "product_title": "Áo len đỏ",
      "variant_id": "44444",
      "variant_title": "XXL",
      "sku": "ALD-XXL",
      "qty_available": 45,
      "last_sale_days_ago": 127,
      "estimated_value": 17955000
    }
  ]
}
```

**Claude tự làm từ output này**:
- **Stock-out Rate**: `out_of_stock / total_variants_analyzed × 100`
- **Dead Stock %**: `dead_stock / total_variants_analyzed × 100`
- **Revenue loss từ stock-out**: `out_of_stock_count × AOV × daily_sales_rate_avg` (ước tính)
- **Capital tied up**: `total_dead_stock_value` — trực tiếp từ output
- **Action prioritization**: Sort top_10_low_stock by `days_of_stock` → item nào hết trước
- **Dead stock action matrix**: Value cao + days_ago lâu → flash sale. Value thấp → bundle/thanh lý
- **Urgency flag**: `days_of_stock < 3` = CRITICAL (hết trước cuối tuần)

**Khi nào DÙNG**: Sức khỏe tổng thể tồn kho, phát hiện SKU hết hàng, dead stock, trước khi ra quyết định nhập hàng.

**Khi nào KHÔNG dùng**: Khi cần kế hoạch nhập cụ thể (qty, timing) → `hrv_stock_reorder_plan`. Khi cần tồn kho theo từng location → `hrv_inventory_imbalance` hoặc `haravan_inventory_locations`.

---

### 6. hrv_stock_reorder_plan

**Mục đích**: Tính toán kế hoạch nhập hàng cụ thể — DSR, reorder point, suggested quantity — cho tất cả variants sắp hết. Server tổng hợp sales history + current stock.

**Input params**:
| Param | Type | Default | Mô tả |
|-------|------|---------|-------|
| `lead_time_days` | integer | 7 | Số ngày từ lúc đặt hàng đến khi nhận hàng |
| `safety_factor` | float | 1.3 | Hệ số an toàn (1.3 = buffer 30%) |
| `date_range_days` | integer | 30 | Số ngày gần nhất để tính DSR |

**Output — cấu trúc JSON**:
```json
{
  "generated_at": "2026-04-06",
  "params": { "lead_time_days": 7, "safety_factor": 1.3, "date_range_days": 30 },
  "reorder_plan": [
    {
      "product_id": "11111",
      "product_title": "Áo thun trắng",
      "variant_id": "22222",
      "variant_title": "M",
      "sku": "ATT-M",
      "qty_available": 2,
      "daily_sales_rate": 3.1,
      "days_of_stock": 0.6,
      "reorder_point": 28,
      "reorder_qty_suggested": 65,
      "urgency": "CRITICAL"
    },
    {
      "product_id": "55555",
      "product_title": "Quần jean đen",
      "variant_id": "66666",
      "variant_title": "32",
      "sku": "QJD-32",
      "qty_available": 0,
      "daily_sales_rate": 2.4,
      "days_of_stock": 0,
      "reorder_point": 22,
      "reorder_qty_suggested": 49,
      "urgency": "CRITICAL"
    }
  ]
}
```

**Claude tự làm từ output này**:
- **Urgency classification**: `days_of_stock < lead_time` = URGENT (sẽ hết trước khi hàng về). `days_of_stock < 2×lead_time` = SOON
- **Total reorder investment estimate**: `SUM(reorder_qty_suggested × price)` — nếu có price từ products_list
- **Priority order**: Sort by `days_of_stock` ASC (hết sớm nhất trước)
- **Supplier grouping**: Group by vendor/supplier để gom đơn đặt hàng (từ products data)
- **Weekly restock schedule**: Items với dos 7-14 ngày → đặt trong tuần này

**Khi nào DÙNG**: Câu hỏi "cần nhập gì", "khi nào nhập", "bao nhiêu units", đề xuất PO.

**Khi nào KHÔNG dùng**: Tổng quan kho (healthy/dead/out) → `hrv_inventory_health`. Cân bằng giữa các kho → `hrv_inventory_imbalance`.

---

### 7. hrv_inventory_imbalance

**Mục đích**: Phát hiện mất cân bằng tồn kho giữa nhiều chi nhánh/kho, đề xuất chuyển hàng. Cần fetch inventory data từ tất cả locations cho mỗi variant.

**Input params**: Không có params (tự động fetch toàn bộ multi-location data).

**Output — cấu trúc JSON**:
```json
{
  "generated_at": "2026-04-06",
  "total_imbalanced_variants": 8,
  "imbalanced_variants": [
    {
      "product_id": "11111",
      "product_title": "Áo thun trắng",
      "variant_id": "22222",
      "variant_title": "M",
      "sku": "ATT-M",
      "imbalance_ratio": 23.5,
      "total_qty": 49,
      "locations": [
        { "location_id": "loc_hn", "location_name": "Kho Hà Nội", "qty": 47 },
        { "location_id": "loc_hcm", "location_name": "Kho HCM", "qty": 2 }
      ],
      "suggested_transfer": {
        "from_location_id": "loc_hn",
        "from_location_name": "Kho Hà Nội",
        "to_location_id": "loc_hcm",
        "to_location_name": "Kho HCM",
        "qty": 22
      }
    }
  ]
}
```

**Claude tự làm từ output này**:
- **Total transfers needed**: Count + tổng qty transfers
- **Priority ranking**: Sort by `imbalance_ratio DESC` — imbalance cao nhất cần xử lý trước
- **Transfer value estimate**: `suggested_transfer.qty × price` per item
- **Operational grouping**: Group transfers by from_location → gom thành 1 lần vận chuyển
- **Urgency layer**: Cross-reference với hrv_inventory_health — variant nào vừa imbalanced VỪA low_stock ở destination = URGENT

**Khi nào DÙNG**: Shop có nhiều kho/chi nhánh, tối ưu phân bổ tồn kho, giảm stock-out tại điểm bán nóng.

**Khi nào KHÔNG dùng**: Shop chỉ có 1 kho (tool không có ý nghĩa). Khi cần kế hoạch nhập từ supplier → `hrv_stock_reorder_plan`.

---

## PHẦN II: BASE TOOLS — Nhóm theo danh mục

Base tools mapping 1:1 với Haravan REST API. Dùng cho tra cứu cụ thể, drill-down sau smart tool, và thực hiện hành động (write).

---

### ORDERS (Đơn hàng)

| Tool | Params chính | Dùng khi |
|------|-------------|---------|
| `haravan_orders_list` | `status`, `date_from`, `date_to`, `limit`, `page`, `fields` | Drill-down đơn hàng, filter theo province/gateway/status cụ thể, COD analysis |
| `haravan_orders_get` | `order_id` | Chi tiết 1 đơn: line_items, shipping, transactions, timeline |
| `haravan_orders_create` | order object | Tạo đơn mới (rare, thường qua storefront) |
| `haravan_orders_update` | `order_id`, fields update | Cập nhật thông tin đơn |
| `haravan_orders_confirm` | `order_id` | **[Write]** Xác nhận đơn pending → processing |
| `haravan_orders_cancel` | `order_id`, `reason` (customer/inventory/fraud/declined) | **[Write]** Hủy đơn |
| `haravan_orders_close` | `order_id` | **[Write]** Đóng đơn hoàn thành |
| `haravan_orders_open` | `order_id` | **[Write]** Mở lại đơn đã đóng |
| `haravan_orders_assign` | `order_id`, `location_id` | **[Write]** Gán đơn cho kho/chi nhánh |

**Lưu ý quan trọng**: `haravan_orders_list` có pagination (100 items/page). Không dùng để đếm/tổng hợp — dùng `hrv_orders_summary`. Chỉ dùng khi cần filter chi tiết mà smart tool không cung cấp (VD: orders by province, by gateway_code cụ thể).

**Fields param tối ưu** cho các use case:
- COD analysis: `fields=id,financial_status,gateway_code,shipping_address,total_price,cancelled_at`
- Geographic analysis: `fields=id,shipping_address,total_price,financial_status,source_name`
- Timeline: `fields=id,created_at,confirmed_at,closed_at,financial_status`

---

### TRANSACTIONS (Giao dịch)

| Tool | Params chính | Dùng khi |
|------|-------------|---------|
| `haravan_transactions_list` | `order_id` | Xem lịch sử giao dịch của 1 đơn |
| `haravan_transactions_get` | `order_id`, `transaction_id` | Chi tiết 1 giao dịch |
| `haravan_transactions_create` | `order_id`, `kind` (Capture/Refund), `amount` | **[Write]** Ghi nhận thanh toán, hoàn tiền |

---

### PRODUCTS (Sản phẩm)

| Tool | Params chính | Dùng khi |
|------|-------------|---------|
| `haravan_products_list` | `limit`, `page`, `fields`, `product_type`, `vendor` | Browse catalog, catalog health scoring (fetch all) |
| `haravan_products_count` | filters | Đếm sản phẩm |
| `haravan_products_get` | `product_id` | Chi tiết đầy đủ: images, variants, SEO, metafields |
| `haravan_products_create` | product object | **[Write]** Tạo sản phẩm mới |
| `haravan_products_update` | `product_id`, fields | **[Write]** Cập nhật sản phẩm |
| `haravan_products_delete` | `product_id` | **[Write]** Xóa sản phẩm |

**Fields param cho catalog health scoring**:
`fields=id,title,body_html,product_type,vendor,tags,images,variants,status`

---

### VARIANTS (Biến thể)

| Tool | Params chính | Dùng khi |
|------|-------------|---------|
| `haravan_variants_list` | `product_id` | Danh sách variants của 1 sản phẩm |
| `haravan_variants_count` | `product_id` | Đếm variants |
| `haravan_variants_get` | `product_id`, `variant_id` | Chi tiết variant: price, SKU, barcode, inventory |
| `haravan_variants_create` | `product_id`, variant object | **[Write]** Thêm variant mới |
| `haravan_variants_update` | `product_id`, `variant_id`, fields | **[Write]** Cập nhật variant (giá, SKU, barcode) |

---

### CUSTOMERS (Khách hàng)

| Tool | Params chính | Dùng khi |
|------|-------------|---------|
| `haravan_customers_list` | `limit`, `page`, `fields` | Browse khách hàng, geographic analysis |
| `haravan_customers_search` | `query` (tên/email/SĐT) | Tìm khách cụ thể |
| `haravan_customers_count` | filters | Đếm khách |
| `haravan_customers_get` | `customer_id` | Chi tiết khách: orders, addresses, tags |
| `haravan_customers_create` | customer object | **[Write]** Tạo khách mới |
| `haravan_customers_update` | `customer_id`, fields | **[Write]** Cập nhật thông tin khách |
| `haravan_customers_delete` | `customer_id` | **[Write]** Xóa khách (cẩn thận!) |
| `haravan_customers_groups` | — | Danh sách nhóm khách hàng |
| `haravan_customer_addresses_list` | `customer_id` | Địa chỉ của khách |

---

### INVENTORY (Tồn kho)

| Tool | Params chính | Dùng khi |
|------|-------------|---------|
| `haravan_inventory_adjustments_list` | `location_id`, `date` | Lịch sử điều chỉnh tồn kho |
| `haravan_inventory_adjustments_get` | `adjustment_id` | Chi tiết 1 lần điều chỉnh |
| `haravan_inventory_adjust_or_set` | `variant_id`, `location_id`, `qty`, `type` (adjust/set) | **[Write]** Điều chỉnh hoặc set tồn kho tuyệt đối |
| `haravan_inventory_locations` | `variant_id` | Tồn kho của 1 variant tại tất cả locations |

---

### SHOP & LOCATIONS (Cửa hàng)

| Tool | Params chính | Dùng khi |
|------|-------------|---------|
| `haravan_shop_get` | — | Thông tin shop: name, domain, currency, timezone, plan |
| `haravan_locations_list` | — | Danh sách kho/chi nhánh |
| `haravan_locations_get` | `location_id` | Chi tiết 1 location |
| `haravan_users_list` | — | Danh sách staff/user (cần Plus plan) |
| `haravan_users_get` | `user_id` | Chi tiết 1 user |
| `haravan_shipping_rates_get` | — | Phương thức vận chuyển đang cấu hình |

---

### CONTENT (Nội dung)

| Tool | Params chính | Dùng khi |
|------|-------------|---------|
| `haravan_pages_list` | — | Danh sách trang tĩnh |
| `haravan_pages_get` | `page_id` | Chi tiết trang |
| `haravan_pages_create` | page object | **[Write]** Tạo trang mới |
| `haravan_pages_update` | `page_id`, fields | **[Write]** Cập nhật trang |
| `haravan_pages_delete` | `page_id` | **[Write]** Xóa trang |
| `haravan_blogs_list` | — | Danh sách blogs |
| `haravan_blogs_get` | `blog_id` | Chi tiết blog |
| `haravan_articles_list` | `blog_id` | Danh sách bài viết trong blog |
| `haravan_articles_get` | `blog_id`, `article_id` | Chi tiết bài viết |
| `haravan_articles_create` | `blog_id`, article object | **[Write]** Tạo bài viết mới |
| `haravan_articles_update` | `blog_id`, `article_id`, fields | **[Write]** Cập nhật bài viết |
| `haravan_articles_delete` | `blog_id`, `article_id` | **[Write]** Xóa bài viết |
| `haravan_script_tags_list` | — | Danh sách script tags |
| `haravan_script_tags_create` | `src`, `event` | **[Write]** Thêm script tag |
| `haravan_script_tags_delete` | `script_tag_id` | **[Write]** Xóa script tag |

---

### WEBHOOKS

| Tool | Params chính | Dùng khi |
|------|-------------|---------|
| `haravan_webhooks_list` | — | Danh sách webhooks đang active |
| `haravan_webhooks_get` | `webhook_id` | Chi tiết 1 webhook |
| `haravan_webhooks_create` | `topic`, `address`, `format` | **[Write]** Tạo webhook mới |
| `haravan_webhooks_update` | `webhook_id`, fields | **[Write]** Cập nhật webhook |
| `haravan_webhooks_delete` | `webhook_id` | **[Write]** Xóa webhook |

---

## PHẦN III: Bảng quyết định — Smart Tool vs Base Tool vs Claude tự làm

| Phân tích cần làm | Cách tối ưu |
|-------------------|------------|
| Tổng DT, số đơn, AOV | `hrv_orders_summary` — 1 call |
| Channel breakdown (web/POS/mobile) | **Claude tự tính** từ `hrv_orders_summary.orders_by_source` |
| Cancel rate + lý do hủy | **Claude tự tính** từ `hrv_orders_summary` |
| ODR (Order Defect Rate) | **Claude tự tính** từ `hrv_orders_summary` |
| Discount penetration + depth | **Claude tự tính** từ `hrv_orders_summary.discount_usage` |
| Sản phẩm bán chạy nhất | `hrv_top_products` — 1 call |
| Tốc độ xử lý đơn, đơn kẹt | `hrv_order_cycle_time` — 1 call |
| RFM segments, retention rate | `hrv_customer_segments` — 1 call |
| Repeat purchase rate | **Claude tự tính** từ `hrv_customer_segments` |
| Tổng quan sức khỏe kho | `hrv_inventory_health` — 1 call |
| Kế hoạch nhập hàng cụ thể | `hrv_stock_reorder_plan` — 1 call |
| Cân bằng kho đa chi nhánh | `hrv_inventory_imbalance` — 1 call |
| Catalog health score | **Claude tự score** từ `haravan_products_list` |
| Geographic revenue | **Claude tự group** từ `haravan_orders_list` (filter fields) |
| COD fail rate tổng quát | **Claude tự ước tính** từ `hrv_orders_summary` |
| COD fail rate by province | `haravan_orders_list` + Claude filter/group |
| Per-code discount ROI | `haravan_orders_list` + Claude parse discount_codes |
| Operations scorecard (all dims) | 4 smart tools song song + **Claude chấm điểm** |

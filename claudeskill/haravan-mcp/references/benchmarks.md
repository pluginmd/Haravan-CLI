# Benchmarks & Scoring — E-commerce Việt Nam

---

## I. Operations Scorecard — Bảng chấm điểm 10 chiều

| Chiều | 9-10 (Xuất sắc) | 7-8 (Tốt) | 5-6 (TB) | 3-4 (Yếu) | 1-2 (Nghiêm trọng) |
|-------|-----------------|------------|----------|------------|---------------------|
| Revenue Growth (MoM) | >20% | 5-20% | 0-5% | -10 đến 0% | <-10% |
| AOV | >500k VND | 300-500k | 200-300k | 100-200k | <100k |
| Processing Speed (median confirm) | <2h | 2-4h | 4-8h | 8-24h | >24h |
| Fulfillment Rate | >98% | 95-98% | 90-95% | 80-90% | <80% |
| Cancel Rate | <1% | 1-3% | 3-5% | 5-10% | >10% |
| Stock-out Rate (% SKU) | <2% | 2-5% | 5-10% | 10-20% | >20% |
| Dead Stock (% SKU) | <2% | 2-5% | 5-10% | 10-15% | >15% |
| Repeat Purchase Rate | >40% | 30-40% | 20-30% | 10-20% | <10% |
| Catalog Health Score | >85 | 70-85 | 55-70 | 40-55 | <40 |
| Collection Rate | >98% | 95-98% | 90-95% | 80-90% | <80% |

### Công thức tổng điểm (có trọng số)

```
Total = (Revenue_Growth×2 + AOV×1 + Processing×1.5 + Fulfillment×2
       + Cancel×1.5 + Stock_out×2 + Dead_Stock×1 + RPR×1.5
       + Catalog×1 + Collection×1.5) / 15

Kết quả: X.X / 10
```

**Trọng số cao** (×2): Revenue Growth, Fulfillment Rate, Stock-out Rate — ảnh hưởng trực tiếp đến doanh thu
**Trọng số TB** (×1.5): Processing Speed, Cancel Rate, RPR, Collection — ảnh hưởng vận hành
**Trọng số thấp** (×1): AOV, Dead Stock, Catalog Health — quan trọng nhưng ít khẩn cấp

---

## II. Benchmark chi tiết theo metric

### Đơn hàng

| Metric | Công thức | Xuất sắc | Tốt | TB | Yếu | Nghiêm trọng |
|--------|----------|----------|------|-----|------|---------------|
| Cancel Rate | cancelled / total × 100% | <1% | 1-3% | 3-5% | 5-10% | >10% |
| ODR (Order Defect Rate) | (cancelled + refunded) / total × 100% | <1% | 1-3% | 3-5% | 5-8% | >8% |
| Time-to-Confirm (median) | confirmed_at − created_at | <2h | 2-4h | 4-8h | 8-24h | >24h |
| Time-to-Close (median) | closed_at − created_at | <24h | 24-48h | 48-72h | 72-120h | >120h |
| Time-to-Close (p90) | p90 | <48h | 48-72h | 72-96h | 96-168h | >168h |
| Fulfillment Rate | fulfilled / (total − cancelled) × 100% | >98% | 95-98% | 90-95% | 80-90% | <80% |
| Collection Rate | paid / (total − cancelled) × 100% | >98% | 95-98% | 90-95% | 80-90% | <80% |

### COD

| Metric | Công thức | Tốt | Cảnh báo | Nguy hiểm | Dừng COD |
|--------|----------|------|----------|-----------|----------|
| COD Fail Rate (tổng) | cancelled_cod / total_cod × 100% | <10% | 10-15% | 15-25% | >25% |
| COD Fail Rate (tỉnh) | cancelled_cod_province / total_cod_province | <15% | 15-20% | 20-30% | >30% |
| COD % tổng đơn | cod_orders / total × 100% | <40% | 40-60% | 60-80% | >80% |

### Tồn kho

| Metric | Công thức | Xuất sắc | Tốt | TB | Yếu | Nghiêm trọng |
|--------|----------|----------|------|-----|------|---------------|
| Stock-out Rate | out_of_stock / total_variants × 100% | <2% | 2-5% | 5-10% | 10-20% | >20% |
| Dead Stock % | dead_stock / total_variants × 100% | <2% | 2-5% | 5-10% | 10-15% | >15% |
| Inventory Turnover | revenue / avg_inventory_value (annual) | >12x | 8-12x | 4-8x | 2-4x | <2x |
| Shrinkage Rate | shrinkage_qty / total_onhand × 100% | <0.5% | 0.5-1% | 1-2% | 2-5% | >5% |
| Days of Stock (avg) | qty_available / daily_sales_rate | 14-30 | 30-60 | 7-14 hoặc 60-90 | <7 hoặc >90 | <3 hoặc >120 |

### Khách hàng

| Metric | Công thức | Xuất sắc | Tốt | TB | Yếu | Nghiêm trọng |
|--------|----------|----------|------|-----|------|---------------|
| Repeat Purchase Rate | (total − new − lost) / total × 100% | >40% | 30-40% | 20-30% | 10-20% | <10% |
| Customer Concentration (top 5%) | top5%_revenue / total_revenue × 100% | <30% | 30-40% | 40-50% | 50-70% | >70% |
| New vs Returning Revenue | returning / total × 100% | >60% | 50-60% | 40-50% | 30-40% | <30% |

### Sản phẩm & Marketing

| Metric | Công thức | Lành mạnh | Cẩn thận | Nguy hiểm |
|--------|----------|-----------|----------|-----------|
| Discount Penetration | orders_with_discount / total × 100% | 5-15% | 15-30% | >40% |
| Discount Depth (avg) | avg(discount / subtotal) × 100% | <10% | 10-25% | >25% |
| Revenue Concentration (top 3 SP) | top3_revenue / total × 100% | <30% | 30-50% | >50% |
| Catalog Health Score | 12 tiêu chí × 100 điểm | >85 | 60-85 | <60 |

### Doanh thu

| Metric | Công thức | Xuất sắc | Tốt | TB | Yếu | Nghiêm trọng |
|--------|----------|----------|------|-----|------|---------------|
| Revenue Growth (MoM) | (current − prior) / prior × 100% | >20% | 5-20% | 0-5% | -10 đến 0% | <-10% |
| AOV | total_revenue / total_paid_orders | >500k | 300-500k | 200-300k | 100-200k | <100k |

---

## III. Context thị trường Việt Nam

### Đặc thù COD
- **60-70% đơn hàng online là COD** — cao hơn nhiều so với Đông Nam Á (30-40%)
- COD fail rate trung bình ngành: 15-20%
- Tỉnh xa (Tây Bắc, Tây Nguyên) fail rate có thể >30%
- HCM & HN: fail rate thường 8-12%

### Kênh bán hàng
- **POS AOV thường cao hơn Web 30-50%** — khách mua trực tiếp upsell tốt hơn
- Mobile conversion rate thấp nhất nhưng đang tăng nhanh
- Web vẫn chiếm 50-65% tổng doanh thu

### Mùa vụ
- **Tết** (tháng 1-2): spike 200-400% — đặc biệt ngành thực phẩm, quà tặng
- **11.11**: spike 150-300% — fashion, beauty, electronics
- **12.12**: spike 100-200%
- **Black Friday**: spike 80-150% — chủ yếu electronics
- **Tháng 3-4**: thường thấp nhất năm (sau Tết)

### Vùng miền
- **HCM + HN**: 50-60% tổng doanh thu, AOV cao hơn 20-30% vs tỉnh
- **Đồng bằng sông Cửu Long**: volume cao nhưng AOV thấp, COD fail cao
- **Đà Nẵng/Miền Trung**: AOV trung bình nhưng cancel rate thấp nhất
- **Tây Nguyên/Tây Bắc**: volume thấp, shipping cost cao, COD fail cao

---

## IV. Cách dùng benchmarks trong phân tích

### Khi trình bày metric
```
✅ Tốt: "Cancel rate 2.1% ✅ (benchmark: <3% = tốt)"
⚠️ Cảnh báo: "Stock-out 8.5% ⚠️ (benchmark: >5% = cần cải thiện)"
🔴 Nghiêm trọng: "Dead stock 18% 🔴 (benchmark: >15% = nghiêm trọng)"
```

### Khi so sánh kỳ trước
```
"AOV 420k ↑12% vs kỳ trước (benchmark: >300k = tốt ✅)"
"Cancel rate 6.2% ↑1.8 điểm 🔴 (vượt ngưỡng 5% — cần hành động ngay)"
```

### Khi chấm scorecard
```
Cancel Rate = 2.1% → nằm trong khoảng 1-3% → Score: 8/10 (Tốt)
Stock-out = 12% → nằm trong khoảng 10-20% → Score: 3/10 (Yếu)
```

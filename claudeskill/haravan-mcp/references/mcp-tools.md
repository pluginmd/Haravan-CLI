# Haravan MCP — Danh mục Tool đầy đủ (auto-generated)

_Tài liệu này được sinh tự động từ schema MCP server (70 tool), KHÔNG sửa tay. Regenerate bằng `make skill-tools`._

Kiến trúc 2 lớp:
- **MCP Server** exposed 70 tool: 7 **smart** (`hrv_*`, aggregate server-side) + 63 **base** (`haravan_*`, 1:1 Haravan REST).
- **Claude Skill** (bạn) chọn đúng tool, truyền đúng params, phân tích & diễn giải.

Nguyên tắc: dùng `hrv_*` cho aggregation lớn (>1000 records), dùng `haravan_*` cho detail/action/CRUD.

---

## 🧠 SMART TOOLS — server-side aggregation  _(7 tool)_

### `hrv_customer_segments`

RFM analysis using quintile scoring. Classifies every customer into
Champions, Loyal, Potential_Loyalists, New, At_Risk, Hibernating, Lost,
or Others, and returns counts + revenue metrics + an action suggestion
per segment.

**Scopes:** `com.read_customers`

| Param | Type | Required | Description |
|---|---|---|---|
| `min_orders` | integer |  | Minimum order count required to include a customer (default 0) |

### `hrv_inventory_health`

Analyze inventory across the first 100 products.
Classifies variants as out_of_stock, low_stock, dead_stock (stock but no
sales in the lookback window) or healthy. Returns summary counts, total
dead-stock value, and top-10 lists.

**Scopes:** `com.read_inventories, com.read_products`

| Param | Type | Required | Description |
|---|---|---|---|
| `days_for_dead_stock` | integer |  | Lookback window in days (default 90) |
| `low_stock_threshold` | integer |  | Qty below which a variant is low (default 5) |

### `hrv_inventory_imbalance`

Detect cross-location imbalances (max/min > 5x)

**Scopes:** `com.read_inventories, com.read_products`

_No parameters._

### `hrv_order_cycle_time`

Median/p90 time-to-confirm and time-to-close, plus stuck-order counts

**Scopes:** `com.read_orders`

| Param | Type | Required | Description |
|---|---|---|---|
| `date_from` | string (ISO 8601) |  | Start date ISO 8601 (default 30 days ago) |
| `date_to` | string (ISO 8601) |  | End date ISO 8601 (default now) |

### `hrv_orders_summary`

Aggregate revenue/AOV/status breakdown with optional prior-period comparison

**Scopes:** `com.read_orders`

| Param | Type | Required | Description |
|---|---|---|---|
| `compare_prior` | boolean |  | Fetch prior equal-length window and add comparison (default true) |
| `date_from` | string (ISO 8601) |  | Start date ISO 8601 (default: 30 days ago) |
| `date_to` | string (ISO 8601) |  | End date ISO 8601 (default: now) |

### `hrv_stock_reorder_plan`

Reorder plan based on daily sales rate, lead time and safety factor

**Scopes:** `com.read_inventories, com.read_products`

| Param | Type | Required | Description |
|---|---|---|---|
| `date_range_days` | integer |  | Days of order history for DSR (default 30) |
| `lead_time_days` | integer |  | Supplier lead time in days (default 7) |
| `safety_factor` | string |  | Safety buffer multiplier (default 1.3) |

### `hrv_top_products`

Top N products by revenue with variant-level breakdown

**Scopes:** `com.read_orders`

| Param | Type | Required | Description |
|---|---|---|---|
| `date_from` | string (ISO 8601) |  | Start date ISO 8601 (default 30 days ago) |
| `date_to` | string (ISO 8601) |  | End date ISO 8601 (default now) |
| `top_n` | integer |  | Number of top products (default 10) |

---

## 📦 ORDERS  _(13 tool)_

### `haravan_orders_assign`

Assign staff to an order

**Scopes:** `com.write_orders`

| Param | Type | Required | Description |
|---|---|---|---|
| `order_id` | integer | **yes** | Order ID |
| `user_id` | integer | **yes** | Staff user ID to assign |

### `haravan_orders_cancel`

Cancel an order. Reason must be one of: customer, fraud, inventory, declined, other.

**Scopes:** `com.write_orders`

| Param | Type | Required | Description |
|---|---|---|---|
| `order_id` | integer | **yes** | Order ID to cancel |
| `email` | boolean |  | Send cancellation email |
| `reason` | string |  | Cancellation reason _(one of: customer, fraud, inventory, declined, other)_ |
| `restock` | boolean |  | Restock cancelled items |

### `haravan_orders_close`

Close an order

**Scopes:** `com.write_orders`

| Param | Type | Required | Description |
|---|---|---|---|
| `order_id` | integer | **yes** | Order ID |

### `haravan_orders_confirm`

Confirm an order

**Scopes:** `com.write_orders`

| Param | Type | Required | Description |
|---|---|---|---|
| `order_id` | integer | **yes** | Order ID |

### `haravan_orders_count`

Get total order count with optional filters

**Scopes:** `com.read_orders`

| Param | Type | Required | Description |
|---|---|---|---|
| `created_at_max` | string (ISO 8601) |  | Created before (ISO 8601) |
| `created_at_min` | string (ISO 8601) |  | Created after (ISO 8601) |
| `financial_status` | string |  | Financial status filter _(one of: pending, authorized, partially_paid, paid, partially_refunded, refunded, voided, any)_ |
| `fulfillment_status` | string |  | Fulfillment status filter _(one of: fulfilled, partial, unshipped, any)_ |
| `status` | string |  | Order status filter _(one of: open, closed, cancelled, any)_ |
| `updated_at_max` | string (ISO 8601) |  | Updated before (ISO 8601) |
| `updated_at_min` | string (ISO 8601) |  | Updated after (ISO 8601) |

### `haravan_orders_create`

Create a new order. Pass the Haravan order payload via --body as inline JSON or @file.
The body MUST be wrapped as {"order": {...}}: line_items is required, billing/shipping
addresses, tags, discount_codes, note, source_name are all supported.

**Scopes:** `com.write_orders`

| Param | Type | Required | Description |
|---|---|---|---|
| `body` | any | **yes** | Haravan order payload, e.g. {"order":{"line_items":[...]}} |

### `haravan_orders_get`

Get a single order by ID. Returns full order details including line_items, shipping, billing, transactions.

**Scopes:** `com.read_orders`

| Param | Type | Required | Description |
|---|---|---|---|
| `order_id` | integer | **yes** | Order ID |
| `fields` | string |  | Comma-separated fields |

### `haravan_orders_list`

List orders. Filter by status, financial_status, fulfillment_status, created_at, updated_at. Supports pagination.

**Scopes:** `com.read_orders`

| Param | Type | Required | Description |
|---|---|---|---|
| `created_at_max` | string (ISO 8601) |  | Created before (ISO 8601) |
| `created_at_min` | string (ISO 8601) |  | Created after (ISO 8601) |
| `fetch_all` | boolean |  | Auto-paginate until all records are returned |
| `fields` | string |  | Comma-separated fields to include |
| `financial_status` | string |  | Financial status filter _(one of: pending, authorized, partially_paid, paid, partially_refunded, refunded, voided, any)_ |
| `fulfillment_status` | string |  | Fulfillment status filter _(one of: fulfilled, partial, unshipped, any)_ |
| `limit` | integer |  | Results per page (default 50, max 250) |
| `page` | integer |  | Page number (default 1) |
| `since_id` | integer |  | Results after this ID |
| `status` | string |  | Order status filter _(one of: open, closed, cancelled, any)_ |
| `updated_at_max` | string (ISO 8601) |  | Updated before (ISO 8601) |
| `updated_at_min` | string (ISO 8601) |  | Updated after (ISO 8601) |

### `haravan_orders_open`

Reopen a closed order

**Scopes:** `com.write_orders`

| Param | Type | Required | Description |
|---|---|---|---|
| `order_id` | integer | **yes** | Order ID |

### `haravan_orders_update`

Update an existing order (note, tags, shipping_address, email, ...). Pass the payload via --body.

**Scopes:** `com.write_orders`

| Param | Type | Required | Description |
|---|---|---|---|
| `body` | any | **yes** | Haravan order patch, e.g. {"order":{"note":"..."}} |
| `order_id` | integer | **yes** | Order ID |

### `haravan_transactions_create`

Create a transaction. Kind: Pending | Authorization | Sale | Capture | Void | Refund. Pass via --body.

**Scopes:** `com.write_orders`

| Param | Type | Required | Description |
|---|---|---|---|
| `body` | any | **yes** | Transaction payload, e.g. {"transaction":{"amount":100,"kind":"Sale"}} |
| `order_id` | integer | **yes** | Order ID |

### `haravan_transactions_get`

Get a specific transaction

**Scopes:** `com.read_orders`

| Param | Type | Required | Description |
|---|---|---|---|
| `order_id` | integer | **yes** | Order ID |
| `transaction_id` | integer | **yes** | Transaction ID |

### `haravan_transactions_list`

List all transactions for an order

**Scopes:** `com.read_orders`

| Param | Type | Required | Description |
|---|---|---|---|
| `order_id` | integer | **yes** | Order ID |

---

## 🛒 PRODUCTS & VARIANTS  _(11 tool)_

### `haravan_products_count`

Get total product count with optional filters

**Scopes:** `com.read_products`

| Param | Type | Required | Description |
|---|---|---|---|
| `collection_id` | integer |  | Filter by collection |
| `created_at_max` | string (ISO 8601) |  | Created before (ISO 8601) |
| `created_at_min` | string (ISO 8601) |  | Created after (ISO 8601) |
| `handle` | string |  | Filter by handle (URL slug) |
| `product_type` | string |  | Filter by product type |
| `published_at_max` | string (ISO 8601) |  | Published before (ISO 8601) |
| `published_at_min` | string (ISO 8601) |  | Published after (ISO 8601) |
| `published_status` | string |  | Published status _(one of: published, unpublished, any)_ |
| `updated_at_max` | string (ISO 8601) |  | Updated before (ISO 8601) |
| `updated_at_min` | string (ISO 8601) |  | Updated after (ISO 8601) |
| `vendor` | string |  | Filter by vendor |

### `haravan_products_create`

Create a product. Pass the full payload via --body (e.g. {"product":{"title":"T","variants":[...]}}). Title is required.

**Scopes:** `com.write_products`

| Param | Type | Required | Description |
|---|---|---|---|
| `body` | any | **yes** | Haravan product payload (wrapped or bare) |

### `haravan_products_delete`

Delete a product by ID

**Scopes:** `com.write_products`

| Param | Type | Required | Description |
|---|---|---|---|
| `product_id` | integer | **yes** | Product ID to delete |

### `haravan_products_get`

Get a product by ID. Returns full details including variants, images, and options.

**Scopes:** `com.read_products`

| Param | Type | Required | Description |
|---|---|---|---|
| `product_id` | integer | **yes** | Product ID |
| `fields` | string |  | Comma-separated fields |

### `haravan_products_list`

List all products with pagination and filtering (collection, type, vendor, handle, publish state, …).

**Scopes:** `com.read_products`

| Param | Type | Required | Description |
|---|---|---|---|
| `collection_id` | integer |  | Filter by collection |
| `created_at_max` | string (ISO 8601) |  | Created before (ISO 8601) |
| `created_at_min` | string (ISO 8601) |  | Created after (ISO 8601) |
| `fetch_all` | boolean |  | Auto-paginate until all results are returned |
| `fields` | string |  | Comma-separated fields |
| `handle` | string |  | Filter by handle (URL slug) |
| `limit` | integer |  | Results per page (max 250) |
| `page` | integer |  | Page number |
| `product_type` | string |  | Filter by product type |
| `published_at_max` | string (ISO 8601) |  | Published before (ISO 8601) |
| `published_at_min` | string (ISO 8601) |  | Published after (ISO 8601) |
| `published_status` | string |  | Published status _(one of: published, unpublished, any)_ |
| `since_id` | integer |  | Results after this ID |
| `updated_at_max` | string (ISO 8601) |  | Updated before (ISO 8601) |
| `updated_at_min` | string (ISO 8601) |  | Updated after (ISO 8601) |
| `vendor` | string |  | Filter by vendor |

### `haravan_products_update`

Update an existing product

**Scopes:** `com.write_products`

| Param | Type | Required | Description |
|---|---|---|---|
| `body` | any | **yes** | Haravan product patch (wrapped or bare) |
| `product_id` | integer | **yes** | Product ID |

### `haravan_variants_count`

Count variants of a product

**Scopes:** `com.read_products`

| Param | Type | Required | Description |
|---|---|---|---|
| `product_id` | integer | **yes** | Product ID |

### `haravan_variants_create`

Create a variant for a product

**Scopes:** `com.write_products`

| Param | Type | Required | Description |
|---|---|---|---|
| `body` | any | **yes** | Variant payload, e.g. {"variant":{"price":99000,"sku":"X"}} |
| `product_id` | integer | **yes** | Product ID |

### `haravan_variants_get`

Fetches /com/variants/{variant_id}.json — no product_id required.

**Scopes:** `com.read_products`

| Param | Type | Required | Description |
|---|---|---|---|
| `variant_id` | integer | **yes** | Variant ID |
| `fields` | string |  | Comma-separated fields |

### `haravan_variants_list`

List variants of a product

**Scopes:** `com.read_products`

| Param | Type | Required | Description |
|---|---|---|---|
| `product_id` | integer | **yes** | Product ID |
| `fields` | string |  | Comma-separated fields |
| `limit` | integer |  | Results per page |
| `page` | integer |  | Page number |

### `haravan_variants_update`

Updates /com/variants/{variant_id}.json. The legacy TS API requires only the variant_id, not the parent product.

**Scopes:** `com.write_products`

| Param | Type | Required | Description |
|---|---|---|---|
| `body` | any | **yes** | Variant patch (wrapped or bare) |
| `variant_id` | integer | **yes** | Variant ID |

---

## 👥 CUSTOMERS & ADDRESSES  _(14 tool)_

### `haravan_customer_addresses_create`

Create a new address for a customer

**Scopes:** `com.write_customers`

| Param | Type | Required | Description |
|---|---|---|---|
| `body` | any | **yes** | Address payload (wrapped or bare) |
| `customer_id` | integer | **yes** | Customer ID |

### `haravan_customer_addresses_delete`

Delete a customer address

**Scopes:** `com.write_customers`

| Param | Type | Required | Description |
|---|---|---|---|
| `address_id` | integer | **yes** | Address ID |
| `customer_id` | integer | **yes** | Customer ID |

### `haravan_customer_addresses_get`

Get a specific address of a customer

**Scopes:** `com.read_customers`

| Param | Type | Required | Description |
|---|---|---|---|
| `address_id` | integer | **yes** | Address ID |
| `customer_id` | integer | **yes** | Customer ID |

### `haravan_customer_addresses_list`

List addresses of a customer

**Scopes:** `com.read_customers`

| Param | Type | Required | Description |
|---|---|---|---|
| `customer_id` | integer | **yes** | Customer ID |
| `limit` | integer |  | Results per page |
| `page` | integer |  | Page number |

### `haravan_customer_addresses_set_default`

Set a customer address as default

**Scopes:** `com.write_customers`

| Param | Type | Required | Description |
|---|---|---|---|
| `address_id` | integer | **yes** | Address ID to set as default |
| `customer_id` | integer | **yes** | Customer ID |

### `haravan_customer_addresses_update`

Update a customer address

**Scopes:** `com.write_customers`

| Param | Type | Required | Description |
|---|---|---|---|
| `address_id` | integer | **yes** | Address ID |
| `body` | any | **yes** | Address patch (wrapped or bare) |
| `customer_id` | integer | **yes** | Customer ID |

### `haravan_customers_count`

Count customers with date filters

**Scopes:** `com.read_customers`

| Param | Type | Required | Description |
|---|---|---|---|
| `created_at_max` | string (ISO 8601) |  | Created before (ISO 8601) |
| `created_at_min` | string (ISO 8601) |  | Created after (ISO 8601) |
| `updated_at_max` | string (ISO 8601) |  | Updated before (ISO 8601) |
| `updated_at_min` | string (ISO 8601) |  | Updated after (ISO 8601) |

### `haravan_customers_create`

Create a customer. Email OR phone is required. Pass the payload via --body (e.g. {"customer":{"email":"x@y.z"}}).

**Scopes:** `com.write_customers`

| Param | Type | Required | Description |
|---|---|---|---|
| `body` | any | **yes** | Customer payload (wrapped or bare) |

### `haravan_customers_delete`

Delete a customer by ID

**Scopes:** `com.write_customers`

| Param | Type | Required | Description |
|---|---|---|---|
| `customer_id` | integer | **yes** | Customer ID |

### `haravan_customers_get`

Get a single customer by ID

**Scopes:** `com.read_customers`

| Param | Type | Required | Description |
|---|---|---|---|
| `customer_id` | integer | **yes** | Customer ID |
| `fields` | string |  | Comma-separated fields |

### `haravan_customers_groups`

List all customer groups

**Scopes:** `com.read_customers`

_No parameters._

### `haravan_customers_list`

List customers with pagination and date filters.

**Scopes:** `com.read_customers`

| Param | Type | Required | Description |
|---|---|---|---|
| `created_at_max` | string (ISO 8601) |  | Created before (ISO 8601) |
| `created_at_min` | string (ISO 8601) |  | Created after (ISO 8601) |
| `fetch_all` | boolean |  | Auto-paginate to fetch every page |
| `fields` | string |  | Comma-separated fields |
| `limit` | integer |  | Results per page (max 250) |
| `page` | integer |  | Page number |
| `since_id` | integer |  | Results after this ID |
| `updated_at_max` | string (ISO 8601) |  | Updated before (ISO 8601) |
| `updated_at_min` | string (ISO 8601) |  | Updated after (ISO 8601) |

### `haravan_customers_search`

Search customers (email, phone, name, …)

**Scopes:** `com.read_customers`

| Param | Type | Required | Description |
|---|---|---|---|
| `query` | string | **yes** | Search query |
| `fields` | string |  | Comma-separated fields |
| `limit` | integer |  | Results per page |
| `page` | integer |  | Page number |

### `haravan_customers_update`

Update an existing customer

**Scopes:** `com.write_customers`

| Param | Type | Required | Description |
|---|---|---|---|
| `body` | any | **yes** | Customer patch (wrapped or bare) |
| `customer_id` | integer | **yes** | Customer ID |

---

## 📊 INVENTORY  _(5 tool)_

### `haravan_inventory_adjust_or_set`

Create an inventory adjustment. type=adjust adds or subtracts quantities,
type=set replaces them. Max 200 line items per request.
Pass the line_items and reason via --body.

**Scopes:** `com.write_inventories`

| Param | Type | Required | Description |
|---|---|---|---|
| `body` | any | **yes** | Line items, e.g. {"line_items":[{"product_id":1,"product_variant_id":2,"quantity":5}]} |
| `location_id` | integer | **yes** | Location/warehouse ID |
| `note` | string |  | Adjustment note |
| `reason` | string |  | Adjustment reason _(one of: newproduct, returned, productionofgoods, damaged, shrinkage, promotion, transfer)_ |
| `tags` | string |  | Comma-separated tags |
| `type` | string |  | adjust\|set (default adjust) _(one of: adjust, set)_ |

### `haravan_inventory_adjustments_count`

Count inventory adjustments

**Scopes:** `com.read_inventories`

| Param | Type | Required | Description |
|---|---|---|---|
| `created_at_max` | string (ISO 8601) |  | Created before (ISO 8601) |
| `created_at_min` | string (ISO 8601) |  | Created after (ISO 8601) |

### `haravan_inventory_adjustments_get`

Get a single inventory adjustment by ID

**Scopes:** `com.read_inventories`

| Param | Type | Required | Description |
|---|---|---|---|
| `adjustment_id` | integer | **yes** | Adjustment ID |

### `haravan_inventory_adjustments_list`

List inventory adjustments with pagination and date filters.

**Scopes:** `com.read_inventories`

| Param | Type | Required | Description |
|---|---|---|---|
| `created_at_max` | string (ISO 8601) |  | Created before (ISO 8601) |
| `created_at_min` | string (ISO 8601) |  | Created after (ISO 8601) |
| `fetch_all` | boolean |  | Auto-paginate to fetch every page |
| `limit` | integer |  | Results per page |
| `page` | integer |  | Page number |
| `since_id` | integer |  | Results after this ID |

### `haravan_inventory_locations`

Get inventory levels by location / variant / product

**Scopes:** `com.read_inventories`

| Param | Type | Required | Description |
|---|---|---|---|
| `limit` | integer |  | Results per page |
| `location_id` | integer |  | Filter by location ID |
| `page` | integer |  | Page number |
| `product_id` | integer |  | Filter by product ID |
| `variant_id` | integer |  | Filter by variant ID |

---

## 🏪 SHOP / LOCATIONS / USERS  _(6 tool)_

### `haravan_locations_get`

Get a single location by ID

**Scopes:** `com.read_shop`

| Param | Type | Required | Description |
|---|---|---|---|
| `location_id` | integer | **yes** | Location ID |

### `haravan_locations_list`

List all locations/warehouses

**Scopes:** `com.read_shop`

_No parameters._

### `haravan_shipping_rates_get`

Get available shipping rates for a destination address. Pass address fields via the --address-* flags; at least one is typically required by the endpoint.

**Scopes:** `com.read_orders`

| Param | Type | Required | Description |
|---|---|---|---|
| `address1` | string |  | Street address |
| `city` | string |  | City |
| `country` | string |  | Country |
| `province` | string |  | Province |
| `zip` | string |  | ZIP / postal code |

### `haravan_shop_get`

Get shop information: name, domain, email, currency, timezone, plan, address, checkout settings.

**Scopes:** `com.read_shop`

_No parameters._

### `haravan_users_get`

Get a single user by ID (Haravan Plus only)

**Scopes:** `com.read_shop`

| Param | Type | Required | Description |
|---|---|---|---|
| `user_id` | integer | **yes** | User ID |

### `haravan_users_list`

List all shop staff users (Haravan Plus only)

**Scopes:** `com.read_shop`

_No parameters._

---

## 📝 CONTENT (Pages / Blogs / Articles / Script tags)  _(11 tool)_

### `haravan_articles_get`

Get a single article

**Scopes:** `web.read_contents`

| Param | Type | Required | Description |
|---|---|---|---|
| `article_id` | integer | **yes** | Article ID |
| `blog_id` | integer | **yes** | Blog ID |

### `haravan_articles_list`

List articles of a blog

**Scopes:** `web.read_contents`

| Param | Type | Required | Description |
|---|---|---|---|
| `blog_id` | integer | **yes** | Blog ID |
| `limit` | integer |  | Results per page |
| `page` | integer |  | Page number |

### `haravan_blogs_list`

List all blogs

**Scopes:** `web.read_contents`

| Param | Type | Required | Description |
|---|---|---|---|
| `limit` | integer |  | Results per page |
| `page` | integer |  | Page number |

### `haravan_pages_create`

Pass the payload via --body (e.g. {"page":{"title":"About","body_html":"..."}}).

**Scopes:** `web.write_contents`

| Param | Type | Required | Description |
|---|---|---|---|
| `body` | any | **yes** | Page payload (wrapped or bare) |

### `haravan_pages_delete`

Delete a page

**Scopes:** `web.write_contents`

| Param | Type | Required | Description |
|---|---|---|---|
| `page_id` | integer | **yes** | Page ID |

### `haravan_pages_get`

Get a single page by ID

**Scopes:** `web.read_contents`

| Param | Type | Required | Description |
|---|---|---|---|
| `page_id` | integer | **yes** | Page ID |

### `haravan_pages_list`

List pages

**Scopes:** `web.read_contents`

| Param | Type | Required | Description |
|---|---|---|---|
| `fields` | string |  | Comma-separated fields |
| `limit` | integer |  | Results per page |
| `page` | integer |  | Page number |
| `since_id` | integer |  | Results after this ID |

### `haravan_pages_update`

Update a page

**Scopes:** `web.write_contents`

| Param | Type | Required | Description |
|---|---|---|---|
| `body` | any | **yes** | Page patch (wrapped or bare) |
| `page_id` | integer | **yes** | Page ID |

### `haravan_script_tags_create`

Requires an HTTPS src URL. Pass via --body: {"script_tag":{"event":"onload","src":"https://…"}}.

**Scopes:** `web.write_script_tags`

| Param | Type | Required | Description |
|---|---|---|---|
| `body` | any | **yes** | Script tag payload (wrapped or bare) |

### `haravan_script_tags_delete`

Delete a script tag

**Scopes:** `web.write_script_tags`

| Param | Type | Required | Description |
|---|---|---|---|
| `script_tag_id` | integer | **yes** | Script tag ID |

### `haravan_script_tags_list`

List script tags

**Scopes:** `web.read_script_tags`

_No parameters._

---

## 🔔 WEBHOOKS  _(3 tool)_

### `haravan_webhooks_list`

List all webhook subscriptions for the current app (hits webhook.haravan.com, not the main API).

**Scopes:** `wh_api`

_No parameters._

### `haravan_webhooks_subscribe`

Subscribe the app to a webhook topic. Valid topics include:
orders/create, orders/updated, orders/paid, orders/cancelled, orders/fulfilled,
products/create, products/update, products/delete,
customers/create, customers/update, customers/delete,
shop/update, user/update, app/uninstalled.

The app must have a verified callback URL configured in the Developer Dashboard.

**Scopes:** `wh_api`

| Param | Type | Required | Description |
|---|---|---|---|
| `topic` | string |  | Webhook topic, e.g. orders/create |

### `haravan_webhooks_unsubscribe`

Unsubscribe from a webhook topic

**Scopes:** `wh_api`

| Param | Type | Required | Description |
|---|---|---|---|
| `topic` | string |  | Webhook topic to unsubscribe |

---

## Tổng: 70 tool

| Nhóm | Số tool | Ghi chú |
|---|---|---|
| 🧠 Smart | 7 | Aggregate server-side, dùng cho summary & RFM & inventory analytics |
| 📦 Orders | 13 | List/count/get/create/update + status transitions + transactions |
| 🛒 Products | 11 | Products CRUD + variants CRUD |
| 👥 Customers | 14 | Customers CRUD + groups + addresses CRUD + set_default |
| 📊 Inventory | 5 | Adjustments list/count/get + adjust_or_set + locations |
| 🏪 Shop | 6 | Shop info, locations, users (Plus), shipping rates |
| 📝 Content | 11 | Pages CRUD, blogs list, articles list/get, script tags |
| 🔔 Webhooks | 3 | list / subscribe / unsubscribe (API Haravan dùng topic, không phải webhook_id) |

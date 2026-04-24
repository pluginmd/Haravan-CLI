package smart

import (
	"context"
	"encoding/json"
	"math"
	"net/url"
	"sort"
	"strconv"

	"github.com/pluginmd/haravan-cli/pkg/tools"
)

var readInventories = []string{"com.read_inventories", "com.read_products"}

// Shared JSON shapes used across inventory smart tools.

type variantJSON struct {
	ID                int64           `json:"id"`
	Title             string          `json:"title"`
	SKU               string          `json:"sku"`
	Price             json.RawMessage `json:"price"`
	InventoryQuantity int             `json:"inventory_quantity"`
}

type productJSON struct {
	ID       int64         `json:"id"`
	Title    string        `json:"title"`
	Type     string        `json:"product_type"`
	Vendor   string        `json:"vendor"`
	Variants []variantJSON `json:"variants"`
}

type invLocationJSON struct {
	VariantID    int64  `json:"variant_id"`
	LocationID   int64  `json:"location_id"`
	LocationName string `json:"location_name"`
	QtyAvailable int    `json:"qty_available"`
	QtyOnhand    int    `json:"qty_onhand"`
}

type locationJSON struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// fetchVariantInventory fetches /com/inventories/locations.json for one variant.
// Errors are swallowed into an empty slice so a single bad variant doesn't kill
// the whole analysis — matches the legacy behaviour.
func fetchVariantInventory(ctx context.Context, deps *tools.Deps, variantID int64) ([]invLocationJSON, error) {
	q := url.Values{}
	q.Set("variant_id", strconv.FormatInt(variantID, 10))
	resp, err := deps.Client.Get(ctx, "/com/inventories/locations.json", q)
	if err != nil {
		return nil, err
	}
	var env struct {
		Inventories []invLocationJSON `json:"inventories"`
		Locations   []invLocationJSON `json:"locations"`
	}
	if err := json.Unmarshal(resp.Body, &env); err != nil {
		return nil, err
	}
	if len(env.Inventories) > 0 {
		return env.Inventories, nil
	}
	return env.Locations, nil
}

// --- hrv_inventory_health ------------------------------------------------

func init() {
	registerInventoryHealth()
	registerStockReorderPlan()
	registerInventoryImbalance()
}

func registerInventoryHealth() {
	tools.Register(&tools.Tool{
		Name:     "hrv_inventory_health",
		Short:    "Classify variants as out_of_stock / low_stock / dead_stock / healthy",
		Long: `Analyze inventory across the first 100 products.
Classifies variants as out_of_stock, low_stock, dead_stock (stock but no
sales in the lookback window) or healthy. Returns summary counts, total
dead-stock value, and top-10 lists.`,
		Category: tools.CatSmart,
		Scopes:   readInventories,
		Flags: []tools.Flag{
			{Name: "low_stock_threshold", Type: tools.FlagInt, Description: "Qty below which a variant is low (default 5)", Default: 5},
			{Name: "days_for_dead_stock", Type: tools.FlagInt, Description: "Lookback window in days (default 90)", Default: 90},
		},
		Handler: inventoryHealthHandler,
	})
}

func inventoryHealthHandler(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
	lowThreshold := in.Int("low_stock_threshold")
	if lowThreshold == 0 {
		lowThreshold = 5
	}
	daysForDead := in.Int("days_for_dead_stock")
	if daysForDead == 0 {
		daysForDead = 90
	}

	// Step 1: first page of products (up to 100).
	prodFetched, err := fetchAll(ctx, deps.Client, "/com/products.json", "products", url.Values{},
		fetchOpts{Fields: "id,title,variants,product_type,vendor", MaxPages: 1})
	if err != nil {
		return nil, err
	}
	allProducts, err := unmarshalProducts(prodFetched.Items)
	if err != nil {
		return nil, err
	}
	products := allProducts
	hasMore := false
	if len(products) > 100 {
		products = products[:100]
		hasMore = true
	}

	// Step 2: inventory per variant.
	variantInv := map[int64][]invLocationJSON{}
	invCalls := 0
	for _, p := range products {
		for _, v := range p.Variants {
			locs, _ := fetchVariantInventory(ctx, deps, v.ID)
			variantInv[v.ID] = locs
			invCalls++
		}
	}

	// Step 3: recent orders to identify variants that have sold.
	orderQ := url.Values{}
	orderQ.Set("created_at_min", parseDate("", daysForDead))
	orderQ.Set("status", "any")
	orderFetched, err := fetchAll(ctx, deps.Client, "/com/orders.json", "orders", orderQ,
		fetchOpts{Fields: "id,line_items"})
	if err != nil {
		return nil, err
	}
	sold := map[int64]struct{}{}
	for _, raw := range orderFetched.Items {
		var o struct {
			LineItems []struct {
				VariantID int64 `json:"variant_id"`
			} `json:"line_items"`
		}
		_ = json.Unmarshal(raw, &o)
		for _, li := range o.LineItems {
			if li.VariantID != 0 {
				sold[li.VariantID] = struct{}{}
			}
		}
	}

	type classified struct {
		ProductID    int64
		ProductTitle string
		VariantID    int64
		VariantTitle string
		SKU          string
		QtyAvail     int
		QtyOnhand    int
		Price        float64
		Category     string
	}

	var list []classified
	for _, p := range products {
		for _, v := range p.Variants {
			locs := variantInv[v.ID]
			qa, qo := 0, 0
			for _, l := range locs {
				qa += l.QtyAvailable
				qo += l.QtyOnhand
			}
			price := parseFloatOr(v.Price, 0)
			var cat string
			switch {
			case qa == 0:
				cat = "out_of_stock"
			case qa < lowThreshold:
				cat = "low_stock"
			case qo > 0:
				if _, ok := sold[v.ID]; !ok {
					cat = "dead_stock"
				} else {
					cat = "healthy"
				}
			default:
				cat = "healthy"
			}
			list = append(list, classified{
				ProductID: p.ID, ProductTitle: p.Title,
				VariantID: v.ID, VariantTitle: v.Title, SKU: v.SKU,
				QtyAvail: qa, QtyOnhand: qo, Price: price, Category: cat,
			})
		}
	}

	byCat := func(name string) []classified {
		out := []classified{}
		for _, c := range list {
			if c.Category == name {
				out = append(out, c)
			}
		}
		return out
	}

	dead := byCat("dead_stock")
	low := byCat("low_stock")
	ooStock := byCat("out_of_stock")
	healthy := byCat("healthy")

	var deadValue float64
	for _, c := range dead {
		deadValue += float64(c.QtyOnhand) * c.Price
	}

	sort.SliceStable(low, func(i, j int) bool { return low[i].QtyAvail < low[j].QtyAvail })
	if len(low) > 10 {
		low = low[:10]
	}
	sort.SliceStable(dead, func(i, j int) bool {
		return float64(dead[i].QtyOnhand)*dead[i].Price > float64(dead[j].QtyOnhand)*dead[j].Price
	})
	if len(dead) > 10 {
		dead = dead[:10]
	}

	type lowOut struct {
		ProductTitle string `json:"product_title"`
		VariantTitle string `json:"variant_title"`
		SKU          string `json:"sku"`
		QtyAvailable int    `json:"qty_available"`
	}
	type deadOut struct {
		ProductTitle   string  `json:"product_title"`
		VariantTitle   string  `json:"variant_title"`
		SKU            string  `json:"sku"`
		QtyOnhand      int     `json:"qty_onhand"`
		DeadStockValue float64 `json:"dead_stock_value"`
	}

	lowList := make([]lowOut, 0, len(low))
	for _, c := range low {
		lowList = append(lowList, lowOut{c.ProductTitle, c.VariantTitle, c.SKU, c.QtyAvail})
	}
	deadList := make([]deadOut, 0, len(dead))
	for _, c := range dead {
		deadList = append(deadList, deadOut{
			ProductTitle:   c.ProductTitle,
			VariantTitle:   c.VariantTitle,
			SKU:            c.SKU,
			QtyOnhand:      c.QtyOnhand,
			DeadStockValue: roundToDecimals(float64(c.QtyOnhand)*c.Price, 2),
		})
	}

	note := ""
	if hasMore {
		note = "Only first 100 products analyzed. Run with more specific filters for full coverage."
	}

	return tools.NewJSON(map[string]any{
		"summary": map[string]any{
			"total_variants_analyzed": len(list),
			"out_of_stock":            len(ooStock),
			"low_stock":               len(byCat("low_stock")),
			"healthy":                 len(healthy),
			"dead_stock":              len(byCat("dead_stock")),
			"total_dead_stock_value":  roundToDecimals(deadValue, 2),
		},
		"top_10_low_stock":  lowList,
		"top_10_dead_stock": deadList,
		"_meta": map[string]any{
			"products_analyzed":    len(products),
			"has_more_products":    hasMore,
			"orders_analyzed":      len(orderFetched.Items),
			"days_for_dead_stock":  daysForDead,
			"low_stock_threshold":  lowThreshold,
			"api_calls":            prodFetched.APICalls + invCalls + orderFetched.APICalls,
			"note":                 note,
		},
	})
}

// --- hrv_stock_reorder_plan ----------------------------------------------

func registerStockReorderPlan() {
	tools.Register(&tools.Tool{
		Name:     "hrv_stock_reorder_plan",
		Short:    "Reorder plan based on daily sales rate, lead time and safety factor",
		Category: tools.CatSmart,
		Scopes:   readInventories,
		Flags: []tools.Flag{
			{Name: "lead_time_days", Type: tools.FlagInt, Description: "Supplier lead time in days (default 7)", Default: 7},
			{Name: "safety_factor", Type: tools.FlagString, Description: "Safety buffer multiplier (default 1.3)"},
			{Name: "date_range_days", Type: tools.FlagInt, Description: "Days of order history for DSR (default 30)", Default: 30},
		},
		Handler: stockReorderPlanHandler,
	})
}

func stockReorderPlanHandler(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
	leadTime := in.Int("lead_time_days")
	if leadTime == 0 {
		leadTime = 7
	}
	safety := 1.3
	if s := in.String("safety_factor"); s != "" {
		if f, err := strconv.ParseFloat(s, 64); err == nil && f > 0 {
			safety = f
		}
	}
	dateRange := in.Int("date_range_days")
	if dateRange == 0 {
		dateRange = 30
	}

	// Orders
	orderQ := url.Values{}
	orderQ.Set("created_at_min", parseDate("", dateRange))
	orderQ.Set("status", "any")
	orderFetched, err := fetchAll(ctx, deps.Client, "/com/orders.json", "orders", orderQ,
		fetchOpts{Fields: "id,line_items,created_at"})
	if err != nil {
		return nil, err
	}
	sales := map[int64]int{}
	for _, raw := range orderFetched.Items {
		var o struct {
			LineItems []struct {
				VariantID int64 `json:"variant_id"`
				Quantity  int   `json:"quantity"`
			} `json:"line_items"`
		}
		_ = json.Unmarshal(raw, &o)
		for _, li := range o.LineItems {
			if li.VariantID == 0 {
				continue
			}
			q := li.Quantity
			if q == 0 {
				q = 1
			}
			sales[li.VariantID] += q
		}
	}

	// Products → variant lookup
	prodFetched, err := fetchAll(ctx, deps.Client, "/com/products.json", "products", url.Values{},
		fetchOpts{Fields: "id,title,variants"})
	if err != nil {
		return nil, err
	}
	products, err := unmarshalProducts(prodFetched.Items)
	if err != nil {
		return nil, err
	}
	type vinfo struct {
		ProductID    int64
		ProductTitle string
		VariantID    int64
		VariantTitle string
		SKU          string
	}
	info := map[int64]vinfo{}
	for _, p := range products {
		for _, v := range p.Variants {
			info[v.ID] = vinfo{p.ID, p.Title, v.ID, v.Title, v.SKU}
		}
	}

	// Top-selling variant IDs (cap at 200 to respect rate limit)
	type pair struct {
		ID  int64
		Qty int
	}
	selling := make([]pair, 0, len(sales))
	for id, q := range sales {
		if q > 0 {
			selling = append(selling, pair{id, q})
		}
	}
	sort.Slice(selling, func(i, j int) bool { return selling[i].Qty > selling[j].Qty })
	if len(selling) > 200 {
		selling = selling[:200]
	}

	inv := map[int64]int{}
	invCalls := 0
	for _, sp := range selling {
		locs, _ := fetchVariantInventory(ctx, deps, sp.ID)
		q := 0
		for _, l := range locs {
			q += l.QtyAvailable
		}
		inv[sp.ID] = q
		invCalls++
	}

	type reorderItem struct {
		ProductTitle        string  `json:"product_title"`
		VariantTitle        string  `json:"variant_title"`
		SKU                 string  `json:"sku"`
		QtyAvailable        int     `json:"qty_available"`
		DailySalesRate      float64 `json:"daily_sales_rate"`
		DaysOfStock         *int    `json:"days_of_stock"`
		ReorderPoint        float64 `json:"reorder_point"`
		ReorderQtySuggested int     `json:"reorder_qty_suggested"`
	}
	var reorder []reorderItem
	for _, sp := range selling {
		vi, ok := info[sp.ID]
		if !ok {
			continue
		}
		dsr := float64(sp.Qty) / float64(dateRange)
		qa := inv[sp.ID]
		reorderPoint := dsr * float64(leadTime) * safety
		reorderQty := int(math.Ceil(reorderPoint - float64(qa)))
		if reorderQty < 0 {
			reorderQty = 0
		}
		var days *int
		if dsr > 0 {
			d := int(math.Floor(float64(qa) / dsr))
			days = &d
		}
		if reorderQty > 0 || (days != nil && *days < leadTime*2) {
			reorder = append(reorder, reorderItem{
				ProductTitle:        vi.ProductTitle,
				VariantTitle:        vi.VariantTitle,
				SKU:                 vi.SKU,
				QtyAvailable:        qa,
				DailySalesRate:      roundToDecimals(dsr, 3),
				DaysOfStock:         days,
				ReorderPoint:        roundToDecimals(reorderPoint, 1),
				ReorderQtySuggested: reorderQty,
			})
		}
	}
	sort.SliceStable(reorder, func(i, j int) bool {
		a, b := 0, 0
		if reorder[i].DaysOfStock != nil {
			a = *reorder[i].DaysOfStock
		}
		if reorder[j].DaysOfStock != nil {
			b = *reorder[j].DaysOfStock
		}
		return a < b
	})
	shown := reorder
	if len(shown) > 30 {
		shown = shown[:30]
	}

	return tools.NewJSON(map[string]any{
		"reorder_plan": shown,
		"_meta": map[string]any{
			"date_range_days":           dateRange,
			"lead_time_days":            leadTime,
			"safety_factor":             safety,
			"orders_analyzed":           len(orderFetched.Items),
			"variants_with_sales":       len(sales),
			"variants_fetched_inventory": len(selling),
			"total_needing_reorder":     len(reorder),
			"showing":                   len(shown),
		},
	})
}

// --- hrv_inventory_imbalance --------------------------------------------

func registerInventoryImbalance() {
	tools.Register(&tools.Tool{
		Name:     "hrv_inventory_imbalance",
		Short:    "Detect cross-location imbalances (max/min > 5x)",
		Category: tools.CatSmart,
		Scopes:   readInventories,
		Handler:  inventoryImbalanceHandler,
	})
}

func inventoryImbalanceHandler(ctx context.Context, deps *tools.Deps, _ tools.Input) (*tools.Result, error) {
	// Locations
	locResp, err := deps.Client.Get(ctx, "/com/locations.json", nil)
	if err != nil {
		return nil, err
	}
	var locEnv struct {
		Locations []locationJSON `json:"locations"`
	}
	_ = json.Unmarshal(locResp.Body, &locEnv)
	if len(locEnv.Locations) < 2 {
		return tools.NewJSON(map[string]any{
			"message":              "At least 2 locations are required to detect imbalance.",
			"locations_found":      len(locEnv.Locations),
			"imbalanced_variants":  []any{},
		})
	}
	locName := map[int64]string{}
	for _, l := range locEnv.Locations {
		locName[l.ID] = l.Name
	}

	// Products — first 50
	prodFetched, err := fetchAll(ctx, deps.Client, "/com/products.json", "products", url.Values{},
		fetchOpts{Fields: "id,title,variants", MaxPages: 1})
	if err != nil {
		return nil, err
	}
	products, err := unmarshalProducts(prodFetched.Items)
	if err != nil {
		return nil, err
	}
	if len(products) > 50 {
		products = products[:50]
	}

	type vlQty struct {
		LocationID   int64  `json:"location_id"`
		LocationName string `json:"location_name"`
		QtyAvailable int    `json:"qty_available"`
	}
	type variantLoc struct {
		ProductTitle string
		VariantID    int64
		VariantTitle string
		SKU          string
		Locations    []vlQty
	}

	var all []variantLoc
	invCalls := 0
	for _, p := range products {
		for _, v := range p.Variants {
			locs, _ := fetchVariantInventory(ctx, deps, v.ID)
			invCalls++
			breakdown := make([]vlQty, 0, len(locs))
			for _, l := range locs {
				name := locName[l.LocationID]
				if name == "" {
					name = "Location " + strconv.FormatInt(l.LocationID, 10)
				}
				breakdown = append(breakdown, vlQty{l.LocationID, name, l.QtyAvailable})
			}
			all = append(all, variantLoc{p.Title, v.ID, v.Title, v.SKU, breakdown})
		}
	}

	type imbalanced struct {
		ProductTitle     string  `json:"product_title"`
		VariantTitle     string  `json:"variant_title"`
		SKU              string  `json:"sku"`
		ImbalanceRatio   float64 `json:"imbalance_ratio"`
		Locations        []vlQty `json:"locations"`
		SuggestedTransfer struct {
			FromLocation string `json:"from_location"`
			ToLocation   string `json:"to_location"`
			Qty          int    `json:"qty"`
		} `json:"suggested_transfer"`
	}
	var out []imbalanced
	for _, vl := range all {
		withStock := make([]vlQty, 0, len(vl.Locations))
		for _, l := range vl.Locations {
			if l.QtyAvailable > 0 {
				withStock = append(withStock, l)
			}
		}
		if len(withStock) < 2 {
			continue
		}
		maxL, minL := withStock[0], withStock[0]
		for _, l := range withStock[1:] {
			if l.QtyAvailable > maxL.QtyAvailable {
				maxL = l
			}
			if l.QtyAvailable < minL.QtyAvailable {
				minL = l
			}
		}
		if minL.QtyAvailable == 0 {
			continue
		}
		ratio := float64(maxL.QtyAvailable) / float64(minL.QtyAvailable)
		if ratio <= 5 {
			continue
		}
		target := (maxL.QtyAvailable + minL.QtyAvailable) / 2
		transfer := maxL.QtyAvailable - target

		sorted := append([]vlQty(nil), vl.Locations...)
		sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].QtyAvailable > sorted[j].QtyAvailable })

		var row imbalanced
		row.ProductTitle = vl.ProductTitle
		row.VariantTitle = vl.VariantTitle
		row.SKU = vl.SKU
		row.ImbalanceRatio = roundToDecimals(ratio, 1)
		row.Locations = sorted
		row.SuggestedTransfer.FromLocation = maxL.LocationName
		row.SuggestedTransfer.ToLocation = minL.LocationName
		row.SuggestedTransfer.Qty = transfer
		out = append(out, row)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ImbalanceRatio > out[j].ImbalanceRatio })

	return tools.NewJSON(map[string]any{
		"imbalanced_variants": out,
		"_meta": map[string]any{
			"locations_count":     len(locEnv.Locations),
			"products_analyzed":   len(products),
			"variants_analyzed":   len(all),
			"imbalanced_count":    len(out),
			"imbalance_threshold": 5,
			"api_calls":           1 + invCalls + prodFetched.APICalls,
		},
	})
}

// unmarshalProducts decodes a slice of raw products with shared structure.
func unmarshalProducts(items []json.RawMessage) ([]productJSON, error) {
	out := make([]productJSON, 0, len(items))
	for _, raw := range items {
		var p productJSON
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

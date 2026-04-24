package content

import (
	"context"
	"fmt"

	"github.com/pluginmd/haravan-cli/pkg/tools"
)

var (
	readContent      = []string{"web.read_contents"}
	writeContent     = []string{"web.write_contents"}
	readScriptTags   = []string{"web.read_script_tags"}
	writeScriptTags  = []string{"web.write_script_tags"}
)

func init() {
	registerPages()
	registerBlogs()
	registerArticles()
	registerScriptTags()
}

func registerPages() {
	tools.Register(&tools.Tool{
		Name:     "haravan_pages_list",
		Short:    "List pages",
		Category: tools.CatContent,
		Scopes:   readContent,
		Flags: []tools.Flag{
			{Name: "page", Type: tools.FlagInt, Description: "Page number"},
			{Name: "limit", Type: tools.FlagInt, Description: "Results per page"},
			{Name: "since_id", Type: tools.FlagInt64, Description: "Results after this ID"},
			{Name: "fields", Type: tools.FlagString, Description: "Comma-separated fields"},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			q := tools.Query(in, "page", "limit", "since_id", "fields")
			resp, err := deps.Client.Get(ctx, "/web/pages.json", q)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_pages_get",
		Short:    "Get a single page by ID",
		Category: tools.CatContent,
		Scopes:   readContent,
		Flags: []tools.Flag{
			{Name: "page_id", Type: tools.FlagInt64, Description: "Page ID", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/web/pages/%d.json", in.Int64("page_id"))
			resp, err := deps.Client.Get(ctx, path, nil)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_pages_create",
		Short:    "Create a new page",
		Long:     `Pass the payload via --body (e.g. {"page":{"title":"About","body_html":"..."}}).`,
		Category: tools.CatContent,
		Scopes:   writeContent,
		Flags: []tools.Flag{
			{Name: "body", Type: tools.FlagJSON, Description: "Page payload (wrapped or bare)", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			body, err := tools.EnvelopeWrap(in.JSON("body"), "page")
			if err != nil {
				return nil, err
			}
			resp, err := deps.Client.Post(ctx, "/web/pages.json", body)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_pages_update",
		Short:    "Update a page",
		Category: tools.CatContent,
		Scopes:   writeContent,
		Flags: []tools.Flag{
			{Name: "page_id", Type: tools.FlagInt64, Description: "Page ID", Required: true},
			{Name: "body", Type: tools.FlagJSON, Description: "Page patch (wrapped or bare)", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			body, err := tools.EnvelopeWrap(in.JSON("body"), "page")
			if err != nil {
				return nil, err
			}
			path := fmt.Sprintf("/web/pages/%d.json", in.Int64("page_id"))
			resp, err := deps.Client.Put(ctx, path, body)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_pages_delete",
		Short:    "Delete a page",
		Category: tools.CatContent,
		Scopes:   writeContent,
		Flags: []tools.Flag{
			{Name: "page_id", Type: tools.FlagInt64, Description: "Page ID", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/web/pages/%d.json", in.Int64("page_id"))
			resp, err := deps.Client.Delete(ctx, path)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerBlogs() {
	tools.Register(&tools.Tool{
		Name:     "haravan_blogs_list",
		Short:    "List all blogs",
		Category: tools.CatContent,
		Scopes:   readContent,
		Flags: []tools.Flag{
			{Name: "page", Type: tools.FlagInt, Description: "Page number"},
			{Name: "limit", Type: tools.FlagInt, Description: "Results per page"},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			q := tools.Query(in, "page", "limit")
			resp, err := deps.Client.Get(ctx, "/web/blogs.json", q)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerArticles() {
	tools.Register(&tools.Tool{
		Name:     "haravan_articles_list",
		Short:    "List articles of a blog",
		Category: tools.CatContent,
		Scopes:   readContent,
		Flags: []tools.Flag{
			{Name: "blog_id", Type: tools.FlagInt64, Description: "Blog ID", Required: true},
			{Name: "page", Type: tools.FlagInt, Description: "Page number"},
			{Name: "limit", Type: tools.FlagInt, Description: "Results per page"},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/web/blogs/%d/articles.json", in.Int64("blog_id"))
			resp, err := deps.Client.Get(ctx, path, tools.Query(in, "page", "limit"))
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_articles_get",
		Short:    "Get a single article",
		Category: tools.CatContent,
		Scopes:   readContent,
		Flags: []tools.Flag{
			{Name: "blog_id", Type: tools.FlagInt64, Description: "Blog ID", Required: true},
			{Name: "article_id", Type: tools.FlagInt64, Description: "Article ID", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/web/blogs/%d/articles/%d.json", in.Int64("blog_id"), in.Int64("article_id"))
			resp, err := deps.Client.Get(ctx, path, nil)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerScriptTags() {
	tools.Register(&tools.Tool{
		Name:     "haravan_script_tags_list",
		Short:    "List script tags",
		Category: tools.CatContent,
		Scopes:   readScriptTags,
		Handler: func(ctx context.Context, deps *tools.Deps, _ tools.Input) (*tools.Result, error) {
			resp, err := deps.Client.Get(ctx, "/web/script_tags.json", nil)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_script_tags_create",
		Short:    "Create a script tag",
		Long:     `Requires an HTTPS src URL. Pass via --body: {"script_tag":{"event":"onload","src":"https://…"}}.`,
		Category: tools.CatContent,
		Scopes:   writeScriptTags,
		Flags: []tools.Flag{
			{Name: "body", Type: tools.FlagJSON, Description: "Script tag payload (wrapped or bare)", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			body, err := tools.EnvelopeWrap(in.JSON("body"), "script_tag")
			if err != nil {
				return nil, err
			}
			resp, err := deps.Client.Post(ctx, "/web/script_tags.json", body)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_script_tags_delete",
		Short:    "Delete a script tag",
		Category: tools.CatContent,
		Scopes:   writeScriptTags,
		Flags: []tools.Flag{
			{Name: "script_tag_id", Type: tools.FlagInt64, Description: "Script tag ID", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/web/script_tags/%d.json", in.Int64("script_tag_id"))
			resp, err := deps.Client.Delete(ctx, path)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

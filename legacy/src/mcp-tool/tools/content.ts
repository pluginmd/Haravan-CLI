import { z } from 'zod';
import { McpTool } from '../types';

export const contentTools: McpTool[] = [
  {
    name: 'haravan_pages_list',
    project: 'content',
    description: 'List all pages.',
    schema: z.object({
      page: z.number().optional(),
      limit: z.number().optional(),
      since_id: z.number().optional(),
      fields: z.string().optional(),
    }),
    httpMethod: 'GET',
    path: '/web/pages.json',
    scopes: ['web.read_contents'],
  },
  {
    name: 'haravan_pages_get',
    project: 'content',
    description: 'Get a single page by ID.',
    schema: z.object({
      page_id: z.number().describe('Page ID'),
    }),
    httpMethod: 'GET',
    path: '/web/pages/{page_id}.json',
    scopes: ['web.read_contents'],
  },
  {
    name: 'haravan_pages_create',
    project: 'content',
    description: 'Create a new page.',
    schema: z.object({
      page: z
        .object({
          title: z.string().describe('Page title'),
          body_html: z.string().optional().describe('Page HTML content'),
          handle: z.string().optional().describe('URL slug'),
          published: z.boolean().optional(),
          template_suffix: z.string().optional(),
        })
        .describe('Page object'),
    }),
    httpMethod: 'POST',
    path: '/web/pages.json',
    scopes: ['web.write_contents'],
  },
  {
    name: 'haravan_pages_update',
    project: 'content',
    description: 'Update a page.',
    schema: z.object({
      page_id: z.number().describe('Page ID'),
      page: z
        .object({
          id: z.number().describe('Page ID'),
          title: z.string().optional(),
          body_html: z.string().optional(),
          handle: z.string().optional(),
          published: z.boolean().optional(),
        })
        .describe('Page fields to update'),
    }),
    httpMethod: 'PUT',
    path: '/web/pages/{page_id}.json',
    scopes: ['web.write_contents'],
  },
  {
    name: 'haravan_pages_delete',
    project: 'content',
    description: 'Delete a page.',
    schema: z.object({
      page_id: z.number().describe('Page ID to delete'),
    }),
    httpMethod: 'DELETE',
    path: '/web/pages/{page_id}.json',
    scopes: ['web.write_contents'],
  },
  {
    name: 'haravan_blogs_list',
    project: 'content',
    description: 'List all blogs.',
    schema: z.object({
      page: z.number().optional(),
      limit: z.number().optional(),
    }),
    httpMethod: 'GET',
    path: '/web/blogs.json',
    scopes: ['web.read_contents'],
  },
  {
    name: 'haravan_articles_list',
    project: 'content',
    description: 'List all articles for a blog.',
    schema: z.object({
      blog_id: z.number().describe('Blog ID'),
      page: z.number().optional(),
      limit: z.number().optional(),
    }),
    httpMethod: 'GET',
    path: '/web/blogs/{blog_id}/articles.json',
    scopes: ['web.read_contents'],
  },
  {
    name: 'haravan_articles_get',
    project: 'content',
    description: 'Get a single article.',
    schema: z.object({
      blog_id: z.number().describe('Blog ID'),
      article_id: z.number().describe('Article ID'),
    }),
    httpMethod: 'GET',
    path: '/web/blogs/{blog_id}/articles/{article_id}.json',
    scopes: ['web.read_contents'],
  },
  {
    name: 'haravan_script_tags_list',
    project: 'content',
    description: 'List all script tags.',
    schema: z.object({}),
    httpMethod: 'GET',
    path: '/web/script_tags.json',
    scopes: ['web.read_script_tags'],
  },
  {
    name: 'haravan_script_tags_create',
    project: 'content',
    description: 'Create a new script tag.',
    schema: z.object({
      script_tag: z
        .object({
          event: z.string().describe('Event trigger (e.g., onload)'),
          src: z.string().url().startsWith('https://').describe('Script URL (HTTPS required)'),
        })
        .describe('Script tag object'),
    }),
    httpMethod: 'POST',
    path: '/web/script_tags.json',
    scopes: ['web.write_script_tags'],
  },
  {
    name: 'haravan_script_tags_delete',
    project: 'content',
    description: 'Delete a script tag.',
    schema: z.object({
      script_tag_id: z.number().describe('Script tag ID'),
    }),
    httpMethod: 'DELETE',
    path: '/web/script_tags/{script_tag_id}.json',
    scopes: ['web.write_script_tags'],
  },
];

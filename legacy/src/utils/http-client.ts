import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios';
import { logger } from './logger';

export interface HaravanApiResponse<T = any> {
  data: T;
  status: number;
  headers: Record<string, string>;
  rateLimitUsed?: number;
  rateLimitMax?: number;
  retryAfter?: number;
}

export interface HaravanClientOptions {
  domain: string;
  accessToken: string;
  timeout?: number;
}

export class HaravanClient {
  private client: AxiosInstance;
  private accessToken: string;

  constructor(options: HaravanClientOptions) {
    this.accessToken = options.accessToken;
    this.client = axios.create({
      baseURL: options.domain,
      timeout: options.timeout || 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    this.client.interceptors.request.use((config) => {
      config.headers['Authorization'] = `Bearer ${this.accessToken}`;
      return config;
    });

    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response) {
          const status = error.response.status;
          const data = error.response.data;
          logger.error(`Haravan API Error [${status}]:`, JSON.stringify(data));
        }
        throw error;
      }
    );
  }

  setAccessToken(token: string) {
    this.accessToken = token;
  }

  private parseRateLimitHeaders(headers: Record<string, any>): {
    used: number;
    max: number;
    retryAfter?: number;
  } {
    const limitHeader = headers['x-haravan-api-call-limit'] || '';
    const [used, max] = limitHeader.split('/').map(Number);
    const retryAfter = headers['retry-after']
      ? parseFloat(headers['retry-after'])
      : undefined;
    return { used: used || 0, max: max || 80, retryAfter };
  }

  async request<T = any>(
    method: string,
    path: string,
    data?: any,
    params?: Record<string, any>
  ): Promise<HaravanApiResponse<T>> {
    const config: AxiosRequestConfig = {
      method: method as any,
      url: path,
      params,
    };

    if (data && ['POST', 'PUT', 'PATCH'].includes(method.toUpperCase())) {
      config.data = data;
    }

    const response: AxiosResponse = await this.client.request(config);
    const rateLimit = this.parseRateLimitHeaders(response.headers);

    logger.debug(
      `API ${method} ${path} -> ${response.status} [Rate: ${rateLimit.used}/${rateLimit.max}]`
    );

    return {
      data: response.data,
      status: response.status,
      headers: response.headers as Record<string, string>,
      rateLimitUsed: rateLimit.used,
      rateLimitMax: rateLimit.max,
      retryAfter: rateLimit.retryAfter,
    };
  }

  async get<T = any>(
    path: string,
    params?: Record<string, any>
  ): Promise<HaravanApiResponse<T>> {
    return this.request<T>('GET', path, undefined, params);
  }

  async post<T = any>(
    path: string,
    data?: any
  ): Promise<HaravanApiResponse<T>> {
    return this.request<T>('POST', path, data);
  }

  async put<T = any>(
    path: string,
    data?: any
  ): Promise<HaravanApiResponse<T>> {
    return this.request<T>('PUT', path, data);
  }

  async delete<T = any>(path: string): Promise<HaravanApiResponse<T>> {
    return this.request<T>('DELETE', path);
  }
}

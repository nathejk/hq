import axios, {
  type InternalAxiosRequestConfig,
  type AxiosInstance,
  type AxiosError, type CreateAxiosDefaults,
} from 'axios';

import bus from '@/plugins/bus';
//import { useAuthStore } from '@/modules/auth/store/auth.store';
import { storeToRefs } from 'pinia';
import type { ToastMessageOptions } from 'primevue/toast';

const createInstance = (config: CreateAxiosDefaults) => {
  const instance: AxiosInstance = axios.create(config);
/*
  instance.interceptors.request.use(async (config: InternalAxiosRequestConfig) => {
    const store = useAuthStore();
    const { token } = storeToRefs(store);

    if (
      !config.url?.includes('://') ||
      (config.baseURL && config.url?.startsWith(config.baseURL)) ||
      config.url?.includes('api/auth/logout')
    ) {
      // relative url
      config.headers.set('Authorization', `Bearer ${token.value}`);
    }

    return config;
  });

  instance.interceptors.response.use(
    undefined,
    async (error: AxiosError<{ message?: string, error?: string }>) => {
      if (error.message !== 'canceled') {
        const is404 = error.status === 400;
        const isRefresh401 = error.status === 401 && error.request?.responseURL?.includes('api/auth/refresh_token');
        if (!is404 && !isRefresh401) {
          const details = [(error.message ?? 'Unknown error')];
          if (error.response?.data?.error) {
            details.push(error.response.data.error);
          }
          if (error.request?.responseURL) {
            details.push(error.request.responseURL);
          }
          bus.emit('toast', {
            severity: 'error',
            closable: true,
            summary: 'Connection error',
            detail: details.join('\n'),
            life: 5000,
          } as ToastMessageOptions);
        }
      }
      return Promise.resolve(error);
    },
  );
*/
  return instance;
};

export const http = createInstance({
  baseURL: '/api/', //import.meta.env.FRONTEND_SAFE_API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

export const http2 = createInstance({
  baseURL: '/api/', //import.meta.env.FRONTEND_SAFE_API_URL.replace('/v1', '/v2'),
  headers: {
    'Content-Type': 'application/json',
  },
});

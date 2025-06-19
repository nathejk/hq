import * as sentry from '@sentry/vue';
import type { App } from 'vue';
import type { Router } from 'vue-router';

export const Sentry = sentry;

const getEnvironmentId = () => {
  try {
    const url = window.location.host;

    if (url === 'localhost') {
      return 'dev';
    }

    if (url === 'recruitment.getwhy.io') {
      return 'production';
    }

    const re = /^.+?\.([a-z]+)\.getwhy\.io$/i;
    const [, env] = url.match(re) ?? [];

    return env ?? 'unknown';
  } catch (e) {
    console.log(e);
  }
  return 'unknown';
};

export const setupSentry = (app: App, router: Router) => {
  const environment = getEnvironmentId();
  if (environment === 'dev' || environment === 'local') {
    return;
  }
  const sentryConfig = {
    app,
    dsn: import.meta.env.FRONTEND_SAFE_SENTRY_DSN,
    integrations: [
      Sentry.browserTracingIntegration({ router }),
      Sentry.replayIntegration(),
    ],
    release: import.meta.env.FRONTEND_SAFE_BUILD_NUMBER ?? 'unknown',
    environment,
    // Tracing
    tracesSampleRate: 1.0, //  Capture 100% of the transactions
    // Set 'tracePropagationTargets' to control for which URLs distributed tracing should be enabled
    tracePropagationTargets: ['localhost', /^https:\/\/recruitment\..*getwhy\.io/],
    // Session Replay
    replaysSessionSampleRate: 0, // This sets the sample rate at 10%. You may want to change it to 100% while in development and then sample at a lower rate in production.
    replaysOnErrorSampleRate: 0, // If you're not already sampling the entire session, change the sample rate to 100% when sampling sessions where errors occur.
  };

  Sentry.init(sentryConfig);
};

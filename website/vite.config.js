import { defineConfig } from 'vite';

export default defineConfig({
  base: './', // Use relative paths for static hosting on GitHub Pages
  define: {
    'process.env.PUBLIC_SITE_URL': JSON.stringify(process.env.PUBLIC_SITE_URL || 'https://netspeedy.github.io/blackhole-threats/'),
    'process.env.PUBLIC_RELEASE_VERSION': JSON.stringify(process.env.PUBLIC_RELEASE_VERSION || 'dev'),
    'process.env.PUBLIC_COMMIT': JSON.stringify(process.env.PUBLIC_COMMIT || 'local'),
    'process.env.PUBLIC_BUILD_DATE': JSON.stringify(process.env.PUBLIC_BUILD_DATE || new Date().toISOString()),
    'process.env.PUBLIC_APT_FINGERPRINT': JSON.stringify(process.env.PUBLIC_APT_FINGERPRINT || 'Published alongside stable signed releases.'),
    'process.env.PUBLIC_RELEASE_HIGHLIGHTS_JSON': JSON.stringify(process.env.PUBLIC_RELEASE_HIGHLIGHTS_JSON || '[]')
  }
});

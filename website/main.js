import './style.css';

const fallbackHighlights = [
  'Normalise local and remote threat feeds into IP prefixes',
  'Publish deterministic route deltas by BGP community',
  'Preserve last-known-good routes when upstream feeds fail',
];

const defaultSiteURL = process.env.PUBLIC_SITE_URL.replace(/\/?$/, '/');
const buildHighlightsValue = process.env.PUBLIC_RELEASE_HIGHLIGHTS_JSON;

function normalizeHighlights(value) {
  if (!Array.isArray(value)) {
    return fallbackHighlights;
  }

  const highlights = value
    .map((item) => `${item ?? ''}`.trim())
    .filter(Boolean)
    .slice(0, 3);

  return highlights.length > 0 ? highlights : fallbackHighlights;
}

function parseBuildHighlights(value) {
  try {
    return normalizeHighlights(JSON.parse(value || '[]'));
  } catch {
    return fallbackHighlights;
  }
}

function hasBuildHighlights(value) {
  try {
    const parsed = JSON.parse(value || '[]');
    return Array.isArray(parsed) && parsed.some((item) => `${item ?? ''}`.trim() !== '');
  } catch {
    return false;
  }
}

const hasExplicitBuildHighlights = hasBuildHighlights(buildHighlightsValue);

const defaultMetadata = {
  siteUrl: defaultSiteURL,
  releaseVersion: process.env.PUBLIC_RELEASE_VERSION,
  commit: process.env.PUBLIC_COMMIT,
  buildDate: process.env.PUBLIC_BUILD_DATE,
  aptFingerprint: process.env.PUBLIC_APT_FINGERPRINT,
  releaseHighlights: parseBuildHighlights(buildHighlightsValue),
  githubUrl: 'https://github.com/netspeedy/blackhole-threats',
  releaseUrl: 'https://github.com/netspeedy/blackhole-threats/releases',
  containerUrl: 'https://ghcr.io/netspeedy/blackhole-threats',
  containerImage: 'ghcr.io/netspeedy/blackhole-threats',
};

function formatBuildDate(value) {
  const parsedBuildDate = new Date(value);
  return Number.isNaN(parsedBuildDate.valueOf())
    ? value
    : `${new Intl.DateTimeFormat('en-GB', {
        dateStyle: 'long',
        timeStyle: 'short',
        timeZone: 'UTC',
      }).format(parsedBuildDate)} UTC`;
}

function parseStableTag(value) {
  const match = `${value}`.match(/^v(\d+)\.(\d+)\.(\d+)$/);
  return match ? match.slice(1).map((part) => Number.parseInt(part, 10)) : null;
}

function compareStableTags(left, right) {
  const leftParts = parseStableTag(left);
  const rightParts = parseStableTag(right);

  if (!leftParts || !rightParts) {
    return 0;
  }

  for (let index = 0; index < leftParts.length; index += 1) {
    if (leftParts[index] !== rightParts[index]) {
      return leftParts[index] - rightParts[index];
    }
  }

  return 0;
}

function normalizeMetadata(value = {}) {
  const siteUrl = `${value.site_url || value.siteUrl || defaultMetadata.siteUrl}`.replace(/\/?$/, '/');
  const releaseVersion = `${value.release_version || value.releaseVersion || defaultMetadata.releaseVersion}`;
  const commit = `${value.commit || defaultMetadata.commit}`;
  const buildDate = `${value.build_date || value.buildDate || defaultMetadata.buildDate}`;
  const aptFingerprint = `${value.apt_fingerprint || value.aptFingerprint || defaultMetadata.aptFingerprint}`;
  const containerImage = `${value.container_image || value.containerImage || defaultMetadata.containerImage}`;

  return {
    siteUrl,
    releaseVersion,
    commit,
    shortCommit: commit.length > 10 ? commit.slice(0, 7) : commit,
    formattedBuildDate: formatBuildDate(buildDate),
    aptFingerprint,
    releaseHighlights: normalizeHighlights(value.release_highlights || value.releaseHighlights),
    githubUrl: value.github_url || value.githubUrl || defaultMetadata.githubUrl,
    releaseUrl: value.release_url || value.releaseUrl || defaultMetadata.releaseUrl,
    containerUrl: value.container_url || value.containerUrl || defaultMetadata.containerUrl,
    containerImage,
  };
}

function shouldRenderFetchedMetadata(metadata) {
  if (parseStableTag(defaultMetadata.releaseVersion) && !parseStableTag(metadata.releaseVersion)) {
    return false;
  }

  return compareStableTags(metadata.releaseVersion, defaultMetadata.releaseVersion) >= 0;
}

function mergeFetchedMetadata(metadata) {
  if (hasExplicitBuildHighlights && metadata.releaseVersion === defaultMetadata.releaseVersion) {
    return {
      ...metadata,
      releaseHighlights: defaultMetadata.releaseHighlights,
    };
  }

  return metadata;
}

function setText(id, value) {
  const element = document.getElementById(id);
  if (element) {
    element.textContent = value;
  }
}

function setHref(id, value) {
  const element = document.getElementById(id);
  if (element && value) {
    element.href = value;
  }
}

function releaseTag(metadata) {
  return parseStableTag(metadata.releaseVersion) ? metadata.releaseVersion : 'latest';
}

function renderCommands(metadata) {
  const keyPath = '/etc/apt/keyrings/blackhole-threats-archive-keyring.gpg';

  setText(
    'apt-command',
    `sudo install -d -m 0755 /etc/apt/keyrings
curl -fsSL ${metadata.siteUrl}blackhole-threats-archive-keyring.gpg \\
  | sudo tee ${keyPath} >/dev/null

curl -fsSL ${metadata.siteUrl}blackhole-threats-archive-keyring.fingerprint.txt

# Verify the fingerprint matches the expected archive key before proceeding.

sudo tee /etc/apt/sources.list.d/blackhole-threats.sources >/dev/null <<'EOF'
Types: deb deb-src
URIs: ${metadata.siteUrl}
Suites: stable
Components: main
Signed-By: ${keyPath}
EOF

sudo apt update
sudo apt install blackhole-threats`,
  );

  setText(
    'container-command',
    `docker pull ${metadata.containerImage}:${releaseTag(metadata)}
docker run -d \\
  -p 179:179 \\
  -v "$PWD/config:/config" \\
  --name blackhole-threats \\
  ${metadata.containerImage}:${releaseTag(metadata)}`,
  );

  setText(
    'apt-fingerprint-row',
    metadata.aptFingerprint ? `Archive fingerprint: ${metadata.aptFingerprint}` : '',
  );
}

function renderMetadata(metadata) {
  setText('release-version', metadata.releaseVersion);
  setText('release-commit', metadata.shortCommit);
  setText('release-date', metadata.formattedBuildDate);
  setText('release-fingerprint', metadata.aptFingerprint);
  setText('footer-version', metadata.releaseVersion);
  setText('footer-commit', metadata.shortCommit);

  setHref('site-home-link', metadata.siteUrl);
  setHref('nav-github-link', metadata.githubUrl);
  setHref('nav-releases-link', metadata.releaseUrl);
  setHref('nav-container-link', metadata.containerUrl);
  setHref('nav-apt-link', `${metadata.siteUrl}#install`);
  setHref('install-apt-link', `${metadata.siteUrl}#install`);
  setHref('install-container-link', metadata.containerUrl);
  setHref('install-release-link', metadata.releaseUrl);
  setHref('install-source-link', metadata.githubUrl);
  setHref('footer-release-link', metadata.releaseUrl);
  setHref('footer-container-link', metadata.containerUrl);
  setHref('footer-apt-link', `${metadata.siteUrl}#install`);

  const highlightsList = document.getElementById('release-highlights');
  if (highlightsList) {
    highlightsList.replaceChildren(
      ...metadata.releaseHighlights.map((highlight) => {
        const item = document.createElement('li');
        item.textContent = highlight;
        return item;
      }),
    );
  }

  renderCommands(metadata);
}

async function loadMetadata() {
  renderMetadata(normalizeMetadata(defaultMetadata));

  try {
    const response = await fetch('./website-metadata.json', { cache: 'no-store' });
    if (!response.ok) {
      return;
    }

    const normalizedMetadata = normalizeMetadata(await response.json());
    if (shouldRenderFetchedMetadata(normalizedMetadata)) {
      renderMetadata(mergeFetchedMetadata(normalizedMetadata));
    }
  } catch {
    // Keep the build-time defaults if the published metadata file is absent.
  }
}

function selectInstallTab(container, tabID) {
  container.querySelectorAll('.tab').forEach((tab) => {
    const active = tab.dataset.tab === tabID;
    tab.classList.toggle('active', active);
    tab.setAttribute('aria-selected', active ? 'true' : 'false');
  });

  container.querySelectorAll('.tab-panel').forEach((panel) => {
    panel.classList.toggle('active', panel.id === `panel-${tabID}`);
  });
}

function wireTabs() {
  document.querySelectorAll('.install-tabs').forEach((container) => {
    container.addEventListener('click', (event) => {
      const button = event.target.closest('.tab');
      if (!button) {
        return;
      }

      selectInstallTab(container, button.dataset.tab);
    });
  });

  document.querySelectorAll('[data-install-tab]').forEach((link) => {
    link.addEventListener('click', () => {
      const container = document.querySelector('.install-tabs');
      if (container) {
        selectInstallTab(container, link.dataset.installTab);
      }
    });
  });
}

function showCopiedFeedback(button) {
  const previous = button.textContent;
  button.classList.add('copied');
  button.textContent = 'Copied';
  setTimeout(() => {
    button.classList.remove('copied');
    button.textContent = previous;
  }, 1400);
}

function wireCopyButtons() {
  document.querySelectorAll('[data-copy-target]').forEach((button) => {
    button.addEventListener('click', async () => {
      const target = document.getElementById(button.dataset.copyTarget);
      if (!target) {
        return;
      }

      const text = target.textContent ?? '';

      try {
        if (navigator.clipboard?.writeText) {
          await navigator.clipboard.writeText(text);
        } else {
          const selection = window.getSelection();
          const range = document.createRange();
          range.selectNodeContents(target);
          selection?.removeAllRanges();
          selection?.addRange(range);
          document.execCommand('copy');
          selection?.removeAllRanges();
        }
        showCopiedFeedback(button);
      } catch {
        showCopiedFeedback(button);
      }
    });
  });
}

document.addEventListener('DOMContentLoaded', () => {
  wireTabs();
  wireCopyButtons();
  void loadMetadata();
});

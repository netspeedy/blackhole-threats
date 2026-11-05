# blackhole-threats website

This folder is the source of blackhole-threats' GitHub Pages site,
<https://netspeedy.github.io/blackhole-threats/>, which is both the landing page and
the front of the signed APT repository. It's a small Vite build rather than
documentation to read here; that lives in the main [README](../README.md) and in
[`docs/`](../docs/README.md).

## Working on it

Run these from the repository root:

```bash
make website-install   # npm ci, from package-lock.json
make website-dev       # local server with live reload
make website-check     # the page's required files, element ids and build-time values
make website-build     # production build into website/dist/
```

`make website-build` fills in the release version, commit, build date and APT key
fingerprint from your local git checkout unless you pass `SITE_RELEASE_VERSION`,
`SITE_COMMIT`, `SITE_BUILD_DATE` and `SITE_APT_FINGERPRINT`, which is what the
workflows do. CI builds with Node 22.

`public/og-image.png` and `public/og-image.svg` are the link-preview card. `make
readme-hero` generates them along with the README banner, so edit
`scripts/readme/build_readme_hero.py` rather than the files.

## How it's published

The Pages root is shared with the APT repository, so the site is always deployed
together with it and must not break the `dists/`, `pool/` or key download paths.
Publish Signed Debian Repository deploys the site at release time, with the new
packages. Deploy Pages Site runs on every push to `main` and after each of those
publish runs: it restores the last published APT snapshot, rebuilds the page with
the latest stable release's version and highlights, and deploys both. Dependabot
checks the npm dependencies here daily.
